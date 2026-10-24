package smt2

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2LogicValidationCPP(t *testing.T) {
	for _, tc := range []struct{ name, setup, command, want string }{
		{"integer-sort", "(set-logic QF_UF)", "(declare-const x Int)", "unknown sort"},
		{"real-sort", "(set-logic QF_LIA)", "(declare-const x Real)", "logic does not support reals"},
		{"bv-sort", "(set-logic QF_LIA)", "(declare-const x (_ BitVec 8))", "unknown sort"},
		{"uf-arity", "(set-logic QF_LIA)", "(declare-fun f (Bool) Bool)", "logic does not support uninterpreted functions"},
		{"uf-domain-first", "(set-logic QF_UF)", "(declare-fun f (Int) Real)", "unknown sort"},
		{"uf-range", "(set-logic QF_UFLIA)", "(declare-fun f (Int) Real)", "logic does not support reals"},
		{"array-sort", "(set-logic QF_LIA)", "(declare-const a (Array Int Int))", "unknown sort"},
		{"array-bv-domain", "(set-logic QF_ABV)", "(declare-const a (Array Bool (_ BitVec 8)))", "logic supports only arrays from bitvectors to bitvectors"},
		{"array-bv-range", "(set-logic QF_ABV)", "(declare-const a (Array (_ BitVec 8) Bool))", "logic supports only arrays from bitvectors to bitvectors"},
		{"array-multiple-domains", "(set-logic QF_ABV)", "(declare-const a (Array (_ BitVec 8) (_ BitVec 8) (_ BitVec 8)))", "logic supports only unidimensional arrays"},
		{"general-array-no-recursion", "(set-logic QF_AUFLIA)", "(declare-const a (Array Int Real))", ""},
		{"hidden-sort-name-can-be-declared", "(set-logic QF_UF)(declare-sort Int 0)", "(declare-const u Int)", ""},
		{"hidden-arithmetic-operator", "(set-logic QF_UF)", "(assert (= (+ 1 2) 3))", "unknown function/constant +"},
		{"hidden-bitvector-operator", "(set-logic QF_LIA)", "(assert (= (bvnot #x01) #xfe))", "unknown function/constant bvnot"},
		{"hidden-sequence-operator", "(set-logic QF_LIA)", "(assert (= (str.len \"x\") 1))", "unknown function/constant str.len"},
		{"user-sort-permissive", "(set-logic QF_LIA)(declare-sort U 0)", "(declare-const u U)", ""},
		{"datatype-functions-rejected", "(set-logic QF_LIA)", "(declare-datatype D ((nil)))", "logic does not support uninterpreted functions"},
		{"datatype-enabled", "(set-logic QF_DT)", "(declare-datatype D ((nil)))", ""},
		{"int-nonlinear", "(set-logic QF_LIA)(declare-const x Int)(declare-const y Int)", "(assert (> (* x y) 0))", "logic does not support nonlinear arithmetic"},
		{"int-divisor", "(set-logic QF_LIA)(declare-const x Int)(declare-const y Int)", "(assert (= (div x y) 0))", "logic does not support nonlinear arithmetic"},
		{"int-modulus", "(set-logic QF_LIA)(declare-const x Int)(declare-const y Int)", "(assert (= (mod x y) 0))", "logic does not support nonlinear arithmetic"},
		{"real-divisor", "(set-logic QF_LRA)(declare-const x Real)(declare-const y Real)", "(assert (= (/ x y) 0))", "logic does not support nonlinear arithmetic"},
		{"negative-coefficient", "(set-logic QF_LIA)(declare-const x Int)", "(assert (> (* (- 3) x) 0))", ""},
		{"rational-coefficient", "(set-logic QF_LRA)(declare-const x Real)", "(assert (> (* (/ 1 3) x) 0))", ""},
		{"nonlinear-enabled", "(set-logic QF_NIA)(declare-const x Int)(declare-const y Int)", "(assert (> (* x y) 0))", ""},
		{"diff-sum", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)", "(assert (<= (+ x y) 0))", "logic only supports difference arithmetic"},
		{"diff-equality", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)", "(assert (= (+ x y) 0))", "logic only supports difference arithmetic"},
		{"diff-weighted", "(set-logic QF_IDL)(declare-const x Int)", "(assert (<= (* 36 x) 264))", "logic only supports difference arithmetic"},
		{"diff-distinct-permissive", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)", "(assert (distinct (+ x y) 0))", ""},
		{"diff-two-vars", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)", "(assert (<= (- x y) 3))", ""},
		{"diff-offset", "(set-logic QF_IDL)(declare-const x Int)", "(assert (<= (+ 1 (- x 2)) 3))", ""},
		{"diff-both-offsets", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)", "(assert (<= (+ x 1) (+ y 2)))", ""},
		{"diff-rational-offset", "(set-logic QF_RDL)(declare-const x Real)", "(assert (<= (+ x (/ 1 3)) 4))", ""},
		{"diff-arithmetic-ite", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)(declare-const p Bool)", "(assert (<= (ite p x y) 3))", ""},
		{"diff-nonlinear-first", "(set-logic QF_IDL)(declare-const x Int)(declare-const y Int)", "(assert (<= (* x y) 0))", "logic does not support nonlinear arithmetic"},
		{"diff-repeated", "(set-logic QF_RDL)(declare-const x Real)(declare-const y Real)", "(assert (<= (- (+ x x) (+ y y)) 3))", ""},
		{"diff-unequal-repeat", "(set-logic QF_RDL)(declare-const x Real)(declare-const y Real)", "(assert (<= (- (+ x x x) (+ y y)) 3))", "logic only supports difference arithmetic"},
		{"diff-mixed-repeat", "(set-logic QF_RDL)(declare-const x Real)(declare-const y Real)", "(assert (<= (- (+ x y) (+ y y)) 3))", "logic only supports difference arithmetic"},
		{"diff-uf-arguments-permissive", "(set-logic QF_UFIDL)(declare-const x Int)(declare-const y Int)(declare-fun f (Int) Int)", "(assert (<= (f (+ x y)) 3))", ""},
		{"unused-quantifier", "(set-logic QF_LIA)", "(assert (forall ((x Int)) true))", "logic does not support quantifiers"},
		{"bound-variable", "(set-logic QF_LIA)", "(assert (forall ((x Int)) (= x 0)))", "logic does not support quantifiers"},
		{"postorder-before-quantifier", "(set-logic QF_LIA)(declare-const x Int)(declare-const y Int)", "(assert (forall ((unused Int)) (> (* x y) 0)))", "logic does not support nonlinear arithmetic"},
		{"quantifiers-enabled", "(set-logic AUFLIA)", "(assert (forall ((x Int)) (= x 0)))", ""},
		{"unused-sort-permissive", "(set-logic AUFLIA)", "(assert (forall ((x Real)) true))", ""},
		{"unknown-profile-permissive", "(set-logic LIA)(declare-const x Int)", "(assert (forall ((y Int)) (> (* x y) 0)))", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := c.Close(); err != nil {
					t.Error(err)
				}
			})
			var output bytes.Buffer
			if err = ParseCommands(c, strings.NewReader(tc.setup), &output); err != nil {
				t.Fatal("setup", err)
			}
			err = ParseCommands(c, strings.NewReader(tc.command), &output)
			if tc.want == "" {
				if err != nil {
					t.Fatal("valid source input rejected", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("wrong rejection", err, tc.want)
			}
			// A refusal must leave the context usable by a later invocation.
			if err = ParseCommands(c, strings.NewReader("(echo \"resumed\")"), &output); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "resumed") {
				t.Fatal("failed to resume", output.String())
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(tc.setup + tc.command)
				var native, diagnostics bytes.Buffer
				cmd.Stdout, cmd.Stderr = &native, &diagnostics
				err := cmd.Run()
				if tc.want == "" {
					if err != nil || strings.Contains(native.String(), "(error") {
						t.Fatal("source rejected expected valid input", err, native.String(), diagnostics.String())
					}
				} else if !strings.Contains(native.String(), tc.want) {
					t.Fatal("source diagnostic mismatch", err, native.String(), diagnostics.String())
				}
			}
		})
	}
}
