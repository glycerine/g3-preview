package ast

import (
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func astUnsignedGrowthPrime(t *testing.T, r *umemory.Runtime) umemory.Block {
	t.Helper()
	astTestMust(t, r.SetMaxSize(0))
	request := int64(99990) - r.State().LocalSize - 8
	if request <= 0 {
		t.Fatal("public bounded TLS priming", r.State())
	}
	h, e := r.Allocate(uint64(request))
	astTestMust(t, e)
	astTestMust(t, r.SetMaxSize(1))
	return h
}

// The diagnostic fixture stays with the AST tests and borrows only public utilities.
func astDebugActionFixture(t *testing.T, text string) (*arena.Budget, *umemory.Runtime, *udiag.DebugContext, *udiag.NumericCharInput, *os.File) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	astTestMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	astTestMust(t, e)
	d, e := udiag.NewDebugContext(region, b)
	astTestMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		astTestMust(t, d.Close())
		astTestMust(t, region.Close())
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("unnamed diagnostic owner", b.Used())
		}
	})
	inputFile, e := os.CreateTemp(t.TempDir(), "debug-action-input-*")
	astTestMust(t, e)
	_, e = inputFile.WriteString(text)
	astTestMust(t, e)
	_, e = inputFile.Seek(0, 0)
	astTestMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "debug-action-output-*")
	astTestMust(t, e)
	streams, e := udiag.NewMemoryStandardStreams(udiag.MemoryStandardStreamsConfig{Budget: b, Stdout: out, Stderr: out})
	astTestMust(t, e)
	input, e := udiag.NewDebuggerCharInput(streams, inputFile)
	astTestMust(t, e)
	action, e := udiag.NewDebugActionContext(udiag.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	astTestMust(t, e)
	astTestMust(t, d.BindAssertionStreams(streams))
	astTestMust(t, d.BindAssertionActions(action))
	t.Cleanup(func() {
		b.FailAfter(-1)
		astTestMust(t, action.Close())
		astTestMust(t, input.Close())
		astTestMust(t, streams.Close())
		astTestMust(t, inputFile.Close())
		astTestMust(t, out.Close())
	})
	return b, r, d, input, out
}
