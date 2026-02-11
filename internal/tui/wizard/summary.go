package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/tui/components"
)

// SummaryModel shows the final results screen.
type SummaryModel struct {
	styles  components.Styles
	results []module.ModuleResult
	err     error // runner-level error
	width   int
	height  int
}

// NewSummaryModel creates a summary view.
func NewSummaryModel(styles components.Styles) SummaryModel {
	return SummaryModel{styles: styles}
}

// SetResults updates the results to display.
func (m SummaryModel) SetResults(results []module.ModuleResult) SummaryModel {
	m.results = results
	return m
}

// SetError sets a runner-level error.
func (m SummaryModel) SetError(err error) SummaryModel {
	m.err = err
	return m
}

// HasError returns true if any module failed or there was a runner error.
func (m SummaryModel) HasError() bool {
	if m.err != nil {
		return true
	}
	for _, r := range m.results {
		if r.Err != nil {
			return true
		}
	}
	return false
}

// Init satisfies tea.Model.
func (m SummaryModel) Init() tea.Cmd {
	return nil
}

// Update handles key events.
func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the summary screen.
func (m SummaryModel) View() string {
	var b strings.Builder

	b.WriteString(components.RenderBanner(m.styles))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(m.styles.Error.Render("Setup Failed"))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Warning.Render("  Fix the issue and re-run — completed steps will be skipped."))
		b.WriteString("\n\n")
	} else if m.HasError() {
		b.WriteString(m.styles.Error.Render("Setup Failed"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(m.styles.Success.Render("Setup Complete!"))
		b.WriteString("\n\n")
	}

	// Per-module results.
	totalCompleted := 0
	totalSkipped := 0
	totalSteps := 0

	for _, r := range m.results {
		totalCompleted += r.Completed
		totalSkipped += r.Skipped
		totalSteps += r.Total

		status := m.styles.Success.Render("done")
		if r.Err != nil {
			status = m.styles.Error.Render(fmt.Sprintf("FAILED at %q", r.FailedStep))
		}

		b.WriteString(fmt.Sprintf("  %s: %s (%d completed, %d skipped)\n",
			r.ModuleID, status, r.Completed, r.Skipped))

		if r.Err != nil {
			b.WriteString(m.styles.Error.Render(fmt.Sprintf("    Error: %v", r.Err)))
			b.WriteString("\n")
		}
	}

	if len(m.results) > 0 {
		b.WriteString(fmt.Sprintf("\n  Total: %d steps (%d completed, %d skipped)\n",
			totalSteps, totalCompleted, totalSkipped))
	}

	if m.HasError() {
		b.WriteString("\n")
		b.WriteString(m.styles.Warning.Render("  Fix the issue and re-run — completed steps will be skipped."))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Footer.Render("  Press enter or q to exit"))

	return b.String()
}
