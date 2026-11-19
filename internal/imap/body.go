package imap

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/emersion/go-message"
	// Register decoders for non-UTF-8 charsets used by mail parts.
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
)

// MaxBodySize caps how many bytes of a message are fetched for previews.
// Larger messages are usually large because of attachments, which previews
// do not show.
const MaxBodySize = 2 << 20

// MessageBody holds the readable parts of a message.
type MessageBody struct {
	// Text is the first text/plain part, if any.
	Text string
	// HTML is the first text/html part, if any.
	HTML string
}

// PlainText returns the message as plain text: the text/plain part when there
// is one, otherwise the HTML part converted to readable text.
func (b *MessageBody) PlainText() string {
	if b == nil {
		return ""
	}
	if strings.TrimSpace(b.Text) != "" {
		return b.Text
	}
	if b.HTML != "" {
		return HTMLToText(b.HTML)
	}
	return ""
}

// parseBody extracts the first plain-text and HTML parts from a raw RFC 5322
// message. Parsing is best effort: a message cut short by MaxBodySize keeps
// whatever parts were read before the cut.
func parseBody(r io.Reader) (*MessageBody, error) {
	mr, err := mail.CreateReader(r)
	if err != nil && !message.IsUnknownCharset(err) {
		return nil, fmt.Errorf("parsing message: %w", err)
	}

	body := &MessageBody{}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !message.IsUnknownCharset(err) {
			// An unknown charset still returns a usable part, read as-is.
			if body.Text != "" || body.HTML != "" {
				// Keep what was read before the message was cut short.
				break
			}
			return nil, fmt.Errorf("reading message part: %w", err)
		}

		inline, ok := part.Header.(*mail.InlineHeader)
		if !ok {
			continue
		}
		mediaType, _, err := inline.ContentType()
		if err != nil {
			mediaType = "text/plain"
		}

		switch {
		case mediaType == "text/plain" && body.Text == "":
			body.Text = readPart(part.Body)
		case mediaType == "text/html" && body.HTML == "":
			body.HTML = readPart(part.Body)
		}
	}

	return body, nil
}

// readPart reads a part body, keeping whatever was read if it ends early.
func readPart(r io.Reader) string {
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// blockElements end a line of text when converting HTML.
var blockElements = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true,
	"br": true, "dd": true, "div": true, "dl": true, "dt": true,
	"footer": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"h5": true, "h6": true, "header": true, "hr": true, "li": true,
	"main": true, "nav": true, "ol": true, "p": true, "pre": true,
	"section": true, "table": true, "td": true, "th": true, "tr": true,
	"ul": true,
}

// skippedElements hold content that is never shown as text.
var skippedElements = map[string]bool{
	"head": true, "script": true, "style": true, "template": true, "title": true,
}

var (
	spaceRun   = regexp.MustCompile(`[ \t\f\r\x{00a0}]+`)
	newlineRun = regexp.MustCompile(`\n{3,}`)
)

// HTMLToText converts an HTML document to readable plain text: block elements
// end lines, scripts and styles are dropped, entities are decoded and runs of
// whitespace are collapsed.
func HTMLToText(s string) string {
	z := html.NewTokenizer(strings.NewReader(s))
	var b strings.Builder
	skipDepth := 0

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return cleanText(b.String())
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if skippedElements[tag] && tt == html.StartTagToken {
				skipDepth++
				continue
			}
			if blockElements[tag] {
				b.WriteString("\n")
			}
			if tag == "td" || tag == "th" {
				b.WriteString(" ")
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if skippedElements[tag] {
				if skipDepth > 0 {
					skipDepth--
				}
				continue
			}
			if blockElements[tag] {
				b.WriteString("\n")
			}
		case html.TextToken:
			if skipDepth == 0 {
				// Text() returns the text with entities already decoded.
				b.WriteString(strings.ReplaceAll(string(z.Text()), "\n", " "))
			}
		}
	}
}

// cleanText collapses whitespace left over from HTML conversion.
func cleanText(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(spaceRun.ReplaceAllString(line, " "))
	}
	out := strings.Join(lines, "\n")
	out = newlineRun.ReplaceAllString(out, "\n\n")
	return strings.TrimSpace(out)
}
