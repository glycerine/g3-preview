// Parameter collectors remain with the C++ definitions they translate.
package smtlogics

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/tactic/smtlogics/qfufbv_tactic_params.pyg
func catalogCollectQfufbvTacticParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "sat_backend", util.ParamBool, "use SAT rather than SMT in qfufbv_ackr_tactic", "false", "ackermannization"); e != nil {
		return e
	}
	if e := catalogInsert(d, "inc_sat_backend", util.ParamBool, "use incremental SAT", "false", "ackermannization"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectQfufbvTacticParams", catalogCollectQfufbvTacticParams)
}
