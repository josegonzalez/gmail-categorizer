package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// contentSecurityPolicy keeps the UI self-contained: no outside scripts,
// fonts or images, so remote images and tracking pixels in mail never load.
// Message HTML renders in a sandboxed srcdoc iframe, which inherits this
// policy; inline styles are allowed so mail keeps its formatting, and the
// sandbox blocks scripts.
const contentSecurityPolicy = "default-src 'none'; " +
	"script-src 'self'; " +
	"connect-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self'; " +
	"frame-src 'self'; " +
	"form-action 'self'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'"

// Options configures a Server.
type Options struct {
	// Username is the Gmail account, shown in the UI and used for Gmail links.
	Username string
	// Now returns the current time, used to format dates. Defaults to time.Now.
	Now func() time.Time
	// Logger receives one line per request and any request errors. Defaults
	// to the standard logger, which writes to stderr.
	Logger *log.Logger
}

// Server serves the web UI. One mutex serializes every request that touches
// the IMAP session or the cached groupings, because the session is a single
// IMAP connection and groupings are shared state.
type Server struct {
	mu        sync.Mutex
	session   *Session
	groupings []*triage.Grouping
	flash     *flash
	opts      Options
	pages     map[string]*template.Template
}

// flash is a one-time notice shown on the next groupings page.
type flash struct {
	Archived      []triage.BatchResult
	Failed        []triage.BatchResult
	ArchivedCount int
	Missing       []string
}

// NewServer creates a server over session.
func NewServer(session *Session, opts Options) (*Server, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}

	pages := make(map[string]*template.Template)
	for _, page := range []string{"groupings", "grouping", "message", "error"} {
		t, err := template.New("layout.html").ParseFS(templateFS, "templates/layout.html", "templates/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("parsing %s template: %w", page, err)
		}
		pages[page] = t
	}

	return &Server{session: session, opts: opts, pages: pages}, nil
}

// Load connects if needed and loads the inbox groupings.
func (s *Server) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reload(ctx)
}

// reload fetches groupings from the inbox. The caller holds s.mu.
func (s *Server) reload(ctx context.Context) error {
	if _, err := s.session.Ensure(ctx); err != nil {
		return err
	}
	groupings, err := s.session.Triager().LoadGroupings(ctx)
	if err != nil {
		return err
	}
	s.groupings = groupings
	return nil
}

// ensure checks the connection, reloading groupings when a reconnect made the
// cached UIDs invalid. The caller holds s.mu.
func (s *Server) ensure(ctx context.Context) error {
	invalidated, err := s.session.Ensure(ctx)
	if err != nil {
		return err
	}
	if invalidated || s.groupings == nil {
		return s.reload(ctx)
	}
	return nil
}

// Handler returns the HTTP handler for the UI.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // the embedded directory always exists
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	mux.HandleFunc("GET /{$}", s.locked(s.handleGroupings))
	mux.HandleFunc("POST /refresh", s.locked(s.handleRefresh))
	mux.HandleFunc("POST /archive", s.locked(s.handleArchive))
	mux.HandleFunc("GET /groupings/{address}", s.locked(s.handleGrouping))
	mux.HandleFunc("GET /groupings/{address}/messages/{uid}", s.locked(s.handleMessage))
	// The catch-all below would otherwise answer other methods on the POST
	// routes with a 404.
	for _, path := range []string{"/refresh", "/archive"} {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Allow", http.MethodPost)
			s.renderError(w, http.StatusMethodNotAllowed, "This page only accepts form submissions.")
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.renderError(w, http.StatusNotFound, "Page not found.")
	})

	// Logging is outermost so rejected cross-origin requests are logged too.
	return logRequests(s.opts.Logger, securityHeaders(http.NewCrossOriginProtection().Handler(mux)))
}

// locked runs h while holding the server mutex.
func (s *Server) locked(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		h(w, r)
	}
}

// securityHeaders sets headers shared by every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// Filter values for the groupings page.
const (
	filterAll      = "all"
	filterBySender = "by-sender"
)

func (s *Server) handleGroupings(w http.ResponseWriter, r *http.Request) {
	if err := s.ensure(r.Context()); err != nil {
		s.serverError(w, err)
		return
	}

	filter := filterAll
	if r.URL.Query().Get("filter") == filterBySender {
		filter = filterBySender
	}

	page := groupingsPage{
		layout:       s.layout("Inbox"),
		Filter:       filter,
		AllURL:       "/",
		BySenderURL:  "/?filter=" + filterBySender,
		ToggleURL:    "/?filter=" + filterBySender,
		Total:        len(s.groupings),
		Flash:        s.takeFlash(),
		RefreshQuery: filterQuery(filter),
	}
	if filter == filterBySender {
		page.ToggleURL = "/"
	}

	now := s.opts.Now()
	for _, g := range s.groupings {
		page.Messages += g.Count
		if g.GroupedByFrom {
			page.BySender++
		}
		if filter == filterBySender && !g.GroupedByFrom {
			continue
		}
		row := groupingRow{
			Address:  g.Address,
			URL:      groupingURL(g.Address),
			Count:    g.Count,
			Folder:   g.DestinationFolder(),
			BySender: g.GroupedByFrom,
		}
		for _, msg := range g.Messages {
			row.Subjects = append(row.Subjects, subjectRow{
				Subject: subjectOrPlaceholder(msg.Subject),
				Date:    triage.ShortDate(msg.Date, now),
				URL:     messageURL(g.Address, msg.UID, triage.SortDateDesc),
			})
		}
		page.Groupings = append(page.Groupings, row)
	}
	page.Shown = len(page.Groupings)

	s.render(w, http.StatusOK, "groupings", page)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if err := s.reload(r.Context()); err != nil {
		s.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/"+filterQuery(r.FormValue("filter")), http.StatusSeeOther)
}

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "Could not read the archive request.")
		return
	}
	if err := s.ensure(r.Context()); err != nil {
		s.serverError(w, err)
		return
	}

	var targets []*triage.Grouping
	var missing []string
	seen := make(map[string]bool)
	for _, address := range r.PostForm["address"] {
		if seen[address] {
			continue
		}
		seen[address] = true
		if g := s.findGrouping(address); g != nil {
			targets = append(targets, g)
		} else {
			missing = append(missing, address)
		}
	}

	redirect := "/" + filterQuery(r.PostForm.Get("filter"))

	// With JavaScript the page asks for progress events instead of a
	// redirect, and shows them as a bar across the top of the page.
	var stream *progressStream
	if wantsProgress(r) {
		stream = newProgressStream(w)
	}

	f := &flash{Missing: missing}
	if len(targets) > 0 {
		var onProgress triage.BatchProgressFunc
		if stream != nil {
			onProgress = stream.progress
		}
		// Finish an archive even if the browser goes away: stopping part way
		// through a move can leave messages copied but still in the inbox.
		results := triage.ArchiveBatch(context.WithoutCancel(r.Context()), s.session.Triager(), targets, onProgress)
		for _, res := range results {
			if res.Err != nil {
				s.opts.Logger.Printf("archiving %s: %v", res.Address, res.Err)
				f.Failed = append(f.Failed, res)
				continue
			}
			f.Archived = append(f.Archived, res)
			if res.Result != nil {
				f.ArchivedCount += res.Result.ArchivedCount
			}
		}
		// Archive leaves each grouping holding only the messages still in
		// the inbox, so fully archived groupings are now empty.
		s.removeEmptyGroupings()
	}
	if len(f.Archived) > 0 || len(f.Failed) > 0 || len(f.Missing) > 0 {
		s.flash = f
	}

	if stream != nil {
		stream.done(redirect)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleGrouping(w http.ResponseWriter, r *http.Request) {
	if err := s.ensure(r.Context()); err != nil {
		s.serverError(w, err)
		return
	}

	g := s.findGrouping(r.PathValue("address"))
	if g == nil {
		s.renderError(w, http.StatusNotFound, "That grouping is not in the inbox. It may have been archived.")
		return
	}
	if err := s.session.Triager().LoadMessages(r.Context(), g); err != nil {
		s.serverError(w, err)
		return
	}

	mode := triage.ParseSortMode(r.URL.Query().Get("sort"))
	now := s.opts.Now()
	page := groupingPage{
		layout:      s.layout(g.Address),
		Address:     g.Address,
		Count:       g.Count,
		Folder:      g.DestinationFolder(),
		BySender:    g.GroupedByFrom,
		GmailURL:    g.GmailSearchURL(s.opts.Username),
		Sort:        mode.Key(),
		NextSortURL: groupingURL(g.Address) + sortQuery(mode.Next()),
		BackURL:     "/",
	}
	for _, m := range []triage.SortMode{triage.SortDateDesc, triage.SortDateAsc, triage.SortSubjectAsc, triage.SortSubjectDesc} {
		page.SortOptions = append(page.SortOptions, sortOption{Key: m.Key(), Label: sortLabel(m), Selected: m == mode})
	}
	for _, msg := range sortedMessages(g, mode) {
		page.Messages = append(page.Messages, messageRow{
			From:    senderName(msg),
			Subject: subjectOrPlaceholder(msg.Subject),
			Date:    triage.ShortDate(msg.Date, now),
			URL:     messageURL(g.Address, msg.UID, mode),
		})
	}

	s.render(w, http.StatusOK, "grouping", page)
}

func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	if err := s.ensure(r.Context()); err != nil {
		s.serverError(w, err)
		return
	}

	g := s.findGrouping(r.PathValue("address"))
	uid, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if g == nil || err != nil {
		s.renderError(w, http.StatusNotFound, "That message is not in the inbox. It may have been archived.")
		return
	}
	if err := s.session.Triager().LoadMessages(r.Context(), g); err != nil {
		s.serverError(w, err)
		return
	}

	mode := triage.ParseSortMode(r.URL.Query().Get("sort"))
	messages := sortedMessages(g, mode)
	idx := -1
	for i, msg := range messages {
		if msg.UID == uint32(uid) {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.renderError(w, http.StatusNotFound, "That message is not in the inbox. It may have been archived.")
		return
	}
	msg := messages[idx]

	body, err := s.session.Triager().LoadMessageBody(r.Context(), g, msg.UID)
	if err != nil {
		s.serverError(w, err)
		return
	}

	page := messagePage{
		layout:   s.layout(subjectOrPlaceholder(msg.Subject)),
		Subject:  subjectOrPlaceholder(msg.Subject),
		From:     imap.FormatAddresses(msg.From),
		To:       imap.FormatAddresses(msg.To),
		Address:  g.Address,
		BackURL:  groupingURL(g.Address) + sortQuery(mode),
		Position: idx + 1,
		Total:    len(messages),
	}
	if !msg.Date.IsZero() {
		page.Date = msg.Date.In(s.opts.Now().Location()).Format("Mon, Jan 2 2006, 3:04 PM")
	}
	if idx > 0 {
		page.PrevURL = messageURL(g.Address, messages[idx-1].UID, mode)
	}
	if idx < len(messages)-1 {
		page.NextURL = messageURL(g.Address, messages[idx+1].UID, mode)
	}
	if strings.TrimSpace(body.Text) != "" {
		page.Text = body.Text
	} else if body.HTML != "" {
		page.HTMLDoc = htmlPreviewDoc(body.HTML)
	}

	s.render(w, http.StatusOK, "message", page)
}

// htmlPreviewDoc wraps message HTML for the sandboxed preview iframe. Links
// open in a new tab rather than inside the small preview frame.
func htmlPreviewDoc(body string) string {
	return `<!doctype html><meta charset="utf-8"><base target="_blank">` + body
}

// findGrouping returns the cached grouping for address. The caller holds s.mu.
func (s *Server) findGrouping(address string) *triage.Grouping {
	for _, g := range s.groupings {
		if g.Address == address {
			return g
		}
	}
	return nil
}

// removeEmptyGroupings drops groupings with nothing left in the inbox from
// the cache. The caller holds s.mu.
func (s *Server) removeEmptyGroupings() {
	kept := s.groupings[:0]
	for _, g := range s.groupings {
		if g.Count > 0 {
			kept = append(kept, g)
		}
	}
	s.groupings = kept
}

// takeFlash returns and clears the pending flash. The caller holds s.mu.
func (s *Server) takeFlash() *flashView {
	f := s.flash
	s.flash = nil
	if f == nil {
		return nil
	}

	v := &flashView{Missing: f.Missing}
	if n := len(f.Archived); n > 0 {
		folders := make([]string, 0, n)
		for _, res := range f.Archived {
			if res.Result != nil {
				folders = append(folders, res.Result.DestinationFolder)
			}
		}
		v.Archived = fmt.Sprintf("Archived %s (%s) to %s",
			plural(n, "grouping", "groupings"),
			plural(f.ArchivedCount, "message", "messages"),
			strings.Join(folders, ", "))
	}
	for _, res := range f.Failed {
		msg := fmt.Sprintf("%s: %v", res.Address, res.Err)
		if res.Result != nil && res.Result.ArchivedCount > 0 {
			msg += fmt.Sprintf(" (%s moved to %s before the error)",
				plural(res.Result.ArchivedCount, "message was", "messages were"), res.Result.DestinationFolder)
		}
		v.Failed = append(v.Failed, msg)
	}
	return v
}

func (s *Server) layout(title string) layout {
	return layout{Title: title, Username: s.opts.Username}
}

// render executes a page template, buffering so a template error does not
// leave a half-written page.
func (s *Server) render(w http.ResponseWriter, status int, page string, data any) {
	var buf strings.Builder
	if err := s.pages[page].Execute(&buf, data); err != nil {
		s.opts.Logger.Printf("rendering %s: %v", page, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(buf.String()))
}

func (s *Server) renderError(w http.ResponseWriter, status int, message string) {
	s.render(w, status, "error", errorPage{
		layout:  s.layout(http.StatusText(status)),
		Status:  status,
		Message: message,
	})
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	s.opts.Logger.Printf("request failed: %v", err)
	status := http.StatusBadGateway
	if errors.Is(err, context.Canceled) {
		status = http.StatusServiceUnavailable
	}
	s.renderError(w, status, "The mail server request failed: "+err.Error())
}

// sortedMessages returns the grouping's messages in mode order without
// changing the cached order, which stays newest first for the groupings page.
func sortedMessages(g *triage.Grouping, mode triage.SortMode) []*imap.Message {
	messages := make([]*imap.Message, len(g.Messages))
	copy(messages, g.Messages)
	triage.SortMessages(messages, mode)
	return messages
}

func groupingURL(address string) string {
	return "/groupings/" + url.PathEscape(address)
}

func messageURL(address string, uid uint32, mode triage.SortMode) string {
	return groupingURL(address) + "/messages/" + strconv.FormatUint(uint64(uid), 10) + sortQuery(mode)
}

func sortQuery(mode triage.SortMode) string {
	if mode == triage.SortDateDesc {
		return ""
	}
	return "?sort=" + mode.Key()
}

func filterQuery(filter string) string {
	if filter == filterBySender {
		return "?filter=" + filterBySender
	}
	return ""
}

func sortLabel(mode triage.SortMode) string {
	switch mode {
	case triage.SortDateAsc:
		return "Oldest first"
	case triage.SortSubjectAsc:
		return "Subject A to Z"
	case triage.SortSubjectDesc:
		return "Subject Z to A"
	default:
		return "Newest first"
	}
}

func subjectOrPlaceholder(subject string) string {
	if strings.TrimSpace(subject) == "" {
		return "(no subject)"
	}
	return subject
}

// senderName returns the first sender's display name, or address.
func senderName(msg *imap.Message) string {
	if len(msg.From) == 0 {
		return "(unknown sender)"
	}
	if msg.From[0].Name != "" {
		return msg.From[0].Name
	}
	return msg.From[0].String()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
