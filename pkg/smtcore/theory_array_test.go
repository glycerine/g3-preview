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
