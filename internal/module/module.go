// Package module defines the core types for shhh modules: Step, Module,
// Category, and a Registry with topological dependency resolution.
package module

import (
	"context"
	"fmt"
)

// Category classifies modules into logical groups.
type Category int

const (
	CategoryBase     Category = iota // Core system setup (shells, dotfiles)
	CategoryLanguage                 // Programming language environments
	CategoryTool                     // Developer tools and utilities
)

// String returns the human-readable name for a Category.
func (c Category) String() string {
	switch c {
	case CategoryBase:
		return "Base"
	case CategoryLanguage:
		return "Language"
	case CategoryTool:
		return "Tool"
	default:
		return fmt.Sprintf("Category(%d)", int(c))
	}
}

// Step represents a single idempotent operation within a module.
type Step struct {
	// Name is a short identifier for this step.
	Name string

	// Description explains what this step does.
	Description string

	// Explain returns a human-readable description of what Run would do,
	// without performing any changes.
	Explain func(ctx context.Context) string

	// Check returns true if the step is already satisfied (i.e. Run can be skipped).
	Check func(ctx context.Context) bool

	// Run executes the step. It should be idempotent.
	Run func(ctx context.Context) error

	// DryRun describes what Run would do without making changes.
	DryRun func(ctx context.Context) string
}

// Module represents a discrete unit of system configuration (e.g. "golang",
// "python", "git"). Each module belongs to a Category and may declare
// Dependencies on other modules that must be applied first.
type Module struct {
	// ID is the unique machine-readable identifier (e.g. "golang").
	ID string

	// Name is the human-readable display name (e.g. "Go").
	Name string

	// Description explains what this module configures.
	Description string

	// Category classifies this module.
	Category Category

	// Dependencies lists module IDs that must be applied before this one.
	Dependencies []string

	// Steps are the ordered operations to apply this module.
	Steps []Step
}

// Registry holds registered modules and provides lookup and dependency
// resolution. It preserves insertion order for deterministic results.
type Registry struct {
	modules map[string]*Module
	order   []string // insertion order for stable iteration
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]*Module),
	}
}

// Register adds a module to the registry. If a module with the same ID
// already exists, it is replaced (but insertion order is preserved).
func (r *Registry) Register(m *Module) {
	if _, exists := r.modules[m.ID]; !exists {
		r.order = append(r.order, m.ID)
	}
	r.modules[m.ID] = m
}

// Get returns the module with the given ID, or nil if not found.
func (r *Registry) Get(id string) *Module {
	return r.modules[id]
}

// All returns every registered module in insertion order.
func (r *Registry) All() []*Module {
	result := make([]*Module, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.modules[id])
	}
	return result
}

// ByCategory returns all modules matching the given category, in insertion order.
func (r *Registry) ByCategory(cat Category) []*Module {
	var result []*Module
	for _, id := range r.order {
		if r.modules[id].Category == cat {
			result = append(result, r.modules[id])
		}
	}
	return result
}

// ResolveDeps performs a topological sort of the requested module IDs and all
// their transitive dependencies using Kahn's algorithm. It returns the IDs in
// an order where every module appears after its dependencies.
//
// When multiple modules have zero in-degree simultaneously, they are emitted
// in insertion order (the order they were registered) for deterministic output.
//
// Returns an error if a dependency is not registered or if a cycle is detected.
func (r *Registry) ResolveDeps(ids []string) ([]string, error) {
	// Collect all needed modules (requested + transitive deps).
	needed := make(map[string]bool)
	var collect func(id string) error
	collect = func(id string) error {
		if needed[id] {
			return nil
		}
		m := r.modules[id]
		if m == nil {
			return fmt.Errorf("module %q not found in registry", id)
		}
		needed[id] = true
		for _, dep := range m.Dependencies {
			if err := collect(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for _, id := range ids {
		if err := collect(id); err != nil {
			return nil, err
		}
	}

	// Build in-degree map scoped to needed modules.
	inDegree := make(map[string]int, len(needed))
	for id := range needed {
		inDegree[id] = 0
	}
	for id := range needed {
		for _, dep := range r.modules[id].Dependencies {
			if needed[dep] {
				inDegree[id]++
			}
		}
	}

	// Seed the queue with zero in-degree nodes in insertion order.
	var queue []string
	for _, id := range r.order {
		if needed[id] && inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		// Pop front.
		cur := queue[0]
		queue = queue[1:]
		sorted = append(sorted, cur)

		// For each module that depends on cur, decrement in-degree.
		// We iterate in insertion order so newly-ready nodes are added
		// in a stable order.
		for _, id := range r.order {
			if !needed[id] {
				continue
			}
			for _, dep := range r.modules[id].Dependencies {
				if dep == cur {
					inDegree[id]--
					if inDegree[id] == 0 {
						queue = append(queue, id)
					}
					break
				}
			}
		}
	}

	if len(sorted) != len(needed) {
		return nil, fmt.Errorf("dependency cycle detected among modules")
	}

	return sorted, nil
}
