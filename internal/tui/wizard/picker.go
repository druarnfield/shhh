package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/tui/components"
)

// pickerItem is a single row in the picker list — either a category header
// or a selectable module.
type pickerItem struct {
	isHeader   bool
	category   string
	module     *module.Module
	required   bool            // base modules
	requiredBy map[string]bool // module names that pulled this in as a dep
}

// PickerConfirmMsg is sent when the user confirms their selection.
type PickerConfirmMsg struct {
	ModuleIDs []string
}

// PickerModel is a multi-select module picker grouped by category.
type PickerModel struct {
	styles   components.Styles
	items    []pickerItem
	selected map[string]bool // module ID → selected
	cursor   int
	width    int
	height   int
}

// NewPickerModel creates a picker from all modules in the registry.
func NewPickerModel(styles components.Styles, reg *module.Registry) PickerModel {
	m := PickerModel{
		styles:   styles,
		selected: make(map[string]bool),
	}

	// Build items grouped by category.
	categories := []module.Category{module.CategoryBase, module.CategoryLanguage, module.CategoryTool}
	for _, cat := range categories {
		mods := reg.ByCategory(cat)
		if len(mods) == 0 {
			continue
		}
		m.items = append(m.items, pickerItem{isHeader: true, category: cat.String()})
		for _, mod := range mods {
			required := cat == module.CategoryBase
			m.items = append(m.items, pickerItem{
				module:     mod,
				required:   required,
				requiredBy: make(map[string]bool),
			})
			if required {
				m.selected[mod.ID] = true
			}
		}
	}

	// Start cursor on first selectable item.
	m.cursor = m.nextSelectable(0, 1)

	return m
}

// SelectedModuleIDs returns the IDs of all selected modules.
func (m PickerModel) SelectedModuleIDs() []string {
	var ids []string
	for _, item := range m.items {
		if item.module != nil && m.selected[item.module.ID] {
			ids = append(ids, item.module.ID)
		}
	}
	return ids
}

// Init satisfies tea.Model.
func (m PickerModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the picker.
func (m PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = m.nextSelectable(m.cursor, -1)
		case "down", "j":
			m.cursor = m.nextSelectable(m.cursor, 1)
		case " ":
			m.toggleCurrent()
		case "enter":
			ids := m.SelectedModuleIDs()
			if len(ids) > 0 {
				return m, func() tea.Msg { return PickerConfirmMsg{ModuleIDs: ids} }
			}
		case "a":
			m.selectAll()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the picker.
func (m PickerModel) View() string {
	var b strings.Builder

	b.WriteString(components.RenderBanner(m.styles))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Title.Render("Select modules to set up"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		if item.isHeader {
			b.WriteString(m.styles.Subtitle.Render(item.category))
			b.WriteString("\n")
			continue
		}

		isCursor := i == m.cursor
		isSelected := m.selected[item.module.ID]

		// Checkbox.
		checkbox := m.styles.CheckboxOff
		if isSelected {
			checkbox = m.styles.CheckboxOn
		}

		// Module line.
		label := item.module.Name
		hint := ""
		if item.required {
			hint = " (required)"
		} else if len(item.requiredBy) > 0 {
			hint = fmt.Sprintf(" (required by %s)", requiredByHint(item.requiredBy))
		}

		line := fmt.Sprintf("  %s %s%s", checkbox, label, hint)

		if isCursor {
			line = m.styles.SelectedItem.Render("> " + line[2:])
		} else if item.required || len(item.requiredBy) > 0 {
			line = m.styles.Muted.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	count := len(m.SelectedModuleIDs())
	b.WriteString(m.styles.Footer.Render(
		fmt.Sprintf("  space: toggle  a: select all  enter: confirm (%d selected)", count),
	))

	return b.String()
}

// requiredByHint formats the set of parent module names for display.
func requiredByHint(parents map[string]bool) string {
	names := make([]string, 0, len(parents))
	for name := range parents {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// toggleCurrent toggles the module under the cursor.
func (m *PickerModel) toggleCurrent() {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	item := m.items[m.cursor]
	if item.isHeader || item.module == nil {
		return
	}
	// Don't allow deselecting required base modules.
	if item.required {
		return
	}

	id := item.module.ID
	if m.selected[id] {
		// Deselect — also remove dep hints that were added by this module.
		delete(m.selected, id)
		m.clearDepHints(id)
	} else {
		// Select — also auto-select dependencies.
		m.selected[id] = true
		m.autoSelectDeps(item.module)
	}
}

// autoSelectDeps ensures all dependencies of mod are selected.
func (m *PickerModel) autoSelectDeps(mod *module.Module) {
	for _, depID := range mod.Dependencies {
		m.selected[depID] = true
		for j := range m.items {
			if m.items[j].module != nil && m.items[j].module.ID == depID {
				m.items[j].requiredBy[mod.Name] = true
				// Recurse for transitive deps.
				m.autoSelectDeps(m.items[j].module)
				break
			}
		}
	}
}

// clearDepHints removes "required by" hints left by a deselected module.
func (m *PickerModel) clearDepHints(moduleID string) {
	name := ""
	for _, item := range m.items {
		if item.module != nil && item.module.ID == moduleID {
			name = item.module.Name
			break
		}
	}
	if name == "" {
		return
	}
	for j := range m.items {
		if m.items[j].module == nil {
			continue
		}
		delete(m.items[j].requiredBy, name)
		// If no one needs it and it's not required, deselect.
		if len(m.items[j].requiredBy) == 0 && !m.items[j].required && !m.isDepOfAnySelected(m.items[j].module.ID) {
			delete(m.selected, m.items[j].module.ID)
		}
	}
}

// isDepOfAnySelected checks if moduleID is a dependency of any currently selected module.
func (m *PickerModel) isDepOfAnySelected(moduleID string) bool {
	for _, item := range m.items {
		if item.module == nil || !m.selected[item.module.ID] {
			continue
		}
		for _, dep := range item.module.Dependencies {
			if dep == moduleID {
				return true
			}
		}
	}
	return false
}

// selectAll selects every module.
func (m *PickerModel) selectAll() {
	for _, item := range m.items {
		if item.module != nil {
			m.selected[item.module.ID] = true
		}
	}
}

// nextSelectable finds the next non-header item index in the given direction.
func (m PickerModel) nextSelectable(from int, dir int) int {
	n := len(m.items)
	if n == 0 {
		return 0
	}

	pos := from + dir
	for i := 0; i < n; i++ {
		if pos < 0 {
			pos = n - 1
		} else if pos >= n {
			pos = 0
		}
		if !m.items[pos].isHeader {
			return pos
		}
		pos += dir
	}
	return from
}
