// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

// Actual sat::config::collect_param_descrs delegates to the generated collector.
// These literal calls preserve sat_params.pyg insertion order, descriptions,
// defaults and module bytes. There is no intermediate metadata slice.
func collectSATConfigParamDescrs(d *ParamDescrs) error {
	if d == nil {
		return ErrParamDescrsPrecondition
	}
	if e := satConfigInsertDescriptor(d, "max_memory", ParamUInt, "maximum amount of memory in megabytes", "4294967295"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "phase", ParamSymbol, "phase selection strategy: always_false, always_true, caching, random", "caching"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "phase.caching.on", ParamUInt, "phase caching on period (in number of conflicts)", "400"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "phase.caching.off", ParamUInt, "phase caching off period (in number of conflicts)", "100"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart", ParamSymbol, "restart strategy: luby or geometric", "luby"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart.initial", ParamUInt, "initial restart (number of conflicts)", "100"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart.max", ParamUInt, "maximal number of restarts.", "4294967295"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart.factor", ParamDouble, "restart increment factor for geometric strategy", "1.5"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "random_freq", ParamDouble, "frequency of random case splits", "0.01"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "random_seed", ParamUInt, "random seed", "0"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "burst_search", ParamUInt, "number of conflicts before first global simplification", "100"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "max_conflicts", ParamUInt, "maximum number of conflicts", "4294967295"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc", ParamSymbol, "garbage collection strategy: psm, glue, glue_psm, dyn_psm", "glue_psm"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.initial", ParamUInt, "learned clauses garbage collection frequence", "20000"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.increment", ParamUInt, "increment to the garbage collection threshold", "500"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.small_lbd", ParamUInt, "learned clauses with small LBD are never deleted (only used in dyn_psm)", "3"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.k", ParamUInt, "learned clauses that are inactive for k gc rounds are permanently deleted (only used in dyn_psm)", "7"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "minimize_lemmas", ParamBool, "minimize learned clauses", "true"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "dyn_sub_res", ParamBool, "dynamic subsumption resolution for minimizing learned clauses", "true"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "core.minimize", ParamBool, "minimize computed core", "false"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "core.minimize_partial", ParamBool, "apply partial (cheap) core minimization", "false"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "parallel_threads", ParamUInt, "number of parallel threads to use", "1"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "dimacs.core", ParamBool, "extract core from DIMACS benchmarks", "false"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "dimacs.display", ParamBool, "display SAT instance in DIMACS format and return unknown instead of solving", "false"); e != nil {
		return e
	}
	return nil
}

func satConfigInsertDescriptor(d *ParamDescrs, name string, kind ParamKind, description, defaultValue string) error {
	module := "sat"
	return d.InsertName(name, kind, &description, &defaultValue, &module)
}
