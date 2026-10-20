// Source: src/ast/format.h:132-162.
package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTFormatSeq3SourceSplitAndNilDocuments(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	doc := func(s string) ASTHandle {
		h, e := m.FormatString(s)
		paramsMust(t, e)
		return h
	}
	nilDoc, err := m.FormatNil()
	paramsMust(t, err)
	print := func(h ASTHandle, width uint32) string {
		paramsMust(t, c.SetUIntName(p, dynAckName("max_width"), width))
		paramsMust(t, m.PrintFormat(out, h, c, p, g))
		s, e := out.String()
		paramsMust(t, e)
		paramsMust(t, out.ResetText())
		return s
	}
	empty, err := m.FormatSeq3("forall", nil, 1, 2)
	paramsMust(t, err)
	if got := print(empty, 80); got != "(forall)" {
		t.Fatalf("empty: %q", got)
	}
	root, err := m.FormatSeq3("forall", []ASTHandle{doc("decls"), nilDoc, doc("body")}, 2, 2)
	paramsMust(t, err)
	if got := print(root, 80); got != "(forall decls body)" {
		t.Fatalf("flat: %q", got)
	}
	if got := print(root, 10); got != "(forall decls\n  body)" {
		t.Fatalf("split: %q", got)
	}
	if _, err = m.FormatSeq3("forall", []ASTHandle{doc("x")}, 0, 2); err == nil {
		t.Fatal("zero split accepted")
	}
}
