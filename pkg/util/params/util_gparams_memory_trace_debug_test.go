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

func TestGlobalParamsModuleCellMemoryTraceAllocationReuseAndReset(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var out bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(tags))
	paramsMust(t, s.InitializeSource())
	out.Reset() // inspect the separate alloc(params_ref) site
	module, e := s.Symbols.Intern("memory-cell-module")
	paramsMust(t, e)
	first, e := s.Current.GetParams(module)
	paramsMust(t, e)
	cell := s.Current.Cells[0]
	if cell == nil || first != &cell.value {
		t.Fatal("source module cell was not published", first, cell)
	}
	again, e := s.Current.GetParams(module)
	paramsMust(t, e)
	if again != first {
		t.Fatal("module lookup allocated a second cell")
	}
	paramsMust(t, s.Current.Reset())
	ptr := fmt.Sprintf("%p", cell)
	want := fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/util/gparams.cpp:192 params_ref %d\n------------------------------------------------\n", ptr, gparamsRefCellBytes) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/util/gparams.cpp:117\n------------------------------------------------\n", ptr)
	if out.String() != want {
		t.Fatalf("source module-cell wrapper records: got %q want %q", out.String(), want)
	}
	out.Reset()
	_, e = s.Current.GetParams(module)
	paramsMust(t, e)
	second := s.Current.Cells[0]
	imp := s.Current
	paramsMust(t, s.FinalizeSource())
	secondPtr := fmt.Sprintf("%p", second)
	want = fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/util/gparams.cpp:192 params_ref %d\n------------------------------------------------\n", secondPtr, gparamsRefCellBytes) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/util/gparams.cpp:117\n------------------------------------------------\n", secondPtr) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %p src/util/gparams.cpp:641\n------------------------------------------------\n", imp)
	if out.String() != want {
		t.Fatalf("source destructor module-cell wrapper records: got %q want %q", out.String(), want)
	}
}

func TestGlobalParamsModuleCellMemoryTraceLateBindings(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var oldOut, newOut bytes.Buffer
	oldTags := udiag.NewTraceTagsWithWriter(&oldOut)
	oldTags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(oldTags))
	paramsMust(t, s.InitializeSource())
	oldOut.Reset()
	newTags := udiag.NewTraceTagsWithWriter(&newOut)
	newTags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(newTags))
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
	module, e := s.Symbols.Intern("late-bound-memory-cell")
	paramsMust(t, e)
	_, e = s.Current.GetParams(module)
	paramsMust(t, e)
	if oldOut.Len() != 0 || !strings.Contains(newOut.String(), "src/util/gparams.cpp:192 params_ref ") {
		t.Fatalf("late trace binding did not reach the current module owner: old %q new %q", oldOut.String(), newOut.String())
	}
	active, e := p.BeginFinalize()
	if e != nil || !active {
		t.Fatal("source finalization guard", active, e)
	}
	paramsMust(t, s.FinalizeSource())
	paramsMust(t, p.CompleteFinalize())
	if oldOut.Len() != 0 || strings.Contains(newOut.String(), "dealloc ") {
		t.Fatalf("late process binding did not suppress deallocation: old %q new %q", oldOut.String(), newOut.String())
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
	module, e := s.Symbols.Intern("refused-memory-cell")
	paramsMust(t, e)
	_, e = s.Current.GetParams(module)
	paramsMust(t, e)
	paramsMust(t, s.Current.Reset())
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
	module, e := s.Symbols.Intern("finalizing-memory-cell")
	paramsMust(t, e)
	_, e = s.Current.GetParams(module)
	paramsMust(t, e)
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
