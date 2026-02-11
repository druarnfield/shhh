package wizard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/druarnfield/shhh/internal/tui/components"
)

// ExplainPanel renders a bordered "What's happening" panel with word-wrapped text.
type ExplainPanel struct {
	styles  components.Styles
	text    string
	visible bool
	width   int
}

// NewExplainPanel creates an explain panel (hidden by default).
func NewExplainPanel(styles components.Styles) ExplainPanel {
	return ExplainPanel{
		styles: styles,
		width:  60,
	}
}

// SetText returns a copy with updated text.
func (p ExplainPanel) SetText(text string) ExplainPanel {
	p.text = text
	return p
}

// SetVisible returns a copy with updated visibility.
func (p ExplainPanel) SetVisible(v bool) ExplainPanel {
	p.visible = v
	return p
}

// SetWidth returns a copy with updated width.
func (p ExplainPanel) SetWidth(w int) ExplainPanel {
	if w > 0 {
		p.width = w
	}
	return p
}

// View renders the panel. Returns empty string when not visible.
func (p ExplainPanel) View() string {
	if !p.visible || p.text == "" {
		return ""
	}

	// Account for panel border + padding (2 border + 2 padding = 4 chars).
	innerWidth := p.width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	wrapped := wordWrap(p.text, innerWidth)

	panel := p.styles.Panel.
		Width(p.width).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				p.styles.Subtitle.Render("What's happening"),
				"",
				wrapped,
			),
		)

	return panel
}

// wordWrap breaks text into lines that fit within the given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	line := words[0]

	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	lines = append(lines, line)

	return strings.Join(lines, "\n")
}
