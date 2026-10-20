package ast

import (
	"strings"
	"testing"
)

func TestLegacySMTPrinterSortParametersAndRenaming(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.RegisterDeclarationPlugins())
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	bvSort, err := m.BitVectorSort(13)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(SliceArguments([]ASTHandle{bvSort}), boolSort)
	astTestMust(t, err)
	ebits, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sbits, err := m.NewParameterInt(24)
	astTestMust(t, err)
	floatSort, err := m.MkFamilySort(m.floatingPointPlugin.family, 0, []ASTParameter{ebits, sbits})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&sbits))
	astTestMust(t, m.DelParameter(&ebits))
	parameter, err := m.NewParameterAST(bvSort)
	astTestMust(t, err)
	userSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "legacy sort"), []ASTParameter{parameter})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameter))
	for _, tc := range []struct {
		sort ASTHandle
		want string
	}{{boolSort, "Bool"}, {bvSort, "(_ BitVec 13)"}, {arraySort, "(Array (_ BitVec 13) Bool)"}, {floatSort, "(_ FloatingPoint 8 24)"}, {userSort, "(|legacy sort| (_ BitVec 13))"}} {
		var out strings.Builder
		astTestMust(t, printer.DisplaySort(&out, tc.sort))
		if out.String() != tc.want {
			t.Fatal("legacy SMT sort", out.String(), tc.want)
		}
	}
}
