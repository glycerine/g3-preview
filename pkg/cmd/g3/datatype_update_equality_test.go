package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSMTLibRunnerDatatypeUpdateEqualityScopes(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "datatype_update_equality_scopes.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	const expected = "sat\n(((= (first changed) u) true) ((= (second changed) v) true))\nunsat\n(matching bad_field)\nsat\nunsat\n(bad_identity empty)\nsat\nsat\n(((= (field changed-one) v) true))\nunsat\nsat\n"
	var out bytes.Buffer
	if err = run(bytes.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if canonicalOutput(out.String()) != canonicalOutput(expected) {
		t.Fatal("Go datatype update output", out.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = bytes.NewReader(script)
		value, err := cmd.CombinedOutput()
		// Both solver facades sort these core expressions by AST identity.
		// The independent managers need not assign the same identities;
		// retain exact membership and every other output in this reference.
		cppExpected := strings.Replace(expected, "(bad_identity empty)", "(empty bad_identity)", 1)
		if err != nil || canonicalOutput(string(value)) != canonicalOutput(cppExpected) {
			t.Fatal("C++ datatype update output", err, string(value))
		}
	}
}
