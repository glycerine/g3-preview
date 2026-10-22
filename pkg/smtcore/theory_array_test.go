package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
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
