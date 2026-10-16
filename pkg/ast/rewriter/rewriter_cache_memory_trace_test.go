package rewriter

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRewriterCacheMemoryWrapperSourceSites(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.Close()) }()
	tags.Enable("memory")
	c, err := newRewriterCore(m, true)
	astTestMust(t, err)
	astTestMust(t, c.beginScope())
	astTestMust(t, c.endScope())
	astTestMust(t, c.beginScope()) // Reuses the source cache; no second allocation.
	astTestMust(t, c.endScope())
	if len(c.cacheStack) != 2 || len(c.proofCacheStack) != 2 {
		t.Fatal("source base and scoped cache owners")
	}
	owners := []*ast.ActivityCache{c.cacheStack[0], c.proofCacheStack[0], c.cacheStack[1], c.proofCacheStack[1]}
	lines := []int{25, 29, 99, 101}
	astTestMust(t, c.Close())
	astTestMust(t, c.Close())
	if !diagnostic.DebugBuild {
		if output.Len() != 0 {
			t.Fatal("source Z3DEBUG memory wrappers must erase", output.String())
		}
		return
	}
	got := output.String()
	if strings.Count(got, "[memory] allocate ") != 4 || strings.Count(got, "[memory] deallocate ") != 4 {
		t.Fatal("one source record per actual cache allocation/deallocation", got)
	}
	size := strconv.FormatUint(uint64(reflect.TypeOf(owners[0]).Elem().Size()), 10)
	for i, cache := range owners {
		id := fmt.Sprintf("%p", cache)
		alloc := "alloc " + id + " src/ast/rewriter/rewriter.cpp:" + strconv.Itoa(lines[i]) + " cache " + size + "\n"
		dealloc := "dealloc " + id + " src/util/util.h:171\n"
		if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 {
			t.Fatalf("source cache %d trace mismatch: %q", i, got)
		}
	}
}
