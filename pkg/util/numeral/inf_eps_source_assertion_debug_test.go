//go:build g3debug

// Source: src/util/inf_eps_rational.h:121-128; src/util/debug.h:63.
package numeral

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestInfEpsAndInfRational64BitSourceAssertionActions(t *testing.T) {
	s, _ := serviceNew(t)
	v := infEpsValue(t, s, "1", "7", "0")
	t.Cleanup(func() { serviceMust(t, s.InfEpsInfDel(&v)) })
	inner := infValue(t, s, 7, 1, "1")
	t.Cleanup(func() { serviceMust(t, s.InfDel(&inner)) })

	budget := arena.NewBudget(^uint64(0))
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	inFile, e := os.CreateTemp(t.TempDir(), "inf-eps-actions-*")
	serviceMust(t, e)
	_, e = inFile.WriteString("CTCT")
	serviceMust(t, e)
	_, e = inFile.Seek(0, 0)
	serviceMust(t, e)
	outFile, e := os.CreateTemp(t.TempDir(), "inf-eps-records-*")
	serviceMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: outFile, Stderr: outFile})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, inFile)
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
		serviceMust(t, inFile.Close())
		serviceMust(t, outFile.Close())
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
	})

	if got, e := s.InfEpsInfInt64(&v); e != nil || got != 7 {
		t.Fatal("Continue must reach nested signed getter", got, e)
	}
	var thrown z3exception.DefaultException
	if _, e := s.InfEpsInfUint64(&v); !errors.As(e, &thrown) {
		t.Fatal("Throw must stop before nested unsigned getter", e)
	}
	if got, e := s.InfInt64(&inner); e != nil || got != 7 {
		t.Fatal("inf_rational Continue must reach FIRST getter", got, e)
	}
	if _, e := s.InfUint64(&inner); !errors.As(e, &thrown) {
		t.Fatal("inf_rational Throw must stop before FIRST getter", e)
	}
	serviceMust(t, d.EnableAssertions(false))
	if got, e := s.InfEpsInfUint64(&v); e != nil || got != 7 {
		t.Fatal("disabled source assertion must reach nested getter", got, e)
	}
	if got, e := s.InfUint64(&inner); e != nil || got != 7 {
		t.Fatal("disabled inf_rational assertion must reach FIRST", got, e)
	}
	state, e := input.State()
	serviceMust(t, e)
	if state.Position != 4 {
		t.Fatal("unexpected debugger action consumption", state.Position)
	}
	got, e := os.ReadFile(outFile.Name())
	serviceMust(t, e)
	prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	want := "ASSERTION VIOLATION\nFile: src/util/inf_eps_rational.h\nLine: 122\nis_int64()\n" + prompt +
		"ASSERTION VIOLATION\nFile: src/util/inf_eps_rational.h\nLine: 127\nis_uint64()\n" + prompt +
		"ASSERTION VIOLATION\nFile: src/util/inf_rational.h\nLine: 126\nis_int64()\n" + prompt +
		"ASSERTION VIOLATION\nFile: src/util/inf_rational.h\nLine: 131\nis_uint64()\n" + prompt
	if string(got) != want {
		t.Fatalf("source assertion records: got %q, want %q", got, want)
	}
}
