package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestColors(t *testing.T) {
	tests := []struct {
		name  string
		color lipgloss.Color
	}{
		{"PrimaryColor", PrimaryColor},
		{"SecondaryColor", SecondaryColor},
		{"SuccessColor", SuccessColor},
		{"WarningColor", WarningColor},
		{"ErrorColor", ErrorColor},
		{"MutedColor", MutedColor},
		{"BgColor", BgColor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.color), "color %s should be defined", tt.name)
		})
	}
}

func TestStyles(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"TitleStyle", TitleStyle},
		{"SubtitleStyle", SubtitleStyle},
		{"SelectedStyle", SelectedStyle},
		{"NormalStyle", NormalStyle},
		{"CountStyle", CountStyle},
		{"HelpStyle", HelpStyle},
		{"SuccessStyle", SuccessStyle},
		{"ErrorStyle", ErrorStyle},
		{"WarningStyle", WarningStyle},
		{"BoxStyle", BoxStyle},
		{"WarningBoxStyle", WarningBoxStyle},
		{"SpinnerStyle", SpinnerStyle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that rendering doesn't panic
			result := tt.style.Render("test content")
			assert.NotEmpty(t, result, "style %s should render content", tt.name)
		})
	}
}

func TestStylesRenderCorrectly(t *testing.T) {
	// Test that styles apply correctly
	text := "Hello World"

	// TitleStyle should be bold
	titleResult := TitleStyle.Render(text)
	assert.Contains(t, titleResult, text)

	// SelectedStyle with padding
	selectedResult := SelectedStyle.Render(text)
	assert.Contains(t, selectedResult, text)

	// BoxStyle with border
	boxResult := BoxStyle.Render(text)
	assert.Contains(t, boxResult, text)
}
