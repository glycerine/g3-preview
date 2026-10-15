package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTSMT2ApplicationSourceShortAndLongNameLayout(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	short, err := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{boolSort, boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	long, err := m.MkFuncDecl(astTestSymbol(t, m, "verylongfunctionname"), []ASTHandle{boolSort, boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	a, err := m.FormatString("p")
	paramsMust(t, err)
	b, err := m.FormatString("q")
	paramsMust(t, err)
	nilDoc, err := m.FormatNil()
	paramsMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	for _, tc := range []struct {
		decl  ASTHandle
		args  []ASTHandle
		width uint32
		want  string
	}{
		{short, []ASTHandle{a, nilDoc, b}, 80, "(f p q)"},
		{short, []ASTHandle{a, b}, 5, "(f p\n   q)"},
		{long, []ASTHandle{a, b}, 80, "(verylongfunctionname p q)"},
		{long, []ASTHandle{a, b}, 8, "(verylongfunctionname\n  p\n  q)"},
	} {
		paramsMust(t, c.SetUIntName(p, dynAckName("max_width"), tc.width))
		format, e := m.SMT2PPAppFormat(tc.decl, tc.args)
		paramsMust(t, e)
		paramsMust(t, m.PrintFormat(out, format, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("width %d: %q, want %q", tc.width, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
	if _, err := m.SMT2PPAppFormat(short, nil); err == nil {
		t.Fatal("constant entered nonconstant application layout")
	}
}
