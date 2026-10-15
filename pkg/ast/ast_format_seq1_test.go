package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTFormatSeq1SourceGroupingAndNilTail(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	first, err := m.FormatString("first")
	paramsMust(t, err)
	second, err := m.FormatString("second")
	paramsMust(t, err)
	nilDoc, err := m.FormatNil()
	paramsMust(t, err)
	for _, tc := range []struct {
		header string
		args   []ASTHandle
		width  uint32
		want   string
	}{
		{"_", nil, 80, "(_)"},
		{"_", []ASTHandle{first, nilDoc, second}, 80, "(_ first second)"},
		{"index", []ASTHandle{first, nilDoc, second}, 9, "(index first\n       second)"},
	} {
		paramsMust(t, c.SetUIntName(p, dynAckName("max_width"), tc.width))
		root, e := m.FormatSeq1(tc.header, tc.args)
		if e != nil {
			t.Fatalf("header %q args %d: %v", tc.header, len(tc.args), e)
		}
		paramsMust(t, m.PrintFormat(out, root, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatalf("header %q width %d: %q, want %q", tc.header, tc.width, got, tc.want)
		}
		paramsMust(t, out.ResetText())
	}
	// The format owner is concrete AST storage; an invalid child is rejected.
	if _, err := m.FormatSeq1("_", []ASTHandle{first, {}}); err == nil {
		t.Fatal("invalid tail format accepted")
	}
}
