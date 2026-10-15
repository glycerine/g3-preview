package ast

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTWellSortedSourceMismatchPrinter(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	otherSort, err := m.MkSort(astTestSymbol(t, m, "OtherGuard"), ASTSortInfo{})
	paramsMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	paramsMust(t, err)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "guard"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	paramsMust(t, err)
	app, err := m.MkApp(decl, []ASTHandle{variable})
	paramsMust(t, err)
	varRef, err := m.internal(variable)
	paramsMust(t, err)
	otherRef, err := m.internal(otherSort)
	paramsMust(t, err)
	id, err := m.GetASTID(variable)
	paramsMust(t, err)
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	old := m.varAt(varRef.slot).sort
	m.varAt(varRef.slot).sort = otherRef
	good, checkErr := m.IsWellSortedWithPrinter(app, warnings, c, p, g)
	m.varAt(varRef.slot).sort = old
	paramsMust(t, checkErr)
	want := fmt.Sprintf("WARNING: Sort mismatch for argument 1 of decl guard :: (-> Bool Bool)\n#%d := (:var 0 OtherGuard)\n(guard #%d)\n\nExpected sort: Bool\nActual sort:   OtherGuard\nFunction sort: (declare-fun guard (Bool) Bool).\n", id, id)
	if good || output.String() != want {
		t.Fatalf("source mismatch warning: good=%v\ngot  %q\nwant %q", good, output.String(), want)
	}
	previous, err := uparams.BindProcessGlobalParamsService(g)
	paramsMust(t, err)
	defer func() { _, e := uparams.BindProcessGlobalParamsService(previous); paramsMust(t, e) }()
	output.Reset()
	m.varAt(varRef.slot).sort = otherRef
	good, checkErr = m.IsWellSorted(app, warnings)
	m.varAt(varRef.slot).sort = old
	paramsMust(t, checkErr)
	if good || output.String() != want {
		t.Fatalf("process-bound source mismatch warning: good=%v\ngot  %q\nwant %q", good, output.String(), want)
	}
}
