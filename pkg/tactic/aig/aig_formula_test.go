package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func aigEvalASTFormula(t *testing.T, m *ast.ASTManager, h ast.ASTHandle, values map[ast.ASTHandle]bool) bool {
	t.Helper()
	if v, ok := values[h]; ok {
		return v
	}
	if m.IsTrue(h) {
		return true
	}
	if m.IsFalse(h) {
		return false
	}
	op, e := m.GetDeclKind(h)
	aigMust(t, e)
	n, e := m.GetNumArgs(h)
	aigMust(t, e)
	child := func(i uint32) bool { a, e := m.GetArg(h, i); aigMust(t, e); return aigEvalASTFormula(t, m, a, values) }
	switch op {
	case ast.ASTOpNot:
		if n != 1 {
			t.Fatal("NOT arity")
		}
		return !child(0)
	case ast.ASTOpOr:
		out := false
		for i := uint32(0); i < n; i++ {
			out = out || child(i)
		}
		return out
	case ast.ASTOpAnd:
		out := true
		for i := uint32(0); i < n; i++ {
			out = out && child(i)
		}
		return out
	}
	t.Fatalf("unexpected output operator %d", op)
	return false
}

func TestAIGToFormulaNaiveIterativeDeMorganAndSmartInversion(t *testing.T) {
	m, budget := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	q, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	truth, e := g.True()
	aigMust(t, e)
	falsehood, e := g.False()
	aigMust(t, e)
	for _, constant := range []struct {
		ref  *Ref
		want bool
	}{{truth, true}, {falsehood, false}} {
		owned, e := g.ToFormula(constant.ref)
		aigMust(t, e)
		value, e := m.RefGet(owned)
		aigMust(t, e)
		if m.IsTrue(value) != constant.want || m.IsFalse(value) == constant.want {
			t.Fatal("AIG constant lost its source AST value")
		}
		aigMust(t, m.DelRef(&owned))
	}
	aigMust(t, falsehood.Close())
	aigMust(t, truth.Close())
	a, e := g.MakeVariable(p)
	aigMust(t, e)
	b, e := g.MakeVariable(q)
	aigMust(t, e)
	conj, e := g.And(a, b)
	aigMust(t, e)
	disj, e := g.Or(a, b)
	aigMust(t, e)
	for _, tc := range []struct {
		ref  *Ref
		op   int32
		want func(bool, bool) bool
	}{
		{conj, ast.ASTOpNot, func(x, y bool) bool { return x && y }},
		{disj, ast.ASTOpOr, func(x, y bool) bool { return x || y }},
	} {
		owned, e := g.ToFormula(tc.ref)
		aigMust(t, e)
		h, e := m.RefGet(owned)
		aigMust(t, e)
		op, e := m.GetDeclKind(h)
		aigMust(t, e)
		if op != tc.op {
			t.Fatalf("source De Morgan root %d want %d", op, tc.op)
		}
		for mask := uint32(0); mask < 4; mask++ {
			pv, qv := mask&1 != 0, mask&2 != 0
			if got := aigEvalASTFormula(t, m, h, map[ast.ASTHandle]bool{p: pv, q: qv}); got != tc.want(pv, qv) {
				t.Fatalf("formula %d mask %02b: %v", tc.op, mask, got)
			}
		}
		aigMust(t, m.DelRef(&owned))
	}
	baseline := budget.Used()
	reached := false
	for prefix := int64(0); prefix < 40; prefix++ {
		budget.FailAfter(prefix)
		owned, failure := g.ToFormula(conj)
		budget.FailAfter(-1)
		if failure == nil {
			aigMust(t, m.DelRef(&owned))
			reached = true
			break
		}
		if owned != (ast.ASTRef{}) || !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatalf("prefix %d result %v failure %v", prefix, owned, failure)
		}
		if budget.Used() != baseline {
			t.Fatalf("prefix %d retained %d bytes", prefix, budget.Used()-baseline)
		}
	}
	if !reached {
		t.Fatal("formula allocation campaign never reached success")
	}
	aigMust(t, disj.Close())
	aigMust(t, conj.Close())
	aigMust(t, b.Close())
	aigMust(t, a.Close())
	aigMust(t, g.Close())
}

func TestAIGToFormulaNestedGraphAndVariableNot(t *testing.T) {
	m, _ := aigFixture(t)
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("negated", sort)
	aigMust(t, e)
	notP, e := m.MkNot(p)
	aigMust(t, e)
	g, e := NewManager(m)
	aigMust(t, e)
	v, e := g.MakeVariable(notP)
	aigMust(t, e)
	nv, e := v.Not()
	aigMust(t, e)
	plain, e := g.ToFormula(nv)
	aigMust(t, e)
	h, e := m.RefGet(plain)
	aigMust(t, e)
	if h != p {
		t.Fatal("source invert did not remove a variable's existing NOT")
	}
	aigMust(t, m.DelRef(&plain))
	aigMust(t, nv.Close())
	aigMust(t, v.Close())
	var vars []*Ref
	for i := 0; i < 128; i++ {
		atom, e := m.MkFreshConst("deep", sort)
		aigMust(t, e)
		r, e := g.MakeVariable(atom)
		aigMust(t, e)
		vars = append(vars, r)
	}
	graph, e := g.AndArgs(vars)
	aigMust(t, e)
	owned, e := g.ToFormula(graph)
	aigMust(t, e)
	formula, e := m.RefGet(owned)
	aigMust(t, e)
	well, e := m.IsWellSorted(formula, nil)
	aigMust(t, e)
	if !well {
		t.Fatal("deep AIG formula is not well sorted")
	}
	aigMust(t, m.DelRef(&owned))
	aigMust(t, graph.Close())
	for i := len(vars) - 1; i >= 0; i-- {
		aigMust(t, vars[i].Close())
	}
	aigMust(t, g.Close())
}
