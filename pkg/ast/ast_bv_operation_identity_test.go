// Source: src/ast/bv_decl_plugin.h:304,313,315; src/ast/ast.h:1308.
package ast

import "testing"

func TestASTBitVectorOperationIdentityErasesMetadata(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	var apps []ASTHandle
	for i, family := range []FamilyID{fid, arith} {
		info, e := m.NewFuncDeclInfo(family, 4, nil, ASTFuncFlags{}, false)
		astTestMust(t, e)
		name := "bv-operation"
		if i == 1 {
			name = "foreign-operation"
		}
		d, e := m.MkFuncDecl(astTestSymbol(t, m, name), nil, sort, info)
		astTestMust(t, e)
		astTestMust(t, m.DelFuncDeclInfo(&info))
		h, e := m.MkApp(d, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		apps = append(apps, h)
	}
	before := m.Stats()
	m.Budget().FailAfter(0)
	yes, e := m.IsBitVectorOperation(apps[0], 4)
	astTestMust(t, e)
	wrong, e := m.IsBitVectorOperation(apps[0], 6)
	astTestMust(t, e)
	foreign, e := m.IsBitVectorOperation(apps[1], 4)
	astTestMust(t, e)
	variable, e := m.IsBitVectorOperation(x, 4)
	astTestMust(t, e)
	m.Budget().FailAfter(-1)
	if !yes || wrong || foreign || variable || m.Stats() != before {
		t.Fatal("identity guard read metadata/allocated")
	}
}
