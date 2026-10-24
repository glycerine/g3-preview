package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"testing"
)

func TestSMTSolverQuantifiedSingletonDomainContradiction(t *testing.T) {
	m, s := smtSolverFixture(t)
	params, e := s.config.Params.NewValue()
	astTestMust(t, e)
	defer s.config.Params.Del(&params)
	key := "mbqi.max_iterations"
	astTestMust(t, s.config.Params.SetUIntName(&params, &key, 3))
	astTestMust(t, s.UpdateParams(&params))
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "U"), nil)
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "a"), sort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	b, err := m.MkConst(astTestSymbol(t, m, "b"), sort)
	astTestMust(t, err)
	astTestPin(t, m, b)
	x, err := m.MkVar(1, sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, y)
	body, err := m.MkEq(x, y)
	astTestMust(t, err)
	astTestPin(t, m, body)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort, sort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x"), astTestSymbol(t, m, "y")}, Body: body, Weight: 1})
	astTestMust(t, err)
	astTestPin(t, m, quantifier)
	eq, err := m.MkEq(a, b)
	astTestMust(t, err)
	astTestPin(t, m, eq)
	distinct, err := m.MkNot(eq)
	astTestMust(t, err)
	astTestPin(t, m, distinct)
	astTestMust(t, s.Assert(quantifier))
	astTestMust(t, s.Assert(distinct))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		reason, e := s.ReasonUnknown()
		astTestMust(t, e)
		q := s.kernel.internalizer.quantifierManager
		t.Fatalf("singleton contradiction: status %v, reason %q, iterations %d", status, reason, q.modelChecker.iteration)
	}
}
