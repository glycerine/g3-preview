// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package lp

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/util/lp/lp_params.pyg
func catalogCollectLpParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "rep_freq", util.ParamUInt, "the report frequency, in how many iterations print the cost and other info ", "0", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "min", util.ParamBool, "minimize cost", "false", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "print_stats", util.ParamBool, "print statistic", "false", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "simplex_strategy", util.ParamUInt, "simplex strategy for the solver", "0", "lp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bprop_on_pivoted_rows", util.ParamBool, "propagate bounds on rows changed by the pivot operation", "true", "lp"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectLpParams", catalogCollectLpParams)
}
