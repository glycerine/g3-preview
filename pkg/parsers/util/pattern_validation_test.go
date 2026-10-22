package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestPatternValidatorVariablesCoverageAndForbiddenOperators(t *testing.T) {
	manager := parserTestManager(t)
	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	x, err := manager.MkVar(0, boolean)
	if err != nil {
		t.Fatal(err)
	}
	y, err := manager.MkVar(1, boolean)
	if err != nil {
		t.Fatal(err)
	}
	free, err := manager.MkVar(2, boolean)
	if err != nil {
		t.Fatal(err)
	}
	name, err := manager.Symbols.Intern("f")
	if err != nil {
		t.Fatal(err)
	}
	decl, err := manager.MkFuncDecl(name, []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	fx, err := manager.MkApp(decl, []ast.ASTHandle{x})
	if err != nil {
		t.Fatal(err)
	}
	fy, err := manager.MkApp(decl, []ast.ASTHandle{y})
	if err != nil {
		t.Fatal(err)
	}
	ffree, err := manager.MkApp(decl, []ast.ASTHandle{free})
	if err != nil {
		t.Fatal(err)
	}
	and, err := manager.MkFamilyApp(manager.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{fx, fy})
	if err != nil {
		t.Fatal(err)
	}
	for _, handle := range []ast.ASTHandle{x, y, free, decl, fx, fy, ffree, and} {
		ref, pinErr := manager.NewRef(handle)
		if pinErr != nil {
			t.Fatal(pinErr)
		}
		defer manager.DelRef(&ref)
	}
	var warnings bytes.Buffer
	validator, err := NewPatternValidator(manager, &warnings)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := validator.Validate(2, 1, fx, 7, 11); err != nil || !ok {
		t.Fatal(ok, err, warnings.String())
	}
	if ok, err := validator.Validate(2, 2, fx, 7, 11); err != nil || ok || !strings.Contains(warnings.String(), "all quantified variables") {
		t.Fatal(ok, err, warnings.String())
	}
	warnings.Reset()
	if ok, err := validator.Validate(2, 1, ffree, 7, 11); err != nil || ok || !strings.Contains(warnings.String(), "free variables") {
		t.Fatal(ok, err, warnings.String())
	}
	warnings.Reset()
	if ok, err := validator.Validate(2, 2, and, 7, 11); err != nil || ok || !strings.Contains(warnings.String(), "'and' cannot be used") {
		t.Fatal(ok, err, warnings.String())
	}
	warnings.Reset()
	if ok, err := validator.Validate(2, 1, x, 7, 11); err != nil || ok || !strings.Contains(warnings.String(), "invalid pattern: variable") {
		t.Fatal(ok, err, warnings.String())
	}
}
