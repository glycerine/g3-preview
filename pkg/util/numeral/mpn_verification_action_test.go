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

func bindMPNDebugActions(t *testing.T, commands string) (*diagnostic.DebugContext, *os.File) {
	t.Helper()
	if !modularSourceDebug {
		return nil, nil
	}
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, err)
	d, err := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, err)
	in, err := os.CreateTemp(t.TempDir(), "mpn-verification-input-*")
	serviceMust(t, err)
	_, err = in.WriteString(commands)
	serviceMust(t, err)
	_, err = in.Seek(0, 0)
	serviceMust(t, err)
	out, err := os.CreateTemp(t.TempDir(), "mpn-verification-output-*")
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
	return d, out
}

func isMPNVerificationThrow(err error) bool {
	var thrown z3exception.DefaultException
	return errors.As(err, &thrown)
}

func TestMPNDivisionVerificationSourceActionAndAssertionSwitch(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG verification only")
	}
	d, out := bindMPNDebugActions(t, "TC")
	m := naturalManager(t)
	view := func(n uint32) limbView { return limbView{external: []uint32{n}, size: 1} }
	bad := func() error { return m.sourceVerifyDivision(view(10), view(2), view(4), view(0)) }
	if err := bad(); !isMPNVerificationThrow(err) || len(m.verificationScratch) != 0 {
		t.Fatal("source Throw and lexical cleanup", err)
	}
	if err := bad(); err != nil || len(m.verificationScratch) != 0 {
		t.Fatal("source Continue and lexical cleanup", err)
	}
	serviceMust(t, d.EnableAssertions(false))
	if err := bad(); err != nil || len(m.verificationScratch) != 0 {
		t.Fatal("disabled source assertion", err)
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	if got := strings.Count(string(contents), "File: src/util/mpn.cpp\nLine: 216\nok\n"); got != 2 {
		t.Fatal("source assertion count/order", got, string(contents))
	}
	serviceMust(t, d.Close())
	if err := m.sourceVerifyDivision(view(10), view(2), view(5), view(0)); !errors.Is(err, arena.ErrClosed) || len(m.verificationScratch) != 0 {
		t.Fatal("successful verification must read source assertion switch", err)
	}
}
