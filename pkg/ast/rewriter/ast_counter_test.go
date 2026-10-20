package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTCounterScalarOperations(t *testing.T) {
	c := NewCounter()
	c.Count([]uint32{3, 1, 3, 8}, 1).Update(8, -2)
	c.Update(5, -4)
	if c.Get(3) != 2 || c.Get(1) != 1 || c.Get(8) != -1 || c.Get(99) != 0 {
		t.Fatal("source signed counts")
	}
	if got := c.PositiveCount(); got != 2 {
		t.Fatal("positive count", got)
	}
	max, found := c.MaxPositive()
	if !found || max != 3 || c.MaxCounterValue() != 2 {
		t.Fatal("positive maxima", max, found, c.MaxCounterValue())
	}
	if value, err := c.MaxPositiveValue(); err != nil || value != 3 {
		t.Fatal("verified maximum", value, err)
	}
	positive := make(map[uint32]struct{})
	c.CollectPositive(positive)
	if len(positive) != 2 {
		t.Fatal("positive collection", positive)
	}
	c.Reset()
	if _, found = c.MaxPositive(); found || c.MaxCounterValue() != 0 {
		t.Fatal("reset")
	}
	if _, err := c.MaxPositiveValue(); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("empty verified maximum", err)
	}
}

func TestASTVarCounterArgumentsAndQuantifierDepth(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	v1, err := m.MkVar(1, boolSort)
	astTestMust(t, err)
	v2, err := m.MkVar(2, boolSort)
	astTestMust(t, err)
	v5, err := m.MkVar(5, boolSort)
	astTestMust(t, err)
	predicate, err := m.MkFamilyApp(0, ast.ASTOpAnd, nil, []ast.ASTHandle{v0, v1, v0})
	astTestMust(t, err)
	c, err := NewVarCounter(m)
	astTestMust(t, err)
	astTestMust(t, c.CountVars(predicate, 1))
	if c.Get(0) != 2 || c.Get(1) != 1 {
		t.Fatal("argument free-variable counts", c.Get(0), c.Get(1))
	}
	body, err := m.MkFamilyApp(0, ast.ASTOpAnd, nil, []ast.ASTHandle{v5, v1})
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true,
		Sorts:  []ast.ASTHandle{boolSort, boolSort},
		Names:  []usymbol.Symbol{astTestSymbol(t, m, "x"), astTestSymbol(t, m, "y")},
		Body:   body,
	})
	astTestMust(t, err)
	root, err := m.MkFamilyApp(0, ast.ASTOpOr, nil, []ast.ASTHandle{v2, q})
	astTestMust(t, err)
	max, err := c.MaxVar(root)
	astTestMust(t, err)
	if max != 3 {
		t.Fatal("binder-adjusted max", max)
	}
	next, err := c.NextVar(root)
	astTestMust(t, err)
	if next != 4 {
		t.Fatal("next variable", next)
	}
	constant, err := m.MkConst(astTestSymbol(t, m, "counter-c"), boolSort)
	astTestMust(t, err)
	next, err = c.NextVar(constant)
	astTestMust(t, err)
	if next != 0 {
		t.Fatal("ground next variable", next)
	}
}

func TestASTIdentityCounter(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "counter-a"), boolSort)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "counter-b"), boolSort)
	astTestMust(t, err)
	c := NewASTCounter()
	c.Inc(a)
	c.Inc(a)
	c.Dec(a)
	c.Dec(b)
	if c.Get(a) != 1 || c.Get(b) != -1 {
		t.Fatal("identity counts", c.Get(a), c.Get(b))
	}
}
