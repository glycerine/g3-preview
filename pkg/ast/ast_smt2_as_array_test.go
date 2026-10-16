package ast

import (
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

// Source: src/ast/ast_smt2_pp.cpp:623-652. A nullary as-array application
// reaches pp_const's indexed declaration branch.
func TestASTSMT2AsArrayNullaryDeclaration(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	_, err = m.RegisterArrayPlugin()
	paramsMust(t, err)
	bs, err := m.MkBoolSort()
	paramsMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "array-source-f"), []ASTHandle{bs}, bs, ASTFuncDeclInfo{})
	paramsMust(t, err)
	array, err := m.MkAsArray(f)
	paramsMust(t, err)
	format, err := m.SMT2PPExprTree(array, nil)
	paramsMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, m.PrintFormat(out, format, c, p, g))
	got, err := out.String()
	paramsMust(t, err)
	if got != "(_ as-array array-source-f)" {
		t.Fatalf("as-array source declaration: %q", got)
	}
}
