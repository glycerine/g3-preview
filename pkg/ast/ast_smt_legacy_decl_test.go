package ast

import (
	"strings"
	"testing"
)

func TestLegacySMTPrinterSortAndFunctionDeclarations(t *testing.T) {
	m := astTestManager(t)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	a, err := m.MkUninterpretedSort(astTestSymbol(t, m, "legacy-A"), nil)
	astTestMust(t, err)
	astTestPin(t, m, a)
	parameter, err := m.NewParameterAST(a)
	astTestMust(t, err)
	b, err := m.MkUninterpretedSort(astTestSymbol(t, m, "legacy-B"), []ASTParameter{parameter})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameter))
	astTestPin(t, m, b)
	function, err := m.MkFuncDecl(astTestSymbol(t, m, "legacy-f"), []ASTHandle{a, b}, a, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, function)

	for _, tc := range []struct {
		write func(*strings.Builder) error
		want  string
	}{
		{func(out *strings.Builder) error { return printer.DisplaySortDeclaration(out, a) }, "(declare-sort legacy-A)\n"},
		{func(out *strings.Builder) error { return printer.DisplaySortDeclaration(out, b) }, "(declare-sort (legacy-B legacy-A))\n"},
		{func(out *strings.Builder) error { return printer.DisplayFuncDeclaration(out, function) }, "(declare-fun legacy-f (legacy-A (legacy-B legacy-A)) legacy-A)"},
	} {
		var out strings.Builder
		astTestMust(t, tc.write(&out))
		if out.String() != tc.want {
			t.Fatalf("legacy SMT declaration %q, want %q", out.String(), tc.want)
		}
	}
}
