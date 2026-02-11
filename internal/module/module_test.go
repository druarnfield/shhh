package module

import (
	"context"
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mod := &Module{
		ID:       "base",
		Name:     "Base",
		Category: CategoryBase,
	}

	reg.Register(mod)

	got := reg.Get("base")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.ID != "base" {
		t.Errorf("ID = %q, want %q", got.ID, "base")
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for missing module")
	}
}

func TestRegistry_ByCategory(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base", Category: CategoryBase})
	reg.Register(&Module{ID: "python", Category: CategoryLanguage})
	reg.Register(&Module{ID: "golang", Category: CategoryLanguage})
	reg.Register(&Module{ID: "tools", Category: CategoryTool})

	langs := reg.ByCategory(CategoryLanguage)
	if len(langs) != 2 {
		t.Errorf("ByCategory(Language) = %d modules, want 2", len(langs))
	}
}

func TestRegistry_ResolveDeps_Simple(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base", Dependencies: nil})
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})

	order, err := reg.ResolveDeps([]string{"python"})
	if err != nil {
		t.Fatalf("ResolveDeps: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("len = %d, want 2", len(order))
	}
	if order[0] != "base" || order[1] != "python" {
		t.Errorf("order = %v, want [base, python]", order)
	}
}

func TestRegistry_ResolveDeps_AlreadyIncluded(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base"})
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})

	order, err := reg.ResolveDeps([]string{"base", "python"})
	if err != nil {
		t.Fatalf("ResolveDeps: %v", err)
	}

	count := 0
	for _, id := range order {
		if id == "base" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("base appears %d times, want 1", count)
	}
}

func TestRegistry_ResolveDeps_Diamond(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "base"})
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})
	reg.Register(&Module{ID: "golang", Dependencies: []string{"base"}})
	reg.Register(&Module{ID: "tools", Dependencies: []string{"python", "golang"}})

	order, err := reg.ResolveDeps([]string{"tools"})
	if err != nil {
		t.Fatalf("ResolveDeps: %v", err)
	}

	idx := make(map[string]int)
	for i, id := range order {
		idx[id] = i
	}

	if idx["base"] >= idx["python"] {
		t.Error("base should come before python")
	}
	if idx["base"] >= idx["golang"] {
		t.Error("base should come before golang")
	}
	if idx["python"] >= idx["tools"] {
		t.Error("python should come before tools")
	}
}

func TestRegistry_ResolveDeps_CycleError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "a", Dependencies: []string{"b"}})
	reg.Register(&Module{ID: "b", Dependencies: []string{"a"}})

	_, err := reg.ResolveDeps([]string{"a"})
	if err == nil {
		t.Error("expected error for cycle")
	}
}

func TestRegistry_ResolveDeps_MissingDepError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Module{ID: "python", Dependencies: []string{"base"}})

	_, err := reg.ResolveDeps([]string{"python"})
	if err == nil {
		t.Error("expected error for missing dependency")
	}
}

func TestStep_CheckSkipsRun(t *testing.T) {
	ran := false
	step := Step{
		Name: "test step",
		Check: func(ctx context.Context) bool {
			return true
		},
		Run: func(ctx context.Context) error {
			ran = true
			return nil
		},
	}

	if !step.Check(context.Background()) {
		t.Error("Check should return true")
	}
	if ran {
		t.Error("Run should not have been called")
	}
}
