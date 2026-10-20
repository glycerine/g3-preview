package normal_forms_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	normalforms "github.com/glycerine/g3/pkg/ast/normal_forms"
)

func TestNNFGeneratedParameterConstruction(t *testing.T) {
	c, g, local, sy := nnfParamsFixture(t)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: c.NumeralsField, Budget: c.BudgetField})
	nnfParamsMust(t, e)
	t.Cleanup(func() { nnfParamsMust(t, m.Close()) })
	names, e := normalforms.NewDefinedNames(m)
	nnfParamsMust(t, e)
	t.Cleanup(func() { nnfParamsMust(t, names.Close()) })
	key := "mode"
	full, e := sy.Intern("full")
	nnfParamsMust(t, e)
	nnfParamsMust(t, c.SetSymbolName(local, &key, full))
	n, e := normalforms.NewNNFFromParams(m, names, c, g, local)
	nnfParamsMust(t, e)
	t.Cleanup(func() { nnfParamsMust(t, n.Close()) })
	b, e := m.MkBoolSort()
	nnfParamsMust(t, e)
	a, e := m.MkFreshConst("parameter-a", b)
	nnfParamsMust(t, e)
	nnfParamsMust(t, m.IncRef(a))
	t.Cleanup(func() { nnfParamsMust(t, m.DecRef(a)) })
	bb, e := m.MkFreshConst("parameter-b", b)
	nnfParamsMust(t, e)
	nnfParamsMust(t, m.IncRef(bb))
	t.Cleanup(func() { nnfParamsMust(t, m.DecRef(bb)) })
	root, e := m.MkImplies(a, bb)
	nnfParamsMust(t, e)
	nnfParamsMust(t, m.IncRef(root))
	t.Cleanup(func() { nnfParamsMust(t, m.DecRef(root)) })
	out, e := m.NewRef(ast.ASTHandle{})
	nnfParamsMust(t, e)
	proof, e := m.NewRef(ast.ASTHandle{})
	nnfParamsMust(t, e)
	t.Cleanup(func() { nnfParamsMust(t, m.DelRef(&proof)); nnfParamsMust(t, m.DelRef(&out)) })
	var definitions, definitionProofs []ast.ASTRef
	nnfParamsMust(t, n.Transform(root, &definitions, &definitionProofs, out, proof))
	result, e := m.RefGet(out)
	nnfParamsMust(t, e)
	if op, x := m.GetDeclKind(result); x != nil || op != ast.ASTOpOr {
		t.Fatal("local full-mode parameter was not applied", op, x)
	}
}
