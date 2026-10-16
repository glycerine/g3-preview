//go:build g3debug

package numeral

import (
	"errors"
	"os"
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
	_, e = in.WriteString("CT")
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
	serviceMust(t, d.EnableAssertions(false))
	if got, e := m.String(&a); e != nil || got != "8" {
		t.Fatal("disabled source assertion must leave numeral readable", got, e)
	}
	data, e := os.ReadFile(out.Name())
	serviceMust(t, e)
	if strings.Count(string(data), "File: src/util/mpzzp.h\nLine: 146\n") != 2 || strings.Count(string(data), "is_p_normalized(a)") != 2 {
		t.Fatal("missing source assertion records", string(data))
	}
	state, e := input.State()
	serviceMust(t, e)
	if state.Position != 2 {
		t.Fatal("disabled assertion consumed an action", state.Position)
	}
}
