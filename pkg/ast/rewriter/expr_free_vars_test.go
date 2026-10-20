package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestExprFreeVarsSourceCollectAccumulateDefaultsAndReverse(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	i, e := m.MkUninterpretedSort(astTestSymbol(t, m, "free-var-sort"), nil)
	astTestMust(t, e)
	v0, e := m.MkVar(0, b)
	astTestMust(t, e)
	v2, e := m.MkVar(2, i)
	astTestMust(t, e)
	pair, e := m.MkFuncDecl(symbol.Symbol{}, []ast.ASTHandle{b, i}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	body, e := m.MkApp(pair, []ast.ASTHandle{v0, v2})
	astTestMust(t, e)
	c, e := NewExprFreeVars(m)
	astTestMust(t, e)
	astTestMust(t, c.Collect(body))
	if c.Size() != 3 || !c.Contains(0) || c.Contains(1) || !c.Contains(2) {
		t.Fatal("free-variable shape", c.Sorts())
	}
	if got, e := c.Get(2); e != nil || got != i {
		t.Fatal("free-variable sort", got, e)
	}
	astTestMust(t, c.SetDefaultSort(b))
	if got, _ := c.Get(1); got != b {
		t.Fatal("default sort", got)
	}
	astTestMust(t, c.Reverse())
	if got, _ := c.Get(0); got != i {
		t.Fatal("reverse", c.Sorts())
	}
	astTestMust(t, c.Accumulate(v0))
	if got, _ := c.Get(0); got != i { // Persistent mark keeps the already-seen v0 out.
		t.Fatal("accumulate mark reset unexpectedly", c.Sorts())
	}
}

func TestExprFreeVarsSourceQuantifierOffsetAndBodyOnly(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	bound, e := m.MkVar(0, b)
	astTestMust(t, e)
	free, e := m.MkVar(1, b)
	astTestMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{{}}, Body: free, NoPatterns: []ast.ASTHandle{bound}})
	astTestMust(t, e)
	c, e := NewExprFreeVars(m)
	astTestMust(t, e)
	astTestMust(t, c.Collect(q))
	if c.Size() != 1 || !c.Contains(0) {
		t.Fatal("binder offset or annotation traversal", c.Sorts())
	}
	copy := *c
	if e := copy.Reset(); e == nil {
		t.Fatal("copied collector accepted")
	}
}
