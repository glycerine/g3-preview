package smt2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2TacticGrammarRejectsMalformedInputBeforeChecking(t *testing.T) {
	for _, text := range []string{
		"(check-sat-using)", "(check-sat-using unknown-tactic)",
		"(check-sat-using (then))", "(check-sat-using (or-else))",
		"(check-sat-using (then smt", "(check-sat-using (try-for smt))",
		"(check-sat-using (repeat smt 2 extra))", "(check-sat-using smt extra)",
		"(check-sat-using (using-params smt :max_conflicts))",
		"(check-sat-using (using-params smt :max_conflicts -1))",
		"(check-sat-using (using-params smt :max_conflicts true))",
		"(check-sat-using (using-params smt :candidate_models 1))",
		"(check-sat-using (using-params smt :unported_parameter 1))",
		"(check-sat-using smt :rlimit false)",
	} {
		t.Run(text, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			var output bytes.Buffer
			if err = ParseCommands(context, strings.NewReader("(assert false)"+text), &output); err == nil {
				t.Fatal("invalid tactic grammar was accepted")
			}
			if output.Len() != 0 {
				t.Fatal("malformed tactic command ran a check", output.String())
			}
			output.Reset()
			if err = ParseCommands(context, strings.NewReader("(check-sat)"), &output); err != nil {
				t.Fatal(err)
			}
			if output.String() != "unsat\n" {
				t.Fatal("malformed tactic command changed assertions", output.String())
			}
		})
	}
}

func TestSMT2TacticCombinatorsAndTypedParameters(t *testing.T) {
	for _, expression := range []string{
		"(then smt)", "(or-else smt)", "(using-params smt)",
		"(using-params smt :restart_factor 1.5 :string_solver seq :random_seed 0.0 :auto_config false)",
		"(! smt :max_conflicts 4294967295)", "(with smt :max-conflicts 4294967295)",
		"(repeat (then simplify smt) 2)", "(fail-if-branching smt 1)",
		"(skip-if-failed smt)", "(or-else fail (then elim-and smt))",
	} {
		t.Run(expression, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			var output bytes.Buffer
			script := "(set-logic QF_UF)(declare-const a Bool)(assert a)(check-sat-using " + expression + ")(get-value (a))"
			if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
				t.Fatal(err)
			}
			if output.String() != "sat\n((a true))\n" {
				t.Fatal("tactic grammar lost the asserted model value", output.String())
			}
		})
	}
}
