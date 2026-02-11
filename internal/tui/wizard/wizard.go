package wizard

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/tui/components"
)

type screen int

const (
	screenPicker screen = iota
	screenProgress
	screenSummary
)

// WizardModel is the top-level tea.Model coordinating picker → progress → summary.
type WizardModel struct {
	styles   components.Styles
	screen   screen
	picker   PickerModel
	progress ProgressModel
	summary  SummaryModel

	bridge   *Bridge
	runner   *module.Runner
	registry *module.Registry
	explain  bool
	dryRun   bool

	width    int
	height   int
	quitting bool
}

// New creates a WizardModel ready to display the picker.
func New(reg *module.Registry, runner *module.Runner, explain, dryRun bool) WizardModel {
	styles := components.DefaultStyles()
	return WizardModel{
		styles:   styles,
		screen:   screenPicker,
		picker:   NewPickerModel(styles, reg),
		progress: NewProgressModel(styles, explain),
		summary:  NewSummaryModel(styles),
		runner:   runner,
		registry: reg,
		explain:  explain,
		dryRun:   dryRun,
	}
}

// Init satisfies tea.Model.
func (m WizardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and delegates to the active screen.
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to children.
		m.picker, _ = m.picker.Update(msg)
		m.progress, _ = m.progress.Update(msg)
		m.summary, _ = m.summary.Update(msg)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if m.bridge != nil {
				m.bridge.Cancel()
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenPicker:
		return m.updatePicker(msg)
	case screenProgress:
		return m.updateProgress(msg)
	case screenSummary:
		return m.updateSummary(msg)
	}

	return m, tea.Batch(cmds...)
}

// View renders the active screen.
func (m WizardModel) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenPicker:
		return m.picker.View()
	case screenProgress:
		return m.progress.View()
	case screenSummary:
		return m.summary.View()
	}
	return ""
}

func (m WizardModel) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case PickerConfirmMsg:
		// Transition to progress screen.
		m.screen = screenProgress

		// Resolve deps to get the full set of modules that will run,
		// then calculate total steps accurately.
		resolved, err := m.registry.ResolveDeps(msg.ModuleIDs)
		if err != nil {
			// If dep resolution fails, the bridge will report the error.
			resolved = msg.ModuleIDs
		}
		total := 0
		for _, id := range resolved {
			mod := m.registry.Get(id)
			if mod != nil {
				total += len(mod.Steps)
			}
		}
		m.progress = m.progress.SetOverallTotal(total)

		// Create and start the bridge.
		m.bridge = NewBridge(m.runner, m.registry, msg.ModuleIDs)
		startCmd := m.bridge.Start()

		return m, tea.Batch(startCmd, m.progress.Init())

	default:
		m.picker, cmd = m.picker.Update(msg)
	}

	return m, cmd
}

func (m WizardModel) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case AllDoneMsg:
		m.screen = screenSummary
		m.summary = m.summary.SetResults(msg.Results)
		return m, nil

	case RunErrorMsg:
		m.screen = screenSummary
		m.summary = m.summary.SetError(msg.Err)
		return m, nil

	case ModuleStartMsg, StepStartMsg, StepDoneMsg, StepErrorMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		cmds = append(cmds, cmd)
		// Request next message from bridge.
		if m.bridge != nil {
			cmds = append(cmds, m.bridge.NextMsg())
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		cmds = append(cmds, cmd)

	default:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m WizardModel) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)
	return m, cmd
}

// Results returns module results from the summary screen (for post-TUI state saving).
func (m WizardModel) Results() []module.ModuleResult {
	return m.summary.results
}

// RunError returns the runner-level error, if any.
func (m WizardModel) RunError() error {
	return m.summary.err
}

// Screen returns the current screen (for testing).
func (m WizardModel) Screen() screen {
	return m.screen
}
