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

func TestSymMuxClassifiesCollectsAndFiltersStateLiterals(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	manager, err := command.Manager()
	spacerMust(t, err)
	boolean, err := manager.MkBoolSort()
	spacerMust(t, err)
	name, err := manager.Symbols.Intern("ClassifiedState")
	spacerMust(t, err)
	prefix, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	spacerMust(t, err)
	mux, err := spacer.NewSymMux(manager, []string{"_n"})
	spacerMust(t, err)
	tuple, err := mux.CreateTuple(prefix, 2)
	spacerMust(t, err)
	thirdDeclaration, err := mux.ConvertDeclaration(tuple[1], 1, 2)
	spacerMust(t, err)
	current, err := manager.MkApp(tuple[0], nil)
	spacerMust(t, err)
	old, err := manager.MkApp(tuple[1], nil)
	spacerMust(t, err)
	third, err := manager.MkApp(thirdDeclaration, nil)
	spacerMust(t, err)
	mixed, err := manager.MkAnd(old, third)
	spacerMust(t, err)
	plain, err := manager.MkTrue()
	spacerMust(t, err)
	notOld, err := manager.MkNot(old)
	spacerMust(t, err)
	if muxed, err := mux.IsMuxedLiteral(notOld, 99); err != nil || !muxed {
		t.Fatal("negated muxed literal classification", muxed, err)
	}

	if homogeneous, err := mux.IsHomogeneousAt(old, 1); err != nil || !homogeneous {
		t.Fatal("old-state homogeneity", homogeneous, err)
	}
	if homogeneous, err := mux.IsHomogeneous(mixed); err != nil || homogeneous {
		t.Fatal("mixed-state homogeneity", homogeneous, err)
	}
	if _, err = mux.CollectIndices(current); err == nil {
		t.Fatal("current-state index collector accepted source index zero")
	}
	indices, err := mux.CollectIndices(mixed)
	spacerMust(t, err)
	if len(indices) != 2 || indices[0] != 0 || indices[1] != 1 {
		t.Fatal("collected state indices", indices)
	}
	variables, err := mux.CollectVariables(mixed)
	spacerMust(t, err)
	if len(variables) != 2 || len(variables[0]) != 1 || variables[0][0] != old || len(variables[1]) != 1 || variables[1][0] != third {
		t.Fatal("collected state variables", variables)
	}
	filtered, err := mux.FilterIndex([]ast.ASTHandle{plain, old, third, mixed}, 1)
	spacerMust(t, err)
	if len(filtered) != 1 || filtered[0] != old {
		t.Fatal("state-index filter", filtered)
	}
	matching, other, err := mux.PartitionIndex([]ast.ASTHandle{plain, old, third, mixed}, 2)
	spacerMust(t, err)
	if len(matching) != 1 || matching[0] != third || len(other) != 3 {
		t.Fatal("state-index partition", matching, other)
	}
	spacerMust(t, mux.MarkNonModel(tuple[1]))
	if nonModel, err := mux.IsNonModel(thirdDeclaration); err != nil || !nonModel {
		t.Fatal("tuple-wide non-model marker", nonModel, err)
	}
	kept, err := mux.FilterNonModel([]ast.ASTHandle{plain, old, third, mixed})
	spacerMust(t, err)
	if len(kept) != 1 || kept[0] != plain {
		t.Fatal("non-model literal filter", kept)
	}
	primaries, err := mux.PrimaryPredicates()
	spacerMust(t, err)
	if len(primaries) != 1 || primaries[0] != tuple[0] {
		t.Fatal("primary predicates", primaries)
	}
	spacerMust(t, mux.Close())
	spacerMust(t, command.Close())
}
