package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTPBConstructorSnapshotTranslationAndAuxiliary(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	target, targetCfg := smtModelGeneratorFixture(t)
	params, err := cfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&params)) })
	set := func(frequency uint32, learn, compile, simplex bool) {
		name := "pb.conflict_frequency"
		astTestMust(t, cfg.Context.SetUIntName(&params, &name, frequency))
		for _, p := range []struct {
			name  string
			value bool
		}{{"pb.learn_complements", learn}, {"pb.enable_compilation", compile}, {"pb.enable_simplex", simplex}} {
			name = p.name
			astTestMust(t, cfg.Context.SetBoolName(&params, &name, p.value))
		}
	}
	set(7, true, true, true)
	s, err := NewSMTSolver(m, &params, "QF_FD", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg, TargetConfig: func(manager *ast.ASTManager) (SMTSolverConfig, error) {
		if manager != target {
			return SMTSolverConfig{}, ast.ErrASTHandle
		}
		return SMTSolverConfig{Params: targetCfg.Context, ModelEvaluator: targetCfg}, nil
	}})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	family, err := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	pb := s.kernel.internalizer.theories[family].(*SMTPseudoBooleanTheory)
	if pb.configured {
		t.Fatal("unused Go placeholder froze constructor controls before source setup")
	}
	set(3, false, false, false)
	astTestMust(t, s.UpdateParams(&params))
	status, err := s.CheckSatCore(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("source setup", status)
	}
	want := SMTPBState{ConflictFrequency: 3}
	if !pb.configured || pb.parameterOwner != nil || pb.controls != want {
		t.Fatal("constructor snapshot did not follow the actual setup boundary", pb.controls)
	}
	set(19, true, true, true)
	astTestMust(t, s.UpdateParams(&params))
	if pb.controls != want {
		t.Fatal("live PB plugin reread constructor-only controls")
	}
	copy, err := s.Translate(target, nil)
	astTestMust(t, err)
	next := copy.(*SMTSolver)
	t.Cleanup(func() { astTestMust(t, next.Close()) })
	targetFamily, err := target.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	np := next.kernel.internalizer.theories[targetFamily].(*SMTPseudoBooleanTheory)
	if np == pb || !np.configured || np.controls != want || np.parameterOwner != nil || len(np.constraints) != 0 {
		t.Fatal("translated PB copied current driver controls instead of the installed snapshot", np.controls)
	}
	status, err = next.CheckSatCore(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("destination setup")
	}
	chained, err := next.Translate(target, nil)
	astTestMust(t, err)
	last := chained.(*SMTSolver)
	t.Cleanup(func() { astTestMust(t, last.Close()) })
	if last.kernel.internalizer.theories[targetFamily].(*SMTPseudoBooleanTheory).controls != want {
		t.Fatal("chained PB snapshot changed")
	}
	checker := s.kernel.internalizer.quantifierManager.modelChecker
	astTestMust(t, checker.initAuxiliaryContext())
	aux := checker.auxiliaryContext.kernel.internalizer.theories[family].(*SMTPseudoBooleanTheory)
	if aux == pb || aux.controls != want || aux.parameterOwner != nil {
		t.Fatal("MBQI auxiliary lost PB constructor snapshot")
	}
	np.controls.ConflictFrequency = 41
	if pb.controls != want || aux.controls != want {
		t.Fatal("constructor snapshots share mutable storage")
	}
}
