// Licensed under the MIT license.
package g3

var catalogDescriptionLiterals = [...]struct{ module, text string }{
	{"ackermannization", "solving UF via ackermannization"},
	{"nnf", "negation normal form"},
	{"pi", "pattern inference (heuristics) for universal formulas (without annotation)"},
	{"pp", "pretty printer"},
	{"rewriter", "new formula simplification module used in the tactic framework, and new solvers"},
	{"interp", "interpolation parameters"},
	{"algebraic", "real algebraic number package"},
	{"rcf", "real closed fields"},
	{"fixedpoint", "fixedpoint parameters"},
	{"nlsat", "nonlinear solver"},
	{"opt", "optimization parameters"},
	{"sat", "propositional SAT solver"},
	{"smt", "smt solver based on lazy smt"},
	{"combined_solver", "combines two solvers: non-incremental (solver1) and incremental (solver2)"},
	{"sls", "Experimental Stochastic Local Search Solver (for QFBV only)."},
	{"ackermannization", "tactics based on solving UF-theories via ackermannization (see also ackr module)"},
}

func (c *GlobalParamsCatalog) dispatch(r GlobalParamsRegistrar) error {
	if e := c.global(r, catalogCollectContextParams); e != nil {
		return e
	}
	if e := c.global(r, catalogCollectEnvParams); e != nil {
		return e
	}
	if e := c.module(r, "ackermannization", catalogCollectAckermannizationParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectAckermannizeBvTacticParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectFpa2bvRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "nnf", catalogCollectNnf); e != nil {
		return e
	}
	if e := c.module(r, "nnf", catalogCollectNnfParams); e != nil {
		return e
	}
	if e := c.module(r, "pi", catalogCollectPatternInferenceParamsHelper); e != nil {
		return e
	}
	if e := c.module(r, "pp", catalogCollectPpParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectArithRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectArrayRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectBoolRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectBvRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectFpaRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectPolyRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "rewriter", catalogCollectRewriterParams); e != nil {
		return e
	}
	if e := c.module(r, "interp", catalogCollectInterpParams); e != nil {
		return e
	}
	if e := c.module(r, "algebraic", catalogCollectAlgebraicParams); e != nil {
		return e
	}
	if e := c.module(r, "rcf", catalogCollectRcfParams); e != nil {
		return e
	}
	if e := c.module(r, "model_evaluator", catalogCollectModelEvaluatorParams); e != nil {
		return e
	}
	if e := c.module(r, "model", catalogCollectModelParams); e != nil {
		return e
	}
	if e := c.module(r, "fixedpoint", catalogCollectFixedpointParams); e != nil {
		return e
	}
	if e := c.module(r, "nlsat", catalogCollectNlsatParams); e != nil {
		return e
	}
	if e := c.module(r, "opt", catalogCollectOptParams); e != nil {
		return e
	}
	if e := c.module(r, "parser", catalogCollectParserParams); e != nil {
		return e
	}
	if e := c.module(r, "sat", catalogCollectSatAsymmBranchParams); e != nil {
		return e
	}
	if e := c.module(r, "sat", collectSATConfigParamDescrs); e != nil {
		return e
	}
	if e := c.module(r, "sat", catalogCollectSatSccParams); e != nil {
		return e
	}
	if e := c.module(r, "sat", catalogCollectSatSimplifierParams); e != nil {
		return e
	}
	if e := c.module(r, "smt", catalogCollectSmtParamsHelper); e != nil {
		return e
	}
	if e := c.module(r, "combined_solver", catalogCollectCombinedSolverParams); e != nil {
		return e
	}
	if e := c.module(r, "sls", catalogCollectSlsParams); e != nil {
		return e
	}
	if e := c.module(r, "ackermannization", catalogCollectQfufbvTacticParams); e != nil {
		return e
	}
	if e := c.module(r, "lp", catalogCollectLpParams); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("ackermannization", c.region, c.descriptions[0], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("nnf", c.region, c.descriptions[1], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("pi", c.region, c.descriptions[2], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("pp", c.region, c.descriptions[3], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("rewriter", c.region, c.descriptions[4], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("interp", c.region, c.descriptions[5], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("algebraic", c.region, c.descriptions[6], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("rcf", c.region, c.descriptions[7], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("fixedpoint", c.region, c.descriptions[8], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("nlsat", c.region, c.descriptions[9], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("opt", c.region, c.descriptions[10], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("sat", c.region, c.descriptions[11], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("smt", c.region, c.descriptions[12], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("combined_solver", c.region, c.descriptions[13], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("sls", c.region, c.descriptions[14], 0); e != nil {
		return e
	}
	if e := r.RegisterModuleDescription("ackermannization", c.region, c.descriptions[15], 0); e != nil {
		return e
	}
	return nil
}
