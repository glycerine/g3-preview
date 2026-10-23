package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var scriptResults = map[string]string{
	"mixed_floor_scopes.smt2":                   "sat\n((i (- 2)))\nunsat\nsat\n",
	"mixed_is_int.smt2":                         "unsat\n",
	"mixed_not_is_int.smt2":                     "unsat\n",
	"mixed_arithmetic_model.smt2":               "sat\n((i 1) (r (/ 3.0 2.0)))\n",
	"mixed_integer_fraction.smt2":               "unsat\n",
	"mixed_negative_epsilon.smt2":               "unsat\n",
	"mixed_arithmetic_scopes.smt2":              "unsat\nsat\n((i 1) (r (/ 1.0 2.0)))\n",
	"all_quantified_arithmetic_after_push.smt2": "unsat\nsat\n",
	"quantified_bounds.smt2":                    "unsat\nsat\n",
	"bit_vector_sharing_scopes.smt2":            "unsat\nsat\nunsat\nsat\n",
	"inferred_trigger.smt2":                     "unsat\n",
	"function_overloads.smt2":                   "sat\n",
	"function_definitions.smt2":                 "unsat\n",
	"definition_scopes.smt2":                    "sat\nsat\n",
	"definition_capture.smt2":                   "unsat\n",
	"definition_coercions.smt2":                 "unsat\n",
	"function_coercions.smt2":                   "unsat\n",
	"global_definitions.smt2":                   "unsat\n",
	"boolean_model.smt2":                        "sat\n(model\n  (define-fun p () Bool true)\n)\n",
	"constant_array.smt2":                       "unsat\n",
	"string_value.smt2":                         "sat\n((x \"a\"\"b\"))\n",
	"global_declarations.smt2":                  "sat\nsat\n",
	"boolean_scopes.smt2":                       "sat\n((|p spaced| true))\nunsat\nsat\na\"b\n",
	"uf_congruence.smt2":                        "unsat\n",
	"linear_integer.smt2":                       "sat\n((x 4))\nunsat\nsat\n",
	"linear_real.smt2":                          "sat\n((x (/ 3.0 2.0)))\n",
	"bit_vector.smt2":                           "sat\n((x #xf))\n",
	"array_read_store.smt2":                     "unsat\n",
	"quantified_boolean.smt2":                   "unsat\n",
	"parallel_let.smt2":                         "sat\n((p true))\n",
	"declaration_scopes.smt2":                   "sat\nsat\n",
	"named_core.smt2":                           "unsat\n(positive negative)\n",
	"quantified_let_capture.smt2":               "unsat\n",
	"indexed_bit_vector.smt2":                   "unsat\n",
	"nary_equality.smt2":                        "unsat\n",
}

func TestSMTLibRunnerScripts(t *testing.T) {
	for name, expected := range scriptResults {
		t.Run(name, func(t *testing.T) {
			script, err := os.ReadFile(filepath.Join("test_vectors", name))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err = run(bytes.NewReader(script), &out); err != nil {
				t.Fatal(err, out.String())
			}
			if canonicalOutput(out.String()) != canonicalOutput(expected) {
				t.Fatal("Go SMT-LIB result", out.String(), "want", expected)
			}
		})
	}
}

// Explicitly opt in to the repository's C++ executable. Ordinary Go semantic
// tests use checked-in vectors and do not require a native reference build.
func TestSMTLibRunnerCPPReference(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the C++ z3 executable")
	}
	for name := range scriptResults {
		t.Run(name, func(t *testing.T) {
			script, err := os.ReadFile(filepath.Join("test_vectors", name))
			if err != nil {
				t.Fatal(err)
			}
			var goOutput bytes.Buffer
			if err = run(bytes.NewReader(script), &goOutput); err != nil {
				t.Fatal(err, goOutput.String())
			}
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = bytes.NewReader(script)
			cppOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal("C++ reference", err, string(cppOutput))
			}
			if canonicalOutput(goOutput.String()) != canonicalOutput(string(cppOutput)) {
				t.Fatalf("Go/C++ mismatch\nGo:\n%s\nC++:\n%s", goOutput.String(), cppOutput)
			}
		})
	}
}

// Preserve whitespace inside quoted names and strings; only printer layout
// outside token spellings may differ between the two executables.
func canonicalOutput(text string) string {
	var out strings.Builder
	var quoted byte
	escape, space := false, false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quoted != 0 {
			out.WriteByte(c)
			if quoted == '"' && c == '"' && i+1 < len(text) && text[i+1] == '"' {
				out.WriteByte(text[i+1])
				i++
				continue
			}
			if c == quoted && !escape {
				quoted = 0
			}
			escape = quoted == '|' && c == '\\'
			continue
		}
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			space = out.Len() != 0
			continue
		}
		if space {
			out.WriteByte(' ')
			space = false
		}
		out.WriteByte(c)
		if c == '"' || c == '|' {
			quoted = c
			escape = false
		}
	}
	return out.String()
}

func TestSMTLibRunnerProofScript(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "proof.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = run(bytes.NewReader(script), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "unsat\n") || !strings.Contains(out.String(), "(asserted") || !strings.Contains(out.String(), "false)") {
		t.Fatal("missing proof output", out.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = bytes.NewReader(script)
		value, err := cmd.CombinedOutput()
		if err != nil || !strings.HasPrefix(string(value), "unsat\n") || !strings.Contains(string(value), "(asserted") || !strings.Contains(string(value), "false)") {
			t.Fatal("C++ proof output", string(value), err)
		}
	}
}

func TestSMTLibRunnerCommandLine(t *testing.T) {
	for _, args := range [][]string{{"-in"}, {"-smt2", "test_vectors/uf_congruence.smt2"}, {"-file:test_vectors/uf_congruence.smt2"}, {"--", "test_vectors/uf_congruence.smt2"}} {
		var out bytes.Buffer
		if err := runArguments(args, strings.NewReader("(assert false)(check-sat)"), &out); err != nil {
			t.Fatal(args, err)
		}
		if out.String() != "unsat\n" {
			t.Fatal(args, out.String())
		}
	}
	for _, args := range [][]string{nil, {"-in", "test_vectors/uf_congruence.smt2"}, {"-dimacs", "-in"}} {
		var out bytes.Buffer
		if err := runArguments(args, strings.NewReader(""), &out); err == nil {
			t.Fatal("invalid frontend selection accepted", args)
		}
	}
	var out bytes.Buffer
	if err := runArguments([]string{"proof=true", "-in"}, strings.NewReader("(assert false)(check-sat)(get-proof)"), &out); err != nil {
		t.Fatal("source command-line parameter refresh", err)
	}
	if !strings.HasPrefix(out.String(), "unsat\n") || !strings.Contains(out.String(), "asserted") {
		t.Fatal(out.String())
	}
}
