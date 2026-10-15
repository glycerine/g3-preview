//go:build g3debug && g3trace

package api

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestAPIASTMapSourceMemoryWrapperReleaseSites(t *testing.T) {
	for _, tc := range []struct {
		name string
		line int
	}{
		{"zero-reference", 54},
		{"context-destructor", 116},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := astTestManager(t)
			c, err := newAPIContextResults(m, false)
			astTestMust(t, err)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("memory")
			c.trace = tags
			a, err := newAPIASTMapObject(c)
			astTestMust(t, err)
			if tc.line == 54 {
				astTestMust(t, c.saveObject(&a.object))
				astTestMust(t, a.object.incRef())
				astTestMust(t, c.assignLastObject(nil))
				astTestMust(t, a.object.decRef())
			}
			astTestMust(t, c.Close())
			ptr := fmt.Sprintf("%p", a)
			size := reflect.TypeOf(apiASTMapObject{}).Size()
			want := fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/api/api_ast_map.cpp:37 Z3_ast_map_ref %d\n------------------------------------------------\n", ptr, size) +
				fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/api/api_context.cpp:%d\n------------------------------------------------\n", ptr, tc.line)
			if got := output.String(); got != want {
				t.Fatalf("source AST-map memory records:\n got %q\nwant %q", got, want)
			}
		})
	}
}

func TestAPIContextMemoryWrappersBindOwnProcessFinalization(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	p := umemory.NewProcessAccounting()
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Errorf("API process owner retained %d bytes", b.Used())
		}
	})
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{
		Memory: r, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, q.Close()) })
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	c, err := newAPIContextResults(m, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	if c.memoryProcess != p {
		t.Fatal("API context did not bind its AST manager's numeral process")
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	c.trace = tags
	_, err = newAPIASTMapObject(c)
	astTestMust(t, err)
	_, err = newAPIGoalObject(c)
	astTestMust(t, err)
	first, err := r.BeginSourceInitialize(math.MaxUint32)
	if err != nil || !first {
		t.Fatal("source process initialization", first, err)
	}
	astTestMust(t, r.CompleteSourceInitialize())
	active, err := p.BeginFinalize()
	if err != nil || !active {
		t.Fatal("source process finalization", active, err)
	}
	astTestMust(t, c.Close())
	astTestMust(t, p.CompleteFinalize())
	if got := output.String(); strings.Count(got, "alloc ") != 2 || strings.Contains(got, "dealloc ") {
		t.Fatal("context wrapper deallocation escaped process finalization guard", got)
	}
}
