package ast

// Source: src/ast/ast_ll_pp.cpp:274-328.

import (
	"strconv"
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTBoundedLowLevelPrinterSourceDepthAndNodeKinds(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	v, err := m.MkVar(7, sort)
	astTestMust(t, err)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{sort}, sort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	one, err := m.MkApp(decl, []ASTHandle{v})
	astTestMust(t, err)
	two, err := m.MkApp(decl, []ASTHandle{one})
	astTestMust(t, err)
	wides := make([]ASTHandle, 17)
	domains := make([]ASTHandle, 17)
	for i := range wides {
		wides[i], domains[i] = v, sort
	}
	wideDecl, err := m.MkFuncDecl(astTestSymbol(t, m, "wide"), domains, sort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	wide, err := m.MkApp(wideDecl, wides)
	astTestMust(t, err)
	id, err := m.GetASTID(one)
	astTestMust(t, err)
	wideID, err := m.GetASTID(wide)
	astTestMust(t, err)
	sortID, err := m.GetASTID(sort)
	astTestMust(t, err)
	for _, tc := range []struct {
		h     ASTHandle
		depth uint32
		want  string
	}{
		{v, 3, "(:var 7)"},
		{two, 3, "(f (f (:var 7)))"},
		{two, 1, "(f #" + strconv.FormatUint(uint64(id), 10) + ")"},
		{wide, 3, "#" + strconv.FormatUint(uint64(wideID), 10)},
		{sort, 3, "#" + strconv.FormatUint(uint64(sortID), 10)},
	} {
		out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.BudgetField})
		astTestMust(t, err)
		astTestMust(t, m.boundedLowLevelPrint(out, astNodeRef{slot: tc.h.slot, generation: tc.h.generation}, tc.depth))
		got, err := out.String()
		astTestMust(t, err)
		astTestMust(t, out.Close())
		if got != tc.want {
			t.Fatalf("bounded depth %d: got %q want %q", tc.depth, got, tc.want)
		}
	}
}
