// Parameter collectors remain with the C++ definitions they translate.
package ackermannization

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/ackermannization/ackermannization_params.pyg
func catalogCollectAckermannizationParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "eager", util.ParamBool, "eagerly instantiate all congruence rules", "true", "ackermannization"); e != nil {
		return e
	}
	return nil
}

// Source: src/ackermannization/ackermannize_bv_tactic_params.pyg
func catalogCollectAckermannizeBvTacticParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "div0_ackermann_limit", util.ParamUInt, "a bound for number of congruence Ackermann lemmas for div0 modelling", "1000", "rewriter"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectAckermannizationParams", catalogCollectAckermannizationParams)
	util.RegisterGlobalParamsCollector("catalogCollectAckermannizeBvTacticParams", catalogCollectAckermannizeBvTacticParams)
}
