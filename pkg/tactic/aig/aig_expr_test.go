package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func aigPinExpr(t *testing.T, m *ast.ASTManager, h ast.ASTHandle) {
	t.Helper()
	r, e := m.NewRef(h)
	aigMust(t, e)
	t.Cleanup(func() { aigMust(t, m.DelRef(&r)) })
}

func aigEvalFormula(t *testing.T, m *ast.ASTManager, l Literal, values map[ast.ASTHandle]bool) bool {
	t.Helper()
	var value bool
	if l.node.id == 0 {
		value = true
	} else if l.node.and {
		value = aigEvalFormula(t, m, l.node.children[0], values) && aigEvalFormula(t, m, l.node.children[1], values)
	} else {
		h, e := m.RefGet(l.node.variable)
		aigMust(t, e)
		var found bool
		value, found = values[h]
		if !found {
			t.Fatalf("unexpected AIG leaf %v", h)
		}
	}
	if l.inverted {
		return !value
	}
	return value
}

func TestAIGExprTraversalSharedLeavesAndBooleanOperations(t *testing.T) {
	m, budget := aigFixture(t)
	boolSort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", boolSort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", boolSort)
	aigMust(t, e)
	shared, e := m.MkAnd(p, p)
	aigMust(t, e)
	aigPinExpr(t, m, shared)
	g, e := NewManager(m)
	aigMust(t, e)
	one, e := g.MkAIG(shared)
	aigMust(t, e)
	count, e := g.NumNodes()
	aigMust(t, e)
	if count != 2 {
		t.Fatalf("repeated AST leaf created %d AIG nodes, want true + one variable", count)
	}
	for _, v := range []bool{false, true} {
		if got := aigEvalFormula(t, m, one.lit, map[ast.ASTHandle]bool{p: v}); got != v {
			t.Fatalf("shared leaf value %v: %v", v, got)
		}
	}
	aigMust(t, one.Close())
	notQ, e := m.MkNot(q)
	aigMust(t, e)
	iff, e := m.MkIff(p, q)
	aigMust(t, e)
	ite, e := m.MkIte(p, notQ, q)
	aigMust(t, e)
	impl, e := m.MkImplies(p, q)
	aigMust(t, e)
	xor, e := m.MkXor(p, q)
	aigMust(t, e)
	or, e := m.MkOr3(iff, ite, impl)
	aigMust(t, e)
	formula, e := m.MkAnd3(or, xor, shared)
	aigMust(t, e)
	aigPinExpr(t, m, formula)
	out, e := g.MkAIG(formula)
	aigMust(t, e)
	for mask := uint32(0); mask < 4; mask++ {
		pv, qv := mask&1 != 0, mask&2 != 0
		iteWant := qv
		if pv {
			iteWant = !qv
		}
		want := (pv == qv || iteWant || !pv || qv) && (pv != qv) && pv
		got := aigEvalFormula(t, m, out.lit, map[ast.ASTHandle]bool{p: pv, q: qv})
		if got != want {
			t.Fatalf("assignment %02b: %v want %v", mask, got, want)
		}
	}
	aigMust(t, out.Close())
	before := budget.Used()
	budget.FailAfter(0)
	if failed, e := g.MkAIG(shared); failed != nil || !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("refused conversion: %v %v", failed, e)
	}
	budget.FailAfter(-1)
	if budget.Used() != before {
		t.Fatalf("refused conversion retained %d bytes", budget.Used()-before)
	}
	succeeded := false
	for prefix := int64(0); prefix < 64; prefix++ {
		budget.FailAfter(prefix)
		attempt, failure := g.MkAIG(formula)
		budget.FailAfter(-1)
		if failure == nil {
			aigMust(t, attempt.Close())
			succeeded = true
			break
		}
		if attempt != nil || !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatalf("conversion prefix %d: result %v, failure %v", prefix, attempt, failure)
		}
		if budget.Used() != before {
			t.Fatalf("conversion prefix %d retained %d bytes", prefix, budget.Used()-before)
		}
	}
	if !succeeded {
		t.Fatal("allocation-prefix campaign never reached completed conversion")
	}
	aigMust(t, g.Close())
}

func TestAIGExprTreatsNonBooleanEqualityAsVariable(t *testing.T) {
	m, _ := aigFixture(t)
	arithmetic, e := m.RegisterArithmeticPlugin()
	aigMust(t, e)
	intSort, e := m.MkFamilySort(arithmetic, 1, nil)
	aigMust(t, e)
	a, e := m.MkFreshConst("a", intSort)
	aigMust(t, e)
	b, e := m.MkFreshConst("b", intSort)
	aigMust(t, e)
	eq, e := m.MkEq(a, b)
	aigMust(t, e)
	aigPinExpr(t, m, eq)
	g, e := NewManager(m)
	aigMust(t, e)
	out, e := g.MkAIG(eq)
	aigMust(t, e)
	if out.lit.node.and {
		t.Fatal("non-Boolean equality was expanded")
	}
	h, e := m.RefGet(out.lit.node.variable)
	aigMust(t, e)
	if h != eq {
		t.Fatal("non-Boolean equality lost its AST expression")
	}
	aigMust(t, out.Close())
	aigMust(t, g.Close())
}

func TestAIGExprDeepNotUsesSourceFrameStack(t *testing.T) {
	m, _ := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("deep", sort)
	aigMust(t, e)
	root := p
	for i := 0; i < 1025; i++ {
		root, e = m.MkNot(root)
		aigMust(t, e)
	}
	aigPinExpr(t, m, root)
	g, e := NewManager(m)
	aigMust(t, e)
	r, e := g.MkAIG(root)
	aigMust(t, e)
	if !r.lit.inverted || r.lit.node.and {
		t.Fatal("odd NOT depth did not invert one variable")
	}
	count, e := g.NumNodes()
	aigMust(t, e)
	if count != 2 {
		t.Fatalf("deep unary chain retained %d nodes", count)
	}
	aigMust(t, r.Close())
	aigMust(t, g.Close())
}
