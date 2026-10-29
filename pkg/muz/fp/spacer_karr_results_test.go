package fp

// Source: src/muz/transforms/dl_mk_karr_invariants.cpp
// add_invariant_model_converter model repair.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/muz/rel"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSpacerKarrConjoinsInvariantWithPredicateModel(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	quantifierResultMust(t, err)
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	quantifierResultMust(t, err)
	t.Cleanup(func() {
		quantifierResultMust(t, numerals.Close())
		quantifierResultMust(t, symbols.Close())
		if used := budget.Used(); used != 0 {
			t.Errorf("owner graph retained %d bytes", used)
		}
	})
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	quantifierResultMust(t, err)
	quantifierResultMust(t, manager.RegisterDeclarationPlugins())
	t.Cleanup(func() { quantifierResultMust(t, manager.Close()) })
	engine, err := NewPropositionalEngine(manager)
	quantifierResultMust(t, err)
	t.Cleanup(func() { quantifierResultMust(t, engine.Close()) })
	boolean, err := manager.MkBoolSort()
	quantifierResultMust(t, err)
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	quantifierResultMust(t, err)
	integer, err := manager.GetSort(zero)
	quantifierResultMust(t, err)
	name, err := manager.Symbols.Intern("karr-model-p")
	quantifierResultMust(t, err)
	predicate, err := manager.MkFuncDecl(name, []ast.ASTHandle{integer}, boolean, ast.ASTFuncDeclInfo{})
	quantifierResultMust(t, err)
	predicateRef, err := manager.NewRef(predicate)
	quantifierResultMust(t, err)
	t.Cleanup(func() { quantifierResultMust(t, manager.DelRef(&predicateRef)) })
	source, err := model.NewModel(manager)
	quantifierResultMust(t, err)
	interpretation, err := model.NewFunctionInterpretation(manager, 1)
	quantifierResultMust(t, err)
	truth, err := manager.MkTrue()
	quantifierResultMust(t, err)
	quantifierResultMust(t, interpretation.SetElse(truth))
	quantifierResultMust(t, source.Core().RegisterFunction(predicate, interpretation))
	matrix := rel.KarrMatrix{A: [][]big.Rat{{*big.NewRat(1, 1)}}, B: []big.Rat{*big.NewRat(-2, 1)}, Eq: []bool{true}}
	engine.spacerKarr = map[ast.ASTHandle]rel.KarrMatrix{predicate: matrix}
	expanded, err := engine.expandSpacerModel(source)
	quantifierResultMust(t, err)
	t.Cleanup(func() { quantifierResultMust(t, expanded.Close()) })
	repaired, err := expanded.Core().FunctionInterpretation(predicate)
	quantifierResultMust(t, err)
	body, err := repaired.Else()
	quantifierResultMust(t, err)
	if kind, getErr := manager.GetDeclKind(body); getErr != nil || kind != ast.ASTOpAnd {
		t.Fatal("Karr model body is not a conjunction", kind, getErr)
	}
	invariant, err := manager.GetArg(body, 1)
	quantifierResultMust(t, err)
	if kind, getErr := manager.GetDeclKind(invariant); getErr != nil || kind != ast.ASTOpEq {
		t.Fatal("Karr equality invariant missing", kind, getErr)
	}
}
