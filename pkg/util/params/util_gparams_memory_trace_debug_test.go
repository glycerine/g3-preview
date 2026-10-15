//go:build g3debug && g3trace

package params

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
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
	descriptorImp := fmt.Sprintf("%p", &current.GlobalDescriptors.entries)
	paramsMust(t, s.FinalizeSource())
	record := func(tag, function, file string, line int, body string) string {
		return fmt.Sprintf("-------- [%s] %s %s:%d ---------\n%s------------------------------------------------\n", tag, function, file, line, body)
	}
	want := record("gparams", "init", "src/util/gparams.cpp", 634, "gparams::init()\n") +
		record("memory", "allocate", "src/util/memory_manager.cpp", 209,
			fmt.Sprintf("alloc %s src/util/gparams.cpp:635 imp %d\n", ptr, size)) +
		record("memory", "allocate", "src/util/memory_manager.cpp", 209,
			fmt.Sprintf("alloc %s src/util/params.cpp:211 imp %d\n", descriptorImp, reflect.TypeOf(ParamDescrs{}).Size())) +
		record("gparams", "finalize", "src/util/gparams.cpp", 639, "gparams::finalize()\n") +
		record("memory", "deallocate", "src/util/memory_manager.cpp", 204,
			fmt.Sprintf("dealloc %s src/util/params.cpp:215\n", descriptorImp)) +
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
	descriptorImp := fmt.Sprintf("%p", &imp.GlobalDescriptors.entries)
	paramsMust(t, s.FinalizeSource())
	secondPtr := fmt.Sprintf("%p", second)
	want = fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/util/gparams.cpp:192 params_ref %d\n------------------------------------------------\n", secondPtr, gparamsRefCellBytes) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/util/gparams.cpp:117\n------------------------------------------------\n", secondPtr) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/util/params.cpp:215\n------------------------------------------------\n", descriptorImp) +
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
	paramsMust(t, s.Current.ensureRegistered())
	if oldOut.Len() != 0 || !strings.Contains(newOut.String(), "src/util/gparams.cpp:192 params_ref ") || !strings.Contains(newOut.String(), "src/util/params.cpp:211 imp ") {
		t.Fatalf("late trace binding did not reach the current module owner: old %q new %q", oldOut.String(), newOut.String())
	}
	newOut.Reset() // duplicate-merge deletes precede process finalization
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

func TestGlobalParamsCatalogDescriptorMemoryTrace(t *testing.T) {
	s, _ := gparamsNamespaceFixture(t)
	var out bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	paramsMust(t, s.BindNamespaceTrace(tags))
	paramsMust(t, s.InitializeSource())
	out.Reset()
	g := s.Current
	paramsMust(t, g.ensureRegistered())
	registered := out.String()
	if len(g.Descriptors) != 19 || strings.Count(registered, " param_descrs ") != 32 || strings.Count(registered, " src/util/gparams.cpp:143\n") != 13 || strings.Count(registered, " src/util/params.cpp:211 imp ") != 34 || strings.Count(registered, " src/util/params.cpp:215\n") != 15 {
		t.Fatalf("catalog descriptor allocation/duplicate deletion count: owners %d outer allocs %d duplicate deletes %d inner allocs %d inner deletes %d", len(g.Descriptors), strings.Count(registered, " param_descrs "), strings.Count(registered, " src/util/gparams.cpp:143\n"), strings.Count(registered, " src/util/params.cpp:211 imp "), strings.Count(registered, " src/util/params.cpp:215\n"))
	}
	first := g.Descriptors[0]
	firstAlloc := fmt.Sprintf("alloc %p src/shell/gparams_register_modules.cpp:40 param_descrs %d\n", first, first.OwnerBytes)
	if !strings.Contains(registered, firstAlloc) {
		t.Fatalf("first source catalog allocation missing: %q", firstAlloc)
	}
	firstInner := fmt.Sprintf("alloc %p src/util/params.cpp:211 imp %d\n", &first.entries, reflect.TypeOf(ParamDescrs{}).Size())
	if outer, inner := strings.Index(registered, firstAlloc), strings.Index(registered, firstInner); outer < 0 || inner <= outer {
		t.Fatal("first module descriptor imp allocation preceded its outer source allocation", outer, inner)
	}
	for line := 40; line <= 71; line++ {
		if strings.Count(registered, fmt.Sprintf(" src/shell/gparams_register_modules.cpp:%d param_descrs ", line)) != 1 {
			t.Fatalf("source catalog allocation line %d absent or repeated", line)
		}
	}
	if a, b, c, d := strings.Index(registered, "gparams_register_modules.cpp:40 param_descrs "), strings.Index(registered, "gparams_register_modules.cpp:41 param_descrs "), strings.Index(registered, "gparams_register_modules.cpp:42 param_descrs "), strings.Index(registered, " src/util/gparams.cpp:143\n"); !(a >= 0 && a < b && b < c && c < d) {
		t.Fatal("source catalog allocation and duplicate-merge deletion order", a, b, c, d)
	}
	out.Reset()
	paramsMust(t, s.FinalizeSource())
	finalized := out.String()
	if strings.Count(finalized, " src/util/gparams.cpp:108\n") != 19 || strings.Count(finalized, " src/util/params.cpp:215\n") != 20 || !strings.Contains(finalized, fmt.Sprintf("dealloc %p src/util/gparams.cpp:108\n", first)) {
		t.Fatal("source destructor did not release retained descriptor owners", finalized)
	}
	if inner, outer := strings.Index(finalized, fmt.Sprintf("dealloc %p src/util/params.cpp:215\n", &first.entries)), strings.Index(finalized, fmt.Sprintf("dealloc %p src/util/gparams.cpp:108\n", first)); inner < 0 || outer <= inner {
		t.Fatal("source descriptor imp destructor did not precede outer deallocation", inner, outer)
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
	paramsMust(t, s.Current.ensureRegistered())
	module, e := s.Symbols.Intern("refused-memory-cell")
	paramsMust(t, e)
	value, e := s.Current.GetParams(module)
	paramsMust(t, e)
	key, e := s.Symbols.Intern("refused-memory-block")
	paramsMust(t, e)
	paramsMust(t, s.Context.SetBool(value, key, true))
	numeralKey, e := s.Symbols.Intern("refused-memory-rational")
	paramsMust(t, e)
	r, e := s.Context.NumeralsField.NewValue()
	paramsMust(t, e)
	paramsMust(t, s.Context.NumeralsField.AssignInt(&r, 7))
	paramsMust(t, s.Context.SetRat(value, numeralKey, &r))
	paramsMust(t, s.Context.NumeralsField.Del(&r))
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
	value, e := s.Current.GetParams(module)
	paramsMust(t, e)
	key, e := s.Symbols.Intern("finalizing-memory-block")
	paramsMust(t, e)
	paramsMust(t, s.Context.SetBool(value, key, true))
	numeralKey, e := s.Symbols.Intern("finalizing-memory-rational")
	paramsMust(t, e)
	rational, e := s.Context.NumeralsField.NewValue()
	paramsMust(t, e)
	paramsMust(t, s.Context.NumeralsField.AssignInt(&rational, 7))
	paramsMust(t, s.Context.SetRat(value, numeralKey, &rational))
	paramsMust(t, s.Context.NumeralsField.Del(&rational))
	paramsMust(t, s.Current.ensureRegistered())
	out.Reset() // duplicate-merge deletions precede process finalization
	active, e := p.BeginFinalize()
	if e != nil || !active {
		t.Fatal("source finalization guard", active, e)
	}
	paramsMust(t, s.FinalizeSource())
	paramsMust(t, p.CompleteFinalize())
	if got := out.String(); strings.Contains(got, "dealloc ") {
		t.Fatalf("finalizing deallocation wrapper fired: %q", got)
	}
}
