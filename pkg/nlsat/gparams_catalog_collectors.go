// Parameter collectors remain with the C++ definitions they translate.
package nlsat

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/nlsat/nlsat_params.pyg
func catalogCollectNlsatParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", util.ParamUInt, "maximum amount of memory in megabytes", "4294967295", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "lazy", util.ParamUInt, "how lazy the solver is.", "0", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "reorder", util.ParamBool, "reorder variables.", "true", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "simplify_conflicts", util.ParamBool, "simplify conflicts using equalities before resolving them in nlsat solver.", "true", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "minimize_conflicts", util.ParamBool, "minimize conflicts", "false", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "randomize", util.ParamBool, "randomize selection of a witness in nlsat.", "true", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_conflicts", util.ParamUInt, "maximum number of conflicts.", "4294967295", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "shuffle_vars", util.ParamBool, "use a random variable order.", "false", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "inline_vars", util.ParamBool, "inline variables that can be isolated from equations", "false", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "seed", util.ParamUInt, "random seed.", "0", "nlsat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "factor", util.ParamBool, "factor polynomials produced during conflict resolution.", "true", "nlsat"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectNlsatParams", catalogCollectNlsatParams)
}
