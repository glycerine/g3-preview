package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2DeclareFunSourceEntrypoint(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	_, err = m.RegisterArithmeticPlugin()
	paramsMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	paramsMust(t, err)
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	bvSort, err := m.BitVectorSort(4)
	paramsMust(t, err)
	constant, err := m.MkFuncDecl(astTestSymbol(t, m, "c"), nil, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	function, err := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{boolSort, bvSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	index, err := m.NewParameterInt(7)
	paramsMust(t, err)
	info, err := m.NewFuncDeclInfo(0, 741, []ASTParameter{index}, ASTFuncFlags{}, false)
	paramsMust(t, err)
	indexed, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "indexed"), []ASTHandle{boolSort}, boolSort, info)
	paramsMust(t, err)
	paramsMust(t, m.DelFuncDeclInfo(&info))
	paramsMust(t, m.DelParameter(&index))
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		decl ASTHandle
		want string
	}{
		{constant, "(declare-fun c () Bool)"},
		{function, "(declare-fun f (Bool (_ BitVec 4)) Bool)"},
		{indexed, "(declare-fun indexed (Bool) Bool)"},
	} {
		format, e := m.SMT2PPDeclareFun(tc.decl)
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("declaration %v: %q, want %q", tc.decl, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
	if _, err := m.SMT2PPDeclareFun(boolSort); err == nil {
		t.Fatal("sort accepted as function declaration")
	}
}
