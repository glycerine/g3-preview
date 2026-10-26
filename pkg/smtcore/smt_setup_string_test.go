package smtcore

import "testing"

func TestSMTQFSSetupInstallsNativeStringAndMixedArithmetic(t *testing.T) {
	for _, profile := range []struct {
		auto   bool
		choice string
	}{
		{true, "seq"}, {true, "z3str3"}, {true, "auto"}, {true, "invalid-generic-choice"}, {false, "seq"},
	} {
		auto, choice := profile.auto, profile.choice
		t.Run(choice+map[bool]string{false: "-logic", true: "-auto"}[auto], func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			params, err := config.Context.NewValue()
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, config.Context.Del(&params)) })
			key := "auto_config"
			astTestMust(t, config.Context.SetBoolName(&params, &key, auto))
			key = "string_solver"
			symbol := astTestSymbol(t, m, choice)
			astTestMust(t, config.Context.SetSymbolName(&params, &key, symbol))
			solver, err := NewSMTSolver(m, &params, "QF_S", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			seq, err := m.SequenceFamilyID()
			astTestMust(t, err)
			str, ok := solver.kernel.internalizer.theories[seq].(*SMTStringTheory)
			if auto && !ok {
				t.Fatal("QF_S profile must select native theory_str independently of generic string_solver")
			}
			if !auto {
				if _, isSeq := solver.kernel.internalizer.theories[seq].(*SMTSequenceTheory); !isSeq {
					t.Fatal("CFG_BASIC default seq selector did not retain generic sequence theory")
				}
			}
			arith, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			if arithmetic, ok := solver.kernel.internalizer.theories[arith].(*SMTLinearArithmeticTheory); !ok || !arithmetic.mixed {
				t.Fatal("QF_S must select native theory_mi_arith")
			}
			if auto && (str.parameterOwner != &solver.kernel.smtParams.str || str.regexExpressionSolver == nil || str.regexExpressionSolver.kernel.smtParams != solver.kernel.smtParams) {
				t.Fatal("QF_S owner/nested kernel lost actual borrowed parameter ownership")
			}
		})
	}
}
