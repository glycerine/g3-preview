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

func TestArbitraryFloatSBVInvalidModeSourceUnreachable(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(r, b)
	serviceMust(t, e)
	in, e := os.CreateTemp(t.TempDir(), "debug-input-*")
	serviceMust(t, e)
	_, e = in.WriteString("T")
	serviceMust(t, e)
	_, e = in.Seek(0, 0)
	serviceMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "debug-output-*")
	serviceMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stdout: out, Stderr: out})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, in)
	serviceMust(t, e)
	actions, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: b, Streams: streams, Input: input})
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
		serviceMust(t, r.Close())
	})
	serviceMust(t, d.EnableAssertions(false))
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 3, 1.5))
	result := NewRational()
	before := result
	var thrown z3exception.DefaultException
	if e := m.ToSBVMPQ(ArbitraryFloatRoundingMode(255), &x, &result); !errors.As(e, &thrown) || result != before {
		t.Fatalf("invalid negative-exponent rounding: %v, %+v", e, result)
	}
	serviceMust(t, m.CleanupToSBVMPQ())
	serviceMust(t, m.Del(&x))
	got, e := os.ReadFile(out.Name())
	serviceMust(t, e)
	const want = "ASSERTION VIOLATION\nFile: src/util/mpf.cpp\nLine: 1211\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(got) != want {
		t.Fatalf("source diagnostic: got %q, want %q", got, want)
	}
}
