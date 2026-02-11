package wizard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/druarnfield/shhh/internal/logging"
	"github.com/druarnfield/shhh/internal/module"
	"github.com/druarnfield/shhh/internal/tui/components"
)

// --- helpers ---

func nopLogger() *slog.Logger {
	return slog.New(logging.NopHandler{})
}

func testRegistry() *module.Registry {
	reg := module.NewRegistry()
	reg.Register(&module.Module{
		ID:       "base",
		Name:     "Base",
		Category: module.CategoryBase,
		Steps: []module.Step{
			{
				Name:    "step-a",
				Explain: "explains step a",
				Check:   func(context.Context) bool { return true }, // already satisfied
				Run:     func(context.Context) error { return nil },
			},
			{
				Name:    "step-b",
				Explain: "explains step b",
				Run:     func(context.Context) error { return nil },
			},
		},
	})
	reg.Register(&module.Module{
		ID:           "python",
		Name:         "Python",
		Category:     module.CategoryLanguage,
		Dependencies: []string{"base"},
		Steps: []module.Step{
			{
				Name: "install-python",
				Run:  func(context.Context) error { return nil },
			},
		},
	})
	reg.Register(&module.Module{
		ID:           "golang",
		Name:         "Go",
		Category:     module.CategoryLanguage,
		Dependencies: []string{"base"},
		Steps: []module.Step{
			{
				Name: "install-go",
				Run:  func(context.Context) error { return nil },
			},
		},
	})
	return reg
}

// --- Explain Panel tests ---

func TestExplainPanel_HiddenByDefault(t *testing.T) {
	s := components.DefaultStyles()
	p := NewExplainPanel(s).SetText("some text")
	if got := p.View(); got != "" {
		t.Errorf("expected empty when hidden, got %q", got)
	}
}

func TestExplainPanel_VisibleWithText(t *testing.T) {
	s := components.DefaultStyles()
	p := NewExplainPanel(s).SetText("some text").SetVisible(true)
	out := p.View()
	if out == "" {
		t.Fatal("expected non-empty when visible")
	}
	if !strings.Contains(out, "some text") {
		t.Errorf("output should contain text, got %q", out)
	}
	if !strings.Contains(out, "What's happening") {
		t.Errorf("output should contain title, got %q", out)
	}
}

func TestExplainPanel_RespectsWidth(t *testing.T) {
	s := components.DefaultStyles()
	p := NewExplainPanel(s).
		SetText("short").
		SetVisible(true).
		SetWidth(40)
	out := p.View()
	if out == "" {
		t.Fatal("expected non-empty")
	}
}

// --- Picker tests ---

func TestPicker_BasePreSelected(t *testing.T) {
	s := components.DefaultStyles()
	reg := testRegistry()
	p := NewPickerModel(s, reg)

	ids := p.SelectedModuleIDs()
	if !sliceContains(ids, "base") {
		t.Error("base should be pre-selected")
	}
	if sliceContains(ids, "python") {
		t.Error("python should not be pre-selected")
	}
}

func TestPicker_TogglePython_AutoSelectsBase(t *testing.T) {
	s := components.DefaultStyles()
	reg := testRegistry()
	p := NewPickerModel(s, reg)

	// Navigate to python and toggle it on.
	p = navigateTo(p, "python")
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	ids := p.SelectedModuleIDs()
	if !sliceContains(ids, "python") {
		t.Error("python should be selected after toggle")
	}
	if !sliceContains(ids, "base") {
		t.Error("base should remain selected (dependency)")
	}
}

func TestPicker_TogglePythonOff(t *testing.T) {
	s := components.DefaultStyles()
	reg := testRegistry()
	p := NewPickerModel(s, reg)

	// Select python.
	p = navigateTo(p, "python")
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	// Deselect python.
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	ids := p.SelectedModuleIDs()
	if sliceContains(ids, "python") {
		t.Error("python should be deselected")
	}
	// Base should still be there (it's required).
	if !sliceContains(ids, "base") {
		t.Error("base should remain (required)")
	}
}

func TestPicker_ConfirmReturnsSelected(t *testing.T) {
	s := components.DefaultStyles()
	reg := testRegistry()
	p := NewPickerModel(s, reg)

	// Select python.
	p = navigateTo(p, "python")
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	// Press enter.
	var cmd tea.Cmd
	p, cmd = p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected a command from enter")
	}
	msg := cmd()
	confirm, ok := msg.(PickerConfirmMsg)
	if !ok {
		t.Fatalf("expected PickerConfirmMsg, got %T", msg)
	}
	if !sliceContains(confirm.ModuleIDs, "python") {
		t.Error("confirm should include python")
	}
	if !sliceContains(confirm.ModuleIDs, "base") {
		t.Error("confirm should include base")
	}
}

func TestPicker_EmptySelectionDoesNotConfirm(t *testing.T) {
	s := components.DefaultStyles()
	// Registry with no base modules (all optional).
	reg := module.NewRegistry()
	reg.Register(&module.Module{
		ID:       "tool",
		Name:     "Tool",
		Category: module.CategoryTool,
		Steps:    []module.Step{{Name: "s1", Run: func(context.Context) error { return nil }}},
	})
	p := NewPickerModel(s, reg)

	// Try enter without selecting anything.
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("should not confirm with zero selection")
	}
}

// --- Progress Model tests ---

func TestProgress_ModuleStart(t *testing.T) {
	s := components.DefaultStyles()
	p := NewProgressModel(s, false)
	p = p.SetOverallTotal(3)

	p, _ = p.Update(ModuleStartMsg{
		ModuleID: "base",
		Name:     "Base",
		Steps: []module.Step{
			{Name: "s1", Explain: "e1"},
			{Name: "s2", Explain: "e2"},
		},
	})

	out := p.View()
	if !strings.Contains(out, "Base") {
		t.Error("should show module name")
	}
}

func TestProgress_StepDoneSkipped(t *testing.T) {
	s := components.DefaultStyles()
	p := NewProgressModel(s, false)
	p = p.SetOverallTotal(2)

	p, _ = p.Update(ModuleStartMsg{
		ModuleID: "base",
		Name:     "Base",
		Steps:    []module.Step{{Name: "s1"}, {Name: "s2"}},
	})

	p, _ = p.Update(StepStartMsg{ModuleID: "base", StepName: "s1", Index: 0, Total: 2})
	p, _ = p.Update(StepDoneMsg{ModuleID: "base", StepName: "s1", Index: 0, Total: 2, Skipped: true})

	out := p.View()
	if !strings.Contains(out, s.StatusSkipped) {
		t.Error("should show skipped icon")
	}
}

func TestProgress_StepDoneCompleted(t *testing.T) {
	s := components.DefaultStyles()
	p := NewProgressModel(s, false)
	p = p.SetOverallTotal(2)

	p, _ = p.Update(ModuleStartMsg{
		ModuleID: "base",
		Name:     "Base",
		Steps:    []module.Step{{Name: "s1"}},
	})

	p, _ = p.Update(StepStartMsg{ModuleID: "base", StepName: "s1", Index: 0, Total: 1})
	p, _ = p.Update(StepDoneMsg{ModuleID: "base", StepName: "s1", Index: 0, Total: 1})

	out := p.View()
	if !strings.Contains(out, s.StatusDone) {
		t.Error("should show done icon")
	}
}

func TestProgress_ToggleExplain(t *testing.T) {
	s := components.DefaultStyles()
	p := NewProgressModel(s, false)

	// Start not showing explain.
	p, _ = p.Update(ModuleStartMsg{
		ModuleID: "base",
		Name:     "Base",
		Steps:    []module.Step{{Name: "s1", Explain: "the explanation"}},
	})
	p, _ = p.Update(StepStartMsg{ModuleID: "base", StepName: "s1", Explain: "the explanation", Index: 0, Total: 1})

	out := p.View()
	if strings.Contains(out, "the explanation") {
		t.Error("explain should be hidden initially")
	}

	// Toggle on.
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	out = p.View()
	if !strings.Contains(out, "the explanation") {
		t.Error("explain should be visible after toggle")
	}
}

// --- Summary Model tests ---

func TestSummary_Success(t *testing.T) {
	s := components.DefaultStyles()
	sm := NewSummaryModel(s).SetResults([]module.ModuleResult{
		{ModuleID: "base", Completed: 2, Skipped: 1, Total: 3},
	})
	out := sm.View()
	if !strings.Contains(out, "Complete") {
		t.Error("should show 'Complete' for successful run")
	}
}

func TestSummary_Failure(t *testing.T) {
	s := components.DefaultStyles()
	sm := NewSummaryModel(s).SetResults([]module.ModuleResult{
		{
			ModuleID:   "base",
			Completed:  1,
			Total:      2,
			FailedStep: "step-b",
			Err:        errors.New("something broke"),
		},
	})
	out := sm.View()
	if !strings.Contains(out, "Failed") {
		t.Error("should show 'Failed'")
	}
	if !strings.Contains(out, "something broke") {
		t.Error("should show error message")
	}
	if !strings.Contains(out, "Fix the issue") {
		t.Error("should show fix hint")
	}
}

func TestSummary_RunnerError(t *testing.T) {
	s := components.DefaultStyles()
	sm := NewSummaryModel(s).SetError(errors.New("dep cycle"))
	out := sm.View()
	if !strings.Contains(out, "Failed") {
		t.Error("should show 'Failed' for runner error")
	}
	if !strings.Contains(out, "dep cycle") {
		t.Error("should show error message")
	}
}

// --- Wizard flow tests ---

func TestWizard_StartsOnPicker(t *testing.T) {
	reg := testRegistry()
	runner := module.NewRunner(nopLogger(), false)
	w := New(reg, runner, false, false)

	if w.Screen() != screenPicker {
		t.Error("wizard should start on picker screen")
	}
}

func TestWizard_PickerToProgress(t *testing.T) {
	reg := testRegistry()
	runner := module.NewRunner(nopLogger(), false)
	w := New(reg, runner, false, false)

	// Simulate picker confirmation.
	updated, _ := w.Update(PickerConfirmMsg{ModuleIDs: []string{"base"}})
	wm := updated.(WizardModel)

	if wm.Screen() != screenProgress {
		t.Errorf("expected progress screen, got %d", wm.Screen())
	}
}

func TestWizard_AllDoneToSummary(t *testing.T) {
	reg := testRegistry()
	runner := module.NewRunner(nopLogger(), false)
	w := New(reg, runner, false, false)

	// Go to progress.
	updated, _ := w.Update(PickerConfirmMsg{ModuleIDs: []string{"base"}})
	wm := updated.(WizardModel)

	// Simulate AllDoneMsg.
	results := []module.ModuleResult{{ModuleID: "base", Completed: 2, Total: 2}}
	updated2, _ := wm.Update(AllDoneMsg{Results: results})
	wm2 := updated2.(WizardModel)

	if wm2.Screen() != screenSummary {
		t.Errorf("expected summary screen, got %d", wm2.Screen())
	}
	if len(wm2.Results()) != 1 {
		t.Errorf("expected 1 result, got %d", len(wm2.Results()))
	}
}

func TestWizard_RunErrorToSummary(t *testing.T) {
	reg := testRegistry()
	runner := module.NewRunner(nopLogger(), false)
	w := New(reg, runner, false, false)

	// Go to progress.
	updated, _ := w.Update(PickerConfirmMsg{ModuleIDs: []string{"base"}})
	wm := updated.(WizardModel)

	// Simulate RunErrorMsg.
	updated2, _ := wm.Update(RunErrorMsg{Err: errors.New("dep failed")})
	wm2 := updated2.(WizardModel)

	if wm2.Screen() != screenSummary {
		t.Errorf("expected summary screen, got %d", wm2.Screen())
	}
	if wm2.RunError() == nil {
		t.Error("expected run error to be set")
	}
}

// --- Bridge tests ---

func TestBridge_MessageOrder(t *testing.T) {
	reg := module.NewRegistry()
	reg.Register(&module.Module{
		ID:       "test",
		Name:     "Test",
		Category: module.CategoryBase,
		Steps: []module.Step{
			{
				Name:    "check-me",
				Explain: "checking",
				Check:   func(context.Context) bool { return true },
				Run:     func(context.Context) error { return nil },
			},
			{
				Name:    "run-me",
				Explain: "running",
				Run:     func(context.Context) error { return nil },
			},
		},
	})

	runner := module.NewRunner(nopLogger(), false)
	bridge := NewBridge(runner, reg, []string{"test"})

	// Collect all messages.
	startCmd := bridge.Start()
	var msgs []tea.Msg

	cmd := startCmd
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		msgs = append(msgs, msg)
		cmd = bridge.NextMsg()
	}

	// Expected order:
	// 1. ModuleStartMsg
	// 2. StepStartMsg (check-me)
	// 3. StepDoneMsg (check-me, skipped)
	// 4. StepStartMsg (run-me)
	// 5. StepDoneMsg (run-me)
	// 6. AllDoneMsg

	if len(msgs) < 6 {
		t.Fatalf("expected at least 6 messages, got %d: %v", len(msgs), msgTypes(msgs))
	}

	assertMsgType[ModuleStartMsg](t, msgs[0], "msg 0")
	assertMsgType[StepStartMsg](t, msgs[1], "msg 1")
	done1 := assertMsgType[StepDoneMsg](t, msgs[2], "msg 2")
	if !done1.Skipped {
		t.Error("first step should be skipped")
	}
	assertMsgType[StepStartMsg](t, msgs[3], "msg 3")
	done2 := assertMsgType[StepDoneMsg](t, msgs[4], "msg 4")
	if done2.Skipped {
		t.Error("second step should not be skipped")
	}
	allDone := assertMsgType[AllDoneMsg](t, msgs[5], "msg 5")
	if len(allDone.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(allDone.Results))
	}
}

func TestBridge_ErrorMessage(t *testing.T) {
	reg := module.NewRegistry()
	reg.Register(&module.Module{
		ID:       "fail",
		Name:     "Fail",
		Category: module.CategoryBase,
		Steps: []module.Step{
			{
				Name: "will-fail",
				Run:  func(context.Context) error { return errors.New("boom") },
			},
		},
	})

	runner := module.NewRunner(nopLogger(), false)
	bridge := NewBridge(runner, reg, []string{"fail"})

	startCmd := bridge.Start()
	var msgs []tea.Msg
	cmd := startCmd
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		msgs = append(msgs, msg)
		cmd = bridge.NextMsg()
	}

	// Should have: ModuleStartMsg, StepStartMsg, StepErrorMsg, AllDoneMsg
	if len(msgs) < 4 {
		t.Fatalf("expected at least 4 messages, got %d: %v", len(msgs), msgTypes(msgs))
	}

	assertMsgType[ModuleStartMsg](t, msgs[0], "msg 0")
	assertMsgType[StepStartMsg](t, msgs[1], "msg 1")
	errMsg := assertMsgType[StepErrorMsg](t, msgs[2], "msg 2")
	if errMsg.Err == nil {
		t.Error("expected error in StepErrorMsg")
	}
	allDone := assertMsgType[AllDoneMsg](t, msgs[3], "msg 3")
	if len(allDone.Results) != 1 || allDone.Results[0].Err == nil {
		t.Error("expected failed result in AllDoneMsg")
	}
}

// --- helpers ---

func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func navigateTo(p PickerModel, moduleID string) PickerModel {
	for i, item := range p.items {
		if item.module != nil && item.module.ID == moduleID {
			p.cursor = i
			return p
		}
	}
	return p
}

func assertMsgType[T any](t *testing.T, msg tea.Msg, label string) T {
	t.Helper()
	v, ok := msg.(T)
	if !ok {
		t.Fatalf("%s: expected %T, got %T", label, v, msg)
	}
	return v
}

func msgTypes(msgs []tea.Msg) []string {
	var types []string
	for _, m := range msgs {
		types = append(types, fmt.Sprintf("%T", m))
	}
	return types
}
