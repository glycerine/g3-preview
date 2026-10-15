package ast

import (
	"math"
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2FunctionDeclarationSourceFormatsAndRecursiveReference(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	plain, err := m.MkFuncDecl(astTestSymbol(t, m, "u"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	integer, err := m.NewParameterInt(3)
	paramsMust(t, err)
	r, err := m.numerals.NewValue()
	paramsMust(t, err)
	paramsMust(t, m.numerals.SetFraction(&r, 7, 3))
	rational, err := m.NewParameterRational(&r)
	paramsMust(t, err)
	paramsMust(t, m.numerals.Del(&r))
	function, err := m.NewParameterAST(plain)
	paramsMust(t, err)
	info, err := m.NewFuncDeclInfo(0, 741, []ASTParameter{integer, rational, function}, ASTFuncFlags{}, false)
	paramsMust(t, err)
	indexed, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "sel"), []ASTHandle{boolSort}, boolSort, info)
	paramsMust(t, err)
	paramsMust(t, m.DelFuncDeclInfo(&info))
	for _, param := range []*ASTParameter{&integer, &rational, &function} {
		paramsMust(t, m.DelParameter(param))
	}
	sortParameter, err := m.NewParameterAST(boolSort)
	paramsMust(t, err)
	info, err = m.NewFuncDeclInfo(0, 742, []ASTParameter{sortParameter}, ASTFuncFlags{}, false)
	paramsMust(t, err)
	asDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "select"), nil, boolSort, info)
	paramsMust(t, err)
	paramsMust(t, m.DelFuncDeclInfo(&info))
	paramsMust(t, m.DelParameter(&sortParameter))
	nestedParam, err := m.NewParameterAST(indexed)
	paramsMust(t, err)
	info, err = m.NewFuncDeclInfo(0, 743, []ASTParameter{nestedParam}, ASTFuncFlags{}, false)
	paramsMust(t, err)
	nested, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "outer"), nil, boolSort, info)
	paramsMust(t, err)
	paramsMust(t, m.DelFuncDeclInfo(&info))
	paramsMust(t, m.DelParameter(&nestedParam))

	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	check := func(format ASTHandle, want string) {
		t.Helper()
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != want {
			t.Fatalf("format %q, want %q", got, want)
		}
		paramsMust(t, out.ResetText())
	}
	for _, tc := range []struct {
		decl   ASTHandle
		want   string
		length uint32
		ref    string
	}{
		{plain, "u", 1, "u"},
		{indexed, "(_ sel 3 7/3 u)", math.MaxUint32, "((_ sel 3 7/3 u) (Bool) Bool)"},
		{asDecl, "(as select Bool)", math.MaxUint32, "(select () Bool)"},
		{nested, "(_ outer ((_ sel 3 7/3 u) (Bool) Bool))", math.MaxUint32, "((_ outer ((_ sel 3 7/3 u) (Bool) Bool)) () Bool)"},
	} {
		format, length, e := m.SMT2Fdecl(tc.decl)
		paramsMust(t, e)
		if length != tc.length {
			t.Fatalf("declaration %v length %d, want %d", tc.decl, length, tc.length)
		}
		check(format, tc.want)
		ref, e := m.SMT2FdeclRef(tc.decl)
		paramsMust(t, e)
		check(ref, tc.ref)
	}
	a, err := m.FormatString("a")
	paramsMust(t, err)
	b, err := m.FormatString("b")
	paramsMust(t, err)
	nilDoc, err := m.FormatNil()
	paramsMust(t, err)
	seq, err := m.FormatSeq5([]ASTHandle{a, nilDoc, b})
	paramsMust(t, err)
	check(seq, "(a b)")
	paramsMust(t, c.SetUIntName(p, dynAckName("max_width"), 3))
	check(seq, "(a\n b)")
	if _, _, err := m.SMT2Fdecl(boolSort); err == nil {
		t.Fatal("sort accepted as declaration")
	}
}
