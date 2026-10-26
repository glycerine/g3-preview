package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// cmd_context's implicit full reset clears the native builtin sort table.
// The legacy extension-command parser does not reinitialize it before the
// next cached sort lookup; an empty check initializes the manager first.
// Keep the unforced native error separate from the strict agreement vectors.
func TestSMTLibRunnerResetAssertionsNativeSortInitializationGap(t *testing.T) {
	script, err := os.ReadFile("test_vectors/pending_reset_assertions_native_sort_init.smt2")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = run(bytes.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	const result = "success\nsuccess\nsuccess\nsuccess\nsuccess\nsuccess\nsuccess\nsuccess\nsuccess\nsuccess\nsat\n((x 3))\n"
	if out.String() != result {
		t.Fatal("owning reset/session initialization changed", out.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = bytes.NewReader(script)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err = cmd.Run(); err == nil || !strings.Contains(stdout.String(), "unknown sort 'Int'") || !strings.Contains(stdout.String(), "sat\n") {
			t.Fatal("native implicit-reset initialization behavior changed; reassess gap", err, stdout.String(), stderr.String())
		}
		forced := strings.Replace(string(script), "(set-logic QF_LIA)", "(set-logic QF_LIA)\n(check-sat-using smt)", 1)
		cmd = exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(forced)
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err = cmd.Run(); err != nil || strings.Contains(stdout.String(), "(error") || !strings.HasSuffix(stdout.String(), "((x 3))\n") {
			t.Fatal("native explicit manager initialization did not repair the reference case", err, stdout.String(), stderr.String())
		}
	}
}
