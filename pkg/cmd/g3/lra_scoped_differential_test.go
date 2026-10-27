package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The files have only SAT status atoms and two get-value responses. Keep model
// validation semantic: each response evaluates the full original conjunction and its negation,
// while the solvers may choose different satisfying rational/function values.
func checkLRAScopedWitnessOutput(t *testing.T, output string) {
	t.Helper()
	tokens := strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(output))
	var statuses, values []string
	depth, bindings, fields, responses := 0, 0, 0, 0
	value := ""
	for _, token := range tokens {
		switch token {
		case "(":
			if depth == 0 {
				bindings = 0
			}
			if depth == 1 {
				bindings++
				fields = 0
				value = ""
			}
			if depth == 2 {
				fields++
			}
			depth++
		case ")":
			if depth == 2 {
				if fields != 2 || value != "true" && value != "false" {
					t.Fatal("malformed model binding", output)
				}
				values = append(values, value)
			}
			if depth == 1 {
				if bindings != 2 {
					t.Fatal("missing conjunction/negation model binding", output)
				}
				responses++
			}
			depth--
			if depth < 0 {
				t.Fatal("unbalanced solver output", output)
			}
		default:
			if depth == 0 {
				statuses = append(statuses, token)
			}
			if depth == 2 {
				fields++
				if fields == 2 {
					value = token
				}
			}
		}
	}
	if depth != 0 || responses != 2 || strings.Join(statuses, " ") != "sat unsat sat" || strings.Join(values, " ") != "true false true false" {
		t.Fatal("SAT/scope/model oracle", output, statuses, values)
	}
}

func TestSMTLibRunnerLRAScopedRationalAndUFModelOracles(t *testing.T) {
	files := []string{"lra_scoped_rational_witness_a.smt2", "lra_scoped_rational_witness_b.smt2", "lra_scoped_uf_witness_a.smt2", "lra_scoped_uf_witness_b.smt2"}
	for _, name := range files {
		for _, tactic := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/tactic=%v", name, tactic), func(t *testing.T) {
				script, err := os.ReadFile(filepath.Join("test_vectors", name))
				if err != nil {
					t.Fatal(err)
				}
				if tactic {
					script = []byte(strings.ReplaceAll(string(script), "(check-sat)", "(check-sat-using smt)"))
				}
				var goOutput bytes.Buffer
				if err = run(bytes.NewReader(script), &goOutput); err != nil {
					t.Fatal(err, goOutput.String())
				}
				checkLRAScopedWitnessOutput(t, goOutput.String())
				if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
					cmd := exec.Command(reference, "-in")
					cmd.Stdin = bytes.NewReader(script)
					var output, diagnostics bytes.Buffer
					cmd.Stdout, cmd.Stderr = &output, &diagnostics
					if err = cmd.Run(); err != nil || diagnostics.Len() != 0 {
						t.Fatal("prebuilt C++", err, output.String(), diagnostics.String())
					}
					checkLRAScopedWitnessOutput(t, output.String())
				}
			})
		}
	}
}
