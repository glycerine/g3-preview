package arith_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestDiffNeqParsedShiftedPipeline(t *testing.T) {
	c, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	globals, err := c.GlobalParameters()
	if err != nil {
		t.Fatal(err)
	}
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := uparams.BindProcessGlobalParamsService(previous); err != nil {
			t.Error(err)
		}
	}()
	script, err := os.ReadFile("../../cmd/g3/test_vectors/diff_neq_shifted_model.smt2")
	if err != nil {
		t.Fatal(err)
	}
	prefix := strings.SplitN(string(script), "(check-sat-using", 2)[0]
	var output bytes.Buffer
	if err = smt2.ParseCommands(c, strings.NewReader(prefix), &output); err != nil {
		t.Fatal(err)
	}
	m, err := c.Manager()
	if err != nil {
		t.Fatal(err)
	}
	goal, err := tactic.NewGoal(m, false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := goal.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err = c.AssertGoal(goal); err != nil {
		t.Fatal(err)
	}
	var result tactic.TacticResult
	defer func() {
		if err := result.Close(); err != nil {
			t.Error(err)
		}
	}()
	var transcript bytes.Buffer
	for _, name := range []string{"simplify", "normalize-bounds", "diff-neq"} {
		transcript.WriteString("before " + name + ":\n")
		if e := goal.Display(&transcript); e != nil {
			t.Fatalf("display before %s: %v\n%s", name, e, transcript.String())
		}
		runner, e := c.NewBuiltinTactic(name)
		if e != nil {
			t.Fatal(e)
		}
		if e = runner.IncRef(); e != nil {
			t.Fatal(e)
		}
		e = runner.Apply(goal, &result)
		closeErr := runner.DecRef()
		if e != nil || closeErr != nil {
			var display bytes.Buffer
			_ = goal.Display(&display)
			t.Fatalf("%s: %v / %v\n%s\n%s", name, e, closeErr, transcript.String(), display.String())
		}
	}
	if !goal.IsDecidedSat() {
		t.Fatal("shifted pipeline did not decide SAT")
	}
}
