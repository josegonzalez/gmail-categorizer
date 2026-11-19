package imap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFixture(t *testing.T, name string) *MessageBody {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	defer f.Close()

	body, err := parseBody(f)
	require.NoError(t, err)
	require.NotNil(t, body)
	return body
}

func TestParseBody_Plain(t *testing.T) {
	body := parseFixture(t, "plain.eml")

	assert.Contains(t, body.Text, "Build #4821 passed.")
	assert.Empty(t, body.HTML)
	assert.Equal(t, body.Text, body.PlainText())
}

func TestParseBody_HTMLOnly(t *testing.T) {
	body := parseFixture(t, "html.eml")

	assert.Empty(t, body.Text)
	assert.Contains(t, body.HTML, "<h1>This week</h1>")

	text := body.PlainText()
	assert.Contains(t, text, "This week")
	assert.Contains(t, text, "Tools & tips")
	assert.NotContains(t, text, "alert")
	assert.NotContains(t, text, "color:red")
	assert.NotContains(t, text, "Ignored")
}

func TestParseBody_MultipartAlternative(t *testing.T) {
	body := parseFixture(t, "alternative.eml")

	assert.Equal(t, "Your invoice is ready.", strings.TrimSpace(body.Text))
	assert.Contains(t, body.HTML, "<b>invoice</b>")
	assert.Equal(t, body.Text, body.PlainText(), "plain part is preferred")
}

func TestParseBody_NonUTF8Charset(t *testing.T) {
	body := parseFixture(t, "latin1.eml")

	assert.Contains(t, body.Text, "café")
}

func TestParseBody_TruncatedKeepsEarlierParts(t *testing.T) {
	body := parseFixture(t, "truncated.eml")

	assert.Contains(t, body.Text, "Your order shipped.")
}

func TestParseBody_Garbage(t *testing.T) {
	_, err := parseBody(strings.NewReader("not a message"))
	assert.Error(t, err)
}

func TestMessageBody_PlainText_Nil(t *testing.T) {
	var body *MessageBody
	assert.Equal(t, "", body.PlainText())
}

func TestMessageBody_PlainText_BlankTextFallsBackToHTML(t *testing.T) {
	body := &MessageBody{Text: "  \r\n", HTML: "<p>Hello</p>"}
	assert.Equal(t, "Hello", body.PlainText())
}

func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"paragraphs become lines", "<p>One</p><p>Two</p>", "One\n\nTwo"},
		{"br breaks lines", "a<br>b<br/>c", "a\nb\nc"},
		{"entities decoded", "Fish &amp; chips &lt;3 &nbsp;ok", "Fish & chips <3 ok"},
		{"script and style dropped", "<style>.x{}</style><script>var a;</script>Hi", "Hi"},
		{"whitespace collapsed", "<div>  lots   of\n\tspace </div>", "lots of space"},
		{"table cells spaced", "<table><tr><td>A</td><td>B</td></tr></table>", "A\n\nB"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HTMLToText(tt.in))
		})
	}
}

func TestGmailClient_FetchMessageBody_NotConnected(t *testing.T) {
	client := &gmailClient{}
	_, err := client.FetchMessageBody(t.Context(), 1)
	assert.Error(t, err)
}

func TestGmailClient_Noop_NotConnected(t *testing.T) {
	client := &gmailClient{}
	assert.Error(t, client.Noop(t.Context()))
}
