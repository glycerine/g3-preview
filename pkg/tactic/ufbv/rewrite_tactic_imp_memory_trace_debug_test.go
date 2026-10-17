//go:build g3debug && g3trace

package ufbv

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVRewriterImpMemoryCleanupSwapAndClose(t *testing.T) {
	b := arena.NewBudget(1 << 30)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	sy, e := symbol.NewManager(symbol.Config{Budget: b})
	cacheMust(t, e)
	q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	cacheMust(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b, TraceTags: tags})
	cacheMust(t, e)
	t.Cleanup(func() {
		cacheMust(t, m.Close())
		cacheMust(t, q.Close())
		cacheMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("UFBV tactic owner leak", b.Used())
		}
	})
	ctx, globals := ufbvTacticServicesFixture(t, m)
	start := b.Used()
	tags.Enable("memory")
	r, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.Close()) })
	first := r.imp
	size := uint64(reflect.TypeOf(rewriteTacticImp{}).Size())
	outerSize := uint64(reflect.TypeOf(RewriterTactic{}).Size())
	if r.header != outerSize {
		t.Fatal("outer tactic owner extent", r.header, outerSize)
	}
	if first == nil || first.header != size || first.boolean == nil {
		t.Fatal("initial concrete implementation owner", first)
	}
	cacheMust(t, r.Cleanup())
	second := r.imp
	if second == first || second == nil || second.header != size || first.header != 0 || len(r.retired) != 0 {
		t.Fatal("cleanup must swap before retiring the old implementation", first, second, r.retired)
	}
	translated, e := r.Translate(m, ctx, globals)
	cacheMust(t, e)
	if translated == r || translated.imp == second {
		t.Fatal("translation must own a distinct tactic and implementation")
	}
	translatedImp := translated.imp
	cacheMust(t, translated.Close())
	cacheMust(t, r.Close())
	if b.Used() != start {
		t.Fatal("implementation and tactic headers not released", b.Used(), start)
	}
	record := func(action string, line int, body string) string {
		return "-------- [memory] " + action + " src/util/memory_manager.cpp:" + strconv.Itoa(line) + " ---------\n" + body + "\n------------------------------------------------\n"
	}
	id, id1, id2 := fmt.Sprintf("%p", r), fmt.Sprintf("%p", first), fmt.Sprintf("%p", second)
	translatedID, translatedImpID := fmt.Sprintf("%p", translated), fmt.Sprintf("%p", translatedImp)
	want := record("allocate", 209, "alloc "+id+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:121 ufbv_rewriter_tactic "+strconv.FormatUint(outerSize, 10)) +
		record("allocate", 209, "alloc "+id1+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:81 imp "+strconv.FormatUint(size, 10)) +
		record("allocate", 209, "alloc "+id2+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:113 imp "+strconv.FormatUint(size, 10)) +
		record("deallocate", 204, "dealloc "+id1+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:115") +
		record("allocate", 209, "alloc "+translatedID+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:85 ufbv_rewriter_tactic "+strconv.FormatUint(outerSize, 10)) +
		record("allocate", 209, "alloc "+translatedImpID+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:81 imp "+strconv.FormatUint(size, 10)) +
		record("deallocate", 204, "dealloc "+translatedImpID+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:89") +
		record("deallocate", 204, "dealloc "+translatedID+" src/tactic/tactic.h:43") +
		record("deallocate", 204, "dealloc "+id2+" src/tactic/ufbv/ufbv_rewriter_tactic.cpp:89") +
		record("deallocate", 204, "dealloc "+id+" src/tactic/tactic.h:43")
	got := output.String()
	if got != want {
		t.Fatalf("concrete imp source memory records:\ngot  %q\nwant %q", got, want)
	}
}
