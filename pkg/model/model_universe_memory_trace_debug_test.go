//go:build g3debug && g3trace

package model_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestModelUniverseMemoryWrapperRegistrationCopyAndClose(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "memory-universe"), nil)
	astTestMust(t, err)
	value, err := m.GetSomeValue(sort)
	astTestMust(t, err)
	original, err := model.NewModel(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, original.Close()) })
	tags.Enable("memory")
	astTestMust(t, original.RegisterUniverse(sort, []ast.ASTHandle{value}))
	first, found := original.Universes.Get2(sort)
	if !found || first == nil {
		t.Fatal("universe header not published")
	}
	copied, err := original.Copy()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, copied.Close()) })
	second, found := copied.Universes.Get2(sort)
	if !found || second == nil || second == first {
		t.Fatal("copy did not acquire an independent universe header")
	}
	astTestMust(t, copied.Close())
	astTestMust(t, original.Close())
	astTestMust(t, original.Close())

	size := strconv.FormatUint(uint64(reflect.TypeOf(first).Elem().Size()), 10)
	got := output.String()
	for _, u := range []any{first, second} {
		id := fmt.Sprintf("%p", u)
		alloc := "alloc " + id + " src/model/model.cpp:139 ptr_vector<expr> " + size + "\n"
		dealloc := "dealloc " + id + " src/model/model.cpp:38\n"
		if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 ||
			strings.Index(got, alloc) > strings.Index(got, dealloc) {
			t.Fatalf("universe memory records for %s: %q", id, got)
		}
	}
}
