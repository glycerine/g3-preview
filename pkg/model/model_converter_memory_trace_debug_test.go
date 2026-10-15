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
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestModelConverterMemoryWrapperFollowsReferenceOwner(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	var out tactic.ModelRef
	t.Cleanup(func() { astTestMust(t, out.Close()) })
	tags.Enable("memory")
	astTestMust(t, tactic.ModelConverterToModel(m, nil, &out))
	if output.Len() != 0 {
		t.Fatalf("nil converter allocated a model: %q", output.String())
	}
	calls := []string{}
	leaf := &modelConverterTestLeaf{name: "memory", calls: &calls}
	astTestMust(t, tactic.ModelConverterToModel(m, leaf, &out))
	core, err := out.Get()
	astTestMust(t, err)
	if core == nil || core.Derived == nil || len(calls) != 1 || calls[0] != "model:memory:0" {
		t.Fatalf("converter did not receive allocated model: %v %v", core, calls)
	}
	md := core.Derived
	id := fmt.Sprintf("%p", md)
	size := strconv.FormatUint(uint64(reflect.TypeOf(model.Model{}).Size()), 10)
	alloc := "alloc " + id + " src/tactic/model_converter.cpp:178 model " + size + "\n"
	dealloc := "dealloc " + id + " src/model/model_core.h:74\n"
	if strings.Count(output.String(), alloc) != 1 || strings.Contains(output.String(), dealloc) {
		t.Fatalf("converter allocation before release: %q", output.String())
	}
	astTestMust(t, out.Close())
	astTestMust(t, out.Close())
	got := output.String()
	if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 ||
		strings.Index(got, alloc) > strings.Index(got, dealloc) {
		t.Fatalf("converter model memory records: %q", got)
	}
}
