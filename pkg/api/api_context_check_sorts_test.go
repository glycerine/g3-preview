package api

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAPIContextSortCheckWarningAndErrorState(t *testing.T) {
	m, c := apiResultsFixture(t, true)
	var output bytes.Buffer
	c.warnings = diagnostic.NewWarningService(&output)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, boolSort)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ast.ASTHandle{boolSort}, boolSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	app, e := m.MkApp(d, []ast.ASTHandle{v})
	astTestMust(t, e)
	astTestMust(t, c.checkSorts(app))
	if c.errorCode != 0 || output.Len() != 0 {
		t.Fatal("valid source application changed API error state", c.errorCode, output.String())
	}
	var calls []uint32
	c.errorHandler = func(owner *apiContextResults, code uint32) {
		if owner != c || owner.errorCode != code || output.String() != "WARNING: expression is null\n" {
			t.Fatalf("handler ran before code or warning: owner=%p code=%d state=%d output=%q", owner, code, owner.errorCode, output.String())
		}
		calls = append(calls, code)
	}
	message, e := c.sortApplicationWarning(app)
	astTestMust(t, e)
	if !strings.Contains(message, " applied to: ") || !strings.Contains(message, "(:var 0) of sort Bool\n") {
		t.Fatal("source application warning formatter", message)
	}
	astTestMust(t, c.checkSorts(ast.ASTHandle{}))
	if c.errorCode != apiSortErrorCode || output.String() != "WARNING: expression is null\n" || len(calls) != 1 || calls[0] != apiSortErrorCode {
		t.Fatal("source first warning and typed null continuation", c.errorCode, output.String())
	}
	c.setErrorCode(0)
	if c.errorCode != 0 || len(calls) != 1 {
		t.Fatal("zero error code invoked handler", c.errorCode, calls)
	}
	c.errorHandler = nil
	astTestMust(t, c.warnings.Enable(false))
	output.Reset()
	astTestMust(t, c.checkSorts(ast.ASTHandle{}))
	if c.errorCode != apiSortErrorCode || output.Len() != 0 {
		t.Fatal("disabled warnings changed sort error", c.errorCode, output.String())
	}
}
