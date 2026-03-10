package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Select", km.Select},
		{"Archive", km.Archive},
		{"Toggle", km.Toggle},
		{"Sort", km.Sort},
		{"Filter", km.Filter},
		{"Back", km.Back},
		{"Quit", km.Quit},
		{"Confirm", km.Confirm},
		{"Cancel", km.Cancel},
		{"Help", km.Help},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := tt.binding.Keys()
			assert.NotEmpty(t, keys, "binding %s should have keys defined", tt.name)
		})
	}
}

func TestDefaultKeyMap_SpecificKeys(t *testing.T) {
	km := DefaultKeyMap()

	// Verify specific key assignments
	assert.Contains(t, km.Up.Keys(), "up")
	assert.Contains(t, km.Up.Keys(), "k")

	assert.Contains(t, km.Down.Keys(), "down")
	assert.Contains(t, km.Down.Keys(), "j")

	assert.Contains(t, km.Select.Keys(), "enter")

	assert.Contains(t, km.Archive.Keys(), "a")

	assert.Contains(t, km.Toggle.Keys(), " ")

	assert.Contains(t, km.Sort.Keys(), "s")

	assert.Contains(t, km.Filter.Keys(), "f")

	assert.Contains(t, km.Back.Keys(), "esc")
	assert.Contains(t, km.Back.Keys(), "backspace")

	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")

	assert.Contains(t, km.Confirm.Keys(), "y")

	assert.Contains(t, km.Cancel.Keys(), "n")
	assert.Contains(t, km.Cancel.Keys(), "esc")

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	shortHelp := km.ShortHelp()

	assert.Equal(t, 9, len(shortHelp), "ShortHelp should return 9 bindings")

	// Verify all expected bindings are included
	expectedBindings := []key.Binding{km.Up, km.Down, km.Select, km.Archive, km.Toggle, km.Sort, km.Filter, km.Back, km.Quit}
	for i, binding := range expectedBindings {
		assert.Equal(t, binding.Keys(), shortHelp[i].Keys())
	}
}

func TestKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap()
	fullHelp := km.FullHelp()

	assert.Equal(t, 3, len(fullHelp), "FullHelp should return 3 groups")

	// First group: navigation
	assert.Equal(t, 2, len(fullHelp[0]))
	assert.Equal(t, km.Up.Keys(), fullHelp[0][0].Keys())
	assert.Equal(t, km.Down.Keys(), fullHelp[0][1].Keys())

	// Second group: actions
	assert.Equal(t, 6, len(fullHelp[1]))
	assert.Equal(t, km.Select.Keys(), fullHelp[1][0].Keys())
	assert.Equal(t, km.Archive.Keys(), fullHelp[1][1].Keys())
	assert.Equal(t, km.Toggle.Keys(), fullHelp[1][2].Keys())
	assert.Equal(t, km.Sort.Keys(), fullHelp[1][3].Keys())
	assert.Equal(t, km.Filter.Keys(), fullHelp[1][4].Keys())
	assert.Equal(t, km.Back.Keys(), fullHelp[1][5].Keys())

	// Third group: help and quit
	assert.Equal(t, 2, len(fullHelp[2]))
	assert.Equal(t, km.Help.Keys(), fullHelp[2][0].Keys())
	assert.Equal(t, km.Quit.Keys(), fullHelp[2][1].Keys())
}

func TestKeyMap_HelpText(t *testing.T) {
	km := DefaultKeyMap()

	// Verify help text is present
	assert.NotEmpty(t, km.Up.Help().Key)
	assert.NotEmpty(t, km.Up.Help().Desc)

	assert.NotEmpty(t, km.Down.Help().Key)
	assert.NotEmpty(t, km.Down.Help().Desc)

	assert.NotEmpty(t, km.Select.Help().Key)
	assert.NotEmpty(t, km.Select.Help().Desc)

	assert.Equal(t, "space", km.Toggle.Help().Key)
	assert.Equal(t, "toggle", km.Toggle.Help().Desc)
}
