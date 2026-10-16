//go:build g3debug

package numeral

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Source: src/util/mpzzp.h:167-170,257-264. Both macros ignore the assertion switch.
func TestIntegerModularExplicitSourceDiagnostics(t *testing.T) {
	m, _, _ := modularManager(t, 7)
	b := arena.NewBudget(1 << 24)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(r, b)
	serviceMust(t, e)
	in, e := os.CreateTemp(t.TempDir(), "diagnostic-input-*")
	serviceMust(t, e)
	_, e = in.WriteString("CTT")
	serviceMust(t, e)
	_, e = in.Seek(0, 0)
	serviceMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "diagnostic-output-*")
	serviceMust(t, e)
	s, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stdout: out, Stderr: out})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(s, in)
	serviceMust(t, e)
	actions, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: b, Streams: s, Input: input})
	serviceMust(t, e)
	serviceMust(t, d.BindAssertionStreams(s))
	serviceMust(t, d.BindAssertionActions(actions))
	old, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(old)
		serviceMust(t, e)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, s.Close())
		serviceMust(t, in.Close())
		serviceMust(t, out.Close())
		serviceMust(t, d.Close())
		serviceMust(t, r.Close())
	})
	serviceMust(t, d.EnableAssertions(false))
	serviceMust(t, m.SetZ())
	a := NewInt(3)
	if e := m.Inv(&a); e != nil || a.Val != 3 {
		t.Fatal("Z inverse Continue must preserve operand", a, e)
	}
	serviceMust(t, m.SetZPUint64(7))
	root := NewInt(99)
	var thrown z3exception.DefaultException
	if _, e := m.IsPerfectSquare(&a, &root); !errors.As(e, &thrown) || root.Val != 99 {
		t.Fatal("finite square Throw must stop before root mutation", root, e)
	}
	var receipt integerPerfectSquareInvocationReceipt
	if _, e := m.IsPerfectSquareWithReceipt(&a, &root, &receipt); !errors.As(e, &thrown) || root.Val != 99 {
		t.Fatal("receipt path must share the finite source diagnostic", root, e)
	}
	state, e := input.State()
	serviceMust(t, e)
	if state.Position != 3 {
		t.Fatal(state)
	}
	got, e := os.ReadFile(out.Name())
	serviceMust(t, e)
	const prompt = "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	const finite = "NOT IMPLEMENTED YET!\nASSERTION VIOLATION\nFile: src/util/mpzzp.h\nLine: 261\nUNREACHABLE CODE WAS REACHED.\n" + prompt
	want := "ASSERTION VIOLATION\nFile: src/util/mpzzp.h\nLine: 169\nUNREACHABLE CODE WAS REACHED.\n" + prompt +
		finite + finite
	if string(got) != want {
		t.Fatalf("diagnostic output: got %q, want %q", got, want)
	}
}
