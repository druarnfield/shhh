package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/druarnfield/shhh/internal/tui/components"
)

type stepState int

const (
	stepPending stepState = iota
	stepRunning
	stepDone
	stepSkipped
	stepFailed
)

type stepStatus struct {
	name    string
	explain string
	state   stepState
	err     error
}

// ProgressModel shows module execution progress.
type ProgressModel struct {
	styles      components.Styles
	spinner     spinner.Model
	explain     ExplainPanel
	showExplain bool

	currentModule string
	steps         []stepStatus
	currentStep   int
	overallDone   int
	overallTotal  int
	width         int
	height        int
}

// NewProgressModel creates a progress view.
func NewProgressModel(styles components.Styles, showExplain bool) ProgressModel {
	return ProgressModel{
		styles:      styles,
		spinner:     components.NewSpinner(styles),
		explain:     NewExplainPanel(styles),
		showExplain: showExplain,
	}
}

// Init starts the spinner.
func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages.
func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "?" {
			m.showExplain = !m.showExplain
			m.explain = m.explain.SetVisible(m.showExplain)
		}

	case ModuleStartMsg:
		m.currentModule = msg.Name
		m.steps = make([]stepStatus, len(msg.Steps))
		for i, s := range msg.Steps {
			m.steps[i] = stepStatus{name: s.Name, explain: s.Explain}
		}
		m.currentStep = 0

	case StepStartMsg:
		if msg.Index < len(m.steps) {
			m.steps[msg.Index].state = stepRunning
			m.currentStep = msg.Index
			m.explain = m.explain.SetText(msg.Explain).SetVisible(m.showExplain)
		}

	case StepDoneMsg:
		if msg.Index < len(m.steps) {
			if msg.Skipped {
				m.steps[msg.Index].state = stepSkipped
			} else {
				m.steps[msg.Index].state = stepDone
			}
			m.overallDone++
		}

	case StepErrorMsg:
		if msg.Index < len(m.steps) {
			m.steps[msg.Index].state = stepFailed
			m.steps[msg.Index].err = msg.Err
			m.overallDone++
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.explain = m.explain.SetWidth(min(msg.Width-4, 70))
	}

	return m, tea.Batch(cmds...)
}

// SetOverallTotal sets the total number of steps across all modules.
func (m ProgressModel) SetOverallTotal(n int) ProgressModel {
	m.overallTotal = n
	return m
}

// View renders the progress screen.
func (m ProgressModel) View() string {
	var b strings.Builder

	b.WriteString(components.RenderBanner(m.styles))
	b.WriteString("\n\n")

	if m.currentModule != "" {
		b.WriteString(m.styles.Title.Render(fmt.Sprintf("Setting up %s", m.currentModule)))
		b.WriteString("\n\n")
	}

	// Progress bar.
	if m.overallTotal > 0 {
		pct := float64(m.overallDone) / float64(m.overallTotal)
		barWidth := 20
		filled := int(pct * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}

		bar := m.styles.ProgressFull.Render(strings.Repeat("█", filled)) +
			m.styles.ProgressEmpty.Render(strings.Repeat("░", barWidth-filled))

		b.WriteString(fmt.Sprintf("  Step %d/%d  %s  %d%%\n\n",
			m.overallDone, m.overallTotal, bar, int(pct*100)))
	}

	// Step list.
	for _, s := range m.steps {
		icon := m.stepIcon(s)
		line := fmt.Sprintf("  %s %s", icon, s.name)

		switch s.state {
		case stepDone:
			line = m.styles.Success.Render(line)
		case stepSkipped:
			line = m.styles.Muted.Render(line)
		case stepFailed:
			line = m.styles.Error.Render(line)
		case stepRunning:
			line = m.styles.Body.Render(line)
		default:
			line = m.styles.Muted.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Explain panel.
	if panel := m.explain.View(); panel != "" {
		b.WriteString("\n")
		b.WriteString(panel)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Footer.Render("  ?: toggle explain"))

	return b.String()
}

func (m ProgressModel) stepIcon(s stepStatus) string {
	switch s.state {
	case stepDone:
		return m.styles.StatusDone
	case stepRunning:
		return m.spinner.View()
	case stepSkipped:
		return m.styles.StatusSkipped
	case stepFailed:
		return m.styles.StatusFailed
	default:
		return m.styles.StatusPending
	}
}
