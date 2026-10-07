// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package sls

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/tactic/sls/sls_params.pyg
func catalogCollectSlsParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", util.ParamUInt, "maximum amount of memory in megabytes", "4294967295", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_restarts", util.ParamUInt, "maximum number of restarts", "4294967295", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat", util.ParamBool, "use walksat assertion selection (instead of gsat)", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb", util.ParamBool, "use bandit heuristic for walksat assertion selection (instead of random)", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_constant", util.ParamDouble, "the ucb constant c in the term score + c * f(touched)", "20.0", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_init", util.ParamBool, "initialize total ucb touched to formula size", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_forget", util.ParamDouble, "scale touched by this factor every base restart interval", "1.0", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_ucb_noise", util.ParamDouble, "add noise 0 <= 256 * ucb_noise to ucb score for assertion selection", "0.0002", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "walksat_repick", util.ParamBool, "repick assertion if randomizing in local minima", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "scale_unsat", util.ParamDouble, "scale score of unsat expressions by this factor", "0.5", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "paws_init", util.ParamUInt, "initial/minimum assertion weights", "40", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "paws_sp", util.ParamUInt, "smooth assertion weights with probability paws_sp / 1024", "52", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "wp", util.ParamUInt, "random walk with probability wp / 1024", "100", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "vns_mc", util.ParamUInt, "in local minima, try Monte Carlo sampling vns_mc many 2-bit-flips per bit", "0", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "vns_repick", util.ParamBool, "in local minima, try picking a different assertion (only for walksat)", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_base", util.ParamUInt, "base restart interval given by moves per run", "100", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "restart_init", util.ParamBool, "initialize to 0 or random value (= 1) after restart", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "early_prune", util.ParamBool, "use early pruning for score prediction", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "random_offset", util.ParamBool, "use random offset for candidate evaluation", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "rescore", util.ParamBool, "rescore/normalize top-level score every base restart interval", "true", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "track_unsat", util.ParamBool, "keep a list of unsat assertions as done in SAT - currently disabled internally", "false", "sls"); e != nil {
		return e
	}
	if e := catalogInsert(d, "random_seed", util.ParamUInt, "random seed", "0", "sls"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectSlsParams", catalogCollectSlsParams)
}
