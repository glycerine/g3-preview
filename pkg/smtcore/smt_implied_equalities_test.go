package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSolverImpliedEqualitiesPartitionsByProofAndSort(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	integer, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_IDL", SMTSolverConfig{Params: config.Context})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	makeConstant := func(name string, sort ast.ASTHandle) ast.ASTHandle {
		value, makeErr := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, makeErr)
		return value
	}
	a := makeConstant("implied-a", integer)
	b := makeConstant("implied-b", integer)
	c := makeConstant("implied-c", integer)
	d := makeConstant("implied-d", integer)
	p := makeConstant("implied-p", boolean)
	ab, err := m.MkEq(a, b)
	astTestMust(t, err)
	bc, err := m.MkEq(b, c)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(ab))
	astTestMust(t, solver.Assert(bc))
	before, err := solver.ScopeLevel()
	astTestMust(t, err)
	status, classes, err := solver.ImpliedEqualities([]ast.ASTHandle{a, b, c, d, p})
	astTestMust(t, err)
	if status != lbool.True || len(classes) != 5 {
		t.Fatal("implied equality result", status, classes)
	}
	if classes[0] != classes[1] || classes[1] != classes[2] || classes[3] == classes[0] || classes[4] == classes[0] || classes[4] == classes[3] {
		t.Fatal("implied equality classes", classes)
	}
	after, err := solver.ScopeLevel()
	astTestMust(t, err)
	if after != before {
		t.Fatal("query changed solver scope", before, after)
	}
}

func TestSMTSolverImpliedEqualitiesUnsatAndEmptyInputs(t *testing.T) {
	m, solver := smtSolverFixture(t)
	emptyStatus, classes, err := solver.ImpliedEqualities(nil)
	astTestMust(t, err)
	if emptyStatus != lbool.True || len(classes) != 0 {
		t.Fatal("empty implied equality query", emptyStatus, classes)
	}
	a := smtRelevancyBool(t, m, "implied-unsat-a")
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(a))
	astTestMust(t, solver.Assert(notA))
	status, classes, err := solver.ImpliedEqualities([]ast.ASTHandle{a})
	astTestMust(t, err)
	if status != lbool.False || classes != nil {
		t.Fatal("unsatisfiable implied equality query", status, classes)
	}
}
