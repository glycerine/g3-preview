package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTLibRunnerMaxBVSharingStepFrontiers(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for limit := 0; limit <= 40; limit++ {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			script := fmt.Sprintf(`(set-logic QF_BV)
(declare-const a (_ BitVec 8))
(declare-const b (_ BitVec 8))
(declare-const c (_ BitVec 8))
(declare-const d (_ BitVec 8))
(assert (= (bvadd a b) c))
(assert (= (bvadd b c a d) (bvadd a b c d)))
(check-sat-using (using-params max-bv-sharing :max_steps %d))
(get-info :reason-unknown)
`, limit)
			var actual bytes.Buffer
			if err := run(strings.NewReader(script), &actual); err != nil {
				t.Fatal(err, actual.String())
			}
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = strings.NewReader(script)
			native, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(native))
			}
			if canonicalOutput(actual.String()) != canonicalOutput(string(native)) {
				t.Fatalf("Go: %s\nC++: %s", actual.String(), native)
			}
		})
	}
}

var scriptResults = map[string]string{
	"max_bv_sharing_atomic_limit.smt2":             "sat\n((p true))\n",
	"max_bv_sharing_model_scopes.smt2":             "sat\n((x #x01) (y #x02) (z #x03) (w #x04))\nunsat\nsat\n((x #x01) (y #x02) (z #x03) (w #x04))\n",
	"max_bv_sharing_limit_recovery.smt2":           "(error \"tactic failed: max. steps exceeded\")\n(:reason-unknown \"max. steps exceeded\")\n(error \"tactic failed: max. steps exceeded\")\n(:reason-unknown \"max. steps exceeded\")\nsat\n((p false))\n",
	"max_bv_sharing_proofs.smt2":                   "unsat\n",
	"max_bv_sharing_cores.smt2":                    "unsat\n(zero one)\n",
	"max_bv_sharing_pb_pipeline.smt2":              "sat\n((x 1) (y 1) (z 1) (w 1))\n",
	"propagate_ineqs_model_scopes.smt2":            "sat\n((x 2.0) (y 5.0))\nunsat\nsat\n((x 2.0) (y 5.0))\n",
	"propagate_ineqs_core_fallback.smt2":           "unknown\n(:reason-unknown \"propagate-ineqs does not support unsat core production\")\nunsat\n(lo hi)\n",
	"propagate_ineqs_pb_pipeline.smt2":             "sat\n((x 7) (y 7))\n",
	"propagate_ineqs_proof_fallback.smt2":          "unknown\n(:reason-unknown \"propagate-ineqs does not support proof production\")\nunsat\n",
	"pb2bv_equal_vector_model.smt2":                "sat\n((x 1) (y 1) (z 0) (w 0))\n",
	"pb2bv_refusal_fallback.smt2":                  "unknown\n(:reason-unknown \"goal is in a fragment unsupported by pb2bv. Offending expression: x\")\nsat\n(((> x 0) true))\n",
	"pb2bv_proof_fallback.smt2":                    "unsat\n",
	"pb2bv_clauses_model_scopes.smt2":              "sat\n((x 1)\n (y 1)\n (z 1))\nunsat\nsat\n((x 1)\n (y 1)\n (z 1))\n",
	"pb2bv_cardinality_model.smt2":                 "sat\n(((= (+ x y z w v) 3) true))\n",
	"pb2bv_bitvector_model.smt2":                   "sat\n((x 1)\n (y 1)\n (z 1)\n (w 1))\n",
	"pb2bv_split_model.smt2":                       "sat\n((x 0)\n (y 0)\n (z 1)\n (w 1))\n",
	"pb2bv_nested_model.smt2":                      "sat\n((x 1)\n (y 1)\n (z 1))\n",
	"pb2bv_shifted_lia_pipeline.smt2":              "sat\n((x 2)\n (y 7))\n",
	"pb2bv_core.smt2":                              "unsat\n(total yhi ylo xlo xhi)\n",
	"bv_weighted_ite_model.smt2":                   "sat\n((a true)\n (b true)\n (c true)\n (d true))\n",
	"arith_negated_weighted_branches.smt2":         "sat\n(((and (>= (+ a (* 2 b) (* 4 c)) 6) (<= (+ a (* 2 b) (* 4 c)) 7)) true))\nunsat\nsat\n(((and (>= (+ a (* 2 b) (* 4 c)) 6) (<= (+ a (* 2 b) (* 4 c)) 7)) true))\n",
	"lia2pb_model_scopes.smt2":                     "sat\n((x 5) (y 2))\nunsat\nsat\n((x 5) (y 2))\n",
	"lia2pb_core.smt2":                             "unsat\n(total yhi ylo xlo xhi)\n",
	"lia2pb_partial.smt2":                          "unknown\n(:reason-unknown \"goal is in a fragment unsupported by lia2pb\")\nsat\n((x 3) (y 2))\n",
	"lia2pb_budget_recovery.smt2":                  "unknown\n(:reason-unknown \"lia2pb failed, number of necessary bits exceeds specified threshold (use option :lia2pb-total-bits to increase threshold)\")\nsat\n(((and (>= x 6) (<= x 7)) true))\n",
	"lia2pb_proof_fallback.smt2":                   "unsat\n",
	"lia2pb_shifted_pipeline.smt2":                 "sat\n((x 2) (y 7))\n",
	"diff_neq_backtrack_model.smt2":                "sat\n((x 1) (y 0))\nsat\n((x 1) (y 0))\n",
	"diff_neq_core_fallback.smt2":                  "unsat\n(lower upper)\n",
	"diff_neq_dl_pipeline_scopes.smt2":             "sat\n((x 0) (y 0) (z 1))\nunsat\nsat\n((x 0) (y 0) (z 1))\n",
	"diff_neq_parameter_recovery.smt2":             "unknown\n(:reason-unknown \"goal is not diff neq\")\nsat\n((x 0))\n",
	"diff_neq_proof_fallback.smt2":                 "unsat\n",
	"diff_neq_real_refusal.smt2":                   "unknown\n(:reason-unknown \"goal is not diff neq\")\nsat\n(((and (>= x 0) (<= x 1)) true))\n",
	"diff_neq_refusal_fallback.smt2":               "unknown\n(:reason-unknown \"goal is not diff neq\")\nsat\n((x 1))\n",
	"diff_neq_shifted_model.smt2":                  "sat\n((x (- 2)) (y 5))\n",
	"diff_neq_signed_offsets.smt2":                 "sat\n((x 0) (y 2))\n",
	"diff_neq_triangle_unsat.smt2":                 "unsat\n",
	"normalize_bounds_core.smt2":                   "unsat\n(lower upper)\n",
	"normalize_bounds_disjunction.smt2":            "sat\n((x 2))\n",
	"normalize_bounds_fix_dl_pipeline.smt2":        "sat\n((x 0) ((<= (- x y) 2) true) ((<= (- x z) 3) true) ((<= (- x w) 4) true))\nunsat\nsat\n",
	"normalize_bounds_int_model_scopes.smt2":       "sat\n(model (define-fun x () Int 7)\n)\nunsat\nsat\n((x 7))\n",
	"normalize_bounds_negative_model.smt2":         "sat\n(model (define-fun x () Int (- 3))\n)\n",
	"normalize_bounds_proof.smt2":                  "unsat\n",
	"normalize_bounds_real_default.smt2":           "sat\n((x (/ 2.0 3.0)))\n",
	"normalize_bounds_real_opt_in.smt2":            "sat\n(((and (> x (/ 1 3)) (<= x (/ 2 3))) true))\nunsat\nsat\n",
	"fix_dl_absolute_refusal.smt2":                 "sat\n((x 5) ((<= (- x y) 2) true))\n",
	"fix_dl_core.smt2":                             "unsat\n(forward backward)\n",
	"fix_dl_foreign_offset.smt2":                   "sat\n((x 0) ((p (+ 2 x)) true))\n",
	"fix_dl_model_scopes.smt2":                     "sat\n((x 0) ((<= (- x y) 2) true))\nunsat\nsat\n((x 0))\n",
	"fix_dl_parameter_recovery.smt2":               "unknown\nsat\n((x 0))\n",
	"fix_dl_proof.smt2":                            "unsat\n",
	"fix_dl_real_models.smt2":                      "sat\n((x 0.0) ((< (- x y) (/ 1 3)) true) ((<= (- x z) (- (/ 1 7))) true))\n",
	"symmetry_ranges_scopes.smt2":                  "sat\n(((distinct x y) true))\nunsat\nsat\n",
	"symmetry_function_support.smt2":               "sat\n(((distinct (f a) (f b)) true))\nunsat\nsat\n",
	"symmetry_asymmetric.smt2":                     "sat\n(((R a b) true) ((R b a) false))\n",
	"symmetry_proof_guard.smt2":                    "unsat\n",
	"qfuf_ranges_scopes.smt2":                      "sat\n(((distinct x y) true))\nunsat\nsat\n",
	"qfuf_combined_ranges_scopes.smt2":             "sat\n(((distinct x y) true))\nunsat\nsat\n",
	"qfuf_core.smt2":                               "unsat\n(different equal)\n",
	"qfuf_combined_core.smt2":                      "unsat\n(equal different)\n",
	"qfuf_proof.smt2":                              "unsat\n",
	"qfuf_parameter_recovery.smt2":                 "unknown\nsat\n(((distinct (f a) (f b)) true))\nsat\n",
	"qflra_combined_assumptions.smt2":              "sat\nsat\nunsat\nsat\nunsat\nsat\n",
	"qflra_combined_core.smt2":                     "unsat\n(positive nonpositive)\n",
	"qflra_core.smt2":                              "unsat\n(positive nonpositive)\n",
	"qflra_proof.smt2":                             "unsat\n",
	"qflra_combined_model_scopes.smt2":             "sat\n(((and (> x 0) (< x (/ 1 3)) (= y (+ x (/ 1 7)))) true))\nunsat\nsat\n",
	"qflra_model_scopes.smt2":                      "sat\n(((and (> x 0) (< x (/ 1 3)) (= y (+ x (/ 1 7)))) true))\nunsat\nsat\n",
	"qflra_parameter_overrides.smt2":               "sat\nsat\n(((and (>= (+ x y) 1) (>= (+ x (* 2 y)) 3)) true))\nsat\n",
	"qfauflia_combined_model_scopes.smt2":          "sat\n((x 2) ((= (select a y) 9) true) ((and (>= y 2) (<= y 4)) true))\nunsat\nsat\n((x 2) ((= (select a y) 9) true) ((and (>= y 2) (<= y 4)) true))\n",
	"qfauflia_combined_assumptions.smt2":           "sat\nsat\nunsat\nsat\nunsat\nsat\n",
	"qfauflia_combined_core.smt2":                  "unsat\n(zero lower)\n",
	"qfauflia_combined_proof.smt2":                 "unsat\n",
	"qfauflia_model_scopes.smt2":                   "sat\n((x 2) ((= (select a y) 9) true) ((and (>= y 2) (<= y 4)) true))\nunsat\nsat\n((x 2) ((= (select a y) 9) true) ((and (>= y 2) (<= y 4)) true))\n",
	"qfauflia_as_array_scopes.smt2":                "sat\nunsat\nsat\n",
	"qfauflia_core.smt2":                           "unsat\n(lower zero)\n",
	"qfauflia_proof.smt2":                          "unsat\n",
	"qfauflia_parameter_recovery.smt2":             "unknown\nsat\n(((and (>= x 0) (<= x 3)) true))\nsat\n(((and (>= x 0) (<= x 3)) true))\n",
	"elim_uncnstr_signed_bounds.smt2":              "sat\n(((bvsle a #x8) true) ((bvsle #x7 b) true) ((bvule c #xf) true) ((bvule #xf d) true) ((<= 4 i) true) ((>= 8 j) true))\n",
	"elim_uncnstr_constructor_singleton.smt2":      "sat\n(((f (box i)) true))\nunsat\nsat\n",
	"elim_uncnstr_uninterpreted_guard.smt2":        "unsat\n",
	"elim_uncnstr_ite_models.smt2":                 "sat\n(((= (ite c1 x y) 4) true) ((= (ite c2 z 2) 5) true) ((= (ite c3 2 w) 6) true))\n",
	"elim_uncnstr_bv_division_models.smt2":         "sat\n(((= (bvudiv a b) #x17) true) ((= (bvsdiv c d) #xfd) true) ((= (bvmul (_ bv3 128) x) (_ bv12 128)) true) ((= (bvmul #x02 y) #x06) true))\n",
	"elim_uncnstr_datatype_scopes.smt2":            "sat\nsat\n(((is-on f) true))\nsat\n",
	"elim_uncnstr_boolean.smt2":                    "sat\n(((or (not a) (= b c)) true))\n",
	"elim_uncnstr_arithmetic.smt2":                 "sat\n(((= (+ (* 2.0 x) y) 17.0) true) (i 3) ((<= j 4) true) ((>= k 8) true))\n",
	"elim_uncnstr_bitvectors.smt2":                 "sat\n(((= (bvmul #x03 x) #x0c) true) ((= ((_ extract 5 2) y) #xa) true) ((= (bvnot z) #x12) true) ((= (concat a b) #xab) true) ((= (bvor c d) #xf) true) ((bvule e #x0) true))\n",
	"elim_uncnstr_arrays.smt2":                     "sat\n(((select a true) 9) ((= (store b false v) ((as const (Array Bool Int)) 7)) true))\n",
	"elim_uncnstr_datatypes.smt2":                  "sat\n(((val b) 7) ((not (= n zero)) true))\n",
	"elim_uncnstr_core_scopes.smt2":                "sat\n(((not a) true) ((or a b) true))\nunsat\n(other negative disjunction)\nsat\n(((not a) true) ((or a b) true))\n",
	"elim_uncnstr_proofs.smt2":                     "sat\nunsat\n",
	"solve_eqs_boolean_scopes.smt2":                "sat\n((a true) (b true))\nunsat\nsat\n((a true) (b true))\n",
	"solve_eqs_real_definitions.smt2":              "sat\n((x 2.0) (y 1.0))\n",
	"solve_eqs_ite_definitions.smt2":               "sat\n(((ite c (= x 2) (= x 3)) true))\n",
	"solve_eqs_core.smt2":                          "unsat\n(positive equality negative)\n",
	"solve_eqs_cycle.smt2":                         "sat\n(((= a (f b)) true) ((= b (f a)) true))\n",
	"solve_eqs_as_array_scopes.smt2":               "sat\nunsat\nsat\n",
	"solve_eqs_proof.smt2":                         "unsat\n",
	"propagate_values_model_scopes.smt2":           "sat\n((a true) (b false) (c true))\nunsat\nsat\n((a true) (b false) (c true))\n",
	"propagate_values_core.smt2":                   "unsat\n(negative implication positive)\n",
	"propagate_values_proof.smt2":                  "unsat\n",
	"tactic_model_scopes.smt2":                     "sat\n((a true))\nunsat\nsat\n((a true))\nsat\n((a true))\n",
	"tactic_unsat_core.smt2":                       "unsat\n(positive negative)\nsat\n",
	"tactic_false_proof.smt2":                      "unsat\n((set-logic QF_UF)\n(proof\n(asserted false)))\n",
	"tactic_pretty_proof.smt2":                     "unsat\n(asserted false)\n",
	"tactic_unknown_recovery.smt2":                 "unknown\nunsat\nunsat\n",
	"tactic_failure_recovery.smt2":                 "unknown\nsat\nsat\n((a true))\n",
	"tactic_resource_limit_recovery.smt2":          "unknown\nsat\nsat\n((a true))\n",
	"array_as_array_direct_core_scopes.smt2":       "unknown\nunknown\nunknown\n",
	"model_macro_cleanup_scopes.smt2":              "sat\n((c false) ((f false) false) ((f true) false))\nsat\n((c true) ((f false) false) ((f true) true))\nsat\n((c false) ((f false) false) ((f true) false))\n",
	"model_compact_live_updates.smt2":              "sat\n(model (define-fun f ((x!0 Bool)) Bool (ite (= x!0 false) false (ite (= x!0 true) false false)))\n)\nsat\n(model (define-fun f ((x!0 Bool)) Bool false)\n)\n(((f false) false) ((f true) false))\nsat\n(model (define-fun f ((x!0 Bool)) Bool (ite (= x!0 false) false (ite (= x!0 true) false false)))\n)\n",
	"array_explicit_extensionality_scopes.smt2":    "sat\nunsat\nsat\nunsat\nsat\n",
	"array_default_store_visitation_scopes.smt2":   "sat\nsat\nunsat\nsat\n",
	"array_function_reference_signatures.smt2":     "unsat\nunsat\nunsat\nsat\n",
	"array_canonical_constructor_reads.smt2":       "sat\n(((select a i) true) ((select b j) false))\nunsat\nsat\nunsat\nunsat\nsat\n(((select a i) true) ((select b j) false))\n",
	"array_constructor_default_models.smt2":        "sat\n(((= a ((as const (Array Bool Bool)) true)) true) ((= b (store a false false)) true) ((select a true) true) ((select b false) false) ((select b true) true))\nunsat\nsat\n(((= a ((as const (Array Bool Bool)) true)) true) ((= b (store a false false)) true) ((select a true) true) ((select b false) false) ((select b true) true))\n",
	"array_store_chain_model_propagation.smt2":     "sat\n(((= b (store a true false)) true) ((= c (store b false true)) true) ((select b false) true) ((select c false) true) ((select c true) false))\nunsat\nsat\n(((= b (store a true false)) true) ((= c (store b false true)) true) ((select b false) true) ((select c false) true) ((select c true) false))\n",
	"array_cyclic_store_model_scopes.smt2":         "sat\n(((select a true) true) ((= a (store a true true)) true))\nunsat\nsat\n(((select a true) true) ((= a (store a true true)) true))\n",
	"array_weak_live_parent_scopes.smt2":           "sat\n(((= a b) true))\nunsat\nunsat\nsat\n(((= a b) true))\n",
	"array_late_default_alias_scopes.smt2":         "sat\nunsat\nsat\n",
	"array_multidimensional_read_queue.smt2":       "sat\nunsat\nsat\nunsat\nsat\n",
	"array_multidimensional_read_queue_proof.smt2": "sat\nunsat\nsat\nunsat\nsat\n",
	"array_nested_interface_scopes.smt2":           "sat\nunsat\nsat\n",
	"array_extensional_disequality_timing.smt2":    "sat\nunsat\nsat\nsat\nsat\n",
	"array_simple_boolean_witness_scopes.smt2":     "sat\n(((distinct a b) true))\nunsat\nsat\n(((distinct a b) true))\n",
	"array_full_boolean_witness_scopes.smt2":       "sat\n(((distinct a b) true))\nunsat\nsat\n(((distinct a b) true))\n",
	"array_extensional_simple_live_updates.smt2":   "sat\nunsat\nsat\nsat\n",
	"array_extensional_live_updates.smt2":          "sat\nunsat\nsat\nsat\n",
	"array_extensional_proof_scopes.smt2":          "unsat\nsat\n",
	"array_extensional_disabled_scopes.smt2":       "sat\nsat\n",
	"array_extensional_enabled_scopes.smt2":        "unsat\nsat\n",
	"bit_vector_symbolic_carry_scopes.smt2":        "sat\n(((= (bvadd x y) #xa) true) ((= (bvsub x y) #x4) true))\nunsat\nsat\n(((= (bvadd x y) #xa) true) ((= (bvsub x y) #x4) true))\n",
	"utvpi_mixed_model_refusal.smt2":               "unknown\n",
	"utvpi_mixed_sum_model_refusal.smt2":           "unknown\n",
	"utvpi_integer_parity_scopes.smt2":             "unsat\nsat\n((x 1) (y 1))\nunsat\nsat\n((x 1) (y 1))\n",
	"utvpi_mixed_models.smt2":                      "sat\n(((>= i 0) true) ((<= i 1) true) ((> r 0.0) true) ((< r (/ 1.0 3.0)) true))\nunsat\nsat\n(((>= i 0) true) ((<= i 1) true) ((> r 0.0) true) ((< r (/ 1.0 3.0)) true))\n",
	"utvpi_real_strict_scopes.smt2":                "sat\n(((> (+ x y) 0.0) true) ((< (+ x y) (/ 1.0 3.0)) true) ((= x y) true))\nunsat\nsat\n(((> (+ x y) 0.0) true) ((< (+ x y) (/ 1.0 3.0)) true) ((= x y) true))\n",
	"utvpi_unknown_first_push.smt2":                "sat\n((x (/ 1.0 2.0)) (y (/ 1.0 2.0)))\nunsat\nsat\n((x (/ 1.0 2.0)) (y (/ 1.0 2.0)))\n",
	"basic_graph_integer.smt2":                     "sat\n((i 1))\nunsat\nsat\n((i 1))\n",
	"basic_graph_real.smt2":                        "sat\n(((> r 0.0) true) ((< r (/ 1.0 3.0)) true))\nunsat\nsat\n(((> r 0.0) true) ((< r (/ 1.0 3.0)) true))\n",
	"basic_dense_mixed.smt2":                       "sat\n(((>= i 0) true) ((<= i 1) true) ((> r 0.0) true) ((< r (/ 1.0 3.0)) true))\nunsat\nsat\n(((>= i 0) true) ((<= i 1) true) ((> r 0.0) true) ((< r (/ 1.0 3.0)) true))\n",
	"basic_disabled_arithmetic_scopes.smt2":        "sat\n((p true))\nunknown\nsat\n((p true))\n",
	"basic_graph_mixed_model_refusal.smt2":         "unknown\n",
	"basic_unknown_first_push_dense.smt2":          "sat\n((r (/ 1.0 3.0)))\nunsat\nsat\n((r (/ 1.0 3.0)))\n",
	"ufidl_congruence_equality_bounds.smt2":        "sat\n(((>= (f u) 3) true) ((<= (f v) 2) true) ((distinct u v) true))\nunsat\nsat\n(((>= (f u) 3) true) ((<= (f v) 2) true) ((distinct u v) true))\n",
	"ufidl_dense_model_scopes.smt2":                "sat\n((x 6) (y 2))\nunsat\nsat\n((x 6) (y 2))\n",
	"ufidl_congruence_models.smt2":                 "sat\n(((f x) 5) ((f y) 7) ((distinct x y) true))\nunsat\nsat\n(((f x) 5) ((f y) 7) ((distinct x y) true))\n",
	"rdl_dense_model_scopes.smt2":                  "sat\n((x (/ 3.0 2.0)) (y (/ 1.0 2.0)))\nunsat\nsat\n((x (/ 3.0 2.0)) (y (/ 1.0 2.0)))\n",
	"rdl_graph_strict_scopes.smt2":                 "sat\n(((> x 0.0) true) ((< x (/ 1.0 3.0)) true))\nunsat\nsat\n(((> x 0.0) true) ((< x (/ 1.0 3.0)) true))\n",
	"idl_dense_model_scopes.smt2":                  "sat\n((x 6) (y 2))\nunsat\nsat\n((x 6) (y 2))\n",
	"idl_first_push_simplex.smt2":                  "sat\n((x 3) (y 1))\nunsat\nsat\n((x 3) (y 1))\n",
	"bit_vector_auto_carry.smt2":                   "sat\n(((bvadd x y) #xa) ((bvsub x y) #x4))\nunsat\nsat\n(((bvadd x y) #xa) ((bvsub x y) #x4))\n",
	"lra_auto_ite_profile.smt2":                    "sat\n((x (/ 3.0 2.0)))\nunsat\nsat\n((x (/ 3.0 2.0)))\n",
	"quantified_mixed_auto_profile.smt2":           "sat\n(((= (f 3.0) 4.0) true))\nunsat\nsat\n(((= (f 7.0) 8.0) true))\n",
	"array_auto_unit_profile.smt2":                 "sat\n(((select a true) true))\nsat\n(((select a true) true) ((select a false) false))\nsat\n(((select a true) true))\nunsat\nsat\n(((select a true) true))\n",
	"array_model_relevancy.smt2":                   "sat\n(((select a 3) 7) ((select a 99) 11))\nsat\n(((select a 3) 7) ((select a 4) 19) ((select a 99) 11))\nsat\n(((select a 3) 7) ((select a 99) 11))\nunsat\nsat\n(((select a 3) 7) ((select a 99) 11))\n",
	"real_auto_rational_profile.smt2":              "sat\n(((= x threshold) true))\nunsat\nsat\n(((= x threshold) true))\n",
	"integer_auto_deep_ite.smt2":                   "sat\n((holds true))\nunsat\nsat\n((holds true))\n",
	"quantified_ematching_saturation.smt2":         "unknown\nunsat\nunknown\n",
	"quantified_ematching_relevancy.smt2":          "unsat\nunsat\n",
	"quantified_model_relevancy.smt2":              "sat\n(((f 7) 7) ((f 11) 11))\nunsat\nsat\n(((f 11) 11))\n",
	"integer_uf_model_relevancy.smt2":              "sat\n(((distinct x y) true) ((distinct (f x) (f y)) true))\nunsat\nsat\n(((distinct x y) true) ((distinct (f x) (f y)) true))\n",
	"arithmetic_eq_options.smt2":                   "unsat\nsat\nunsat\nsat\nunsat\nsat\n",
	"rational_square_interval_scopes.smt2":         "unsat\nsat\nsat\n((d (/ 121.0 225.0)) (|squared difference| (/ 121.0 225.0)) ((let ((q d)) (+ q 0)) (/ 121.0 225.0)))\nunsat\nsat\nsat\n",
	"shared_bound_queue.smt2":                      "unknown\nsat\n((x 2.0) ((f x) 0.0) ((f z) 0.0))\nsat\n",
	"shared_nonlinear_model.smt2":                  "sat\nunknown\nsat\nsat\n((x 2.0) (y 1.0) (z (- 1.0)) ((f z) 0.0))\n",
	"row_conflict_relaxation.smt2":                 "unsat\nsat\nsat\n",
	"grobner_factor_dependencies.smt2":             "unsat\nsat\nsat\n",
	"nested_bound_dependency_scopes.smt2":          "unsat\nsat\nunsat\nsat\n",
	"nonlinear_bound_provenance_scopes.smt2":       "sat\nunsat\nsat\nunsat\nsat\n",
	"integer_bound_provenance_scopes.smt2":         "unsat\nunsat\nsat\n",
	"mixed_cut_bound_provenance_scopes.smt2":       "unsat\nsat\n",
	"integer_euclidean_scopes.smt2":                "unsat\nsat\n((x 1) (y 1))\n",
	"mixed_euclidean_large_integer.smt2":           "sat\n((x 1) (y 1))\n",
	"integer_equality_branch_scopes.smt2":          "unsat\nsat\n((x 1) (y 1))\n",
	"integer_branch_cut_scopes.smt2":               "sat\n((x 0) (y 1))\nunsat\nsat\n",
	"mixed_gomory_scopes.smt2":                     "unsat\nsat\n",
	"integer_freedom_patch_scopes.smt2":            "sat\n((x 1) (y 2))\nunsat\nsat\n",
	"mixed_integer_freedom_patch.smt2":             "sat\n((x 1) (y 2))\nsat\nunsat\nsat\n",
	"shared_linear_expression_bounds.smt2":         "unsat\nunsat\nsat\n",
	"mixed_equality_and_ite_join.smt2":             "sat\nunsat\nunsat\nunsat\nsat\n",
	"integer_gcd_scopes.smt2":                      "unsat\nsat\n",
	"integer_extended_gcd.smt2":                    "unsat\nsat\n",
	"mixed_integer_gcd_scopes.smt2":                "sat\nunsat\nunsat\nsat\n",
	"nonlinear_nested_row_aliases.smt2":            "unsat\nsat\n",
	"nonlinear_nested_integer_row_aliases.smt2":    "unsat\nsat\n",
	"shared_arithmetic_equality_cycle.smt2":        "sat\nunsat\nsat\n",
	"shared_arithmetic_integer_cycle.smt2":         "sat\nunsat\nsat\n",
	"nonlinear_max_min_bounds.smt2":                "unsat\n",
	"nonlinear_max_min_model.smt2":                 "sat\n",
	"shared_arithmetic_fixed_rows.smt2":            "unsat\nsat\n",
	"shared_arithmetic_fixed_integer.smt2":         "unsat\n",
	"nonlinear_numeric_progress.smt2":              "unsat\nsat\n((x 3) (y 4))\n",
	"nonlinear_cluster_inequalities.smt2":          "unsat\nsat\n((x 1.0) (y 1.0) (z (- 1.0)))\n",
	"nonlinear_cluster_strict.smt2":                "unsat\n",
	"nonlinear_horner_bound.smt2":                  "unsat\n",
	"nonlinear_completed_square.smt2":              "unsat\n",
	"nonlinear_nested_multivariate.smt2":           "unsat\nsat\n",
	"nonlinear_nested_integer.smt2":                "unsat\n",
	"linearized_product_real_scopes.smt2":          "sat\n((y 1.0))\nsat\n((y (/ 1.0 2.0)))\nsat\n((y (- 1.0)))\n",
	"linearized_product_integer_scopes.smt2":       "unsat\nsat\n((y 1))\n",
	"nonlinear_integer_factor_scopes.smt2":         "unsat\nsat\n",
	"nonlinear_integer_zero_split.smt2":            "sat\n",
	"nonlinear_integer_upper_split.smt2":           "sat\n",
	"nonlinear_grobner_expanded_square.smt2":       "unsat\nsat\n",
	"nonlinear_grobner_rational_square.smt2":       "unsat\n",
	"nonlinear_grobner_coupled.smt2":               "unsat\nsat\n",
	"nonlinear_zero_candidate.smt2":                "sat\n((x 0.0) ((* x x) 0.0))\n",
	"nonlinear_interval_candidate.smt2":            "sat\n",
	"nonlinear_product_scopes.smt2":                "unsat\nsat\n((x 2.0) (y 3.0) ((* x y) 6.0))\n",
	"nonlinear_positive_product.smt2":              "unsat\n",
	"integer_zero_divisor.smt2":                    "sat\n(((div x 0) (div 1 0)) ((mod x 0) (mod 1 0)) ((rem x 0) (rem 1 0)))\n",
	"integer_division_model.smt2":                  "sat\n((x (- 7)) ((div x 3) (- 3)) ((mod x 3) 2) ((rem x 3) 2))\n",
	"integer_negative_divisor.smt2":                "sat\n((x (- 7)) ((div x (- 3)) 3) ((mod x (- 3)) 2) ((rem x (- 3)) (- 2)))\n",
	"integer_division_scopes.smt2":                 "unsat\nsat\n((x 5))\n",
	"mixed_floor_scopes.smt2":                      "sat\n((i (- 2)))\nunsat\nsat\n",
	"mixed_is_int.smt2":                            "unsat\n",
	"mixed_not_is_int.smt2":                        "unsat\n",
	"mixed_arithmetic_model.smt2":                  "sat\n((i 1) (r (/ 3.0 2.0)))\n",
	"mixed_integer_fraction.smt2":                  "unsat\n",
	"mixed_negative_epsilon.smt2":                  "unsat\n",
	"mixed_arithmetic_scopes.smt2":                 "unsat\nsat\n((i 1) (r (/ 1.0 2.0)))\n",
	"all_quantified_arithmetic_after_push.smt2":    "unsat\nsat\n",
	"quantified_bounds.smt2":                       "unsat\nsat\n",
	"bit_vector_sharing_scopes.smt2":               "unsat\nsat\nunsat\nsat\n",
	"inferred_trigger.smt2":                        "unsat\n",
	"function_overloads.smt2":                      "sat\n",
	"function_definitions.smt2":                    "unsat\n",
	"definition_scopes.smt2":                       "sat\nsat\n",
	"definition_capture.smt2":                      "unsat\n",
	"definition_coercions.smt2":                    "unsat\n",
	"function_coercions.smt2":                      "unsat\n",
	"global_definitions.smt2":                      "unsat\n",
	"boolean_model.smt2":                           "sat\n(model\n  (define-fun p () Bool true)\n)\n",
	"constant_array.smt2":                          "unsat\n",
	"string_value.smt2":                            "sat\n((x \"a\"\"b\"))\n",
	"global_declarations.smt2":                     "sat\nsat\n",
	"boolean_scopes.smt2":                          "sat\n((|p spaced| true))\nunsat\nsat\na\"b\n",
	"uf_congruence.smt2":                           "unsat\n",
	"linear_integer.smt2":                          "sat\n((x 4))\nunsat\nsat\n",
	"linear_real.smt2":                             "sat\n((x (/ 3.0 2.0)))\n",
	"bit_vector.smt2":                              "sat\n((x #xf))\n",
	"array_read_store.smt2":                        "unsat\n",
	"quantified_boolean.smt2":                      "unsat\n",
	"parallel_let.smt2":                            "sat\n((p true))\n",
	"declaration_scopes.smt2":                      "sat\nsat\n",
	"named_core.smt2":                              "unsat\n(positive negative)\n",
	"quantified_let_capture.smt2":                  "unsat\n",
	"indexed_bit_vector.smt2":                      "unsat\n",
	"nary_equality.smt2":                           "unsat\n",
}

func TestSMTLibRunnerScripts(t *testing.T) {
	for name, expected := range scriptResults {
		t.Run(name, func(t *testing.T) {
			script, err := os.ReadFile(filepath.Join("test_vectors", name))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err = run(bytes.NewReader(script), &out)
			if sourceDebugMacroCoercionRefusal(t, name, err, &out) {
				return
			}
			if err != nil {
				t.Fatal(err, out.String())
			}
			if canonicalOutput(out.String()) != canonicalOutput(expected) {
				t.Fatal("Go SMT-LIB result", out.String(), "want", expected)
			}
		})
	}
}

// The native executable is a release build. Its macro substitution accepts
// Int arguments in Real binders, while rewriter_def.h:49 asserts exact binder
// and replacement sorts in source debug builds, before application coercion.
func sourceDebugMacroCoercionRefusal(t *testing.T, name string, err error, out *bytes.Buffer) bool {
	t.Helper()
	if !ast.ASTSourceDebug || name != "definition_coercions.smt2" {
		return false
	}
	if err == nil || err.Error() != "line 3 column 26: "+ast.ErrASTDomain.Error() || out.Len() != 0 {
		t.Fatal("source debug macro-binding sort assertion", err, out.String())
	}
	return true
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
			err = run(bytes.NewReader(script), &goOutput)
			debugRefusal := sourceDebugMacroCoercionRefusal(t, name, err, &goOutput)
			if err != nil && !debugRefusal {
				t.Fatal(err, goOutput.String())
			}
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = bytes.NewReader(script)
			cppOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal("C++ reference", err, string(cppOutput))
			}
			if debugRefusal {
				if canonicalOutput(string(cppOutput)) != canonicalOutput(scriptResults[name]) {
					t.Fatal("native macro coercion result", string(cppOutput))
				}
				return
			}
			if canonicalOutput(goOutput.String()) != canonicalOutput(string(cppOutput)) {
				t.Fatalf("Go/C++ mismatch\nGo:\n%s\nC++:\n%s", goOutput.String(), cppOutput)
			}
		})
	}
}

// Keep this reference limitation separate from the agreement set. Direct smt
// inherits QF_LIA after pb2bv has introduced BV terms, so the release reference
// omits the required BV theory and can return an invalid integer model. The
// corresponding ALL vector checks the complete mixed-theory model in Go/C++;
// source QF_IDL's eventual strategy instead bit-blasts before its SAT backend.
func TestSMTLibRunnerPB2BVArithmeticLogicReferenceGap(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	script, err := os.ReadFile(filepath.Join("test_vectors", "pb2bv_qflia_direct_smt_reference_gap.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(output))
	}
	want := "sat\n(((>= (+ (* 2 x) (* 3 y) (* 5 z) (* 7 w)) 17) false))\n"
	if canonicalOutput(string(output)) != canonicalOutput(want) {
		t.Fatalf("reference logic-profile behavior changed; reassess this isolated gap: %s", output)
	}
}

// The native combined solver uses its strategic tactic pipeline before the
// incremental core is selected. This remains an integration gap: the native
// nonincremental tactic solves this mixed sum, while the direct UTVPI core
// refuses its fractional Int candidate. The strict comparison set includes
// the matching incremental variant in utvpi_mixed_sum_model_refusal.smt2.
func TestSMTLibRunnerUTVPINonincrementalStrategyGap(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "utvpi_mixed_sum_model.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run(bytes.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if canonicalOutput(output.String()) != "unknown" {
		t.Fatal("update the recorded UTVPI strategy gap after checking the native pipeline", output.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = bytes.NewReader(script)
		native, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(native))
		}
		if canonicalOutput(string(native)) != "sat" {
			t.Fatal("native strategic-solver result changed", string(native))
		}
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

// The retained C++ backend gives up on the unbounded satisfiable scopes in
// proof mode. Preserve this case separately from strict output comparisons;
// x=p=q=0 is an independently checked rational witness for those scopes.
func TestSMTLibRunnerNestedUnboundedProofModeReferenceUnknown(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "nested_bound_dependency_scopes.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	unbounded := strings.Replace(string(script), "(assert (>= x 2.0))\n", "", 1)
	var out bytes.Buffer
	if err = run(strings.NewReader(unbounded), &out); err != nil {
		t.Fatal(err)
	}
	if canonicalOutput(out.String()) != canonicalOutput("unsat\nsat\nunsat\nsat\n") {
		t.Fatal(out.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		for _, entry := range []struct{ script, expected string }{
			{unbounded, "unsat\nunknown\nunsat\nunknown\n"},
			{strings.Replace(unbounded, "(push 1)", "(assert (= x 0.0))\n(push 1)", 1), "unsat\nsat\nunsat\nsat\n"},
		} {
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = strings.NewReader(entry.script)
			value, err := cmd.CombinedOutput()
			if err != nil || canonicalOutput(string(value)) != canonicalOutput(entry.expected) {
				t.Fatal("C++ unbounded scope or zero witness", string(value), err)
			}
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

// The retained C++ executable can return models violating p and (or a b)
// after changing relevancy from 0 to 2. Keep this case outside the strict
// comparison table: require valid Go models and expose the reference defect.
func TestSMTLibRunnerRelevancyUpdatesKeepModelsValidWithKnownCPPDefect(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("test_vectors", "relevancy_update_scopes.smt2"))
	if err != nil {
		t.Fatal(err)
	}
	valid := "sat\n((p true) ((or a b) true))\nsat\n((p true) ((or a b) true))\nsat\n((p true) ((or a b) true))\nunsat\nsat\n((p true) ((or a b) true))\n"
	var out bytes.Buffer
	if err = run(bytes.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if canonicalOutput(out.String()) != canonicalOutput(valid) {
		t.Fatal("Go model violates an active assertion after option update or pop", out.String())
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = bytes.NewReader(script)
		value, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal("C++ reference", err, string(value))
		}
		if canonicalOutput(string(value)) == canonicalOutput(valid) {
			return // A reference with the model defect fixed agrees with Go.
		}
		knownDefect := "sat\n((p true) ((or a b) true))\nsat\n((p false) ((or a b) false))\nsat\n((p false) ((or a b) false))\nunsat\nsat\n((p false) ((or a b) false))\n"
		if canonicalOutput(string(value)) != canonicalOutput(knownDefect) {
			t.Fatal("unexpected C++ option-update behavior", string(value))
		}
		t.Log("C++ reference returned models violating asserted formulas after the relevancy update; Go models remain valid")
	}
}
