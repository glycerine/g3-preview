package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
)

func TestSMTNestedMixedCoefficientSortGuards(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "ALL", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	astTestMust(t, s.Push())
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
	integer, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	real, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	i, err := m.MkConst(astTestSymbol(t, m, "nested-mixed-i"), integer)
	astTestMust(t, err)
	r, err := m.MkConst(astTestSymbol(t, m, "nested-mixed-r"), real)
	astTestMust(t, err)
	square, err := m.MkArithmeticMul(i, i)
	astTestMust(t, err)
	for _, h := range []ast.ASTHandle{square, r} {
		_, err := theory.InternalizeTerm(h)
		astTestMust(t, err)
	}
	n := smtNestedArithmetic{theory: theory, solver: simplex.NewSolver(nil), refs: smtArithmeticDivisionCall{theory: theory}}
	defer func() { astTestMust(t, n.refs.close()) }()
	form := newSMTLinearForm()
	for _, spec := range []struct {
		h           ast.ASTHandle
		coefficient *big.Rat
	}{{square, big.NewRat(1, 2)}, {i, big.NewRat(-1, 1)}, {r, big.NewRat(1, 1)}} {
		column, found := n.column(spec.h)
		if !found {
			t.Fatal("mixed polynomial column")
		}
		form.addTerm(column, spec.coefficient)
	}
	astTestMust(t, m.EnableIntRealCoercions(false))
	_, supported, err := n.terms(&form)
	astTestMust(t, err)
	if supported {
		t.Fatal("mixed row bypassed source coercion guard")
	}
	astTestMust(t, m.EnableIntRealCoercions(true))
	polynomial, supported, err := n.terms(&form)
	astTestMust(t, err)
	if !supported {
		t.Fatal("mixed row with coercions")
	}
	expression, err := n.crossNested(polynomial, i)
	astTestMust(t, err)
	// A fractional leading coefficient cannot create an Int numeral.
	// The Horner fallback still preserves (1/2)*i*i - i + r exactly.
	for _, spec := range []struct {
		h     ast.ASTHandle
		value int64
	}{{i, 2}, {r, 1}} {
		column, _ := n.column(spec.h)
		bound := simplex.RationalInfinitesimal(big.NewRat(spec.value, 1))
		astTestMust(t, n.solver.SetLower(column, bound))
		astTestMust(t, n.solver.SetUpper(column, bound))
	}
	interval, err := n.evaluate(expression)
	astTestMust(t, err)
	if interval.minusInfinity() || interval.plusInfinity() || interval.lower.value.Cmp(big.NewRat(1, 1)) != 0 || interval.upper.value.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatal("mixed nested polynomial", interval)
	}
}
