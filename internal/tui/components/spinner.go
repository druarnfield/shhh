package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// NewSpinner returns a spinner.Model pre-configured with shhh accent styling.
func NewSpinner(styles Styles) spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.AccentColor)
	return s
}
