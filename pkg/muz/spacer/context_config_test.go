package spacer

import (
	"testing"

	"github.com/glycerine/g3/pkg/smtcore"
)

func TestContextConsumesPropSolverFarkasParameters(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.new_unsat_core", "true"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.farkas_optimized", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.split_farkas_literals", "true"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.farkas_a_const", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.keep_proxy", "false"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if context.config.Itp.CoreMode != ItpCoreNewFarkas || !context.config.Itp.SplitLiterals || context.config.Itp.FarkasAConst || context.config.KeepProxy || !context.config.UseInductiveGeneralizer || !context.config.UseArrayEqGeneralizer || context.config.UseEqClassGeneralizer {
		t.Fatal("prop-solver parameter policy", context.config)
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextCanDisableInductiveGeneralizer(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.pdr.use_inductive_generalizer", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.use_array_eq_generalizer", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.use_eqclass", "true"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.lemma_sanity_check", "true"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if context.config.UseInductiveGeneralizer {
		t.Fatal("disabled inductive generalizer was retained as enabled")
	}
	if context.config.UseArrayEqGeneralizer {
		t.Fatal("disabled array equality generalizer was retained as enabled")
	}
	if !context.config.UseEqClassGeneralizer {
		t.Fatal("enabled equivalence-class generalizer was retained as disabled")
	}
	if !context.config.LemmaSanityCheck {
		t.Fatal("enabled lemma sanity checker was retained as disabled")
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextConsumesMinCutCoreMode(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.minimize_unsat_core", "true"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if !context.config.Itp.MinimizeUnsatCore {
		t.Fatal("min-cut core parameter was not retained")
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextConsumesReachFactPolicies(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.elim_aux", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.reach_dnf", "false"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if context.eliminateAux || context.reachDNF {
		t.Fatal("reach-fact parameter policy", context.eliminateAux, context.reachDNF)
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextConsumesRestartPolicy(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.restarts", "true"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.restart_initial_threshold", "7"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if !context.useRestarts || context.restartInitial != 7 {
		t.Fatal("restart parameter policy", context.useRestarts, context.restartInitial)
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSearchRestartPolicyUsesStrictLubyThresholds(t *testing.T) {
	policy := newSearchRestartPolicy(true, 1, 0)
	for _, test := range []struct {
		lemmas  uint64
		restart bool
	}{
		{1, false},
		{2, true},
		{3, false},
		{4, true},
		{6, false},
		{7, true},
	} {
		restart, err := policy.observe(test.lemmas)
		if err != nil || restart != test.restart {
			t.Fatal("restart policy", test.lemmas, restart, err)
		}
	}
}

func TestContextConsumesQuantifiedLemmaPolicies(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.instantiate", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.qlemmas", "false"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if context.instantiate || context.quantifiedLemmas {
		t.Fatal("quantified lemma parameter policy", context.instantiate, context.quantifiedLemmas)
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextConsumesGroundCTIPolicy(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.ground_cti", "false"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	if context.groundCTI {
		t.Fatal("ground CTI parameter policy")
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextMapsSpacerSMTBackendParameters(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.random_seed", "17"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.mbqi", "false"); err != nil {
		t.Fatal(err)
	}
	if err := globals.SourceSetName("fixedpoint.spacer.eq_prop", "false"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	seedName := "random_seed"
	seed, err := globals.Context.GetUIntName(&context.owner.solverParams, &seedName, 0)
	if err != nil || seed != 17 {
		t.Fatal("Spacer SMT random seed", seed, err)
	}
	mbqiName := "mbqi"
	mbqi, err := globals.Context.GetBoolName(&context.owner.solverParams, &mbqiName, true)
	if err != nil || mbqi {
		t.Fatal("Spacer SMT MBQI", mbqi, err)
	}
	propagationName := "arith.propagation_mode"
	propagation, err := globals.Context.GetUIntName(&context.owner.solverParams, &propagationName, 2)
	if err != nil || propagation != 0 {
		t.Fatal("Spacer SMT bound propagation", propagation, err)
	}
	equalitiesName := "arith.propagate_eqs"
	equalities, err := globals.Context.GetBoolName(&context.owner.solverParams, &equalitiesName, true)
	if err != nil || equalities {
		t.Fatal("Spacer SMT equality propagation", equalities, err)
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}
