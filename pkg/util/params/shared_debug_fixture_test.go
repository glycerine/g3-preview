package params

import (
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func debugFixture(t *testing.T) (*udiag.DebugContext, *arena.BytesRegion, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	d, e := udiag.NewDebugContext(r, b)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, d.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("remaining debug graph", b.Used())
		}
	})
	return d, r, b
}

func bindProcessAssertionRefusal(t *testing.T) {
	t.Helper()
	d, _, _ := debugFixture(t)
	old, e := udiag.BindProcessDebugContext(d)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := udiag.BindProcessDebugContext(old)
		serviceMust(t, e)
	})
}

func bindProcessAssertionActions(t *testing.T, commands string) (*udiag.NumericCharInput, *os.File) {
	t.Helper()
	d, _, budget := debugFixture(t)
	in, e := os.CreateTemp(t.TempDir(), "params-debug-input-*")
	serviceMust(t, e)
	_, e = in.WriteString(commands)
	serviceMust(t, e)
	_, e = in.Seek(0, 0)
	serviceMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "params-debug-output-*")
	serviceMust(t, e)
	streams, e := udiag.NewMemoryStandardStreams(udiag.MemoryStandardStreamsConfig{Budget: budget, Stdout: out, Stderr: out})
	serviceMust(t, e)
	input, e := udiag.NewDebuggerCharInput(streams, in)
	serviceMust(t, e)
	actions, e := udiag.NewDebugActionContext(udiag.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	serviceMust(t, e)
	serviceMust(t, d.BindAssertionStreams(streams))
	serviceMust(t, d.BindAssertionActions(actions))
	old, e := udiag.BindProcessDebugContext(d)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := udiag.BindProcessDebugContext(old)
		serviceMust(t, e)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, in.Close())
		serviceMust(t, out.Close())
	})
	return input, out
}
