// Source: src/ast/bv_decl_plugin.h:298.
package ast

import "testing"

func TestASTBitVectorNumeralPredicateOnlyReadsIdentity(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	for _, test := range []struct {
		family FamilyID
		kind   int32
		want   bool
	}{{fid, 0, true}, {fid, 4, false}, {arith, 0, false}, {42, 0, false}} {
		info, e := m.NewFuncDeclInfo(test.family, test.kind, nil, ASTFuncFlags{}, false)
		astTestMust(t, e)
		d, e := m.MkFuncDecl(astTestSymbol(t, m, "raw-numeral-predicate"), nil, sort, info)
		astTestMust(t, e)
		astTestMust(t, m.DelFuncDeclInfo(&info))
		h, e := m.MkApp(d, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		before := m.Stats()
		m.Budget().FailAfter(0)
		got, e := m.IsBitVectorNumeral(h)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if got != test.want || m.Stats() != before {
			t.Fatal("numeral family/kind predicate", test.family, test.kind, got)
		}
	}
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	yes, e := m.IsBitVectorNumeral(x)
	astTestMust(t, e)
	if yes {
		t.Fatal("variable matched numeral")
	}
}
