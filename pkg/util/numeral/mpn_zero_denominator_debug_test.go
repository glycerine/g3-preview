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

func TestNaturalSourceAllZeroDenominatorDebugAction(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, err)
	d, err := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, err)
	in, err := os.CreateTemp(t.TempDir(), "mpn-debug-input-*")
	serviceMust(t, err)
	_, err = in.WriteString("TTC")
	serviceMust(t, err)
	_, err = in.Seek(0, 0)
	serviceMust(t, err)
	out, err := os.CreateTemp(t.TempDir(), "mpn-debug-output-*")
	serviceMust(t, err)
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: out, Stderr: out})
	serviceMust(t, err)
	input, err := diagnostic.NewDebuggerCharInput(streams, in)
	serviceMust(t, err)
	actions, err := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	serviceMust(t, err)
	serviceMust(t, d.BindAssertionStreams(streams))
	serviceMust(t, d.BindAssertionActions(actions))
	old, err := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, err)
	t.Cleanup(func() {
		_, err := diagnostic.BindProcessDebugContext(old)
		serviceMust(t, err)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, in.Close())
		serviceMust(t, out.Close())
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
	})
	serviceMust(t, d.EnableAssertions(false))
	n := naturalManager(t)
	q, r := []uint32{99}, []uint32{88}
	_, err = n.divide(limbView{external: []uint32{7}, size: 1}, limbView{external: []uint32{0}, size: 1}, limbView{external: q, size: 1}, limbView{external: r, size: 1})
	var thrown z3exception.DefaultException
	if !errors.As(err, &thrown) || q[0] != 99 || r[0] != 88 {
		t.Fatal("debug all-zero denominator guard", err, q, r)
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	if !strings.Contains(string(contents), "File: src/util/mpn.cpp\nLine: 174\nUNREACHABLE CODE WAS REACHED.") {
		t.Fatal("wrong source diagnostic", string(contents))
	}
	serviceMust(t, d.EnableAssertions(true))
	bad := limbView{external: []uint32{1, 0}, size: 2}
	a := limbView{external: []uint32{7, 1}, size: 2}
	q, r = []uint32{99}, []uint32{88, 88}
	_, err = n.divide(a, bad, limbView{external: q, size: 1}, limbView{external: r, size: 2})
	if !errors.As(err, &thrown) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("top-limb source assertion Throw", err, q, r)
	}
	contents, err = os.ReadFile(out.Name())
	serviceMust(t, err)
	if !strings.Contains(string(contents), "File: src/util/mpn.cpp\nLine: 179\ndenom[lden-1] != 0") {
		t.Fatal("wrong top-limb assertion", string(contents))
	}
	_, err = n.divide(a, bad, limbView{external: q, size: 1}, limbView{external: r, size: 2})
	if !errors.Is(err, arena.ErrBounds) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("Continue must stop at first undefined shift", err, q, r)
	}
	serviceMust(t, d.EnableAssertions(false))
	_, err = n.divide(a, bad, limbView{external: q, size: 1}, limbView{external: r, size: 2})
	if !errors.Is(err, arena.ErrBounds) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("disabled assertion must stop at first undefined shift", err, q, r)
	}
}
