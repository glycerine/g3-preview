//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTTraceFileMemoryWrapperSourceLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	name := uparams.LiteralParamsStringView(filepath.Join(t.TempDir(), "ast-trace.log"))
	f, err := newASTTraceFileBorrowed(b, name, tags)
	astTestMust(t, err)
	astTestMust(t, f.Close())
	if output.Len() != 0 || b.Used() != 0 {
		t.Fatalf("disabled source memory tag: output=%q used=%d", output.String(), b.Used())
	}
	b.FailAfter(0)
	tags.Enable("memory")
	if refused, err := newASTTraceFileBorrowed(b, name, tags); refused != nil || err == nil || output.Len() != 0 {
		t.Fatalf("refused header emitted alloc: owner=%p err=%v output=%q", refused, err, output.String())
	}
	b.FailAfter(-1)
	f, err = newASTTraceFileBorrowed(b, name, tags)
	astTestMust(t, err)
	astTestMust(t, f.Close())
	astTestMust(t, f.Close())
	size := strconv.FormatUint(uint64(reflect.TypeOf(astTraceFile{}).Size()), 10)
	id := fmt.Sprintf("%p", f)
	want := "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n" +
		"alloc " + id + " src/ast/ast.cpp:1308 std::fstream " + size + "\n" +
		"------------------------------------------------\n" +
		"-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\n" +
		"dealloc " + id + " src/ast/ast.cpp:1478\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want || b.Used() != 0 {
		t.Fatalf("source file memory records:\n got %q\nwant %q\nused %d", got, want, b.Used())
	}
}

func TestASTTraceFileMemoryWrapperFinalizationSuppressesDeallocation(t *testing.T) {
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: arena.NewBudget(1024), Process: p})
	astTestMust(t, err)
	defer r.Close()
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatalf("source initialization: first=%v err=%v", first, err)
	}
	astTestMust(t, r.CompleteSourceInitialize())
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	astTestMust(t, tags.BindMemoryProcess(p))
	tags.Enable("memory")
	b := arena.NewBudget(1 << 20)
	name := uparams.LiteralParamsStringView(filepath.Join(t.TempDir(), "ast-finalize.log"))
	f, err := newASTTraceFileBorrowed(b, name, tags)
	astTestMust(t, err)
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatalf("source finalization: active=%v err=%v", active, err)
	}
	astTestMust(t, f.Close())
	if got := output.String(); !bytes.Contains([]byte(got), []byte("alloc ")) || bytes.Contains([]byte(got), []byte("dealloc ")) || b.Used() != 0 {
		t.Fatalf("finalizing memory records: output=%q used=%d", got, b.Used())
	}
	astTestMust(t, p.CompleteFinalize())
}
