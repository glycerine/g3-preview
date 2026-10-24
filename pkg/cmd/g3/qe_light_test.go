package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSMTLibRunnerQELightProjectionModelAndCore(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"qe_light_projection.smt2", "unsat\n"},
		{"qe_light_fm.smt2", "unsat\n"},
		{"qe_light_model.smt2", "sat\n((k 3))\n"},
		{"qe_light_core.smt2", "unsat\n(quantified negated)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script, err := os.ReadFile(filepath.Join("test_vectors", tc.name))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err = run(bytes.NewReader(script), &out); err != nil {
				t.Fatal(err, out.String())
			}
			if out.String() != tc.want {
				t.Fatalf("Go result: %q, want %q", out.String(), tc.want)
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = bytes.NewReader(script)
				cpp, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(cpp))
				}
				if string(cpp) != out.String() {
					t.Fatalf("Go/C++ mismatch\nGo: %s\nC++: %s", out.String(), cpp)
				}
			}
		})
	}
}
