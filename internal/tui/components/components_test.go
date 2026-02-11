package components

import "testing"

func TestDefaultStyles(t *testing.T) {
	s := DefaultStyles()

	if s.CheckboxOn == "" {
		t.Error("CheckboxOn is empty")
	}
	if s.CheckboxOff == "" {
		t.Error("CheckboxOff is empty")
	}
	if s.StatusDone == "" {
		t.Error("StatusDone is empty")
	}
	if s.StatusPending == "" {
		t.Error("StatusPending is empty")
	}
	if s.StatusFailed == "" {
		t.Error("StatusFailed is empty")
	}
}

func TestRenderBanner(t *testing.T) {
	s := DefaultStyles()
	out := RenderBanner(s)
	if out == "" {
		t.Error("RenderBanner returned empty string")
	}
	if len(out) < 50 {
		t.Error("RenderBanner output seems too short")
	}
}

func TestNewSpinner(t *testing.T) {
	s := DefaultStyles()
	sp := NewSpinner(s)
	// Spinner should produce a non-empty frame.
	if sp.View() == "" {
		t.Error("spinner View() is empty")
	}
}
