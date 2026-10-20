package ast

import (
	"strconv"
	"strings"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestLegacySMTPrinterLetInsertionSharedAndLargeTerms(t *testing.T) {
	m := astTestManager(t)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "legacy-la"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	b, err := m.MkConst(astTestSymbol(t, m, "legacy-lb"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, b)
	shared, err := m.MkIff(a, b)
	astTestMust(t, err)
	astTestPin(t, m, shared)
	root, err := m.MkOr(shared, shared)
	astTestMust(t, err)
	astTestPin(t, m, root)
	id, err := m.GetASTID(shared)
	astTestMust(t, err)
	alias := "$x" + strconv.FormatUint(uint64(id), 10)
	var out strings.Builder
	astTestMust(t, printer.DisplayExpr(&out, root, nil))
	want := "(let ((" + alias + " (= legacy-la legacy-lb)))\n(or " + alias + " " + alias + "))"
	if out.String() != want {
		t.Fatalf("legacy SMT shared let %q, want %q", out.String(), want)
	}

	longName := "legacy-" + strings.Repeat("long", 22)
	longDecl, err := m.MkFuncDecl(astTestSymbol(t, m, longName), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, longDecl)
	large, err := m.MkApp(longDecl, []ASTHandle{a})
	astTestMust(t, err)
	astTestPin(t, m, large)
	largeRoot, err := m.MkIff(large, b)
	astTestMust(t, err)
	astTestPin(t, m, largeRoot)
	id, err = m.GetASTID(large)
	astTestMust(t, err)
	alias = "$x" + strconv.FormatUint(uint64(id), 10)
	out.Reset()
	astTestMust(t, printer.DisplayExpr(&out, largeRoot, nil))
	want = "(let ((" + alias + " (" + longName + " legacy-la)))\n(= " + alias + " legacy-lb))"
	if out.String() != want {
		t.Fatalf("legacy SMT large-term let %q, want %q", out.String(), want)
	}
}

func TestLegacySMTPrinterQuantifierLocalLetScope(t *testing.T) {
	m := astTestManager(t)
	printer, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	astTestPin(t, m, truth)
	shared, err := m.MkIff(variable, truth)
	astTestMust(t, err)
	astTestPin(t, m, shared)
	body, err := m.MkAnd(shared, shared)
	astTestMust(t, err)
	astTestPin(t, m, body)
	quantifier, err := m.MkQuantifier(ASTQuantifierSpec{
		Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "legacy-lx")}, Body: body,
	})
	astTestMust(t, err)
	astTestPin(t, m, quantifier)
	id, err := m.GetASTID(shared)
	astTestMust(t, err)
	alias := "$x" + strconv.FormatUint(uint64(id), 10)
	var out strings.Builder
	astTestMust(t, printer.DisplayExpr(&out, quantifier, nil))
	want := "(forall ((legacy-lx Bool) )(let ((" + alias + " (= legacy-lx true)))\n(and " + alias + " " + alias + ")))\n"
	if out.String() != want {
		t.Fatalf("legacy SMT quantified local let %q, want %q", out.String(), want)
	}
}
