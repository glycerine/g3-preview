package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Source justify allocates a region theory axiom used by an auxiliary lemma.
// The prebuilt reference prints a complete proof, then aborts freeing it.
// Keep this exact defect separate from the ordinary clean-exit proof oracle.
func TestSMTLibRunnerPBStrengtheningProofKnownCPPTeardownDefect(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "pb_strengthening_proof.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	check := func(label, value string) {
		if !strings.HasPrefix(canonicalOutput(value), "unsat (cut_A cut_B) ") || !strings.Contains(value, "(proof") || !strings.Contains(value, "(asserted") || !strings.Contains(value, "false)") || strings.Count(value, "(") != strings.Count(value, ")") {
			t.Fatal(label, "missing complete PB cut proof/core", value)
		}
	}
	var native bytes.Buffer
	if err = run(bytes.NewReader(script), &native); err != nil {
		t.Fatal("Go PB proof failed", err, native.String())
	}
	check("Go", native.String())
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		// Avoid leaving a core file when exercising the observed teardown bug.
		cmd := exec.CommandContext(ctx, "sh", "-c", `ulimit -c 0; exec "$1" -in`, "g3-cpp-reference", reference)
		cmd.Stdin = bytes.NewReader(script)
		var out, diagnostics bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diagnostics
		err = cmd.Run()
		check("C++", out.String())
		if err != nil && (!strings.HasPrefix(err.Error(), "signal: aborted") || strings.TrimSpace(diagnostics.String()) != "free(): invalid pointer") {
			t.Fatal("unexpected C++ PB proof failure", err, diagnostics.String())
		}
	}
}
