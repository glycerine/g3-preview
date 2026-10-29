package fp

// Source: src/muz/transforms/dl_mk_bit_blast.cpp
// bit_blast_model_converter reconstruction and fresh-predicate filtering.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSpacerBitBlastReconstructsBitVectorPredicateModel(t *testing.T) {
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
	bv, err := manager.BitVectorSort(2)
	quantifierResultMust(t, err)
	oldName, err := manager.Symbols.Intern("bb-model-old")
	quantifierResultMust(t, err)
	old, err := manager.MkFuncDecl(oldName, []ast.ASTHandle{bv}, boolean, ast.ASTFuncDeclInfo{})
	quantifierResultMust(t, err)
	newName, err := manager.Symbols.Intern("bb-model-new")
	quantifierResultMust(t, err)
	derived, err := manager.MkFuncDecl(newName, []ast.ASTHandle{boolean, boolean}, boolean, ast.ASTFuncDeclInfo{})
	quantifierResultMust(t, err)
	oldRef, err := manager.NewRef(old)
	quantifierResultMust(t, err)
	derivedRef, err := manager.NewRef(derived)
	quantifierResultMust(t, err)
	t.Cleanup(func() {
		quantifierResultMust(t, manager.DelRef(&derivedRef))
		quantifierResultMust(t, manager.DelRef(&oldRef))
	})
	source, err := model.NewModel(manager)
	quantifierResultMust(t, err)
	derivedInterpretation, err := model.NewFunctionInterpretation(manager, 2)
	quantifierResultMust(t, err)
	firstBit, err := manager.MkVar(0, boolean)
	quantifierResultMust(t, err)
	quantifierResultMust(t, derivedInterpretation.SetElse(firstBit))
	quantifierResultMust(t, source.Core().RegisterFunction(derived, derivedInterpretation))
	engine.spacerBitBlasted = map[ast.ASTHandle]ast.ASTHandle{old: derived}
	expanded, err := engine.expandSpacerModel(source)
	quantifierResultMust(t, err)
	t.Cleanup(func() { quantifierResultMust(t, expanded.Close()) })
	if absent, getErr := expanded.Core().FunctionInterpretation(derived); getErr != nil || absent != nil {
		t.Fatal("derived interpretation remained registered", absent, getErr)
	}
	reconstructed, err := expanded.Core().FunctionInterpretation(old)
	quantifierResultMust(t, err)
	if reconstructed == nil || reconstructed.Arity() != 1 {
		t.Fatal("original interpretation was not reconstructed")
	}
	body, err := reconstructed.Else()
	quantifierResultMust(t, err)
	argument, err := manager.MkVar(0, bv)
	quantifierResultMust(t, err)
	want, err := manager.MkBitVectorBit2Bool(0, argument)
	quantifierResultMust(t, err)
	if body != want {
		t.Fatalf("reconstructed body %v, want %v", body, want)
	}
}
