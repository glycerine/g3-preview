package spacer

import (
	"strings"
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
	if context.config.Itp.CoreMode != ItpCoreNewFarkas || !context.config.Itp.SplitLiterals || context.config.Itp.FarkasAConst || context.config.KeepProxy {
		t.Fatal("prop-solver parameter policy", context.config)
	}
	if err = context.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextRejectsPendingMinCutCoreMode(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	if err := globals.SourceSetName("fixedpoint.spacer.minimize_unsat_core", "true"); err != nil {
		t.Fatal(err)
	}
	context, err := NewContext(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if context != nil || err == nil || !strings.Contains(err.Error(), "minimize_unsat_core is not ported") {
		t.Fatal("pending min-cut mode accepted", context, err)
	}
}
