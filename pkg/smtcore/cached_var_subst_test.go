package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTCachedVarSubstitutionSourceOrderCacheAndReset(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	v1, err := m.MkVar(1, boolean)
	astTestMust(t, err)
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{v0, v1})
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolean, boolean},
		Names: []symbol.Symbol{symbol.Numerical(2200), symbol.Numerical(2201)}, Body: body,
	})
	astTestMust(t, err)
	leftOwner, left := smtEnodeConstant(t, m, 2202, boolean, 0)
	rightOwner, right := smtEnodeConstant(t, m, 2203, boolean, 0)
	for _, handle := range []ast.ASTHandle{quantifier, leftOwner, rightOwner} {
		astTestMust(t, m.IncRef(handle))
		h := handle
		t.Cleanup(func() { astTestMust(t, m.DecRef(h)) })
	}
	cache, err := NewSMTCachedVarSubstitution(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cache.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	cached, err := cache.Instantiate(quantifier, []*SMTEnode{left, right}, out)
	astTestMust(t, err)
	if cached || cache.Size() != 1 {
		t.Fatal("first substitution cache state", cached, cache.Size())
	}
	result, err := m.RefGet(out)
	astTestMust(t, err)
	first, err := m.GetArg(result, 0)
	astTestMust(t, err)
	second, err := m.GetArg(result, 1)
	astTestMust(t, err)
	if first != rightOwner || second != leftOwner {
		t.Fatal("standard reverse de Bruijn binding order", first, second)
	}
	astTestMust(t, m.SetRef(out, leftOwner))
	cached, err = cache.Instantiate(quantifier, []*SMTEnode{left, right}, out)
	astTestMust(t, err)
	reused, err := m.RefGet(out)
	astTestMust(t, err)
	if !cached || reused != result || cache.Size() != 1 {
		t.Fatal("cached result reuse", cached, reused, result, cache.Size())
	}
	astTestMust(t, cache.Reset())
	if cache.Size() != 0 {
		t.Fatal("reset retained entries")
	}
	cached, err = cache.Instantiate(quantifier, []*SMTEnode{right, left}, out)
	astTestMust(t, err)
	if cached || cache.Size() != 1 {
		t.Fatal("post-reset insertion", cached, cache.Size())
	}
}

func TestSMTCachedVarSubstitutionGuards(t *testing.T) {
	m := astTestManager(t)
	cache, err := NewSMTCachedVarSubstitution(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cache.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	truth, err := m.MkBoolVal(true)
	astTestMust(t, err)
	if _, err = cache.Instantiate(truth, nil, out); err != ast.ErrASTDomain {
		t.Fatal("non-quantifier accepted", err)
	}
}
