package ast

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTWellSortedWrongAritySourceTraceBeforeWarning(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("ws")
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	arg, err := m.MkVar(0, boolSort)
	paramsMust(t, err)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "guard"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	app, err := m.MkApp(decl, []ASTHandle{arg})
	paramsMust(t, err)
	internal, err := m.internal(app)
	paramsMust(t, err)
	warnings := diagnostic.NewWarningService(&output)
	m.appAt(internal.slot).count = 0
	good, checkErr := m.IsWellSortedWithPrinter(app, warnings, c, p, g)
	m.appAt(internal.slot).count = 1
	paramsMust(t, checkErr)
	if good {
		t.Fatal("wrong arity accepted")
	}
	if tags.ShouldTrace("ws") {
		got := output.String()
		if !strings.HasPrefix(got, "-------- [ws] operator() src/ast/well_sorted.cpp:49 ---------\nunexpected number of arguments.\n") ||
			!strings.Contains(got, "guard") ||
			!strings.Contains(got, "------------------------------------------------\nWARNING: unexpected number of arguments.\n") {
			t.Fatalf("source trace/warning order %q", got)
		}
	} else if got := output.String(); got != "WARNING: unexpected number of arguments.\n" {
		t.Fatalf("erased trace emitted %q", got)
	}
}

func TestASTWellSortedSortMismatchSourceTraceBeforeWarning(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("tc")
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	otherSort, err := m.MkSort(astTestSymbol(t, m, "OtherTrace"), ASTSortInfo{})
	paramsMust(t, err)
	arg, err := m.MkVar(0, boolSort)
	paramsMust(t, err)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "guard"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	app, err := m.MkApp(decl, []ASTHandle{arg})
	paramsMust(t, err)
	id, err := m.GetASTID(arg)
	paramsMust(t, err)
	varRef, err := m.internal(arg)
	paramsMust(t, err)
	otherRef, err := m.internal(otherSort)
	paramsMust(t, err)
	warnings := diagnostic.NewWarningService(&output)
	old := m.varAt(varRef.slot).sort
	m.varAt(varRef.slot).sort = otherRef
	good, checkErr := m.IsWellSortedWithPrinter(app, warnings, c, p, g)
	m.varAt(varRef.slot).sort = old
	paramsMust(t, checkErr)
	if good {
		t.Fatal("sort mismatch accepted")
	}
	got := output.String()
	warning := fmt.Sprintf("WARNING: Sort mismatch for argument 1 of decl guard :: (-> Bool Bool)\n#%d := (:var 0 OtherTrace)\n(guard #%d)\n\nExpected sort: Bool\nActual sort:   OtherTrace\nFunction sort: (declare-fun guard (Bool) Bool).\n", id, id)
	if tags.ShouldTrace("tc") {
		want := "-------- [tc] operator() src/ast/well_sorted.cpp:59 ---------\n" +
			"sort mismatch on argument #0.\n" +
			"(guard (:var 0))Sort mismatch for argument 1 of (guard (:var 0))\n" +
			"Expected sort: Bool\nActual sort:   OtherTrace\nFunction sort: (declare-fun guard (Bool) Bool)." +
			"------------------------------------------------\n" + warning
		if got != want {
			t.Fatalf("source trace/warning order:\ngot %q\nwant %q", got, want)
		}
	} else if got != warning {
		t.Fatalf("erased trace emitted %q", got)
	}
}

func TestASTWellSortedNonBooleanRootTraceProfiles(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("ws")
	tags.Enable("tc")
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	sort, err := m.MkSort(astTestSymbol(t, m, "Value"), ASTSortInfo{})
	paramsMust(t, err)
	other, err := m.MkSort(astTestSymbol(t, m, "OtherValue"), ASTSortInfo{})
	paramsMust(t, err)
	arg, err := m.MkVar(0, sort)
	paramsMust(t, err)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{sort}, sort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	app, err := m.MkApp(decl, []ASTHandle{arg})
	paramsMust(t, err)
	appRef, err := m.internal(app)
	paramsMust(t, err)
	varRef, err := m.internal(arg)
	paramsMust(t, err)
	otherRef, err := m.internal(other)
	paramsMust(t, err)
	warnings := diagnostic.NewWarningService(&output)
	m.appAt(appRef.slot).count = 0
	good, checkErr := m.IsWellSortedWithPrinter(app, warnings, c, p, g)
	m.appAt(appRef.slot).count = 1
	paramsMust(t, checkErr)
	if good {
		t.Fatal("non-Boolean wrong arity accepted")
	}
	if tags.ShouldTrace("ws") && !strings.Contains(output.String(), "unexpected number of arguments.\nf") {
		t.Fatalf("non-Boolean ws expression missing: %q", output.String())
	}
	output.Reset()
	old := m.varAt(varRef.slot).sort
	m.varAt(varRef.slot).sort = otherRef
	good, checkErr = m.IsWellSortedWithPrinter(app, warnings, c, p, g)
	m.varAt(varRef.slot).sort = old
	paramsMust(t, checkErr)
	if good {
		t.Fatal("non-Boolean sort mismatch accepted")
	}
	if tags.ShouldTrace("tc") {
		got := output.String()
		if !strings.Contains(got, "(f (:var 0))Sort mismatch for argument 1 of (f (:var 0))\n") ||
			!strings.Contains(got, "Expected sort: Value\nActual sort:   OtherValue\n") ||
			!strings.Contains(got, "------------------------------------------------\nWARNING: Sort mismatch for argument 1") {
			t.Fatalf("non-Boolean tc expression missing: %q", got)
		}
	}
}
