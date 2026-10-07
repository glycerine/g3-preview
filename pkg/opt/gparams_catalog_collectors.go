// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package opt

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/opt/opt_params.pyg
func catalogCollectOptParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "optsmt_engine", util.ParamSymbol, "select optimization engine: 'basic', 'farkas', 'symba'", "basic", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxsat_engine", util.ParamSymbol, "select engine for maxsat: 'core_maxsat', 'wmax', 'maxres', 'pd-maxres'", "maxres", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "priority", util.ParamSymbol, "select how to priortize objectives: 'lex' (lexicographic), 'pareto', or 'box'", "lex", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "dump_benchmarks", util.ParamBool, "dump benchmarks for profiling", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "solution_prefix", util.ParamSymbol, "path prefix to dump intermediary, but non-optimal, solutions", "", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "timeout", util.ParamUInt, "timeout (in milliseconds) (UINT_MAX and 0 mean no timeout)", "4294967295", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rlimit", util.ParamUInt, "resource limit (0 means no limit)", "0", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "enable_sls", util.ParamBool, "enable SLS tuning during weighted maxsast", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "enable_sat", util.ParamBool, "enable the new SAT core for propositional constraints", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_01", util.ParamBool, "eliminate 01 variables", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pp.neat", util.ParamBool, "use neat (as opposed to less readable, but faster) pretty printer when displaying context", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pb.compile_equality", util.ParamBool, "compile arithmetical equalities into pseudo-Boolean equality (instead of two inequalites)", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.hill_climb", util.ParamBool, "give preference for large weight cores", "true", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.add_upper_bound_block", util.ParamBool, "restict upper bound with constraint", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.max_num_cores", util.ParamUInt, "maximal number of cores per round", "4294967295", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.max_core_size", util.ParamUInt, "break batch of generated cores if size reaches this number", "3", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.maximize_assignment", util.ParamBool, "find an MSS/MCS to improve current assignment", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.max_correction_set_size", util.ParamUInt, "allow generating correction set constraints up to maximal size", "3", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.wmax", util.ParamBool, "use weighted theory solver to constrain upper bounds", "false", "opt"); e != nil {
		return e
	}
	if e := catalogInsert(d, "maxres.pivot_on_correction_set", util.ParamBool, "reduce soft constraints if the current correction set is smaller than current core", "true", "opt"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectOptParams", catalogCollectOptParams)
}
