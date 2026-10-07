// Parameter collectors remain with the C++ definitions they translate.
package realclosure

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/math/realclosure/rcf_params.pyg
func catalogCollectRcfParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "use_prem", util.ParamBool, "use pseudo-remainder instead of remainder when computing GCDs and Sturm-Tarski sequences", "true", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "clean_denominators", util.ParamBool, "clean denominators before root isolation", "true", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "initial_precision", util.ParamUInt, "a value k that is the initial interval size (as 1/2^k) when creating transcendentals and approximated division", "24", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "inf_precision", util.ParamUInt, "a value k that is the initial interval size (i.e., (0, 1/2^l)) used as an approximation for infinitesimal values", "24", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_precision", util.ParamUInt, "during sign determination we switch from interval arithmetic to complete methods when the interval size is less than 1/2^k, where k is the max_precision", "128", "rcf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "lazy_algebraic_normalization", util.ParamBool, "during sturm-seq and square-free polynomial computations, only normalize algebraic polynomial expressions when the definining polynomial is monic", "true", "rcf"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectRcfParams", catalogCollectRcfParams)
}
