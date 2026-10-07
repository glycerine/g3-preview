// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package smtcore

import (
	"github.com/glycerine/g3/pkg/util"
)

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/cmd_context/context_params.cpp
func catalogCollectContextParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "timeout", util.ParamUInt, "default timeout (in milliseconds) used for solvers", "4294967295", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "rlimit", util.ParamUInt, "default resource limit used for solvers. Unrestricted when set to 0.", "0", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "well_sorted_check", util.ParamBool, "type checker", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "type_check", util.ParamBool, "type checker (alias for well_sorted_check)", "true", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "auto_config", util.ParamBool, "use heuristics to automatically select solver and configure it", "true", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "model_validate", util.ParamBool, "validate models produced by solvers", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "dump_models", util.ParamBool, "dump models whenever check-sat returns sat", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "trace", util.ParamBool, "trace generation for VCC", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "trace_file_name", util.ParamString, "trace out file name (see option 'trace')", "z3.log", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "dot_proof_file", util.ParamString, "file in which to output graphical proofs", "proof.dot", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "debug_ref_count", util.ParamBool, "debug support for AST reference counting", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "smtlib2_compliant", util.ParamBool, "enable/disable SMT-LIB 2.0 compliance", "false", ""); e != nil {
		return e
	}
	return catalogCollectContextSolverParams(d)
}

// Source: src/cmd_context/context_params.cpp
func catalogCollectContextSolverParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "proof", util.ParamBool, "proof generation, it must be enabled when the Z3 context is created", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "model", util.ParamBool, "model generation for solvers, this parameter can be overwritten when creating a solver", "true", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "unsat_core", util.ParamBool, "unsat-core generation for solvers, this parameter can be overwritten when creating a solver, not every solver in Z3 supports unsat core generation", "false", ""); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/pattern/pattern_inference_params_helper.pyg
func catalogCollectPatternInferenceParamsHelper(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_multi_patterns", util.ParamUInt, "when patterns are not provided, the prover uses a heuristic to infer them, this option sets the threshold on the number of extra multi-patterns that can be created; by default, the prover creates at most one multi-pattern when there is no unary pattern", "0", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "block_loop_patterns", util.ParamBool, "block looping patterns during pattern inference", "true", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith", util.ParamUInt, "0 - do not infer patterns with arithmetic terms, 1 - use patterns with arithmetic terms if there is no other pattern, 2 - always use patterns with arithmetic terms", "1", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "use_database", util.ParamBool, "use pattern database", "false", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith_weight", util.ParamUInt, "default weight for quantifiers where the only available pattern has nested arithmetic terms", "5", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "non_nested_arith_weight", util.ParamUInt, "default weight for quantifiers where the only available pattern has non nested arithmetic terms", "10", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pull_quantifiers", util.ParamBool, "pull nested quantifiers, if no pattern was found", "true", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "warnings", util.ParamBool, "enable/disable warning messages in the pattern inference module", "false", "pi"); e != nil {
		return e
	}
	return nil
}

// Source: src/interp/interp_params.pyg
func catalogCollectInterpParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "profile", util.ParamBool, "(INTERP) profile interpolation", "false", "interp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "check", util.ParamBool, "(INTERP) check interpolants", "false", "interp"); e != nil {
		return e
	}
	return nil
}

// Source: src/smt/params/smt_params_helper.pyg
func catalogCollectSmtParamsHelper(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "auto_config", util.ParamBool, "automatically configure solver", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "logic", util.ParamSymbol, "logic used to setup the SMT solver", "", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "random_seed", util.ParamUInt, "random seed for the smt solver", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "relevancy", util.ParamUInt, "relevancy propagation heuristic: 0 - disabled, 1 - relevancy is tracked by only affects quantifier instantiation, 2 - relevancy is tracked, and an atom is only asserted if it is relevant", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "macro_finder", util.ParamBool, "try to find universally quantified formulas that can be viewed as macros", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "quasi_macros", util.ParamBool, "try to find universally quantified formulas that are quasi-macros", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restricted_quasi_macros", util.ParamBool, "try to find universally quantified formulas that are restricted quasi-macros", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ematching", util.ParamBool, "E-Matching based quantifier instantiation", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "phase_selection", util.ParamUInt, "phase selection heuristic: 0 - always false, 1 - always true, 2 - phase caching, 3 - phase caching conservative, 4 - phase caching conservative 2, 5 - random, 6 - number of occurrences", "3", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_strategy", util.ParamUInt, "0 - geometric, 1 - inner-outer-geometric, 2 - luby, 3 - fixed, 4 - arithmetic", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_factor", util.ParamDouble, "when using geometric (or inner-outer-geometric) progression of restarts, it specifies the constant used to multiply the currect restart threshold", "1.1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "case_split", util.ParamUInt, "0 - case split based on variable activity, 1 - similar to 0, but delay case splits created during the search, 2 - similar to 0, but cache the relevancy, 3 - case split based on relevancy (structural splitting), 4 - case split on relevancy and activity, 5 - case split on relevancy and current goal, 6 - activity-based case split with theory-aware branching activity", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "delay_units", util.ParamBool, "if true then z3 will not restart when a unit clause is learned", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "delay_units_threshold", util.ParamUInt, "maximum number of learned unit clauses before restarting, ignored if delay_units is false", "32", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pull_nested_quantifiers", util.ParamBool, "pull nested quantifiers", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "refine_inj_axioms", util.ParamBool, "refine injectivity axioms", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "timeout", util.ParamUInt, "timeout (in milliseconds) (UINT_MAX and 0 mean no timeout)", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rlimit", util.ParamUInt, "resource limit (0 means no limit)", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_conflicts", util.ParamUInt, "maximum number of conflicts before giving up.", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi", util.ParamBool, "model based quantifier instantiation (MBQI)", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.max_cexs", util.ParamUInt, "initial maximal number of counterexamples used in MBQI, each counterexample generates a quantifier instantiation", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.max_cexs_incr", util.ParamUInt, "increment for MBQI_MAX_CEXS, the increment is performed after each round of MBQI", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.max_iterations", util.ParamUInt, "maximum number of rounds of MBQI", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.trace", util.ParamBool, "generate tracing messages for Model Based Quantifier Instantiation (MBQI). It will display a message before every round of MBQI, and the quantifiers that were not satisfied", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.force_template", util.ParamUInt, "some quantifiers can be used as templates for building interpretations for functions. Z3 uses heuristics to decide whether a quantifier will be used as a template or not. Quantifiers with weight >= mbqi.force_template are forced to be used as a template", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.id", util.ParamString, "Only use model-based instantiation for quantifiers with id's beginning with string", "", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.profile", util.ParamBool, "profile quantifier instantiation", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.profile_freq", util.ParamUInt, "how frequent results are reported by qi.profile", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.max_instances", util.ParamUInt, "maximum number of quantifier instantiations", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.eager_threshold", util.ParamDouble, "threshold for eager quantifier instantiation", "10.0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.lazy_threshold", util.ParamDouble, "threshold for lazy quantifier instantiation", "20.0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.cost", util.ParamString, "expression specifying what is the cost of a given quantifier instantiation", "(+ weight generation)", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.max_multi_patterns", util.ParamUInt, "specify the number of extra multi patterns", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv.reflect", util.ParamBool, "create enode for every bit-vector term", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv.enable_int2bv", util.ParamBool, "enable support for int2bv and bv2int operators", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.random_initial_value", util.ParamBool, "use random initial values in the simplex-based procedure for linear arithmetic", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.solver", util.ParamUInt, "arithmetic solver: 0 - no solver, 1 - bellman-ford based solver (diff. logic only), 2 - simplex based solver, 3 - floyd-warshall based solver (diff. logic only) and no theory combination", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl", util.ParamBool, "(incomplete) nonlinear arithmetic support based on Groebner basis and interval propagation", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl.gb", util.ParamBool, "groebner Basis computation, this option is ignored when arith.nl=false", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl.branching", util.ParamBool, "branching on integer variables in non linear clusters", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl.rounds", util.ParamUInt, "threshold for number of (nested) final checks for non linear arithmetic", "1024", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.euclidean_solver", util.ParamBool, "eucliean solver for linear integer arithmetic", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.propagate_eqs", util.ParamBool, "propagate (cheap) equalities", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.propagation_mode", util.ParamUInt, "0 - no propagation, 1 - propagate existing literals, 2 - refine bounds", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.reflect", util.ParamBool, "reflect arithmetical operators to the congruence closure", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.branch_cut_ratio", util.ParamUInt, "branch/cut ratio for linear integer arithmetic", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.int_eq_branch", util.ParamBool, "branching using derived integer equations", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.ignore_int", util.ParamBool, "treat integer variables as real", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.dump_lemmas", util.ParamBool, "dump arithmetic theory lemmas to files", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.greatest_error_pivot", util.ParamBool, "Pivoting strategy", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.conflict_frequency", util.ParamUInt, "conflict frequency for Pseudo-Boolean theory", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.learn_complements", util.ParamBool, "learn complement literals for Pseudo-Boolean theory", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.enable_compilation", util.ParamBool, "enable compilation into sorting circuits for Pseudo-Boolean", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.enable_simplex", util.ParamBool, "enable simplex to check rational feasibility", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "array.weak", util.ParamBool, "weak array theory", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "array.extensional", util.ParamBool, "extensional array theory", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack", util.ParamUInt, "0 - disable dynamic ackermannization, 1 - expand Leibniz's axiom if a congruence is the root of a conflict, 2 - expand Leibniz's axiom if a congruence is used during conflict resolution", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.eq", util.ParamBool, "enable dynamic ackermannization for transtivity of equalities", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.factor", util.ParamDouble, "number of instance per conflict", "0.1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.gc", util.ParamUInt, "Dynamic ackermannization garbage collection frequency (per conflict)", "2000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.gc_inv_decay", util.ParamDouble, "Dynamic ackermannization garbage collection decay", "0.8", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.threshold", util.ParamUInt, " number of times the congruence rule must be used before Leibniz's axiom is expanded", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "theory_case_split", util.ParamBool, "Allow the context to use heuristics involving theory case splits, which are a set of literals of which exactly one can be assigned True. If this option is false, the context will generate extra axioms to enforce this instead.", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "string_solver", util.ParamSymbol, "solver for string/sequence theories. options are: 'z3str3' (specialized string solver), 'seq' (sequence solver), 'auto' (use static features to choose best solver)", "seq", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.validate", util.ParamBool, "validate unsat core produced by SMT context", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.strong_arrangements", util.ParamBool, "assert equivalences instead of implications when generating string arrangement axioms", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.aggressive_length_testing", util.ParamBool, "prioritize testing concrete length values over generating more options", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.aggressive_value_testing", util.ParamBool, "prioritize testing concrete string constant values over generating more options", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.aggressive_unroll_testing", util.ParamBool, "prioritize testing concrete regex unroll counts over generating more options", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.fast_length_tester_cache", util.ParamBool, "cache length tester constants instead of regenerating them", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.fast_value_tester_cache", util.ParamBool, "cache value tester constants instead of regenerating them", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.string_constant_cache", util.ParamBool, "cache all generated string constants generated from anywhere in theory_str", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.use_binary_search", util.ParamBool, "use a binary search heuristic for finding concrete length values for free variables in theory_str (set to False to use linear search)", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.binary_search_start", util.ParamUInt, "initial upper bound for theory_str binary search", "64", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "theory_aware_branching", util.ParamBool, "Allow the context to use extra information from theory solvers regarding literal branching prioritization.", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.finite_overlap_models", util.ParamBool, "attempt a finite model search for overlapping variables instead of completely giving up on the arrangement", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.overlap_priority", util.ParamDouble, "theory-aware priority for overlapping variable cases; use smt.theory_aware_branching=true", "-0.1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata", util.ParamBool, "use automata-based reasoning for regular expressions (Z3str3 only)", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_difficulty_threshold", util.ParamUInt, "difficulty threshold for regex automata heuristics", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_intersection_difficulty_threshold", util.ParamUInt, "difficulty threshold for regex intersection heuristics", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_failed_automaton_threshold", util.ParamUInt, "number of failed automaton construction attempts after which a full automaton is automatically built", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_failed_intersection_threshold", util.ParamUInt, "number of failed automaton intersection attempts after which intersection is always computed", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_length_attempt_threshold", util.ParamUInt, "number of length/path constraint attempts before checking unsatisfiability of regex terms", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.minimize", util.ParamBool, "minimize unsat core produced by SMT context", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.extend_patterns", util.ParamBool, "extend unsat core with literals that trigger (potential) quantifier instances", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.extend_patterns.max_distance", util.ParamUInt, "limits the distance of a pattern-extended unsat core", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.extend_nonlocal_patterns", util.ParamBool, "extend unsat cores with literals that have quantifiers with patterns that contain symbols which are not in the quantifier's body", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "lemma_gc_strategy", util.ParamUInt, "lemma garbage collection strategy: 0 - fixed, 1 - geometric, 2 - at restart, 3 - none", "0", "smt"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectContextParams", catalogCollectContextParams)
	util.RegisterGlobalParamsCollector("catalogCollectContextSolverParams", catalogCollectContextSolverParams)
	util.RegisterGlobalParamsCollector("catalogCollectPatternInferenceParamsHelper", catalogCollectPatternInferenceParamsHelper)
	util.RegisterGlobalParamsCollector("catalogCollectInterpParams", catalogCollectInterpParams)
	util.RegisterGlobalParamsCollector("catalogCollectSmtParamsHelper", catalogCollectSmtParamsHelper)
}
