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

func TestArbitraryFloatExtremaMixedZeroSourceUnreachable(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(r, b)
	serviceMust(t, e)
	in, e := os.CreateTemp(t.TempDir(), "debug-input-*")
	serviceMust(t, e)
	_, e = in.WriteString("TT")
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
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.Close()) }()
	x, y := extremaTestNumeral(0), extremaTestNumeral(0x80000000)
	dst := extremaTestNumeral(0x40800000)
	before := dst
	for _, tc := range []struct {
		line int
		call func() error
	}{{1493, func() error { return m.Maximum(&x, &y, &dst) }}, {1509, func() error { return m.Minimum(&x, &y, &dst) }}} {
		var thrown z3exception.DefaultException
		if e := tc.call(); !errors.As(e, &thrown) || dst != before {
			t.Fatalf("mixed zero line %d: error %v, output %+v", tc.line, e, dst)
		}
	}
	got, e := os.ReadFile(out.Name())
	serviceMust(t, e)
	const prompt = "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	want := ""
	for _, line := range []string{"1493", "1509"} {
		want += "ASSERTION VIOLATION\nFile: src/util/mpf.cpp\nLine: " + line + "\nUNREACHABLE CODE WAS REACHED.\n" + prompt
	}
	if string(got) != want {
		t.Fatalf("source diagnostic: got %q, want %q", got, want)
	}
}
