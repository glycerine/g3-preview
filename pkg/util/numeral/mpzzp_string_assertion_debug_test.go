//go:build g3debug

package numeral

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Source: src/util/mpzzp.h:146. The assertion precedes integer formatting.
func TestIntegerModularStringSourceAssertionActions(t *testing.T) {
	m, _, _ := modularManager(t, 7)
	a := NewInt(8) // outside the centered [-3, 3] residue interval
	budget := arena.NewBudget(^uint64(0))
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	in, e := os.CreateTemp(t.TempDir(), "mpzzp-assert-input-*")
	serviceMust(t, e)
	_, e = in.WriteString("CTCTCTCTCTCTCT")
	serviceMust(t, e)
	_, e = in.Seek(0, 0)
	serviceMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "mpzzp-assert-output-*")
	serviceMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: out, Stderr: out})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, in)
	serviceMust(t, e)
	actions, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	serviceMust(t, e)
	serviceMust(t, d.BindAssertionStreams(streams))
	serviceMust(t, d.BindAssertionActions(actions))
	old, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(old)
		serviceMust(t, e)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, in.Close())
		serviceMust(t, out.Close())
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
	})

	if got, e := m.String(&a); e != nil || got != "8" {
		t.Fatal("Continue must reach raw numeral formatting", got, e)
	}
	var thrown z3exception.DefaultException
	if _, e := m.String(&a); !errors.As(e, &thrown) {
		t.Fatal("Throw must stop before numeral formatting", e)
	}
	b := NewInt(0)
	calls := []func() error{
		func() error { _, e := m.IsZero(&a); return e },
		func() error { _, e := m.IsOne(&a); return e },
		func() error { _, e := m.IsPos(&a); return e },
		func() error { _, e := m.IsNeg(&a); return e },
		func() error { _, e := m.IsNonPos(&a); return e },
		func() error { _, e := m.IsNonNeg(&a); return e },
		func() error { _, e := m.IsMinusOne(&a); return e },
		func() error { _, e := m.Equal(&a, &b); return e },
		func() error { _, e := m.Less(&a, &b); return e },
		func() error { _, e := m.LessEqual(&a, &b); return e },
		func() error { _, e := m.Greater(&a, &b); return e },
		func() error { _, e := m.GreaterEqual(&a, &b); return e },
	}
	for i, call := range calls {
		e := call()
		if i%2 == 0 {
			if e != nil {
				t.Fatal("Continue must reach the raw scalar operation", i, e)
			}
		} else if !errors.As(e, &thrown) {
			t.Fatal("Throw must stop before the raw scalar operation", i, e)
		}
	}
	serviceMust(t, d.EnableAssertions(false))
	if got, e := m.String(&a); e != nil || got != "8" {
		t.Fatal("disabled source assertion must leave numeral readable", got, e)
	}
	if _, e := m.Greater(&a, &b); e != nil {
		t.Fatal("disabled pair assertion must reach comparison", e)
	}
	data, e := os.ReadFile(out.Name())
	serviceMust(t, e)
	if strings.Count(string(data), "File: src/util/mpzzp.h\nLine: 146\n") != 2 ||
		strings.Count(string(data), "is_p_normalized(a)\n") != 9 ||
		strings.Count(string(data), "is_p_normalized(a) && is_p_normalized(b)\n") != 5 {
		t.Fatal("missing source assertion records", string(data))
	}
	for line := 133; line <= 144; line++ {
		if strings.Count(string(data), "Line: "+strconv.Itoa(line)+"\n") != 1 {
			t.Fatal("missing scalar assertion source line", line, string(data))
		}
	}
	state, e := input.State()
	serviceMust(t, e)
	if state.Position != 14 {
		t.Fatal("disabled assertion consumed an action", state.Position)
	}
}
