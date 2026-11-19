package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	const url = "https://mail.google.com/mail/#search/x"

	tests := []struct {
		goos string
		name string
		args []string
	}{
		{"darwin", "open", []string{url}},
		{"linux", "xdg-open", []string{url}},
		{"freebsd", "xdg-open", []string{url}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", url}},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, args, err := command(tt.goos, url)
			require.NoError(t, err)
			assert.Equal(t, tt.name, name)
			assert.Equal(t, tt.args, args)
		})
	}
}

func TestCommand_Unsupported(t *testing.T) {
	_, _, err := command("plan9", "https://example.com")
	assert.Error(t, err)
}
