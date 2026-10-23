package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtFixedEqualityGuardFixture(t *testing.T) (*SMTLinearArithmeticTheory, *simplex.Solver, uint32, uint32) {
	t.Helper()
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_LRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
	var columns [2]uint32
	solver := simplex.NewSolver(nil)
	for index, name := range []string{"guard-x", "guard-y"} {
		term, err := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, err)
		columns[index], err = theory.ensureVariable(term)
		astTestMust(t, err)
		value := simplex.RationalInfinitesimal(big.NewRat(2, 1))
		astTestMust(t, solver.SetLowerWithDependencies(columns[index], value, nil))
		astTestMust(t, solver.SetUpperWithDependencies(columns[index], value, nil))
	}
	return theory, solver, columns[0], columns[1]
}

func TestSMTArithmeticEqualityPropagationHonorsConfigurationAndConflictThreshold(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		enabled              bool
		threshold, conflicts uint32
		want                 bool
	}{
		{"disabled", false, 2, 0, false},
		{"zero-threshold", true, 0, 0, false},
		{"below-threshold", true, 2, 1, true},
		{"at-threshold", true, 1, 1, false},
		{"past-threshold", true, 1, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			theory, solver, first, second := smtFixedEqualityGuardFixture(t)
			astTestMust(t, theory.ConfigureArithmetic(SMTArithState{PropagateEqs: tc.enabled, PropagationThreshold: tc.threshold}))
			for index := uint32(0); index < tc.conflicts; index++ {
				theory.recordArithmeticConflict()
			}
			propagated, err := theory.propagateFixedEquality(solver, first, second)
			astTestMust(t, err)
			if propagated != tc.want {
				t.Fatal("fixed equality propagation", propagated, tc.want)
			}
		})
	}
}

func TestSMTArithmeticEqualityConflictThresholdSurvivesPopAndResetsForSearch(t *testing.T) {
	theory, solver, first, second := smtFixedEqualityGuardFixture(t)
	astTestMust(t, theory.ConfigureArithmetic(SMTArithState{PropagateEqs: true, PropagationThreshold: 1}))
	astTestMust(t, theory.PushScope())
	theory.recordArithmeticConflict()
	astTestMust(t, theory.PopScopes(1))
	astTestMust(t, theory.Restart())
	propagated, err := theory.propagateFixedEquality(solver, first, second)
	astTestMust(t, err)
	if propagated || theory.conflictCount != 1 || theory.statistics.conflicts != 1 {
		t.Fatal("scope pop or restart reset the search conflict threshold")
	}
	astTestMust(t, theory.InitializeSearch())
	propagated, err = theory.propagateFixedEquality(solver, first, second)
	astTestMust(t, err)
	if !propagated || theory.conflictCount != 0 || theory.statistics.conflicts != 1 {
		t.Fatal("new search failed to reset its count independently of lifetime statistics")
	}
}

func TestSMTArithmeticEqualityOptionControlsImpliedRowPropagation(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			s, err := NewSMTSolver(m, nil, "QF_UFLRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			s.kernel.preprocess = false
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "option-x"), sort)
			astTestMust(t, err)
			y, err := m.MkConst(astTestSymbol(t, m, "option-y"), sort)
			astTestMust(t, err)
			f, err := m.MkFuncDecl(astTestSymbol(t, m, "option-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
			astTestMust(t, err)
			fx, err := m.MkApp(f, []ast.ASTHandle{x})
			astTestMust(t, err)
			fy, err := m.MkApp(f, []ast.ASTHandle{y})
			astTestMust(t, err)
			equal, err := m.MkEq(fx, fy)
			astTestMust(t, err)
			different, err := m.MkNot(equal)
			astTestMust(t, err)
			astTestMust(t, s.Assert(different))
			for _, args := range [][]ast.ASTHandle{{x, y}, {y, x}} {
				bound, err := m.MkFamilyApp(family, smtArithOpLE, nil, args)
				astTestMust(t, err)
				astTestMust(t, s.Assert(bound))
			}
			theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
			config, err := s.kernel.smtParams.ArithmeticState()
			astTestMust(t, err)
			config.PropagateEqs = enabled
			astTestMust(t, theory.ConfigureArithmetic(config))
			status, err := s.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("x=y must contradict distinct UF applications", status)
			}
			if (theory.statistics.offsetEqualities != 0) != enabled {
				t.Fatal("implied row equality ignored propagate_eqs", enabled, theory.statistics.offsetEqualities)
			}
		})
	}
}
