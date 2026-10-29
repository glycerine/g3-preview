package spacer_test

// Source: inductive_property::to_model and inductive_property::to_expr in
// src/muz/spacer/spacer_manager.cpp.

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/spacer"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSpacerInductivePropertyBuildsNullaryAndAbstractedModels(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	manager, err := command.Manager()
	spacerMust(t, err)
	globals, err := command.GlobalParameters()
	spacerMust(t, err)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name := func(value string) usymbol.Symbol {
		symbol, internErr := manager.Symbols.Intern(value)
		spacerMust(t, internErr)
		return symbol
	}
	variable, err := manager.MkFuncDecl(name("inductive-x"), nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	predicate, err := manager.MkFuncDecl(name("inductive-p"), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	constantPredicate, err := manager.MkFuncDecl(name("inductive-c"), nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	body, err := manager.MkApp(variable, nil)
	spacerMust(t, err)
	falsity, err := manager.MkFalse()
	spacerMust(t, err)
	property, err := spacer.NewInductiveProperty(manager, []spacer.RelationInfo{
		{Predicate: predicate, Variables: []ast.ASTHandle{variable}, Body: body},
		{Predicate: constantPredicate, Body: falsity},
	})
	spacerMust(t, err)
	md, err := property.Model()
	spacerMust(t, err)
	interpretation, err := md.Core().FunctionInterpretation(predicate)
	spacerMust(t, err)
	if interpretation == nil || interpretation.Arity() != 1 {
		t.Fatal("missing unary inductive interpretation")
	}
	otherwise, err := interpretation.Else()
	spacerMust(t, err)
	if kind, err := manager.GetKind(otherwise); err != nil || kind != ast.ASTVar {
		t.Fatal("signature constant was not abstracted", kind, err)
	}
	value, err := md.Core().ConstantInterpretation(constantPredicate)
	spacerMust(t, err)
	if value != falsity {
		t.Fatal("nullary inductive interpretation", value, falsity)
	}
	spacerMust(t, md.Close())
	formula, err := property.Expr()
	spacerMust(t, err)
	formulaHandle, err := manager.RefGet(formula)
	spacerMust(t, err)
	if formulaHandle.IsNull() {
		t.Fatal("empty inductive expression")
	}
	spacerMust(t, manager.DelRef(&formula))
	printed, err := property.SMT2(globals.Context, globals)
	spacerMust(t, err)
	if !strings.Contains(printed, "define-fun inductive-p") || !strings.Contains(printed, "define-fun inductive-c") {
		t.Fatal("inductive model SMT2", printed)
	}
	spacerMust(t, property.Close())
	spacerMust(t, command.Close())
}
