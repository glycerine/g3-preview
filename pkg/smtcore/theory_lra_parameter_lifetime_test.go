package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTLRAPropagationModeUpdatesDoNotReinitializeLPTracking(t *testing.T) {
	for _, initial := range []uint32{0, 1, 2} {
		for _, current := range []uint32{0, 1, 2} {
			t.Run(fmt.Sprintf("%d-to-%d", initial, current), func(t *testing.T) {
				f := newLRAFrontendFixtureWithPropagation(t, true, initial, ^uint32(0))
				config := f.a.config
				config.BoundProp = current
				astTestMust(t, f.a.ConfigureArithmetic(config))
				if f.a.numeric.solver.Settings.BoundPropagation != (initial != 0) || f.a.propagationEnabled() != (current != 0) {
					t.Fatal("initial LP setting and live callback mode conflated")
				}
				target := prepareLRALPPropagation(t, f)
				tracked := len(f.a.numeric.solver.RowsWithChangedBounds.Indices())
				if (tracked != 0) != (initial != 0) {
					t.Fatal("actual feasibility ignored frozen row tracking", tracked)
				}
				astTestMust(t, f.a.propagateLPBounds())
				literal, err := f.s.Literal(target)
				astTestMust(t, err)
				value, err := f.s.Assignment(literal)
				astTestMust(t, err)
				want := lbool.Undef
				if initial != 0 && current != 0 {
					want = lbool.True
				}
				if value != want {
					t.Fatal("LP propagation did not follow source initialization/mode split", value, want)
				}
				if current == 0 && len(f.a.numeric.solver.RowsWithChangedBounds.Indices()) != tracked {
					t.Fatal("disabled callback discarded tracked rows")
				}
				astTestMust(t, f.a.InitializeSearch())
				if f.a.numeric.solver.Settings.BoundPropagation != (initial != 0) {
					t.Fatal("new search reinitialized frozen LP setting")
				}
			})
		}
	}
}

func TestSMTLRAPropagationThresholdUpdateDoesNotEnableLPTracking(t *testing.T) {
	f := newLRAFrontendFixtureWithPropagation(t, true, 2, 0)
	config := f.a.config
	config.PropagationThreshold = ^uint32(0)
	astTestMust(t, f.a.ConfigureArithmetic(config))
	if !f.a.propagationEnabled() || f.a.numeric.solver.Settings.BoundPropagation {
		t.Fatal("live threshold reinitialized LP")
	}
	target := prepareLRALPPropagation(t, f)
	astTestMust(t, f.a.propagateLPBounds())
	literal, err := f.s.Literal(target)
	astTestMust(t, err)
	value, err := f.s.Assignment(literal)
	astTestMust(t, err)
	if value != lbool.Undef || f.a.lpPropagations != 0 || len(f.a.numeric.solver.RowsWithChangedBounds.Indices()) != 0 {
		t.Fatal("rows fabricated after initialization with zero threshold")
	}
}

func TestSMTLRAKernelParameterUpdatePreservesInitialTracking(t *testing.T) {
	for _, initial := range []uint32{0, 2} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			p, err := cfg.Context.NewValue()
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
			key := "arith.propagation_mode"
			astTestMust(t, cfg.Context.SetUIntName(&p, &key, initial))
			k := smtTestKernel(t, m)
			astTestMust(t, k.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			a, err := NewSMTLRATheory(family)
			astTestMust(t, err)
			astTestMust(t, k.RegisterTheory(a))
			solver := a.numeric.solver
			astTestMust(t, k.internalizer.PushScope())
			updated := uint32(2)
			if initial != 0 {
				updated = 0
			}
			astTestMust(t, cfg.Context.SetUIntName(&p, &key, updated))
			astTestMust(t, k.UpdateParams(&p))
			if a.config.BoundProp != updated || a.numeric.solver != solver || solver.Settings.BoundPropagation != (initial != 0) {
				t.Fatal("kernel update replaced/reconfigured initialized LAR")
			}
			astTestMust(t, k.internalizer.PopScopes(1))
			if a.config.BoundProp != updated || a.numeric.solver.Settings.BoundPropagation != (initial != 0) {
				t.Fatal("scope restored non-scoped parameter or changed initialization setting")
			}
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			f := &lraFrontendFixture{t, m, k.internalizer, a, sort}
			x, y := f.variable("parameter-lifetime-x"), f.variable("parameter-lifetime-y")
			f.assert(f.app(smtArithOpGE, f.app(smtArithOpAdd, x, y), f.number(5)), true)
			f.assert(f.app(smtArithOpLE, y, f.number(2)), true)
			f.propagate(true)
		})
	}
}

func TestSMTLRAFreshCopyInitializesTrackingFromCurrentParameters(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	target, targetCfg := smtModelGeneratorFixture(t)
	p, err := cfg.Context.NewValue()
	astTestMust(t, err)
	q, err := targetCfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, targetCfg.Context.Del(&q)); astTestMust(t, cfg.Context.Del(&p)) })
	key := "arith.propagation_mode"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 0))
	source, err := NewSMTSolver(m, &p, "QF_UFLRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg, TargetConfig: func(manager *ast.ASTManager) (SMTSolverConfig, error) {
		if manager != target {
			return SMTSolverConfig{}, ast.ErrASTDomain
		}
		return SMTSolverConfig{Params: targetCfg.Context, ModelEvaluator: targetCfg}, nil
	}})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, source.Close()) })
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	pin := &lraFrontendFixture{t: t, m: m, s: source.kernel.internalizer, sort: sort}
	x := pin.variable("copy-tracking-x")
	one := pin.number(1)
	lower := pin.pin(m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{x, one}))
	astTestMust(t, source.Assert(lower))
	status, err := source.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	a, ok := source.kernel.internalizer.Theory(family).(*SMTLRATheory)
	if !ok || a.numeric.solver.Settings.BoundPropagation {
		t.Fatal("source initialization", ok)
	}
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 2))
	astTestMust(t, source.UpdateParams(&p))
	if a.config.BoundProp != 2 || a.numeric.solver.Settings.BoundPropagation {
		t.Fatal("live update changed old LP flag")
	}
	translated, err := source.Translate(target, &q)
	astTestMust(t, err)
	copied := translated.(*SMTSolver)
	t.Cleanup(func() { astTestMust(t, copied.Close()) })
	targetFamily, err := target.ArithmeticFamilyID()
	astTestMust(t, err)
	fresh, ok := copied.kernel.internalizer.Theory(targetFamily).(*SMTLRATheory)
	if !ok || fresh == a || fresh.numeric.solver == a.numeric.solver || !fresh.numeric.solver.Settings.BoundPropagation {
		t.Fatal("fresh copy inherited stale initialization flag", ok)
	}
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 0))
	astTestMust(t, source.UpdateParams(&p))
	astTestMust(t, source.Close())
	status, err = copied.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True || fresh.config.BoundProp != 0 || !fresh.numeric.solver.Settings.BoundPropagation {
		t.Fatal("borrowed live mode or fresh frozen flag lost after source close", status, fresh.config.BoundProp)
	}
}
