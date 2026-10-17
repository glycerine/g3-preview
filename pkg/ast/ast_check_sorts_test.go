package ast

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTCheckSortsShallowWarningsAndAssociativeChild(t *testing.T) {
	m := astTestManager(t)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	otherSort, e := m.MkSort(astTestSymbol(t, m, "OtherCheck"), ASTSortInfo{})
	astTestMust(t, e)
	v, e := m.MkVar(0, boolSort)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(NullFamilyID, -1, nil, ASTFuncFlags{RightAssociative: true}, false)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelFuncDeclInfo(&info)) })
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "right"), []ASTHandle{boolSort, boolSort}, boolSort, info)
	astTestMust(t, e)
	child, e := m.MkApp(d, []ASTHandle{v, v})
	astTestMust(t, e)
	parent, e := m.MkApp(d, []ASTHandle{v, child})
	astTestMust(t, e)
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	check := func(h ASTHandle, want bool, message string) {
		t.Helper()
		output.Reset()
		good, err := m.CheckSorts(h, warnings)
		astTestMust(t, err)
		if good != want || output.String() != message {
			t.Fatalf("check_sorts(%v): got %v, %q; want %v, %q", h, good, output.String(), want, message)
		}
	}
	check(ASTHandle{}, false, "WARNING: expression is null\n")
	check(otherSort, true, "") // Source accepts any non-application AST.
	check(parent, true, "")
	cr, e := m.internal(child)
	astTestMust(t, e)
	m.appAt(cr.slot).count = 1
	check(parent, false, "WARNING: invalid function application, wrong number of arguments\n")
	m.appAt(cr.slot).count = 2
	vr, e := m.internal(v)
	astTestMust(t, e)
	or, e := m.internal(otherSort)
	astTestMust(t, e)
	old := m.varAt(vr.slot).sort
	m.varAt(vr.slot).sort = or
	output.Reset()
	good, e := m.CheckSorts(parent, warnings)
	m.varAt(vr.slot).sort = old
	astTestMust(t, e)
	if good || !strings.Contains(output.String(), "WARNING: invalid function application for right, sort mismatch on argument at position 1, expected Bool but given OtherCheck\n") {
		t.Fatal("source sort mismatch warning", good, output.String())
	}
	astTestMust(t, warnings.Enable(false))
	output.Reset()
	m.varAt(vr.slot).sort = or
	good, e = m.CheckSorts(parent, warnings)
	m.varAt(vr.slot).sort = old
	astTestMust(t, e)
	if good || output.Len() != 0 {
		t.Fatal("disabled warning still reports sort failure", good, output.String())
	}
}

func TestASTCheckSortsArithmeticCoercionUsesSourceFamily(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	intSort := m.external(m.arithmeticPlugin.fields[1])
	realSort := m.external(m.arithmeticPlugin.fields[0])
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, intSort)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "takes_int"), []ASTHandle{intSort}, boolSort, ASTFuncDeclInfo{})
	astTestMust(t, e)
	app, e := m.MkApp(d, []ASTHandle{v})
	astTestMust(t, e)
	vr, e := m.internal(v)
	astTestMust(t, e)
	realRef, e := m.internal(realSort)
	astTestMust(t, e)
	old := m.varAt(vr.slot).sort
	m.varAt(vr.slot).sort = realRef
	defer func() { m.varAt(vr.slot).sort = old }()
	astTestMust(t, m.EnableIntRealCoercions(false))
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	good, e := m.CheckSorts(app, warnings)
	astTestMust(t, e)
	if good || !strings.Contains(output.String(), "expected Int but given Real") {
		t.Fatal("disabled source int/real compatibility", good, output.String())
	}
	astTestMust(t, m.EnableIntRealCoercions(true))
	output.Reset()
	good, e = m.CheckSorts(app, warnings)
	astTestMust(t, e)
	if !good || output.Len() != 0 {
		t.Fatal("enabled source int/real compatibility", good, output.String())
	}
}
