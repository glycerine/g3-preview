//go:build g3debug && g3trace

package numeral

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalGlobalMemoryTraceAllocationReuseAndFinalization(t *testing.T) {
	g, _, _ := rationalGlobalFixture(t)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	serviceMust(t, g.BindMemoryTrace(tags))
	serviceMust(t, g.InitializeSource())
	m, e := g.Manager()
	serviceMust(t, e)
	serviceMust(t, g.InitializeSource())
	serviceMust(t, g.FinalizeSource())
	ptr := fmt.Sprintf("%p", m)
	want := fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/util/rational.cpp:66 synch_mpq_manager %d\n------------------------------------------------\n", ptr, m.HeaderBytes) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/util/rational.cpp:82\n------------------------------------------------\n", ptr)
	if out.String() != want {
		t.Fatalf("source rational manager memory wrappers: got %q want %q", out.String(), want)
	}
}

func TestRationalGlobalMemoryTraceFinalizingSuppressesDeallocation(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	p := umemory.NewProcessAccounting()
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	serviceMust(t, e)
	g, e := NewRationalGlobalService(RationalServiceConfig{Integers: IntegerConfig{Memory: r,
		Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		rationalGlobalFinish(t, g)
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Error("actual rational global roots", b.Used())
		}
	})
	first, e := r.BeginSourceInitialize(1 << 25)
	if e != nil || !first {
		t.Fatal("source process initialization", first, e)
	}
	serviceMust(t, r.CompleteSourceInitialize())
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	serviceMust(t, g.BindMemoryTrace(tags))
	serviceMust(t, g.InitializeSource())
	active, e := p.BeginFinalize()
	if e != nil || !active {
		t.Fatal("source process finalization", active, e)
	}
	serviceMust(t, g.FinalizeSource())
	serviceMust(t, p.CompleteFinalize())
	if !strings.Contains(out.String(), "src/util/rational.cpp:66 synch_mpq_manager ") || strings.Contains(out.String(), "dealloc ") {
		t.Fatal("source rational manager deallocation escaped finalizing guard", out.String())
	}
}

type rationalGlobalTraceRefusalWriter struct{ calls int }

func (w *rationalGlobalTraceRefusalWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, errors.New("trace stream refused")
}

func TestRationalGlobalMemoryTraceRefusalDoesNotStopLifecycle(t *testing.T) {
	g, _, _ := rationalGlobalFixture(t)
	out := &rationalGlobalTraceRefusalWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("memory")
	serviceMust(t, g.BindMemoryTrace(tags))
	serviceMust(t, g.InitializeSource())
	if _, e := g.Manager(); e != nil || out.calls == 0 {
		t.Fatal("refused allocation trace stopped manager publication", e, out.calls)
	}
	serviceMust(t, g.FinalizeSource())
}
