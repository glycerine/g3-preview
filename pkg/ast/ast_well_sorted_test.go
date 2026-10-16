package ast

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTWellSortedQuantifierWarningAndChildOrder(t *testing.T) {
	m := astTestManager(t)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	otherSort, e := m.MkSort(astTestSymbol(t, m, "Other"), ASTSortInfo{})
	astTestMust(t, e)
	decl, e := m.MkFuncDecl(astTestSymbol(t, m, "c"), nil, otherSort, ASTFuncDeclInfo{})
	astTestMust(t, e)
	constant, e := m.MkApp(decl, nil)
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{boolSort}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: constant})
	astTestMust(t, e)
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	good, e := m.IsWellSorted(constant, warnings)
	astTestMust(t, e)
	if !good || output.Len() != 0 {
		t.Fatalf("ordinary application: good=%v warning=%q", good, output.String())
	}
	good, e = m.IsWellSorted(q, warnings)
	astTestMust(t, e)
	if good || output.String() != "WARNING: quantifier's body must be a boolean.\n" {
		t.Fatalf("nonboolean body: good=%v warning=%q", good, output.String())
	}
	astTestMust(t, warnings.Enable(false))
	output.Reset()
	good, e = m.IsWellSorted(q, warnings)
	astTestMust(t, e)
	if good || output.Len() != 0 {
		t.Fatalf("disabled warning: good=%v warning=%q", good, output.String())
	}
	if _, e = m.IsWellSorted(otherSort, warnings); e == nil {
		t.Fatalf("sort accepted as expression: %v", e)
	}
}

func TestASTWellSortedApplicationGuardBranches(t *testing.T) {
	m := astTestManager(t)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	otherSort, e := m.MkSort(astTestSymbol(t, m, "OtherGuard"), ASTSortInfo{})
	astTestMust(t, e)
	variable, e := m.MkVar(0, boolSort)
	astTestMust(t, e)
	decl, e := m.MkFuncDecl(astTestSymbol(t, m, "guard"), []ASTHandle{boolSort}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, e)
	app, e := m.MkApp(decl, []ASTHandle{variable})
	astTestMust(t, e)
	ar, e := m.internal(app)
	astTestMust(t, e)
	vr, e := m.internal(variable)
	astTestMust(t, e)
	or, e := m.internal(otherSort)
	astTestMust(t, e)
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	good, e := m.IsWellSorted(app, warnings)
	astTestMust(t, e)
	if !good || output.Len() != 0 {
		t.Fatalf("well-sorted app: %v %q", good, output.String())
	}
	// Public factories refuse malformed applications. Temporarily corrupt the
	// already-owned test node to reach the source checker guards, then restore
	// it before manager destruction.
	m.appAt(ar.slot).count = 0
	good, e = m.IsWellSorted(app, warnings)
	m.appAt(ar.slot).count = 1
	astTestMust(t, e)
	if good || output.String() != "WARNING: unexpected number of arguments.\n" {
		t.Fatalf("arity guard: %v %q", good, output.String())
	}
	output.Reset()
	oldSort := m.varAt(vr.slot).sort
	id, e := m.GetASTID(variable)
	astTestMust(t, e)
	m.varAt(vr.slot).sort = or
	good, e = m.IsWellSorted(app, warnings)
	m.varAt(vr.slot).sort = oldSort
	astTestMust(t, e)
	want := fmt.Sprintf("WARNING: Sort mismatch for argument 1 of decl guard :: (-> Bool Bool)\n#%d := (:var 0 OtherGuard)\n(guard #%d)\n\nExpected sort: Bool\nActual sort:   OtherGuard\nFunction sort: (declare-fun guard (Bool) Bool).\n", id, id)
	if good || output.String() != want {
		t.Fatalf("sort guard: %v %q", good, output.String())
	}
}
