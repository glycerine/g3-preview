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
