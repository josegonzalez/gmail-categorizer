package web

// layout holds the fields every page's shared header uses.
type layout struct {
	Title    string
	Username string
}

// groupingsPage is the inbox index.
type groupingsPage struct {
	layout
	Filter       string
	AllURL       string
	BySenderURL  string
	ToggleURL    string
	RefreshQuery string
	Groupings    []groupingRow
	// Total and BySender count every cached grouping; Shown counts the
	// groupings that match the filter.
	Total    int
	BySender int
	Shown    int
	Messages int
	Flash    *flashView
}

type groupingRow struct {
	Address  string
	URL      string
	Count    int
	Folder   string
	BySender bool
	Subjects []subjectRow
}

type subjectRow struct {
	Subject string
	Date    string
	URL     string
}

type flashView struct {
	Archived string
	Failed   []string
	Missing  []string
}

// groupingPage lists the messages in one grouping.
type groupingPage struct {
	layout
	Address     string
	Count       int
	Folder      string
	BySender    bool
	GmailURL    string
	Sort        string
	SortOptions []sortOption
	NextSortURL string
	BackURL     string
	Messages    []messageRow
}

type sortOption struct {
	Key      string
	Label    string
	Selected bool
}

type messageRow struct {
	From    string
	Subject string
	Date    string
	URL     string
}

// messagePage shows one message.
type messagePage struct {
	layout
	Subject  string
	From     string
	To       string
	Date     string
	Address  string
	BackURL  string
	PrevURL  string
	NextURL  string
	Position int
	Total    int
	// Text is the plain-text body, preferred when present.
	Text string
	// HTMLDoc is the HTML body as a document for the sandboxed iframe.
	HTMLDoc string
}

type errorPage struct {
	layout
	Status  int
	Message string
}
