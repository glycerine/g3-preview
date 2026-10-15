//go:build g3debug && g3trace

package tactic

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

func TestFilterModelMemoryWrapperFollowsReplacementOwner(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	old, err := model.NewModel(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, old.Close()) })
	var out ModelRef
	astTestMust(t, out.Set(old.Core()))
	t.Cleanup(func() { astTestMust(t, out.Close()) })
	filter, err := NewFilterModel(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, filter.Close()) })
	tags.Enable("memory")
	astTestMust(t, filter.ApplyUnindexed(&out))
	core, err := out.Get()
	astTestMust(t, err)
	if core == nil || core.DerivedModel() == nil || core.DerivedModel() == old {
		t.Fatal("filter did not publish an independent model")
	}
	newModel := core.DerivedModel()
	id := fmt.Sprintf("%p", newModel)
	size := strconv.FormatUint(uint64(reflect.TypeOf(model.Model{}).Size()), 10)
	alloc := "alloc " + id + " src/tactic/filter_model_converter.cpp:31 model " + size + "\n"
	dealloc := "dealloc " + id + " src/model/model_core.h:74\n"
	if strings.Count(output.String(), alloc) != 1 || strings.Contains(output.String(), dealloc) {
		t.Fatalf("filter allocation before release: %q", output.String())
	}
	astTestMust(t, out.Close())
	astTestMust(t, out.Close())
	got := output.String()
	if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 ||
		strings.Index(got, alloc) > strings.Index(got, dealloc) {
		t.Fatalf("filter model memory records: %q", got)
	}
}
