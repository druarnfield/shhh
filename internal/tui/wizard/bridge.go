package wizard

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/druarnfield/shhh/internal/module"
)

// Bridge runs modules in a background goroutine and produces tea.Msg values
// for the TUI via a channel.
type Bridge struct {
	runner    *module.Runner
	registry  *module.Registry
	moduleIDs []string
	msgs      chan tea.Msg
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewBridge creates a Bridge that will run the given modules.
func NewBridge(runner *module.Runner, reg *module.Registry, ids []string) *Bridge {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bridge{
		runner:    runner,
		registry:  reg,
		moduleIDs: ids,
		msgs:      make(chan tea.Msg, 64),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Cancel signals the runner goroutine to stop.
func (b *Bridge) Cancel() {
	b.cancel()
}

// send delivers a message on the channel, respecting context cancellation
// to prevent deadlocks if the TUI has been shut down.
func (b *Bridge) send(msg tea.Msg) bool {
	select {
	case b.msgs <- msg:
		return true
	case <-b.ctx.Done():
		return false
	}
}

// Start launches module execution in a background goroutine and returns a
// tea.Cmd that delivers the first message.
func (b *Bridge) Start() tea.Cmd {
	// Install pre-step callback for StepStartMsg.
	b.runner.SetPreStepCallback(func(mod *module.Module, step *module.Step, index int, total int) {
		b.send(StepStartMsg{
			ModuleID: mod.ID,
			StepName: step.Name,
			Explain:  step.Explain,
			Index:    index,
			Total:    total,
		})
	})

	// Install post-step callback for StepDoneMsg / StepErrorMsg.
	b.runner.SetCallback(func(mod *module.Module, step *module.Step, index int, total int, skipped bool, err error) {
		if err != nil {
			b.send(StepErrorMsg{
				ModuleID: mod.ID,
				StepName: step.Name,
				Index:    index,
				Total:    total,
				Err:      err,
			})
			return
		}
		b.send(StepDoneMsg{
			ModuleID: mod.ID,
			StepName: step.Name,
			Index:    index,
			Total:    total,
			Skipped:  skipped,
		})
	})

	go b.run()

	return b.NextMsg()
}

// run executes modules one at a time, sending ModuleStartMsg before each.
// It resolves dependencies itself (rather than using runner.RunModules) so it
// can inject ModuleStartMsg between modules.
func (b *Bridge) run() {
	defer close(b.msgs)

	sorted, err := b.registry.ResolveDeps(b.moduleIDs)
	if err != nil {
		b.send(RunErrorMsg{Err: err})
		return
	}

	var results []module.ModuleResult
	for _, id := range sorted {
		mod := b.registry.Get(id)
		if mod == nil {
			b.send(RunErrorMsg{Err: fmt.Errorf("module %q not found", id)})
			return
		}

		if !b.send(ModuleStartMsg{
			ModuleID: mod.ID,
			Name:     mod.Name,
			Steps:    mod.Steps,
		}) {
			return
		}

		result := b.runner.RunModule(b.ctx, mod)
		results = append(results, result)

		if result.Err != nil {
			b.send(AllDoneMsg{Results: results})
			return
		}
	}

	b.send(AllDoneMsg{Results: results})
}

// NextMsg returns a tea.Cmd that waits for the next message from the channel.
func (b *Bridge) NextMsg() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-b.msgs
		if !ok {
			return nil
		}
		return msg
	}
}
