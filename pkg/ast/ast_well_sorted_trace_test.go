package ast

import (
	"bytes"
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
