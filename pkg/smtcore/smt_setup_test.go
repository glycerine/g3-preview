package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTQFUFBVSetupUsesNativeArrayAndBitVectorRoute(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	arrayFamily, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	bvFamily, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_UFBV", SMTSolverConfig{
		Params:         modelConfig.Context,
		ModelEvaluator: modelConfig,
	})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	if _, ok := solver.kernel.internalizer.theories[arrayFamily].(*SMTArrayTheory); !ok {
		t.Fatal("QF_UFBV did not install the native shared array theory")
	}
	if _, ok := solver.kernel.internalizer.theories[bvFamily].(*SMTBitVectorTheory); !ok {
		t.Fatal("QF_UFBV did not install the bit-vector theory")
	}
}

func TestSMTQFAUFLIASetupUsesIntegerArithmeticAndArrays(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	arithmeticFamily, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	arrayFamily, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	integerSort, err := m.MkFamilySort(arithmeticFamily, 1, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "qf-auflia-x"), integerSort)
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "qf-auflia-y"), integerSort)
	astTestMust(t, err)
	less, err := m.MkFamilyApp(arithmeticFamily, smtArithOpLT, nil, []ast.ASTHandle{x, y})
	astTestMust(t, err)
	greaterEqual, err := m.MkFamilyApp(arithmeticFamily, smtArithOpGE, nil, []ast.ASTHandle{x, y})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_AUFLIA", SMTSolverConfig{
		Params:         modelConfig.Context,
		ModelEvaluator: modelConfig,
	})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	arithmetic, ok := solver.kernel.internalizer.theories[arithmeticFamily].(*SMTLinearArithmeticTheory)
	if !ok || !arithmetic.integer {
		t.Fatal("QF_AUFLIA did not install concrete integer linear arithmetic")
	}
	if _, ok = solver.kernel.internalizer.theories[arrayFamily].(*SMTArrayTheory); !ok {
		t.Fatal("QF_AUFLIA did not install the array theory")
	}
	astTestMust(t, solver.Assert(less))
	astTestMust(t, solver.Assert(greaterEqual))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("QF_AUFLIA linear contradiction", status)
	}
}

func TestSMTLinearRealLogicSetupUsesConcreteArithmetic(t *testing.T) {
	for _, test := range []struct {
		logic      string
		wantArrays bool
	}{
		{logic: "LRA"},
		{logic: "UFLRA", wantArrays: true},
	} {
		t.Run(test.logic, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			arithmeticFamily, err := m.RegisterArithmeticPlugin()
			astTestMust(t, err)
			arrayFamily, err := m.RegisterArrayPlugin()
			astTestMust(t, err)
			realSort, err := m.MkFamilySort(arithmeticFamily, 0, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "linear-real-setup-x"), realSort)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "linear-real-setup-y"), realSort)
			astTestMust(t, err)
			less, err := m.MkFamilyApp(arithmeticFamily, smtArithOpLT, nil, []ast.ASTHandle{x, y})
			astTestMust(t, err)
			greaterEqual, err := m.MkFamilyApp(arithmeticFamily, smtArithOpGE, nil, []ast.ASTHandle{x, y})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, test.logic, SMTSolverConfig{
				Params:         modelConfig.Context,
				ModelEvaluator: modelConfig,
			})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			arithmetic, ok := solver.kernel.internalizer.theories[arithmeticFamily].(*SMTLinearArithmeticTheory)
			if !ok || arithmetic.integer {
				t.Fatal(test.logic, "did not install concrete rational linear arithmetic")
			}
			_, hasArrays := solver.kernel.internalizer.theories[arrayFamily].(*SMTArrayTheory)
			if hasArrays != test.wantArrays {
				t.Fatal(test.logic, "array route", hasArrays, "want", test.wantArrays)
			}
			astTestMust(t, solver.Assert(less))
			astTestMust(t, solver.Assert(greaterEqual))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal(test.logic, "linear contradiction", status)
			}
		})
	}
}

func TestSMTQuantifiedIntegerLogicSetupUsesConcreteLinearSubset(t *testing.T) {
	for _, logic := range []string{"AUFLIA", "AUFLIA+", "AUFLIA-", "UFNIA"} {
		t.Run(logic, func(t *testing.T) {
			m, modelConfig := smtModelGeneratorFixture(t)
			arithmeticFamily, err := m.RegisterArithmeticPlugin()
			astTestMust(t, err)
			arrayFamily, err := m.RegisterArrayPlugin()
			astTestMust(t, err)
			integerSort, err := m.MkFamilySort(arithmeticFamily, 1, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "quantified-integer-setup-x"), integerSort)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "quantified-integer-setup-y"), integerSort)
			astTestMust(t, err)
			less, err := m.MkFamilyApp(arithmeticFamily, smtArithOpLT, nil, []ast.ASTHandle{x, y})
			astTestMust(t, err)
			greaterEqual, err := m.MkFamilyApp(arithmeticFamily, smtArithOpGE, nil, []ast.ASTHandle{x, y})
			astTestMust(t, err)
			solver, err := NewSMTSolver(m, nil, logic, SMTSolverConfig{
				Params:         modelConfig.Context,
				ModelEvaluator: modelConfig,
			})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			arithmetic, ok := solver.kernel.internalizer.theories[arithmeticFamily].(*SMTLinearArithmeticTheory)
			if !ok || !arithmetic.integer {
				t.Fatal(logic, "did not install the concrete integer linear subset")
			}
			if _, ok = solver.kernel.internalizer.theories[arrayFamily].(*SMTArrayTheory); !ok {
				t.Fatal(logic, "did not install the native shared array theory")
			}
			astTestMust(t, solver.Assert(less))
			astTestMust(t, solver.Assert(greaterEqual))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal(logic, "linear contradiction", status)
			}
		})
	}
}
