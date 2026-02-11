package components

import "github.com/charmbracelet/lipgloss"

// Styles holds all shared Lipgloss styles used across TUI screens.
type Styles struct {
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	Body           lipgloss.Style
	Muted          lipgloss.Style
	Success        lipgloss.Style
	Error          lipgloss.Style
	Warning        lipgloss.Style
	Panel          lipgloss.Style
	SelectedItem   lipgloss.Style
	UnselectedItem lipgloss.Style
	CheckboxOn     string
	CheckboxOff    string
	StatusDone     string
	StatusRunning  string
	StatusPending  string
	StatusSkipped  string
	StatusFailed   string
	Footer         lipgloss.Style
	AccentColor    lipgloss.AdaptiveColor
	ProgressFull   lipgloss.Style
	ProgressEmpty  lipgloss.Style
}

// DefaultStyles returns a Styles populated with the shhh color palette.
// Uses AdaptiveColor to work in both light and dark terminals.
func DefaultStyles() Styles {
	accent := lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: "#B476F0"}
	cyan := lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"}
	muted := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	success := lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}
	errColor := lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}
	warn := lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}

	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		Subtitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan),

		Body: lipgloss.NewStyle(),

		Muted: lipgloss.NewStyle().
			Foreground(muted),

		Success: lipgloss.NewStyle().
			Foreground(success),

		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(errColor),

		Warning: lipgloss.NewStyle().
			Foreground(warn),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1),

		SelectedItem: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		UnselectedItem: lipgloss.NewStyle().
			Foreground(muted),

		CheckboxOn:    "[x]",
		CheckboxOff:   "[ ]",
		StatusDone:    "✓",
		StatusRunning: "●",
		StatusPending: "○",
		StatusSkipped: "~",
		StatusFailed:  "✗",

		Footer: lipgloss.NewStyle().
			Foreground(muted),

		AccentColor: accent,

		ProgressFull: lipgloss.NewStyle().
			Foreground(accent),

		ProgressEmpty: lipgloss.NewStyle().
			Foreground(muted),
	}
}
