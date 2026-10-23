package smtcore

import "testing"

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
