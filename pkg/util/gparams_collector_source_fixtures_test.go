// Test fixtures copied from the source-owned collectors. These supply upper-package
// registrations to the utility dispatcher tests without reversing the dependency.
// Licensed under the MIT license.
package util

func catalogCollectQfufbvTacticParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "sat_backend", ParamBool, "use SAT rather than SMT in qfufbv_ackr_tactic", "false", "ackermannization"); e != nil {
		return e
	}
	if e := catalogInsert(d, "inc_sat_backend", ParamBool, "use incremental SAT", "false", "ackermannization"); e != nil {
		return e
	}
	return nil
}

func catalogCollectSlsParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_restarts", ParamUInt, "maximum number of restarts", "4294967295", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat", ParamBool, "use walksat assertion selection (instead of gsat)", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb", ParamBool, "use bandit heuristic for walksat assertion selection (instead of random)", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_constant", ParamDouble, "the ucb constant c in the term score + c * f(touched)", "20.0", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_init", ParamBool, "initialize total ucb touched to formula size", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_forget", ParamDouble, "scale touched by this factor every base restart interval", "1.0", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_noise", ParamDouble, "add noise 0 <= 256 * ucb_noise to ucb score for assertion selection", "0.0002", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_repick", ParamBool, "repick assertion if randomizing in local minima", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "scale_unsat", ParamDouble, "scale score of unsat expressions by this factor", "0.5", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "paws_init", ParamUInt, "initial/minimum assertion weights", "40", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "paws_sp", ParamUInt, "smooth assertion weights with probability paws_sp / 1024", "52", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "wp", ParamUInt, "random walk with probability wp / 1024", "100", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "vns_mc", ParamUInt, "in local minima, try Monte Carlo sampling vns_mc many 2-bit-flips per bit", "0", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "vns_repick", ParamBool, "in local minima, try picking a different assertion (only for walksat)", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_base", ParamUInt, "base restart interval given by moves per run", "100", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_init", ParamBool, "initialize to 0 or random value (= 1) after restart", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "early_prune", ParamBool, "use early pruning for score prediction", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "random_offset", ParamBool, "use random offset for candidate evaluation", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rescore", ParamBool, "rescore/normalize top-level score every base restart interval", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "track_unsat", ParamBool, "keep a list of unsat assertions as done in SAT - currently disabled internally", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "random_seed", ParamUInt, "random seed", "0", "sls"); e != nil {
		return e
	}
	return nil
}

func catalogCollectOptParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "optsmt_engine", ParamSymbol, "select optimization engine: 'basic', 'farkas', 'symba'", "basic", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxsat_engine", ParamSymbol, "select engine for maxsat: 'core_maxsat', 'wmax', 'maxres', 'pd-maxres'", "maxres", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "priority", ParamSymbol, "select how to priortize objectives: 'lex' (lexicographic), 'pareto', or 'box'", "lex", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dump_benchmarks", ParamBool, "dump benchmarks for profiling", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "solution_prefix", ParamSymbol, "path prefix to dump intermediary, but non-optimal, solutions", "", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "timeout", ParamUInt, "timeout (in milliseconds) (UINT_MAX and 0 mean no timeout)", "4294967295", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rlimit", ParamUInt, "resource limit (0 means no limit)", "0", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "enable_sls", ParamBool, "enable SLS tuning during weighted maxsast", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "enable_sat", ParamBool, "enable the new SAT core for propositional constraints", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_01", ParamBool, "eliminate 01 variables", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pp.neat", ParamBool, "use neat (as opposed to less readable, but faster) pretty printer when displaying context", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.compile_equality", ParamBool, "compile arithmetical equalities into pseudo-Boolean equality (instead of two inequalites)", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.hill_climb", ParamBool, "give preference for large weight cores", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.add_upper_bound_block", ParamBool, "restict upper bound with constraint", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.max_num_cores", ParamUInt, "maximal number of cores per round", "4294967295", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.max_core_size", ParamUInt, "break batch of generated cores if size reaches this number", "3", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.maximize_assignment", ParamBool, "find an MSS/MCS to improve current assignment", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.max_correction_set_size", ParamUInt, "allow generating correction set constraints up to maximal size", "3", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.wmax", ParamBool, "use weighted theory solver to constrain upper bounds", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.pivot_on_correction_set", ParamBool, "reduce soft constraints if the current correction set is smaller than current core", "true", "opt"); e != nil {
		return e
	}
	return nil
}

func catalogCollectCombinedSolverParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "solver2_timeout", ParamUInt, "fallback to solver 1 after timeout even when in incremental model", "4294967295", "combined_solver"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_solver1", ParamBool, "if true, solver 2 is always used", "false", "combined_solver"); e != nil {
		return e
	}
	if e := catalogInsert(d, "solver2_unknown", ParamUInt, "what should be done when solver 2 returns unknown: 0 - just return unknown, 1 - execute solver 1 if quantifier free problem, 2 - execute solver 1", "1", "combined_solver"); e != nil {
		return e
	}
	return nil
}

func catalogCollectLpParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "rep_freq", ParamUInt, "the report frequency, in how many iterations print the cost and other info ", "0", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "min", ParamBool, "minimize cost", "false", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_stats", ParamBool, "print statistic", "false", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "simplex_strategy", ParamUInt, "simplex strategy for the solver", "0", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bprop_on_pivoted_rows", ParamBool, "propagate bounds on rows changed by the pivot operation", "true", "lp"); e != nil {
		return e
	}
	return nil
}

func catalogCollectModelEvaluatorParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_steps", ParamUInt, "maximum number of steps", "4294967295", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "completion", ParamBool, "assigns an interptetation to symbols that do not have one in the current model, when evaluating expressions in the current model", "false", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "cache", ParamBool, "cache intermediate results in the model evaluator", "true", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "array_equalities", ParamBool, "evaluate array equalities", "true", "model_evaluator"); e != nil {
		return e
	}
	return nil
}

// Source: src/model/model_params.pyg
func catalogCollectModelParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "partial", ParamBool, "enable/disable partial function interpretations", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "v1", ParamBool, "use Z3 version 1.x pretty printer", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "v2", ParamBool, "use Z3 version 2.x (x <= 16) pretty printer", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "compact", ParamBool, "try to compact function graph (i.e., function interpretations that are lookup tables)", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "completion", ParamBool, "enable/disable model completion", "false", "model"); e != nil {
		return e
	}
	return nil
}

func catalogCollectSatAsymmBranchParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "asymm_branch", ParamBool, "asymmetric branching", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "asymm_branch.rounds", ParamUInt, "maximum number of rounds of asymmetric branching", "32", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "asymm_branch.limit", ParamUInt, "approx. maximum number of literals visited during asymmetric branching", "100000000", "sat"); e != nil {
		return e
	}
	return nil
}

// Source: src/sat/sat_scc_params.pyg
func catalogCollectSatSccParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "scc", ParamBool, "eliminate Boolean variables by computing strongly connected components", "true", "sat"); e != nil {
		return e
	}
	return nil
}

// Source: src/sat/sat_simplifier_params.pyg
func catalogCollectSatSimplifierParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "elim_blocked_clauses", ParamBool, "eliminate blocked clauses", "false", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_blocked_clauses_at", ParamUInt, "eliminate blocked clauses only once at the given simplification round", "2", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blocked_clause_limit", ParamUInt, "maximum number of literals visited during blocked clause elimination", "100000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution", ParamBool, "eliminate boolean variables using resolution", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.limit", ParamUInt, "approx. maximum number of literals visited during variable elimination", "500000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff", ParamUInt, "first cutoff (on number of positive/negative occurrences) for Boolean variable elimination", "10", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff_range1", ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing less than res_cls_cutoff1 clauses", "8", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff_range2", ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing more than res_cls_cutoff1 and less than res_cls_cutoff2", "5", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff_range3", ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing more than res_cls_cutoff2", "3", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.lit_cutoff_range1", ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing less than res_cls_cutoff1 clauses", "700", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.lit_cutoff_range2", ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing more than res_cls_cutoff1 and less than res_cls_cutoff2", "400", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.lit_cutoff_range3", ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing more than res_cls_cutoff2", "300", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.cls_cutoff1", ParamUInt, "limit1 - total number of problems clauses for the second cutoff of Boolean variable elimination", "100000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.cls_cutoff2", ParamUInt, "limit2 - total number of problems clauses for the second cutoff of Boolean variable elimination", "700000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_vars", ParamBool, "enable variable elimination during simplification", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "subsumption", ParamBool, "eliminate subsumed clauses", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "subsumption.limit", ParamUInt, "approx. maximum number of literals visited during subsumption (and subsumption resolution)", "100000000", "sat"); e != nil {
		return e
	}
	return nil
}

func catalogCollectParserParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "ignore_user_patterns", ParamBool, "ignore patterns provided by the user", "false", "parser"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_bad_patterns", ParamBool, "ignore malformed patterns", "true", "parser"); e != nil {
		return e
	}
	if e := catalogInsert(d, "error_for_visual_studio", ParamBool, "display error messages in Visual Studio format", "false", "parser"); e != nil {
		return e
	}
	return nil
}

func catalogCollectContextParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "timeout", ParamUInt, "default timeout (in milliseconds) used for solvers", "4294967295", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "rlimit", ParamUInt, "default resource limit used for solvers. Unrestricted when set to 0.", "0", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "well_sorted_check", ParamBool, "type checker", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "type_check", ParamBool, "type checker (alias for well_sorted_check)", "true", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "auto_config", ParamBool, "use heuristics to automatically select solver and configure it", "true", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "model_validate", ParamBool, "validate models produced by solvers", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "dump_models", ParamBool, "dump models whenever check-sat returns sat", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "trace", ParamBool, "trace generation for VCC", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "trace_file_name", ParamString, "trace out file name (see option 'trace')", "z3.log", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "dot_proof_file", ParamString, "file in which to output graphical proofs", "proof.dot", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "debug_ref_count", ParamBool, "debug support for AST reference counting", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "smtlib2_compliant", ParamBool, "enable/disable SMT-LIB 2.0 compliance", "false", ""); e != nil {
		return e
	}
	return catalogCollectContextSolverParams(d)
}

// Source: src/cmd_context/context_params.cpp
func catalogCollectContextSolverParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "proof", ParamBool, "proof generation, it must be enabled when the Z3 context is created", "false", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "model", ParamBool, "model generation for solvers, this parameter can be overwritten when creating a solver", "true", ""); e != nil {
		return e
	}
	if e := catalogInsert(d, "unsat_core", ParamBool, "unsat-core generation for solvers, this parameter can be overwritten when creating a solver, not every solver in Z3 supports unsat core generation", "false", ""); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/pattern/pattern_inference_params_helper.pyg
func catalogCollectPatternInferenceParamsHelper(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_multi_patterns", ParamUInt, "when patterns are not provided, the prover uses a heuristic to infer them, this option sets the threshold on the number of extra multi-patterns that can be created; by default, the prover creates at most one multi-pattern when there is no unary pattern", "0", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "block_loop_patterns", ParamBool, "block looping patterns during pattern inference", "true", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith", ParamUInt, "0 - do not infer patterns with arithmetic terms, 1 - use patterns with arithmetic terms if there is no other pattern, 2 - always use patterns with arithmetic terms", "1", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "use_database", ParamBool, "use pattern database", "false", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith_weight", ParamUInt, "default weight for quantifiers where the only available pattern has nested arithmetic terms", "5", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "non_nested_arith_weight", ParamUInt, "default weight for quantifiers where the only available pattern has non nested arithmetic terms", "10", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pull_quantifiers", ParamBool, "pull nested quantifiers, if no pattern was found", "true", "pi"); e != nil {
		return e
	}
	if e := catalogInsert(d, "warnings", ParamBool, "enable/disable warning messages in the pattern inference module", "false", "pi"); e != nil {
		return e
	}
	return nil
}

// Source: src/interp/interp_params.pyg
func catalogCollectInterpParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "profile", ParamBool, "(INTERP) profile interpolation", "false", "interp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "check", ParamBool, "(INTERP) check interpolants", "false", "interp"); e != nil {
		return e
	}
	return nil
}

// Source: src/smt/params/smt_params_helper.pyg
func catalogCollectSmtParamsHelper(d *ParamDescrs) error {
	if e := catalogInsert(d, "auto_config", ParamBool, "automatically configure solver", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "logic", ParamSymbol, "logic used to setup the SMT solver", "", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "random_seed", ParamUInt, "random seed for the smt solver", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "relevancy", ParamUInt, "relevancy propagation heuristic: 0 - disabled, 1 - relevancy is tracked by only affects quantifier instantiation, 2 - relevancy is tracked, and an atom is only asserted if it is relevant", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "macro_finder", ParamBool, "try to find universally quantified formulas that can be viewed as macros", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "quasi_macros", ParamBool, "try to find universally quantified formulas that are quasi-macros", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restricted_quasi_macros", ParamBool, "try to find universally quantified formulas that are restricted quasi-macros", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ematching", ParamBool, "E-Matching based quantifier instantiation", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "phase_selection", ParamUInt, "phase selection heuristic: 0 - always false, 1 - always true, 2 - phase caching, 3 - phase caching conservative, 4 - phase caching conservative 2, 5 - random, 6 - number of occurrences", "3", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_strategy", ParamUInt, "0 - geometric, 1 - inner-outer-geometric, 2 - luby, 3 - fixed, 4 - arithmetic", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_factor", ParamDouble, "when using geometric (or inner-outer-geometric) progression of restarts, it specifies the constant used to multiply the currect restart threshold", "1.1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "case_split", ParamUInt, "0 - case split based on variable activity, 1 - similar to 0, but delay case splits created during the search, 2 - similar to 0, but cache the relevancy, 3 - case split based on relevancy (structural splitting), 4 - case split on relevancy and activity, 5 - case split on relevancy and current goal, 6 - activity-based case split with theory-aware branching activity", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "delay_units", ParamBool, "if true then z3 will not restart when a unit clause is learned", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "delay_units_threshold", ParamUInt, "maximum number of learned unit clauses before restarting, ignored if delay_units is false", "32", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pull_nested_quantifiers", ParamBool, "pull nested quantifiers", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "refine_inj_axioms", ParamBool, "refine injectivity axioms", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "timeout", ParamUInt, "timeout (in milliseconds) (UINT_MAX and 0 mean no timeout)", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rlimit", ParamUInt, "resource limit (0 means no limit)", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_conflicts", ParamUInt, "maximum number of conflicts before giving up.", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi", ParamBool, "model based quantifier instantiation (MBQI)", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.max_cexs", ParamUInt, "initial maximal number of counterexamples used in MBQI, each counterexample generates a quantifier instantiation", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.max_cexs_incr", ParamUInt, "increment for MBQI_MAX_CEXS, the increment is performed after each round of MBQI", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.max_iterations", ParamUInt, "maximum number of rounds of MBQI", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.trace", ParamBool, "generate tracing messages for Model Based Quantifier Instantiation (MBQI). It will display a message before every round of MBQI, and the quantifiers that were not satisfied", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.force_template", ParamUInt, "some quantifiers can be used as templates for building interpretations for functions. Z3 uses heuristics to decide whether a quantifier will be used as a template or not. Quantifiers with weight >= mbqi.force_template are forced to be used as a template", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mbqi.id", ParamString, "Only use model-based instantiation for quantifiers with id's beginning with string", "", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.profile", ParamBool, "profile quantifier instantiation", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.profile_freq", ParamUInt, "how frequent results are reported by qi.profile", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.max_instances", ParamUInt, "maximum number of quantifier instantiations", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.eager_threshold", ParamDouble, "threshold for eager quantifier instantiation", "10.0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.lazy_threshold", ParamDouble, "threshold for lazy quantifier instantiation", "20.0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.cost", ParamString, "expression specifying what is the cost of a given quantifier instantiation", "(+ weight generation)", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "qi.max_multi_patterns", ParamUInt, "specify the number of extra multi patterns", "0", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv.reflect", ParamBool, "create enode for every bit-vector term", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv.enable_int2bv", ParamBool, "enable support for int2bv and bv2int operators", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.random_initial_value", ParamBool, "use random initial values in the simplex-based procedure for linear arithmetic", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.solver", ParamUInt, "arithmetic solver: 0 - no solver, 1 - bellman-ford based solver (diff. logic only), 2 - simplex based solver, 3 - floyd-warshall based solver (diff. logic only) and no theory combination", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl", ParamBool, "(incomplete) nonlinear arithmetic support based on Groebner basis and interval propagation", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl.gb", ParamBool, "groebner Basis computation, this option is ignored when arith.nl=false", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl.branching", ParamBool, "branching on integer variables in non linear clusters", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.nl.rounds", ParamUInt, "threshold for number of (nested) final checks for non linear arithmetic", "1024", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.euclidean_solver", ParamBool, "eucliean solver for linear integer arithmetic", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.propagate_eqs", ParamBool, "propagate (cheap) equalities", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.propagation_mode", ParamUInt, "0 - no propagation, 1 - propagate existing literals, 2 - refine bounds", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.reflect", ParamBool, "reflect arithmetical operators to the congruence closure", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.branch_cut_ratio", ParamUInt, "branch/cut ratio for linear integer arithmetic", "2", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.int_eq_branch", ParamBool, "branching using derived integer equations", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.ignore_int", ParamBool, "treat integer variables as real", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.dump_lemmas", ParamBool, "dump arithmetic theory lemmas to files", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith.greatest_error_pivot", ParamBool, "Pivoting strategy", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.conflict_frequency", ParamUInt, "conflict frequency for Pseudo-Boolean theory", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.learn_complements", ParamBool, "learn complement literals for Pseudo-Boolean theory", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.enable_compilation", ParamBool, "enable compilation into sorting circuits for Pseudo-Boolean", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.enable_simplex", ParamBool, "enable simplex to check rational feasibility", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "array.weak", ParamBool, "weak array theory", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "array.extensional", ParamBool, "extensional array theory", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack", ParamUInt, "0 - disable dynamic ackermannization, 1 - expand Leibniz's axiom if a congruence is the root of a conflict, 2 - expand Leibniz's axiom if a congruence is used during conflict resolution", "1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.eq", ParamBool, "enable dynamic ackermannization for transtivity of equalities", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.factor", ParamDouble, "number of instance per conflict", "0.1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.gc", ParamUInt, "Dynamic ackermannization garbage collection frequency (per conflict)", "2000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.gc_inv_decay", ParamDouble, "Dynamic ackermannization garbage collection decay", "0.8", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dack.threshold", ParamUInt, " number of times the congruence rule must be used before Leibniz's axiom is expanded", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "theory_case_split", ParamBool, "Allow the context to use heuristics involving theory case splits, which are a set of literals of which exactly one can be assigned True. If this option is false, the context will generate extra axioms to enforce this instead.", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "string_solver", ParamSymbol, "solver for string/sequence theories. options are: 'z3str3' (specialized string solver), 'seq' (sequence solver), 'auto' (use static features to choose best solver)", "seq", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.validate", ParamBool, "validate unsat core produced by SMT context", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.strong_arrangements", ParamBool, "assert equivalences instead of implications when generating string arrangement axioms", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.aggressive_length_testing", ParamBool, "prioritize testing concrete length values over generating more options", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.aggressive_value_testing", ParamBool, "prioritize testing concrete string constant values over generating more options", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.aggressive_unroll_testing", ParamBool, "prioritize testing concrete regex unroll counts over generating more options", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.fast_length_tester_cache", ParamBool, "cache length tester constants instead of regenerating them", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.fast_value_tester_cache", ParamBool, "cache value tester constants instead of regenerating them", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.string_constant_cache", ParamBool, "cache all generated string constants generated from anywhere in theory_str", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.use_binary_search", ParamBool, "use a binary search heuristic for finding concrete length values for free variables in theory_str (set to False to use linear search)", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.binary_search_start", ParamUInt, "initial upper bound for theory_str binary search", "64", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "theory_aware_branching", ParamBool, "Allow the context to use extra information from theory solvers regarding literal branching prioritization.", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.finite_overlap_models", ParamBool, "attempt a finite model search for overlapping variables instead of completely giving up on the arrangement", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.overlap_priority", ParamDouble, "theory-aware priority for overlapping variable cases; use smt.theory_aware_branching=true", "-0.1", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata", ParamBool, "use automata-based reasoning for regular expressions (Z3str3 only)", "true", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_difficulty_threshold", ParamUInt, "difficulty threshold for regex automata heuristics", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_intersection_difficulty_threshold", ParamUInt, "difficulty threshold for regex intersection heuristics", "1000", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_failed_automaton_threshold", ParamUInt, "number of failed automaton construction attempts after which a full automaton is automatically built", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_failed_intersection_threshold", ParamUInt, "number of failed automaton intersection attempts after which intersection is always computed", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "str.regex_automata_length_attempt_threshold", ParamUInt, "number of length/path constraint attempts before checking unsatisfiability of regex terms", "10", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.minimize", ParamBool, "minimize unsat core produced by SMT context", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.extend_patterns", ParamBool, "extend unsat core with literals that trigger (potential) quantifier instances", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.extend_patterns.max_distance", ParamUInt, "limits the distance of a pattern-extended unsat core", "4294967295", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "core.extend_nonlocal_patterns", ParamBool, "extend unsat cores with literals that have quantifiers with patterns that contain symbols which are not in the quantifier's body", "false", "smt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "lemma_gc_strategy", ParamUInt, "lemma garbage collection strategy: 0 - fixed, 1 - geometric, 2 - at restart, 3 - none", "0", "smt"); e != nil {
		return e
	}
	return nil
}

func catalogCollectAlgebraicParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "zero_accuracy", ParamUInt, "one of the most time-consuming operations in the real algebraic number module is determining the sign of a polynomial evaluated at a sample point with non-rational algebraic number values. Let k be the value of this option. If k is 0, Z3 uses precise computation. Otherwise, the result of a polynomial evaluation is considered to be 0 if Z3 can show it is inside the interval (-1/2^k, 1/2^k)", "0", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "min_mag", ParamUInt, "Z3 represents algebraic numbers using a (square-free) polynomial p and an isolating interval (which contains one and only one root of p). This interval may be refined during the computations. This parameter specifies whether to cache the value of a refined interval or not. It says the minimal size of an interval for caching purposes is 1/2^16", "16", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor", ParamBool, "use polynomial factorization to simplify polynomials representing algebraic numbers", "true", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor_max_prime", ParamUInt, "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. This parameter limits the maximum prime number p to be used in the first step", "31", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor_num_primes", ParamUInt, "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. The search space may be reduced by factoring the polynomial in different GF(p)'s. This parameter specify the maximum number of finite factorizations to be considered, before lifiting and searching", "1", "algebraic"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor_search_size", ParamUInt, "parameter for the polynomial factorization procedure in the algebraic number module. Z3 polynomial factorization is composed of three steps: factorization in GF(p), lifting and search. This parameter can be used to limit the search space", "5000", "algebraic"); e != nil {
		return e
	}
	return nil
}

func catalogCollectRcfParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "use_prem", ParamBool, "use pseudo-remainder instead of remainder when computing GCDs and Sturm-Tarski sequences", "true", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "clean_denominators", ParamBool, "clean denominators before root isolation", "true", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "initial_precision", ParamUInt, "a value k that is the initial interval size (as 1/2^k) when creating transcendentals and approximated division", "24", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "inf_precision", ParamUInt, "a value k that is the initial interval size (i.e., (0, 1/2^l)) used as an approximation for infinitesimal values", "24", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_precision", ParamUInt, "during sign determination we switch from interval arithmetic to complete methods when the interval size is less than 1/2^k, where k is the max_precision", "128", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "lazy_algebraic_normalization", ParamBool, "during sturm-seq and square-free polynomial computations, only normalize algebraic polynomial expressions when the definining polynomial is monic", "true", "rcf"); e != nil {
		return e
	}
	return nil
}

func catalogCollectPpParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_indent", ParamUInt, "max. indentation in pretty printer", "4294967295", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_num_lines", ParamUInt, "max. number of lines to be displayed in pretty printer", "4294967295", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_width", ParamUInt, "max. width in pretty printer", "80", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_ribbon", ParamUInt, "max. ribbon (width - indentation) in pretty printer", "80", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_depth", ParamUInt, "max. term depth (when pretty printing SMT2 terms/formulas)", "5", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "min_alias_size", ParamUInt, "min. size for creating an alias for a shared term (when pretty printing SMT2 terms/formulas)", "10", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "decimal", ParamBool, "pretty print real numbers using decimal notation (the output may be truncated). Z3 adds a ? if the value is not precise", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "decimal_precision", ParamUInt, "maximum number of decimal places to be used when pp.decimal=true", "10", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_literals", ParamBool, "use Bit-Vector literals (e.g, #x0F and #b0101) during pretty printing", "true", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "fp_real_literals", ParamBool, "use real-numbered floating point literals (e.g, +1.0p-1) during pretty printing", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_neg", ParamBool, "use bvneg when displaying Bit-Vector literals where the most significant bit is 1", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat_assoc", ParamBool, "flat associative operators (when pretty printing SMT2 terms/formulas)", "true", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "fixed_indent", ParamBool, "use a fixed indentation for applications", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "single_line", ParamBool, "ignore line breaks when true", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bounded", ParamBool, "ignore characters exceeding max width", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pretty_proof", ParamBool, "use slower, but prettier, printer for proofs", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "simplify_implies", ParamBool, "simplify nested implications for pretty printing", "true", "pp"); e != nil {
		return e
	}
	return nil
}

func catalogCollectFpa2bvRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "hi_fp_unspecified", ParamBool, "use the 'hardware interpretation' for unspecified values in fp.min, fp.max, fp.to_ubv, fp.to_sbv, and fp.to_real", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

func catalogCollectNnf(d *ParamDescrs) error { return catalogCollectNnfParams(d) }

// Source: src/ast/normal_forms/nnf_params.pyg
func catalogCollectNnfParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "sk_hack", ParamBool, "hack for VCC", "false", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mode", ParamSymbol, "NNF translation mode: skolem (skolem normal form), quantifiers (skolem normal form + quantifiers in NNF), full", "skolem", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_labels", ParamBool, "remove/ignore labels in the input formula, this option is ignored if proofs are enabled", "false", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "skolemize", ParamBool, "skolemize (existential force) quantifiers", "true", "nnf"); e != nil {
		return e
	}
	return nil
}

func catalogCollectArithRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "algebraic_number_evaluator", ParamBool, "simplify/evaluate expressions containing (algebraic) irrational numbers.", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mul_to_power", ParamBool, "collpase (* t ... t) into (^ t k), it is ignored if expand_power is true.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "expand_power", ParamBool, "expand (^ t k) into (* t ... t) if  1 < k <= max_degree.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "expand_tan", ParamBool, "replace (tan x) with (/ (sin x) (cos x)).", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_degree", ParamUInt, "max degree of algebraic numbers (and power operators) processed by simplifier.", "64", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "sort_sums", ParamBool, "sort the arguments of + application.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "gcd_rounding", ParamBool, "use gcd rounding on integer arithmetic atoms.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith_lhs", ParamBool, "all monomials are moved to the left-hand-side, and the right-hand-side is just a constant.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "arith_ineq_lhs", ParamBool, "rewrite inequalities so that right-hand-side is a constant.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_to_real", ParamBool, "eliminate to_real from arithmetic predicates that contain only integers.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "push_to_real", ParamBool, "distribute to_real over * and +.", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "eq2ineq", ParamBool, "expand equalities into two inequalities", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_rem", ParamBool, "replace (rem x y) with (ite (>= y 0) (mod x y) (- (mod x y))).", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/array_rewriter_params.pyg
func catalogCollectArrayRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "expand_select_store", ParamBool, "replace a (select (store ...) ...) term by an if-then-else term", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "expand_store_eq", ParamBool, "reduce (store ...) = (store ...) with a common base into selects", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "sort_store", ParamBool, "sort nested stores when the indices are known to be different", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/bool_rewriter_params.pyg
func catalogCollectBoolRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "ite_extra_rules", ParamBool, "extra ite simplifications, these additional simplifications may reduce size locally but increase globally", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat", ParamBool, "create nary applications for and,or,+,*,bvadd,bvmul,bvand,bvor,bvxor", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_and", ParamBool, "conjunctions are rewritten using negation and disjunctions", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "local_ctx", ParamBool, "perform local (i.e., cheap) context simplifications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "local_ctx_limit", ParamUInt, "limit for applying local context simplifier", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blast_distinct", ParamBool, "expand a distinct predicate into a quadratic number of disequalities", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blast_distinct_threshold", ParamUInt, "when blast_distinct is true, only distinct expressions with less than this number of arguments are blasted", "4294967295", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/bv_rewriter_params.pyg
func catalogCollectBvRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "udiv2mul", ParamBool, "convert constant udiv to mul", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "split_concat_eq", ParamBool, "split equalities of the form (= (concat t1 t2) t3)", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bit2bool", ParamBool, "try to convert bit-vector terms of size 1 into Boolean terms", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blast_eq_value", ParamBool, "blast (some) Bit-vector equalities into bits", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_sign_ext", ParamBool, "expand sign-ext operator using concat and extract", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "hi_div0", ParamBool, "use the 'hardware interpretation' for division by zero (for bit-vector terms)", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mul2concat", ParamBool, "replace multiplication by a power of two into a concatenation", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bvnot2arith", ParamBool, "replace (bvnot x) with (bvsub -1 x)", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_sort_ac", ParamBool, "sort the arguments of all AC operators", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_trailing", ParamBool, "lean removal of trailing zeros", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_extract_prop", ParamBool, "attempt to partially propagate extraction inwards", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_not_simpl", ParamBool, "apply simplifications for bvnot", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_ite2id", ParamBool, "rewrite ite that can be simplified to identity", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_le_extra", ParamBool, "additional bu_(u/s)le simplifications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_urem_simpl", ParamBool, "additional simplification for bvurem", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/fpa_rewriter_params.pyg
func catalogCollectFpaRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "hi_fp_unspecified", ParamBool, "use the 'hardware interpretation' for unspecified values in fp.to_ubv, fp.to_sbv, fp.to_real, and fp.to_ieee_bv", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/poly_rewriter_params.pyg
func catalogCollectPolyRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "som", ParamBool, "put polynomials in som-of-monomials form", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "som_blowup", ParamUInt, "maximum number of monomials generated when putting a polynomial in sum-of-monomials normal form", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "hoist_mul", ParamBool, "hoist multiplication over summation to minimize number of multiplications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "hoist_cmul", ParamBool, "hoist constant multiplication over summation to minimize number of multiplications", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat", ParamBool, "create nary applications for and,or,+,*,bvadd,bvmul,bvand,bvor,bvxor", "true", "rewriter"); e != nil {
		return e
	}
	return nil
}

// Source: src/ast/rewriter/rewriter_params.pyg
func catalogCollectRewriterParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_steps", ParamUInt, "maximum number of steps", "4294967295", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat", ParamBool, "create nary applications for and,or,+,*,bvadd,bvmul,bvand,bvor,bvxor", "true", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "push_ite_arith", ParamBool, "push if-then-else over arithmetic terms.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "push_ite_bv", ParamBool, "push if-then-else over bit-vector terms.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pull_cheap_ite", ParamBool, "pull if-then-else terms when cheap.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_ineq_consistency_test_max", ParamUInt, "max size of conjunctions on which to perform consistency test based on inequalities on bitvectors.", "0", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "cache_all", ParamBool, "cache all intermediate results.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rewrite_patterns", ParamBool, "rewrite patterns.", "false", "rewriter"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_patterns_on_ground_qbody", ParamBool, "ignores patterns on quantifiers that don't mention their bound variables.", "true", "rewriter"); e != nil {
		return e
	}
	return nil
}

func catalogCollectNlsatParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "lazy", ParamUInt, "how lazy the solver is.", "0", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "reorder", ParamBool, "reorder variables.", "true", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "simplify_conflicts", ParamBool, "simplify conflicts using equalities before resolving them in nlsat solver.", "true", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "minimize_conflicts", ParamBool, "minimize conflicts", "false", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "randomize", ParamBool, "randomize selection of a witness in nlsat.", "true", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_conflicts", ParamUInt, "maximum number of conflicts.", "4294967295", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "shuffle_vars", ParamBool, "use a random variable order.", "false", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "inline_vars", ParamBool, "inline variables that can be isolated from equations", "false", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "seed", ParamUInt, "random seed.", "0", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor", ParamBool, "factor polynomials produced during conflict resolution.", "true", "nlsat"); e != nil {
		return e
	}
	return nil
}

func catalogCollectFixedpointParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "timeout", ParamUInt, "set timeout", "4294967295", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "engine", ParamSymbol, "Select: auto-config, datalog, duality, pdr, bmc, spacer", "auto-config", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.default_table", ParamSymbol, "default table implementation: sparse, hashtable, bitvector, interval", "sparse", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.default_relation", ParamSymbol, "default relation implementation: external_relation, pentagon", "pentagon", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.generate_explanations", ParamBool, "produce explanations for produced facts when using the datalog engine", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.use_map_names", ParamBool, "use names from map files when displaying tuples", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.magic_sets_for_queries", ParamBool, "magic set transformation will be used for queries", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.explanations_on_relation_level", ParamBool, "if true, explanations are generated as history of each relation, rather than per fact (generate_explanations must be set to true for this option to have any effect)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.unbound_compressor", ParamBool, "auxiliary relations will be introduced to avoid unbound variables in rule heads", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.similarity_compressor", ParamBool, "rules that differ only in values of constants will be merged into a single rule", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.similarity_compressor_threshold", ParamUInt, "if similarity_compressor is on, this value determines how many similar rules there must be in order for them to be merged", "11", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.all_or_nothing_deltas", ParamBool, "compile rules so that it is enough for the delta relation in union and widening operations to determine only whether the updated relation was modified or not", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.compile_with_widening", ParamBool, "widening will be used to compile recursive rules", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.default_table_checked", ParamBool, "if true, the default table will be default_table inside a wrapper that checks that its results are the same as of default_table_checker table", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.default_table_checker", ParamSymbol, "see default_table_checked", "null", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.check_relation", ParamSymbol, "name of default relation to check. operations on the default relation will be verified using SMT solving", "null", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.initial_restart_timeout", ParamUInt, "length of saturation run before the first restart (in ms), zero means no restarts", "0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.output_profile", ParamBool, "determines whether profile information should be output when outputting Datalog rules or instructions", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.print.tuples", ParamBool, "determines whether tuples for output predicates should be output", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.profile_timeout_milliseconds", ParamUInt, "instructions and rules that took less than the threshold will not be printed when printed the instruction/rule list", "0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.dbg_fpr_nonempty_relation_signature", ParamBool, "if true, finite_product_relation will attempt to avoid creating inner relation with empty signature by putting in half of the table columns, if it would have been empty otherwise", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "datalog.subsumption", ParamBool, "if true, removes/filters predicates with total transitions", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.full_expand", ParamBool, "Fully expand derivation trees", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.no_conj", ParamBool, "No forced covering (conjectures)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.feasible_edges", ParamBool, "Don't expand definitely infeasible edges", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.use_underapprox", ParamBool, "Use underapproximations", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.stratified_inlining", ParamBool, "Use stratified inlining", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.recursion_bound", ParamUInt, "Recursion bound for stratified inlining", "4294967295", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.profile", ParamBool, "profile run time", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.mbqi", ParamBool, "use model-based quantifier instantiation", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.batch_expand", ParamBool, "use batch expansion", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.conjecture_file", ParamString, "save conjectures to file", "", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.bfs_model_search", ParamBool, "use BFS strategy for expanding model search", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.farkas", ParamBool, "use lemma generator based on Farkas (for linear real arithmetic)", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "generate_proof_trace", ParamBool, "trace for 'sat' answer as proof object", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.flexible_trace", ParamBool, "allow PDR generate long counter-examples by extending candidate trace within search area", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.flexible_trace_depth", ParamUInt, "Controls the depth (below the current level) at which flexible trace can be applied", "4294967295", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.use_model_generalizer", ParamBool, "use model for backwards propagation (instead of symbolic simulation)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.validate_result", ParamBool, "validate result (by proof checking or model checking)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.simplify_formulas_pre", ParamBool, "simplify derived formulas before inductive propagation", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.simplify_formulas_post", ParamBool, "simplify derived formulas after inductive propagation", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.use_multicore_generalizer", ParamBool, "extract multiple cores for blocking states", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.use_inductive_generalizer", ParamBool, "generalize lemmas using induction strengthening", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.use_arith_inductive_generalizer", ParamBool, "generalize lemmas using arithmetic heuristics for induction strengthening", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.use_convex_closure_generalizer", ParamBool, "generalize using convex closures of lemmas", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.use_convex_interior_generalizer", ParamBool, "generalize using convex interiors of lemmas", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.cache_mode", ParamUInt, "use no (0), symbolic (1) or explicit cache (2) for model search", "0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.inductive_reachability_check", ParamBool, "assume negation of the cube on the previous level when checking for reachability (not only during cube weakening)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.max_num_contexts", ParamUInt, "maximal number of contexts to create", "500", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.try_minimize_core", ParamBool, "try to reduce core size (before inductive minimization)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pdr.utvpi", ParamBool, "Enable UTVPI strategy", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_fixedpoint_extensions", ParamBool, "use SMT-LIB2 fixedpoint extensions, instead of pure SMT2, when printing rules", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_low_level_smt2", ParamBool, "use (faster) low-level SMT2 printer (the printer is scalable but the result may not be as readable)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_with_variable_declarations", ParamBool, "use variable declarations when displaying rules (instead of attempting to use original names)", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_answer", ParamBool, "print answer instance(s) to query", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_certificate", ParamBool, "print certificate for reachability or non-reachability", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_boogie_certificate", ParamBool, "print certificate for reachability or non-reachability using a format understood by Boogie", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_statistics", ParamBool, "print statistics", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_aig", ParamSymbol, "Dump clauses in AIG text format (AAG) to the given file name", "", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "tab.selection", ParamSymbol, "selection method for tabular strategy: weight (default), first, var-use", "weight", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.bit_blast", ParamBool, "bit-blast bit-vectors", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.magic", ParamBool, "perform symbolic magic set transformation", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.scale", ParamBool, "add scaling variable to linear real arithmetic clauses", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.inline_linear", ParamBool, "try linear inlining method", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.inline_eager", ParamBool, "try eager inlining of rules", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.inline_linear_branch", ParamBool, "try linear inlining method with potential expansion", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.compress_unbound", ParamBool, "compress tails with unbound variables", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.fix_unbound_vars", ParamBool, "fix unbound variables in tail", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.unfold_rules", ParamUInt, "unfold rules statically using iterative squarring", "0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.slice", ParamBool, "simplify clause set using slicing", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.karr", ParamBool, "Add linear invariants to clauses using Karr's method", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.use_eqclass", ParamBool, "Generalizes equalities to equivalence classes", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.transform_arrays", ParamBool, "Rewrites arrays equalities and applies select over store", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.instantiate_arrays", ParamBool, "Transforms P(a) into P(i, a[i] a)", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.instantiate_arrays.enforce", ParamBool, "Transforms P(a) into P(i, a[i]), discards a from predicate", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.instantiate_arrays.nb_quantifier", ParamUInt, "Gives the number of quantifiers per array", "1", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.instantiate_arrays.slice_technique", ParamSymbol, "<no-slicing>=> GetId(i) = i, <smash> => GetId(i) = true", "no-slicing", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.quantify_arrays", ParamBool, "create quantified Horn clauses from clauses with arrays", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.instantiate_quantifiers", ParamBool, "instantiate quantified Horn clauses using E-matching heuristic", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.coalesce_rules", ParamBool, "coalesce rules", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.tail_simplifier_pve", ParamBool, "propagate_variable_equivalences", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.subsumption_checker", ParamBool, "Enable subsumption checker (no support for model conversion)", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.coi", ParamBool, "use cone of influence simplification", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "duality.enable_restarts", ParamBool, "DUALITY: enable restarts", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.order_children", ParamUInt, "SPACER: order of enqueuing children in non-linear rules : 0 (original), 1 (reverse)", "0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.eager_reach_check", ParamBool, "SPACER: eagerly check if a query is reachable using reachability facts of predecessors", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.use_lemma_as_cti", ParamBool, "SPACER: use a lemma instead of a CTI in flexible_trace", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.reset_obligation_queue", ParamBool, "SPACER: reset obligation queue when entering a new level", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.init_reach_facts", ParamBool, "SPACER: initialize reachability facts with false", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.use_array_eq_generalizer", ParamBool, "SPACER: attempt to generalize lemmas with array equalities", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.use_derivations", ParamBool, "SPACER: using derivation mechanism to cache intermediate results for non-linear rules", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.array_blast", ParamBool, "try to eliminate local array terms using Ackermannization -- some array terms may remain", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "xform.array_blast_full", ParamBool, "eliminate all local array variables by QE", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.skip_propagate", ParamBool, "Skip propagate/pushing phase. Turns PDR into a BMC that returns either reachable or unknown", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.max_level", ParamUInt, "Maximum level to explore", "4294967295", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.elim_aux", ParamBool, "Eliminate auxiliary variables in reachability facts", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.reach_as_init", ParamBool, "Extend initial rules with computed reachability facts", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.blast_term_ite", ParamBool, "Expand non-Boolean ite-terms", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.nondet_tie_break", ParamBool, "Break ties in obligation queue non-deterministically", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.reach_dnf", ParamBool, "Restrict reachability facts to DNF", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bmc.linear_unrolling_depth", ParamUInt, "Maximal level to explore", "4294967295", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.split_farkas_literals", ParamBool, "Split Farkas literals", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.native_mbp", ParamBool, "Use native mbp of Z3", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.eq_prop", ParamBool, "Enable equality and bound propagation in arithmetic", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.weak_abs", ParamBool, "Weak abstraction", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.restarts", ParamBool, "Enable reseting obligation queue", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.restart_initial_threshold", ParamUInt, "Intial threshold for restarts", "10", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.random_seed", ParamUInt, "Random seed to be used by SMT solver", "0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.ground_cti", ParamBool, "Require CTI to be ground", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.vs.dump_benchmarks", ParamBool, "dump benchmarks in virtual solver", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.vs.dump_min_time", ParamDouble, "min time to dump benchmark", "5.0", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.vs.recheck", ParamBool, "re-check locally during benchmark dumping", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.mbqi", ParamBool, "use model-based quantifier instantiation", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.keep_proxy", ParamBool, "keep proxy variables (internal parameter)", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.instantiate", ParamBool, "instantiate quantified lemmas", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.qlemmas", ParamBool, "allow quantified lemmas in frames", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.new_unsat_core", ParamBool, "use the new implementation of unsat-core-generation", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.minimize_unsat_core", ParamBool, "compute unsat-core by min-cut", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.farkas_optimized", ParamBool, "use the optimized farkas plugin, which performs gaussian elimination", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.farkas_a_const", ParamBool, "if the unoptimized farkas plugin is used, use the constants from A while constructing unsat_cores", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.lemma_sanity_check", ParamBool, "check during generalization whether lemma is actually correct", "false", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.reuse_pobs", ParamBool, "reuse POBs", "true", "fixedpoint"); e != nil {
		return e
	}
	if e := catalogInsert(d, "spacer.simplify_pob", ParamBool, "simplify POBs by removing redundant constraints", "false", "fixedpoint"); e != nil {
		return e
	}
	return nil
}

func catalogCollectAckermannizationParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "eager", ParamBool, "eagerly instantiate all congruence rules", "true", "ackermannization"); e != nil {
		return e
	}
	return nil
}

// Source: src/ackermannization/ackermannize_bv_tactic_params.pyg
func catalogCollectAckermannizeBvTacticParams(d *ParamDescrs) error {
	if e := catalogInsert(d, "div0_ackermann_limit", ParamUInt, "a bound for number of congruence Ackermann lemmas for div0 modelling", "1000", "rewriter"); e != nil {
		return e
	}
	return nil
}

func CollectSATConfigParamDescrs(d *ParamDescrs) error {
	if d == nil {
		return ErrParamDescrsPrecondition
	}
	if e := catalogSATFixtureInsert(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "phase", ParamSymbol, "phase selection strategy: always_false, always_true, caching, random", "caching"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "phase.caching.on", ParamUInt, "phase caching on period (in number of conflicts)", "400"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "phase.caching.off", ParamUInt, "phase caching off period (in number of conflicts)", "100"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "restart", ParamSymbol, "restart strategy: luby or geometric", "luby"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "restart.initial", ParamUInt, "initial restart (number of conflicts)", "100"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "restart.max", ParamUInt, "maximal number of restarts.", "4294967295"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "restart.factor", ParamDouble, "restart increment factor for geometric strategy", "1.5"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "random_freq", ParamDouble, "frequency of random case splits", "0.01"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "random_seed", ParamUInt, "random seed", "0"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "burst_search", ParamUInt, "number of conflicts before first global simplification", "100"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "max_conflicts", ParamUInt, "maximum number of conflicts", "4294967295"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "gc", ParamSymbol, "garbage collection strategy: psm, glue, glue_psm, dyn_psm", "glue_psm"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "gc.initial", ParamUInt, "learned clauses garbage collection frequence", "20000"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "gc.increment", ParamUInt, "increment to the garbage collection threshold", "500"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "gc.small_lbd", ParamUInt, "learned clauses with small LBD are never deleted (only used in dyn_psm)", "3"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "gc.k", ParamUInt, "learned clauses that are inactive for k gc rounds are permanently deleted (only used in dyn_psm)", "7"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "minimize_lemmas", ParamBool, "minimize learned clauses", "true"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "dyn_sub_res", ParamBool, "dynamic subsumption resolution for minimizing learned clauses", "true"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "core.minimize", ParamBool, "minimize computed core", "false"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "core.minimize_partial", ParamBool, "apply partial (cheap) core minimization", "false"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "parallel_threads", ParamUInt, "number of parallel threads to use", "1"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "dimacs.core", ParamBool, "extract core from DIMACS benchmarks", "false"); e != nil {
		return e
	}
	if e := catalogSATFixtureInsert(d, "dimacs.display", ParamBool, "display SAT instance in DIMACS format and return unknown instead of solving", "false"); e != nil {
		return e
	}
	return nil
}
func catalogSATFixtureInsert(d *ParamDescrs, name string, kind ParamKind, description, defaultValue string) error {
	module := "sat"
	return d.InsertName(name, kind, &description, &defaultValue, &module)
}
func init() {
	RegisterGlobalParamsCollector("catalogCollectQfufbvTacticParams", catalogCollectQfufbvTacticParams)
	RegisterGlobalParamsCollector("catalogCollectSlsParams", catalogCollectSlsParams)
	RegisterGlobalParamsCollector("catalogCollectOptParams", catalogCollectOptParams)
	RegisterGlobalParamsCollector("catalogCollectCombinedSolverParams", catalogCollectCombinedSolverParams)
	RegisterGlobalParamsCollector("catalogCollectLpParams", catalogCollectLpParams)
	RegisterGlobalParamsCollector("catalogCollectModelEvaluatorParams", catalogCollectModelEvaluatorParams)
	RegisterGlobalParamsCollector("catalogCollectModelParams", catalogCollectModelParams)
	RegisterGlobalParamsCollector("catalogCollectSatAsymmBranchParams", catalogCollectSatAsymmBranchParams)
	RegisterGlobalParamsCollector("catalogCollectSatSccParams", catalogCollectSatSccParams)
	RegisterGlobalParamsCollector("catalogCollectSatSimplifierParams", catalogCollectSatSimplifierParams)
	RegisterGlobalParamsCollector("catalogCollectParserParams", catalogCollectParserParams)
	RegisterGlobalParamsCollector("catalogCollectContextParams", catalogCollectContextParams)
	RegisterGlobalParamsCollector("catalogCollectContextSolverParams", catalogCollectContextSolverParams)
	RegisterGlobalParamsCollector("catalogCollectPatternInferenceParamsHelper", catalogCollectPatternInferenceParamsHelper)
	RegisterGlobalParamsCollector("catalogCollectInterpParams", catalogCollectInterpParams)
	RegisterGlobalParamsCollector("catalogCollectSmtParamsHelper", catalogCollectSmtParamsHelper)
	RegisterGlobalParamsCollector("catalogCollectAlgebraicParams", catalogCollectAlgebraicParams)
	RegisterGlobalParamsCollector("catalogCollectRcfParams", catalogCollectRcfParams)
	RegisterGlobalParamsCollector("catalogCollectPpParams", catalogCollectPpParams)
	RegisterGlobalParamsCollector("catalogCollectFpa2bvRewriterParams", catalogCollectFpa2bvRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectNnf", catalogCollectNnf)
	RegisterGlobalParamsCollector("catalogCollectNnfParams", catalogCollectNnfParams)
	RegisterGlobalParamsCollector("catalogCollectArithRewriterParams", catalogCollectArithRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectArrayRewriterParams", catalogCollectArrayRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectBoolRewriterParams", catalogCollectBoolRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectBvRewriterParams", catalogCollectBvRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectFpaRewriterParams", catalogCollectFpaRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectPolyRewriterParams", catalogCollectPolyRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectRewriterParams", catalogCollectRewriterParams)
	RegisterGlobalParamsCollector("catalogCollectNlsatParams", catalogCollectNlsatParams)
	RegisterGlobalParamsCollector("catalogCollectFixedpointParams", catalogCollectFixedpointParams)
	RegisterGlobalParamsCollector("catalogCollectAckermannizationParams", catalogCollectAckermannizationParams)
	RegisterGlobalParamsCollector("catalogCollectAckermannizeBvTacticParams", catalogCollectAckermannizeBvTacticParams)
	RegisterGlobalParamsCollector("CollectSATConfigParamDescrs", CollectSATConfigParamDescrs)
}
