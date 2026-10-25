package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func TestSMTDatatypeSingletonExcludesDisequality(t *testing.T) {
	m, prior := smtSolverFixture(t)
	s, err := NewSMTSolver(m, nil, "ALL", SMTSolverConfig{Params: prior.config.Params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	family, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: astTestSymbol(t, m, "Single"), Constructors: []ast.DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "only"), Recognizer: astTestSymbol(t, m, "is-only")}}}})
	astTestMust(t, err)
	size, err := m.GetSortSize(sorts[0])
	astTestMust(t, err)
	count, err := size.Size()
	astTestMust(t, err)
	if count != 1 {
		t.Fatal("singleton size", count)
	}
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	only, err := m.MkApp(constructors[0], nil)
	astTestMust(t, err)
	astTestPin(t, m, only)
	symbol := astTestSymbol(t, m, "s")
	value, err := m.MkConst(symbol, sorts[0])
	astTestMust(t, err)
	astTestPin(t, m, value)
	eq, err := m.MkEq(value, only)
	astTestMust(t, err)
	astTestPin(t, m, eq)
	neq, err := m.MkNot(eq)
	astTestMust(t, err)
	astTestPin(t, m, neq)
	astTestMust(t, s.Assert(neq))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		theory := s.kernel.internalizer.theories[family].(*SMTDatatypeTheory)
		t.Fatalf("singleton disequality: %v variables=%v reconstructions=%v splits=%v", status, len(theory.data), theory.statistics.constructorAxioms, theory.statistics.splits)
	}
}
