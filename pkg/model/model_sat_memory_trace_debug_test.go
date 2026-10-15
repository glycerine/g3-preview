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
	sattactic "github.com/glycerine/g3/pkg/sat/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATBooleanModelMemoryWrapperFollowsCoreOwner(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	solver, _, _ := satSolverCheckFixture(t)
	translator, err := sattactic.NewGoalTranslator(m, solver)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, translator.Close()) })
	atom := goalTestAtom(t, m, "sat-model-memory")
	astTestMust(t, translator.Compile([]ast.ASTHandle{atom}))
	result, err := solver.Check(nil)
	astTestMust(t, err)
	if result != lbool.True {
		t.Fatal("SAT fixture did not produce a model", result)
	}
	values, err := solver.AppendModel(nil)
	astTestMust(t, err)
	tags.Enable("memory")
	core, err := sattactic.BuildBooleanModel(translator, values)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, core.Close()) })
	derived := core.DerivedModel()
	if derived == nil {
		t.Fatal("SAT fragment returned no derived model")
	}
	id := fmt.Sprintf("%p", derived)
	size := strconv.FormatUint(uint64(reflect.TypeOf(model.Model{}).Size()), 10)
	alloc := "alloc " + id + " src/sat/tactic/sat_tactic.cpp:93 model " + size + "\n"
	dealloc := "dealloc " + id + " src/model/model_core.h:74\n"
	if strings.Count(output.String(), alloc) != 1 || strings.Contains(output.String(), dealloc) {
		t.Fatalf("SAT model allocation before release: %q", output.String())
	}
	astTestMust(t, core.Close())
	astTestMust(t, core.Close())
	got := output.String()
	if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 ||
		strings.Index(got, alloc) > strings.Index(got, dealloc) {
		t.Fatalf("SAT model memory records: %q", got)
	}
}
