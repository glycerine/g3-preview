package spacer_test

// Source: src/muz/spacer/spacer_sym_mux.cpp tuple growth, declaration
// conversion, formula rewriting, and shifting.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/spacer"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSymMuxCreatesAndRewritesStateVariants(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	manager, err := command.Manager()
	spacerMust(t, err)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name, err := manager.Symbols.Intern("State")
	spacerMust(t, err)
	prefix, err := manager.MkFuncDecl(name, []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	mux, err := spacer.NewSymMux(manager, []string{"_n"})
	spacerMust(t, err)
	tuple, err := mux.CreateTuple(prefix, 2)
	spacerMust(t, err)
	if len(tuple) != 2 {
		t.Fatal("tuple length", len(tuple))
	}
	zero, err := manager.MkTrue()
	spacerMust(t, err)
	current, err := manager.MkApp(tuple[0], []ast.ASTHandle{zero})
	spacerMust(t, err)
	old, err := manager.MkApp(tuple[1], []ast.ASTHandle{zero})
	spacerMust(t, err)
	formula, err := manager.MkAnd(current, old)
	spacerMust(t, err)
	if _, err = mux.ConvertFormula(formula, 0, 1, true); err == nil {
		t.Fatal("homogeneous conversion accepted a mixed state formula")
	}
	converted, err := mux.ConvertFormula(current, 0, 1, true)
	spacerMust(t, err)
	convertedHandle, err := manager.RefGet(converted)
	spacerMust(t, err)
	convertedDecl, err := manager.GetDecl(convertedHandle)
	spacerMust(t, err)
	if convertedDecl != tuple[1] {
		t.Fatal("current-to-old declaration", convertedDecl, tuple[1])
	}
	spacerMust(t, manager.DelRef(&converted))
	shifted, err := mux.ShiftFormula(old, 1)
	spacerMust(t, err)
	shiftedHandle, err := manager.RefGet(shifted)
	spacerMust(t, err)
	shiftedDecl, err := manager.GetDecl(shiftedHandle)
	spacerMust(t, err)
	index, found, err := mux.Index(shiftedDecl)
	spacerMust(t, err)
	if !found || index != 2 {
		t.Fatal("shifted declaration index", index, found)
	}
	spacerMust(t, manager.DelRef(&shifted))
	variable, err := manager.MkVar(0, boolean)
	spacerMust(t, err)
	currentVariable, err := manager.MkApp(tuple[0], []ast.ASTHandle{variable})
	spacerMust(t, err)
	pattern, err := manager.MkPattern([]ast.ASTHandle{currentVariable})
	spacerMust(t, err)
	quantified, err := manager.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{name},
		Body: currentVariable, Patterns: []ast.ASTHandle{pattern},
	})
	spacerMust(t, err)
	converted, err = mux.ConvertFormula(quantified, 0, 1, true)
	spacerMust(t, err)
	convertedHandle, err = manager.RefGet(converted)
	spacerMust(t, err)
	convertedBody, err := manager.GetQuantifierBody(convertedHandle)
	spacerMust(t, err)
	convertedDecl, err = manager.GetDecl(convertedBody)
	spacerMust(t, err)
	if convertedDecl != tuple[1] {
		t.Fatal("quantifier body declaration", convertedDecl, tuple[1])
	}
	convertedPattern, err := manager.GetPattern(convertedHandle, 0)
	spacerMust(t, err)
	patternTerm, err := manager.GetArg(convertedPattern, 0)
	spacerMust(t, err)
	convertedDecl, err = manager.GetDecl(patternTerm)
	spacerMust(t, err)
	if convertedDecl != tuple[1] {
		t.Fatal("quantifier pattern declaration", convertedDecl, tuple[1])
	}
	spacerMust(t, manager.DelRef(&converted))
	spacerMust(t, mux.Close())
	spacerMust(t, command.Close())
}
