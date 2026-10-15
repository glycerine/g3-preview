//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTDeclCollectorDependencySetMemoryTrace(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	a, err := m.MkUninterpretedSort(astTestSymbol(t, m, "A"), nil)
	astTestMust(t, err)
	astTestPin(t, m, a)
	p, err := m.NewParameterAST(a)
	astTestMust(t, err)
	b, err := m.MkUninterpretedSort(astTestSymbol(t, m, "B"), []ASTParameter{p})
	astTestMust(t, err)
	astTestPin(t, m, b)
	astTestMust(t, m.DelParameter(&p))
	c := newASTDeclCollector(m, true)
	c.sorts = []ASTHandle{b, a}
	astTestMust(t, c.orderDeps())
	if !equalSortHandles(c.sorts, []ASTHandle{a, b}) || output.Len() != 0 {
		t.Fatalf("disabled memory trace: sorts=%v output=%q", c.sorts, output.String())
	}

	tags.Enable("memory")
	c.sorts = []ASTHandle{b, a}
	astTestMust(t, c.orderDeps())
	if !equalSortHandles(c.sorts, []ASTHandle{a, b}) {
		t.Fatalf("dependency order = %v", c.sorts)
	}
	// Allocation occurs in collected order; top_sort owns each set until the
	// completed dependency order has replaced the collector's sort list.
	size := strconv.FormatUint(uint64(unsafe.Sizeof([]ASTHandle(nil))), 10)
	record := func(function string, line int, body string) string {
		return "-------- [memory] " + function + " src/util/memory_manager.cpp:" + strconv.Itoa(line) + " ---------\n" +
			body + "\n------------------------------------------------\n"
	}
	// Each orderDeps call has its own lexical order owner. Extract the two
	// stable identities from the allocation records, then check their use in
	// the destructor records.
	var owner string
	if _, err := fmt.Sscanf(output.String(), "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s", &owner); err != nil {
		t.Fatalf("allocation record: %q: %v", output.String(), err)
	}
	// Sscanf's string ends at the space following the first owner/slot.
	base, ok := strings.CutSuffix(owner, "/0")
	if !ok {
		t.Fatalf("first dependency-set identity = %q", owner)
	}
	want := record("allocate", 209, "alloc "+base+"/0 src/ast/decl_collector.cpp:127 sort_set "+size) +
		record("allocate", 209, "alloc "+base+"/1 src/ast/decl_collector.cpp:127 sort_set "+size) +
		record("deallocate", 204, "dealloc "+base+"/0 src/util/top_sort.h:79") +
		record("deallocate", 204, "dealloc "+base+"/1 src/util/top_sort.h:79")
	if got := output.String(); got != want {
		t.Fatalf("dependency-set memory records:\n got %q\nwant %q", got, want)
	}
}
