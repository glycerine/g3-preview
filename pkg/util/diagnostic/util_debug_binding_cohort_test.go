package diagnostic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// The generated process has one std::cerr owner for notification and debugger
// prompts. Borrowed bindings must preserve that identity in either order.
func TestDebugInteractiveNotificationActionStreamCohort(t *testing.T) {
	_, d, action, _, streams := debugActionFixture(t, "C")
	before := d.BudgetField.Used()
	other, err := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: d.BudgetField})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, other.Close()) })
	if err = d.BindAssertionStreams(other); !errors.Is(err, arena.ErrConfig) || d.assertionStreams != streams {
		t.Fatal("notification rebind detached the interactive action", err)
	}
	if err = d.BindAssertionStreams(nil); !errors.Is(err, arena.ErrConfig) || d.assertionStreams != streams {
		t.Fatal("notification unbind detached the interactive action", err)
	}
	paramsMust(t, d.BindAssertionActions(nil))
	paramsMust(t, d.BindAssertionStreams(other))
	if err = d.BindAssertionActions(action); !errors.Is(err, arena.ErrConfig) || d.assertionActions != nil {
		t.Fatal("action rebind crossed same-budget standard streams", err)
	}
	paramsMust(t, d.BindAssertionStreams(nil))
	if err = d.BindAssertionActions(action); !errors.Is(err, arena.ErrConfig) || d.assertionActions != nil {
		t.Fatal("action bound without its notification stream", err)
	}
	paramsMust(t, d.BindAssertionStreams(streams))
	paramsMust(t, d.BindAssertionActions(action))
	paramsMust(t, d.BindAssertionActions(nil))
	paramsMust(t, d.BindAssertionStreams(nil))
	noDebugger, err := NewDebugActionContext(DebugActionConfig{Budget: d.BudgetField, Profile: DebugNoDebugger})
	paramsMust(t, err)
	paramsMust(t, d.BindAssertionActions(noDebugger))
	paramsMust(t, d.BindAssertionStreams(other))
	paramsMust(t, d.BindAssertionStreams(nil))
	paramsMust(t, d.BindAssertionActions(nil))
	paramsMust(t, noDebugger.Close())
	paramsMust(t, d.BindAssertionStreams(streams))
	paramsMust(t, d.BindAssertionActions(action))
	paramsMust(t, other.Close())
	if d.BudgetField.Used() != before {
		t.Fatal("borrowed rebindings changed the charged graph", before, d.BudgetField.Used())
	}
}

func TestDebugInteractiveInputMustStartTiedToItsOwnStdout(t *testing.T) {
	_, d, _, input, streams := debugActionFixture(t, "C")
	before := d.BudgetField.Used()
	other, err := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: d.BudgetField})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, other.Close()) })
	foreignInput, err := NewDebuggerCharInput(other, nil)
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, foreignInput.Close()) })
	if action, err := NewDebugActionContext(DebugActionConfig{Budget: d.BudgetField, Streams: streams, Input: foreignInput}); action != nil || !errors.Is(err, arena.ErrConfig) {
		t.Fatal("same-budget foreign stdout tie became the process input", action, err)
	}
	old, err := input.SetTie(nil)
	paramsMust(t, err)
	if old != streams.stdout {
		t.Fatal("lost original source stdout tie")
	}
	if action, err := NewDebugActionContext(DebugActionConfig{Budget: d.BudgetField, Streams: streams, Input: input}); action != nil || !errors.Is(err, arena.ErrConfig) {
		t.Fatal("untied input became the process input", action, err)
	}
	_, err = input.SetTie(streams.stdout)
	paramsMust(t, err)
	paramsMust(t, foreignInput.Close())
	paramsMust(t, other.Close())
	if d.BudgetField.Used() != before {
		t.Fatal("rejected action charged the process", before, d.BudgetField.Used())
	}
}
