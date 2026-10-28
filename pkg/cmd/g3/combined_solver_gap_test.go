package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The native incremental solver gives up on retained as-array terms, then
// combined_solver::check_sat falls back to its tactic solver. Preserve that
// fallback across nested scopes in the public Go command path.
func TestSMTLibRunnerAsArrayCombinedSolverFallback(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "array_as_array_combined_fallback_gap.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err = run(bytes.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if canonicalOutput(output.String()) != "sat unsat sat" {
		t.Fatal("combined solver did not preserve the native as-array fallback", output.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		command := exec.Command(reference, "-in")
		command.Stdin = bytes.NewReader(script)
		native, err := command.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(native))
		}
		if canonicalOutput(string(native)) != "sat unsat sat" {
			t.Fatal("native combined-solver result changed", string(native))
		}
	}
}
