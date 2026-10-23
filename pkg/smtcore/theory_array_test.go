package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func TestSMTArrayReadOverWrite(t *testing.T) {
	for _, test := range []struct {
		name      string
		sameIndex bool
		disequal  bool
		want      lbool.LBool
	}{
		{name: "same-index-conflict", sameIndex: true, disequal: true, want: lbool.False},
		{name: "different-index-conflict", disequal: true, want: lbool.False},
		{name: "different-index-consistent", want: lbool.True},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := astTestManager(t)
			family, err := m.RegisterArrayPlugin()
			astTestMust(t, err)
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "array-index-sort"), nil)
			astTestMust(t, err)
			arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
			astTestMust(t, err)
			base, err := m.MkConst(astTestSymbol(t, m, "array-base"), arraySort)
			astTestMust(t, err)
			first, err := m.MkConst(astTestSymbol(t, m, "array-first-index"), indexSort)
			astTestMust(t, err)
			second := first
			if !test.sameIndex {
				second, err = m.MkConst(astTestSymbol(t, m, "array-second-index"), indexSort)
				astTestMust(t, err)
			}
			value, err := m.MkConst(astTestSymbol(t, m, "array-value"), boolean)
			astTestMust(t, err)
			stored, err := m.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, first, value})
			astTestMust(t, err)
			selected, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{stored, second})
			astTestMust(t, err)
			wantValue := value
			if !test.sameIndex {
				wantValue, err = m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{base, second})
				astTestMust(t, err)
			}
			equality, err := m.MkEq(selected, wantValue)
			astTestMust(t, err)
			if test.disequal {
				equality, err = m.MkNot(equality)
				astTestMust(t, err)
			}
			params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
			if !test.sameIndex {
				indexesEqual, makeErr := m.MkEq(first, second)
				astTestMust(t, makeErr)
				indexesDiffer, makeErr := m.MkNot(indexesEqual)
				astTestMust(t, makeErr)
				astTestMust(t, solver.Assert(indexesDiffer))
			}
			astTestMust(t, solver.Assert(equality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != test.want {
				reason, _ := solver.ReasonUnknown()
				theory, _ := solver.kernel.internalizer.theories[family].(*SMTArrayTheory)
				t.Fatalf("array status %v, want %v (reason %q, unsupported %v, stores %d, selects %d)", status, test.want, reason, theory.unsupported, len(theory.storeTrail), len(theory.selectTrail))
			}
		})
	}
}

func TestSMTArrayReadOverWriteThroughEquality(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "array-alias-index-sort"), nil)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
	astTestMust(t, err)
	base, err := m.MkConst(astTestSymbol(t, m, "array-alias-base"), arraySort)
	astTestMust(t, err)
	alias, err := m.MkConst(astTestSymbol(t, m, "array-alias"), arraySort)
	astTestMust(t, err)
	storedIndex, err := m.MkConst(astTestSymbol(t, m, "array-alias-stored-index"), indexSort)
	astTestMust(t, err)
	selectedIndex, err := m.MkConst(astTestSymbol(t, m, "array-alias-selected-index"), indexSort)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "array-alias-value"), boolean)
	astTestMust(t, err)
	stored, err := m.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, storedIndex, value})
	astTestMust(t, err)
	aliasDefinition, err := m.MkEq(alias, stored)
	astTestMust(t, err)
	indexesEqual, err := m.MkEq(storedIndex, selectedIndex)
	astTestMust(t, err)
	indexesDiffer, err := m.MkNot(indexesEqual)
	astTestMust(t, err)
	aliasRead, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{alias, selectedIndex})
	astTestMust(t, err)
	baseRead, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{base, selectedIndex})
	astTestMust(t, err)
	readsEqual, err := m.MkEq(aliasRead, baseRead)
	astTestMust(t, err)
	readsDiffer, err := m.MkNot(readsEqual)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(aliasDefinition))
	astTestMust(t, solver.Assert(indexesDiffer))
	astTestMust(t, solver.Assert(readsDiffer))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("read-over-write through equal array", status)
	}
}

func TestSMTArrayDefaultThroughEquality(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "array-alias-default-value"), boolean)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(arraySort)
	astTestMust(t, err)
	constant, err := m.MkFamilyApp(family, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{value})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameter))
	alias, err := m.MkConst(astTestSymbol(t, m, "array-default-alias"), arraySort)
	astTestMust(t, err)
	aliasDefinition, err := m.MkEq(alias, constant)
	astTestMust(t, err)
	actual, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{alias})
	astTestMust(t, err)
	defaultsEqual, err := m.MkEq(actual, value)
	astTestMust(t, err)
	defaultsDiffer, err := m.MkNot(defaultsEqual)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(aliasDefinition))
	astTestMust(t, solver.Assert(defaultsDiffer))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("default through equal array", status)
	}
}

func TestSMTArrayConstantRead(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "constant-array-index"), nil)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "constant-array-value"), boolean)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(arraySort)
	astTestMust(t, err)
	constant, err := m.MkFamilyApp(family, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{value})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameter))
	index, err := m.MkConst(astTestSymbol(t, m, "constant-array-read-index"), indexSort)
	astTestMust(t, err)
	selected, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{constant, index})
	astTestMust(t, err)
	equality, err := m.MkEq(selected, value)
	astTestMust(t, err)
	disequality, err := m.MkNot(equality)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(disequality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("constant array read disequality", status)
	}
}

func TestSMTArrayExtensionality(t *testing.T) {
	for _, test := range []struct {
		name         string
		equalWitness bool
		want         lbool.LBool
	}{
		{name: "disequal-arrays", want: lbool.True},
		{name: "equal-extensional-read", equalWitness: true, want: lbool.False},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := astTestManager(t)
			family, err := m.RegisterArrayPlugin()
			astTestMust(t, err)
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "extensional-index"), nil)
			astTestMust(t, err)
			arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
			astTestMust(t, err)
			left, err := m.MkConst(astTestSymbol(t, m, "extensional-left"), arraySort)
			astTestMust(t, err)
			right, err := m.MkConst(astTestSymbol(t, m, "extensional-right"), arraySort)
			astTestMust(t, err)
			equality, err := m.MkEq(left, right)
			astTestMust(t, err)
			disequality, err := m.MkNot(equality)
			astTestMust(t, err)
			params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
			astTestMust(t, solver.Assert(disequality))
			if test.equalWitness {
				extDeclaration, makeErr := m.MkArrayExtDecl(arraySort, 0)
				astTestMust(t, makeErr)
				witness, makeErr := m.MkApp(extDeclaration, []ast.ASTHandle{left, right})
				astTestMust(t, makeErr)
				leftSelect, makeErr := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{left, witness})
				astTestMust(t, makeErr)
				rightSelect, makeErr := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{right, witness})
				astTestMust(t, makeErr)
				selectEquality, makeErr := m.MkEq(leftSelect, rightSelect)
				astTestMust(t, makeErr)
				astTestMust(t, solver.Assert(selectEquality))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != test.want {
				t.Fatalf("array extensionality status %v, want %v", status, test.want)
			}
		})
	}
}

func TestSMTArrayMapRead(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "map-array-index"), nil)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
	astTestMust(t, err)
	input, err := m.MkConst(astTestSymbol(t, m, "map-array-input"), arraySort)
	astTestMust(t, err)
	function, err := m.MkFuncDecl(astTestSymbol(t, m, "map-array-function"), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	mapped, err := m.MkArrayMap(function, ast.SliceArguments([]ast.ASTHandle{input}))
	astTestMust(t, err)
	index, err := m.MkConst(astTestSymbol(t, m, "map-array-read-index"), indexSort)
	astTestMust(t, err)
	selected, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{mapped, index})
	astTestMust(t, err)
	inputValue, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{input, index})
	astTestMust(t, err)
	want, err := m.MkApp(function, []ast.ASTHandle{inputValue})
	astTestMust(t, err)
	equality, err := m.MkEq(selected, want)
	astTestMust(t, err)
	disequality, err := m.MkNot(equality)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(disequality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("mapped array read disequality", status)
	}
	stats := statistics.New()
	astTestMust(t, solver.CollectStatistics(stats))
	found := false
	for index := 0; index < stats.Size(); index++ {
		if stats.Key(index) == "array map ax" {
			found = stats.IsUint(index) && stats.UintValue(index) != 0
		}
	}
	if !found {
		t.Fatal("missing array map statistics")
	}
	astTestMust(t, solver.kernel.ResetStatistics())
	stats.Reset()
	astTestMust(t, solver.CollectStatistics(stats))
	for index := 0; index < stats.Size(); index++ {
		if stats.Key(index) == "array map ax" {
			t.Fatal("array statistics survived reset")
		}
	}
}

func TestSMTArrayDefaultConstAndMap(t *testing.T) {
	for _, mapCase := range []bool{false, true} {
		name := "const"
		if mapCase {
			name = "map"
		}
		t.Run(name, func(t *testing.T) {
			m := astTestManager(t)
			family, err := m.RegisterArrayPlugin()
			astTestMust(t, err)
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "default-array-index"), nil)
			astTestMust(t, err)
			arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
			astTestMust(t, err)
			value, err := m.MkConst(astTestSymbol(t, m, "default-array-value"), boolean)
			astTestMust(t, err)
			array := ast.ASTHandle{}
			want := value
			if mapCase {
				input, makeErr := m.MkConst(astTestSymbol(t, m, "default-map-input"), arraySort)
				astTestMust(t, makeErr)
				function, makeErr := m.MkFuncDecl(astTestSymbol(t, m, "default-map-function"), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
				astTestMust(t, makeErr)
				array, makeErr = m.MkArrayMap(function, ast.SliceArguments([]ast.ASTHandle{input}))
				astTestMust(t, makeErr)
				inputDefault, makeErr := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{input})
				astTestMust(t, makeErr)
				want, makeErr = m.MkApp(function, []ast.ASTHandle{inputDefault})
				astTestMust(t, makeErr)
			} else {
				parameter, makeErr := m.NewParameterAST(arraySort)
				astTestMust(t, makeErr)
				array, makeErr = m.MkFamilyApp(family, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{value})
				astTestMust(t, makeErr)
				astTestMust(t, m.DelParameter(&parameter))
			}
			actual, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{array})
			astTestMust(t, err)
			equality, err := m.MkEq(actual, want)
			astTestMust(t, err)
			disequality, err := m.MkNot(equality)
			astTestMust(t, err)
			params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
			astTestMust(t, solver.Assert(disequality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("array default disequality", status)
			}
		})
	}
}

func TestSMTArrayDefaultStoreLargeDomain(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "default-store-index"), nil)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
	astTestMust(t, err)
	base, err := m.MkConst(astTestSymbol(t, m, "default-store-base"), arraySort)
	astTestMust(t, err)
	index, err := m.MkConst(astTestSymbol(t, m, "default-store-location"), indexSort)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "default-store-value"), boolean)
	astTestMust(t, err)
	stored, err := m.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, index, value})
	astTestMust(t, err)
	storedDefault, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{stored})
	astTestMust(t, err)
	baseDefault, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{base})
	astTestMust(t, err)
	equality, err := m.MkEq(storedDefault, baseDefault)
	astTestMust(t, err)
	disequality, err := m.MkNot(equality)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(disequality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("large-domain store changed default", status)
	}
}

func TestSMTArrayDefaultStoreFiniteDomain(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	astTestMust(t, err)
	base, err := m.MkConst(astTestSymbol(t, m, "finite-default-store-base"), arraySort)
	astTestMust(t, err)
	index, err := m.MkConst(astTestSymbol(t, m, "finite-default-store-index"), boolean)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "finite-default-store-value"), boolean)
	astTestMust(t, err)
	stored, err := m.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, index, value})
	astTestMust(t, err)
	storedDefault, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{stored})
	astTestMust(t, err)
	baseDefault, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{base})
	astTestMust(t, err)
	baseValue, err := m.MkEq(baseDefault, value)
	astTestMust(t, err)
	storedValue, err := m.MkEq(storedDefault, value)
	astTestMust(t, err)
	storedNotValue, err := m.MkNot(storedValue)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(baseValue))
	astTestMust(t, solver.Assert(storedNotValue))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("finite-domain store default ITE", status)
	}
}

func TestSMTArrayPublishesModelValues(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	_, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "model-array-index"), nil)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "model-array-left"), arraySort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "model-array-right"), arraySort)
	astTestMust(t, err)
	equality, err := m.MkEq(left, right)
	astTestMust(t, err)
	disequality, err := m.MkNot(equality)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.SetProduceModels(true))
	astTestMust(t, solver.Assert(disequality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("array model status", status)
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	if generated == nil {
		t.Fatal("array solver omitted model")
	}
	t.Cleanup(func() { astTestMust(t, generated.DecRef()) })
	leftDeclaration, err := m.GetDecl(left)
	astTestMust(t, err)
	rightDeclaration, err := m.GetDecl(right)
	astTestMust(t, err)
	leftValue, err := generated.ConstantInterpretation(leftDeclaration)
	astTestMust(t, err)
	rightValue, err := generated.ConstantInterpretation(rightDeclaration)
	astTestMust(t, err)
	if leftValue.IsNull() || rightValue.IsNull() || leftValue == rightValue {
		t.Fatal("array model did not separate disequal roots", leftValue, rightValue)
	}
}

func TestSMTArrayModelPreservesObservedSelect(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "observed-select-index-sort"), nil)
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort}), boolean)
	astTestMust(t, err)
	array, err := m.MkConst(astTestSymbol(t, m, "observed-select-array"), arraySort)
	astTestMust(t, err)
	index, err := m.MkConst(astTestSymbol(t, m, "observed-select-index"), indexSort)
	astTestMust(t, err)
	selected, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{array, index})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.SetProduceModels(true))
	astTestMust(t, solver.Assert(selected))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("observed select model status", status)
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generated.DecRef()) })
	out := astTestPin(t, m, ast.ASTHandle{})
	evaluator, err := model.NewModelEvaluator(generated, modelConfig)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, evaluator.Close()) })
	astTestMust(t, evaluator.SetModelCompletion(true))
	astTestMust(t, evaluator.Evaluate(selected, out))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if !m.IsTrue(result) {
		t.Fatal("array model lost observed select", result)
	}
}

func TestSMTArrayModelPreservesObservedDefault(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	astTestMust(t, err)
	array, err := m.MkConst(astTestSymbol(t, m, "observed-default-array"), arraySort)
	astTestMust(t, err)
	defaultValue, err := m.MkFamilyApp(family, ast.ASTArrayOpDefault, nil, []ast.ASTHandle{array})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.SetProduceModels(true))
	astTestMust(t, solver.Assert(defaultValue))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("observed default model status", status)
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generated.DecRef()) })
	arrayDeclaration, err := m.GetDecl(array)
	astTestMust(t, err)
	arrayValue, err := generated.ConstantInterpretation(arrayDeclaration)
	astTestMust(t, err)
	function, err := m.AsArrayFunction(arrayValue)
	astTestMust(t, err)
	interpretation, err := generated.FunctionInterpretation(function)
	astTestMust(t, err)
	if interpretation == nil {
		t.Fatal("array model omitted observed default interpretation")
	}
	fallback, err := interpretation.Else()
	astTestMust(t, err)
	if !m.IsTrue(fallback) {
		t.Fatal("array model lost observed default", fallback)
	}
}

func TestSMTArrayConstructorModelValue(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	arraySort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	astTestMust(t, err)
	base, err := m.MkConst(astTestSymbol(t, m, "constructor-model-base"), arraySort)
	astTestMust(t, err)
	index, err := m.MkConst(astTestSymbol(t, m, "constructor-model-index"), boolean)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "constructor-model-value"), boolean)
	astTestMust(t, err)
	stored, err := m.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, index, value})
	astTestMust(t, err)
	selected, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{stored, index})
	astTestMust(t, err)
	equality, err := m.MkEq(selected, value)
	astTestMust(t, err)
	astTestPin(t, m, equality) // Evaluated again after preprocessing reduces it.
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.SetProduceModels(true))
	astTestMust(t, solver.Assert(equality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("array constructor model status", status)
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generated.DecRef()) })
	out := astTestPin(t, m, ast.ASTHandle{})
	evaluator, err := model.NewModelEvaluator(generated, modelConfig)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, evaluator.Close()) })
	astTestMust(t, evaluator.SetModelCompletion(true))
	astTestMust(t, evaluator.Evaluate(equality, out))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if !m.IsTrue(result) {
		t.Fatal("constructor-aware array model falsified store read", result)
	}
}

func TestSMTArrayAsArrayReadAxiom(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	indexSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "as-array-index"), nil)
	astTestMust(t, err)
	function, err := m.MkFuncDecl(astTestSymbol(t, m, "as-array-function"), []ast.ASTHandle{indexSort}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	array, err := m.MkAsArray(function)
	astTestMust(t, err)
	index, err := m.MkConst(astTestSymbol(t, m, "as-array-read-index"), indexSort)
	astTestMust(t, err)
	selected, err := m.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{array, index})
	astTestMust(t, err)
	application, err := m.MkApp(function, []ast.ASTHandle{index})
	astTestMust(t, err)
	equality, err := m.MkEq(selected, application)
	astTestMust(t, err)
	disequality, err := m.MkNot(equality)
	astTestMust(t, err)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AX", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()); astTestMust(t, params.Close()) })
	astTestMust(t, solver.Assert(disequality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("as-array read axiom", status)
	}
}
