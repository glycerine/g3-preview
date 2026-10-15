//go:build g3debug && g3trace

package params

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestGlobalParamsImpMemoryTraceAllocationAndDeallocationOrder(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var out bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&out)
	tags.Enable("gparams")
	tags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(tags))
	paramsMust(t, s.InitializeSource())
	current := s.Current
	if current == nil || !current.constructorReady {
		t.Fatal("source imp constructor did not finish")
	}
	ptr, size := fmt.Sprintf("%p", current), current.OwnerBytes
	paramsMust(t, s.FinalizeSource())
	record := func(tag, function, file string, line int, body string) string {
		return fmt.Sprintf("-------- [%s] %s %s:%d ---------\n%s------------------------------------------------\n", tag, function, file, line, body)
	}
	want := record("gparams", "init", "src/util/gparams.cpp", 634, "gparams::init()\n") +
		record("memory", "allocate", "src/util/memory_manager.cpp", 209,
			fmt.Sprintf("alloc %s src/util/gparams.cpp:635 imp %d\n", ptr, size)) +
		record("gparams", "finalize", "src/util/gparams.cpp", 639, "gparams::finalize()\n") +
		record("memory", "deallocate", "src/util/memory_manager.cpp", 204,
			fmt.Sprintf("dealloc %s src/util/gparams.cpp:641\n", ptr))
	if out.String() != want {
		t.Fatalf("source wrapper order: got %q want %q", out.String(), want)
	}
}

func TestGlobalParamsImpMemoryTraceRefusalDoesNotStopLifecycle(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	out := &gparamsTraceBodyRefusalWriter{failAt: 1, cause: errors.New("memory trace refused")}
	tags := udiag.NewTraceTagsWithWriter(out)
	tags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(tags))
	paramsMust(t, s.InitializeSource())
	if s.Current == nil || !s.Current.constructorReady || out.calls == 0 {
		t.Fatal("refused allocation trace stopped source constructor", s.Current, out.calls)
	}
	paramsMust(t, s.FinalizeSource())
	if s.Current != nil {
		t.Fatal("refused deallocation trace stopped source finalization")
	}
}

func TestGlobalParamsImpMemoryTraceFinalizingSuppressesDeallocation(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var out bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(tags))
	p := umemory.NewProcessAccounting()
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: s.Config.Budget, Process: p, SizeTBits: 64})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, r.Close()) })
	first, e := r.BeginSourceInitialize(1 << 25)
	if e != nil || !first {
		t.Fatal("source process initialization", first, e)
	}
	paramsMust(t, r.CompleteSourceInitialize())
	paramsMust(t, s.BindMemoryProcess(p))
	paramsMust(t, s.InitializeSource())
	active, e := p.BeginFinalize()
	if e != nil || !active {
		t.Fatal("source finalization guard", active, e)
	}
	paramsMust(t, s.FinalizeSource())
	paramsMust(t, p.CompleteFinalize())
	if got := out.String(); !strings.Contains(got, "alloc ") || strings.Contains(got, "dealloc ") {
		t.Fatalf("finalizing deallocation wrapper fired: %q", got)
	}
}
