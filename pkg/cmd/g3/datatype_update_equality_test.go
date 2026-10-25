package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSMTLibRunnerDatatypeUpdateEqualityScopes(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "datatype_update_equality_scopes.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	const expected = "sat\n(((= (first changed) u) true) ((= (second changed) v) true))\nunsat\n(matching bad_field)\nsat\nunsat\n(empty bad_identity)\nsat\nsat\n(((= (field changed-one) v) true))\nunsat\nsat\n"
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
		if err != nil || canonicalOutput(string(value)) != canonicalOutput(expected) {
			t.Fatal("C++ datatype update output", err, string(value))
		}
	}
}
