package pdr_test

// Source: src/muz/pdr/pdr_sym_mux.cpp tuple growth, declaration conversion,
// index collection and literal filtering.

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/muz/pdr"
)

func pdrFunction(t *testing.T, manager *ast.ASTManager, name string, domain []ast.ASTHandle, rangeSort ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	symbol, err := manager.Symbols.Intern(name)
	pdrMust(t, err)
	declaration, err := manager.MkFuncDecl(symbol, domain, rangeSort, ast.ASTFuncDeclInfo{})
	pdrMust(t, err)
	return declaration
}

func TestSymbolMuxTupleGrowthAndFormulaConversion(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 0)
	pdrMust(t, contexts.Close())
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	prefix := pdrFunction(t, manager, "state", []ast.ASTHandle{boolean}, boolean)
	mux, err := pdr.NewSymbolMux(manager)
	pdrMust(t, err)
	tuple, err := mux.CreateTuple(prefix, []ast.ASTHandle{boolean}, boolean, 2, nil)
	pdrMust(t, err)
	if len(tuple) != 2 {
		t.Fatal("tuple length", len(tuple))
	}
	for index, want := range []string{"state_n", "state_0"} {
		name, x := manager.GetASTName(tuple[index])
		pdrMust(t, x)
		got, present, x := manager.Symbols.BareString(name)
		pdrMust(t, x)
		if !present || got != want {
			t.Fatal("tuple name", index, got, present, want)
		}
	}
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	state0, err := manager.MkApp(tuple[0], []ast.ASTHandle{truth})
	pdrMust(t, err)
	state1, err := manager.MkApp(tuple[1], []ast.ASTHandle{truth})
	pdrMust(t, err)
	mixed, err := manager.MkAnd(state0, state1)
	pdrMust(t, err)

	converted, err := mux.ConvertFormula(mixed, 0, 2, false)
	pdrMust(t, err)
	left, err := manager.GetArg(converted, 0)
	pdrMust(t, err)
	leftDecl, err := manager.GetDecl(left)
	pdrMust(t, err)
	index, found, err := mux.TryIndex(leftDecl)
	pdrMust(t, err)
	if !found || index != 2 {
		t.Fatal("converted declaration index", index, found)
	}
	right, err := manager.GetArg(converted, 1)
	pdrMust(t, err)
	if right != state1 {
		t.Fatal("conversion changed another state index")
	}
	if _, err = mux.ConvertFormula(mixed, 0, 2, true); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("homogeneous conversion accepted mixed indices", err)
	}
	shifted, err := mux.ShiftFormula(state1, -1)
	pdrMust(t, err)
	shiftedDecl, err := manager.GetDecl(shifted)
	pdrMust(t, err)
	if shiftedDecl != tuple[0] {
		t.Fatal("negative shift declaration", shiftedDecl, tuple[0])
	}
	if _, err = mux.ShiftFormula(state0, -1); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("negative index shift", err)
	}
	third, err := mux.Convert(tuple[0], 0, 2)
	pdrMust(t, err)
	if third != leftDecl {
		t.Fatal("tuple growth was not canonical", third, leftDecl)
	}
	primary, err := mux.Primary(third)
	pdrMust(t, err)
	if primary != tuple[0] {
		t.Fatal("primary declaration", primary, tuple[0])
	}
	byPrefix, found, err := mux.TryPrimaryByPrefix(prefix)
	pdrMust(t, err)
	if !found || byPrefix != tuple[0] {
		t.Fatal("prefix lookup", byPrefix, found)
	}

	pdrMust(t, mux.Close())
	pdrMust(t, command.Close())
}

func TestSymbolMuxClassificationCollectionAndFiltering(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 0)
	pdrMust(t, contexts.Close())
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	prefix := pdrFunction(t, manager, "cell", nil, boolean)
	mux, err := pdr.NewSymbolMux(manager)
	pdrMust(t, err)
	tuple, err := mux.CreateTuple(prefix, nil, boolean, 3, nil)
	pdrMust(t, err)
	values := make([]ast.ASTHandle, len(tuple))
	for index := range tuple {
		values[index], err = manager.MkApp(tuple[index], nil)
		pdrMust(t, err)
	}
	mixed, err := manager.MkAnd(values[1], values[2])
	pdrMust(t, err)
	plain := pdrBool(t, manager, "plain")

	if found, x := mux.Contains(mixed, 2); x != nil || !found {
		t.Fatal("contains", found, x)
	}
	if homogeneous, x := mux.IsHomogeneous(mixed); x != nil || homogeneous {
		t.Fatal("mixed homogeneity", homogeneous, x)
	}
	if homogeneous, x := mux.IsHomogeneousAt(values[1], 1); x != nil || !homogeneous {
		t.Fatal("indexed homogeneity", homogeneous, x)
	}
	indices, err := mux.CollectIndices(mixed)
	pdrMust(t, err)
	if len(indices) != 2 || indices[0] != 0 || indices[1] != 1 {
		t.Fatal("collected source-relative indices", indices)
	}
	variables, err := mux.CollectVariables(mixed)
	pdrMust(t, err)
	if len(variables) != 2 || len(variables[0]) != 1 || variables[0][0] != values[1] || len(variables[1]) != 1 || variables[1][0] != values[2] {
		t.Fatal("collected variables", variables)
	}
	if _, err = mux.CollectIndices(values[0]); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("primary index collection", err)
	}

	filtered, err := mux.FilterIndex([]ast.ASTHandle{plain, values[1], values[2]}, 1)
	pdrMust(t, err)
	if len(filtered) != 1 || filtered[0] != values[1] {
		t.Fatal("index filter", filtered)
	}
	matching, other, err := mux.PartitionIndex([]ast.ASTHandle{plain, values[1], values[2]}, 2)
	pdrMust(t, err)
	if len(matching) != 1 || matching[0] != values[2] || len(other) != 2 {
		t.Fatal("index partition", matching, other)
	}
	pdrMust(t, mux.MarkNonModel(tuple[1]))
	if found, x := mux.HasNonModelSymbol(mixed); x != nil || !found {
		t.Fatal("non-model occurrence", found, x)
	}
	filtered, err = mux.FilterNonModel([]ast.ASTHandle{plain, values[1], values[2]})
	pdrMust(t, err)
	if len(filtered) != 1 || filtered[0] != plain {
		t.Fatal("non-model tuple filter", filtered)
	}
	negated, err := manager.MkNot(values[2])
	pdrMust(t, err)
	if muxed, x := mux.IsMuxedLiteral(negated, 0); x != nil || !muxed {
		t.Fatal("negated muxed literal", muxed, x)
	}
	primaries, err := mux.PrimaryPredicates()
	pdrMust(t, err)
	if len(primaries) != 1 || primaries[0] != tuple[0] {
		t.Fatal("primary predicates", primaries)
	}
	modelCore, err := model.NewModelCore(manager)
	pdrMust(t, err)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	falsity, err := manager.MkFalse()
	pdrMust(t, err)
	plainDecl, err := manager.GetDecl(plain)
	pdrMust(t, err)
	pdrMust(t, modelCore.RegisterConstant(tuple[2], truth))
	pdrMust(t, modelCore.RegisterConstant(plainDecl, truth))
	pdrMust(t, modelCore.RegisterConstant(tuple[0], falsity))
	text, err := mux.ModelString(modelCore)
	pdrMust(t, err)
	first := strings.Index(text, "cell_n -> false")
	second := strings.Index(text, "cell_1 -> true")
	last := strings.Index(text, "plain -> true")
	if first < 0 || second <= first || last <= second {
		t.Fatal("model order", text)
	}
	pdrMust(t, modelCore.Close())

	pdrMust(t, mux.Close())
	if _, err = mux.Contains(plain, 0); !errors.Is(err, pdr.ErrSymbolMuxOwner) {
		t.Fatal("closed symbol mux", err)
	}
	pdrMust(t, command.Close())
}
