//go:build g3debug && g3trace

package symbol

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestSymbolServiceMemoryTraceAllocReuseAndFinalize(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, err := NewService(Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	if err = s.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeSource(); err != nil || out.Len() != 0 {
		t.Fatal("null source pointer deallocation", err, out.String())
	}
	if err = s.InitializeSource(); err != nil {
		t.Fatal(err)
	}
	m, err := s.Manager()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InitializeSource(); err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeSource(); err != nil {
		t.Fatal(err)
	}
	ptr := fmt.Sprintf("%p", m)
	want := fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/util/symbol.cpp:67 internal_symbol_table %d\n------------------------------------------------\n", ptr, reflect.TypeOf(Manager{}).Size()) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/util/symbol.cpp:72\n------------------------------------------------\n", ptr)
	if out.String() != want {
		t.Fatalf("symbol table source memory wrappers: got %q want %q", out.String(), want)
	}
	if err = s.Close(); err != nil || b.Used() != 0 || out.String() != want {
		t.Fatal("closed symbol service replayed deallocation", err, b.Used(), out.String())
	}
}

func TestSymbolServiceMemoryTraceFinalizingSuppressesDeallocation(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.BeginSourceInitialize(1 << 25)
	if err != nil || !first {
		t.Fatal("source process initialization", first, err)
	}
	if err = r.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
	s, err := NewService(Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	if err = s.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err = s.BindMemoryProcess(p); err != nil {
		t.Fatal(err)
	}
	if err = s.InitializeSource(); err != nil {
		t.Fatal(err)
	}
	active, err := p.BeginFinalize()
	if err != nil || !active {
		t.Fatal("source process finalization", active, err)
	}
	if err = s.FinalizeSource(); err != nil {
		t.Fatal(err)
	}
	if err = p.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "src/util/symbol.cpp:67 internal_symbol_table ") || strings.Contains(out.String(), "dealloc ") {
		t.Fatal("symbol deallocation escaped process-finalizing guard", out.String())
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil || b.Used() != 0 {
		t.Fatal("symbol/process owner cleanup", err, b.Used())
	}
}

type symbolMemoryRefusalWriter struct{ calls int }

func (w *symbolMemoryRefusalWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, errors.New("trace stream refused")
}

func TestSymbolServiceMemoryTraceRefusalDoesNotStopLifecycle(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, err := NewService(Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	out := &symbolMemoryRefusalWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("memory")
	if err = s.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err = s.InitializeSource(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Manager(); err != nil || out.calls == 0 {
		t.Fatal("refused allocation trace stopped constructor", err, out.calls)
	}
	if err = s.FinalizeSource(); err != nil {
		t.Fatal("refused deallocation trace stopped destructor", err)
	}
	if err = s.Close(); err != nil || b.Used() != 0 {
		t.Fatal("refused trace prevented owner release", err, b.Used())
	}
}
