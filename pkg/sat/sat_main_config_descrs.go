// Source: src/sat/sat_params.pyg; src/sat/sat_config.cpp.
// Licensed under the MIT license.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

// Actual sat::config::collect_param_descrs delegates to the generated collector.
// These literal calls preserve sat_params.pyg insertion order, descriptions,
// defaults and module bytes. There is no intermediate metadata slice.
func collectSATConfigParamDescrs(d *util.ParamDescrs) error {
	if d == nil {
		return util.ErrParamDescrsPrecondition
	}
	if e := satConfigInsertDescriptor(d, "max_memory", util.ParamUInt, "maximum amount of memory in megabytes", "4294967295"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "phase", util.ParamSymbol, "phase selection strategy: always_false, always_true, caching, random", "caching"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "phase.caching.on", util.ParamUInt, "phase caching on period (in number of conflicts)", "400"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "phase.caching.off", util.ParamUInt, "phase caching off period (in number of conflicts)", "100"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart", util.ParamSymbol, "restart strategy: luby or geometric", "luby"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart.initial", util.ParamUInt, "initial restart (number of conflicts)", "100"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart.max", util.ParamUInt, "maximal number of restarts.", "4294967295"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "restart.factor", util.ParamDouble, "restart increment factor for geometric strategy", "1.5"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "random_freq", util.ParamDouble, "frequency of random case splits", "0.01"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "random_seed", util.ParamUInt, "random seed", "0"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "burst_search", util.ParamUInt, "number of conflicts before first global simplification", "100"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "max_conflicts", util.ParamUInt, "maximum number of conflicts", "4294967295"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc", util.ParamSymbol, "garbage collection strategy: psm, glue, glue_psm, dyn_psm", "glue_psm"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.initial", util.ParamUInt, "learned clauses garbage collection frequence", "20000"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.increment", util.ParamUInt, "increment to the garbage collection threshold", "500"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.small_lbd", util.ParamUInt, "learned clauses with small LBD are never deleted (only used in dyn_psm)", "3"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "gc.k", util.ParamUInt, "learned clauses that are inactive for k gc rounds are permanently deleted (only used in dyn_psm)", "7"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "minimize_lemmas", util.ParamBool, "minimize learned clauses", "true"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "dyn_sub_res", util.ParamBool, "dynamic subsumption resolution for minimizing learned clauses", "true"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "core.minimize", util.ParamBool, "minimize computed core", "false"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "core.minimize_partial", util.ParamBool, "apply partial (cheap) core minimization", "false"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "parallel_threads", util.ParamUInt, "number of parallel threads to use", "1"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "dimacs.core", util.ParamBool, "extract core from DIMACS benchmarks", "false"); e != nil {
		return e
	}
	if e := satConfigInsertDescriptor(d, "dimacs.display", util.ParamBool, "display SAT instance in DIMACS format and return unknown instead of solving", "false"); e != nil {
		return e
	}
	return nil
}

func satConfigInsertDescriptor(d *util.ParamDescrs, name string, kind util.ParamKind, description, defaultValue string) error {
	module := "sat"
	return d.InsertName(name, kind, &description, &defaultValue, &module)
}

func init() {
	util.RegisterGlobalParamsCollector("CollectSATConfigParamDescrs", collectSATConfigParamDescrs)
}
