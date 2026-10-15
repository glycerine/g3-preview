//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTBuiltinPluginMemoryWrappersFollowBothManagerDestructors(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	child := m.Format
	if child == nil {
		t.Fatal("source format manager was not constructed")
	}
	size := strconv.FormatUint(uint64(unsafe.Sizeof(astPluginBinding{})), 10)
	for _, owner := range []*ASTManager{child, m} {
		for id, site := range astBuiltinPluginAllocation {
			alloc := "alloc " + owner.builtinPluginTraceIdentity(id) + " src/ast/ast.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + size + "\n"
			if strings.Count(output.String(), alloc) != 1 {
				t.Fatalf("builtin allocation before close: %q: %q", alloc, output.String())
			}
		}
	}
	astTestMust(t, m.Close())
	astTestMust(t, m.Close())
	got := output.String()
	for _, owner := range []*ASTManager{child, m} {
		previous := -1
		for id, site := range astBuiltinPluginAllocation {
			identity := owner.builtinPluginTraceIdentity(id)
			alloc := "alloc " + identity + " src/ast/ast.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + size + "\n"
			dealloc := "dealloc " + identity + " src/ast/ast.cpp:1412\n"
			ai, di := strings.Index(got, alloc), strings.Index(got, dealloc)
			if ai < 0 || di <= ai || strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 || di <= previous {
				t.Fatalf("builtin lifetime/order for %q: %q", identity, got)
			}
			previous = di
		}
	}
}
