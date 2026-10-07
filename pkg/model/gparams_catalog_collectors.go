// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package model

import (
	"github.com/glycerine/g3/pkg/util"
)

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/model/model_evaluator_params.pyg
func catalogCollectModelEvaluatorParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", util.ParamUInt, "maximum amount of memory in megabytes", "4294967295", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_steps", util.ParamUInt, "maximum number of steps", "4294967295", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "completion", util.ParamBool, "assigns an interptetation to symbols that do not have one in the current model, when evaluating expressions in the current model", "false", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "cache", util.ParamBool, "cache intermediate results in the model evaluator", "true", "model_evaluator"); e != nil {
		return e
	}
	if e := catalogInsert(d, "array_equalities", util.ParamBool, "evaluate array equalities", "true", "model_evaluator"); e != nil {
		return e
	}
	return nil
}

// Source: src/model/model_params.pyg
func catalogCollectModelParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "partial", util.ParamBool, "enable/disable partial function interpretations", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "v1", util.ParamBool, "use Z3 version 1.x pretty printer", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "v2", util.ParamBool, "use Z3 version 2.x (x <= 16) pretty printer", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "compact", util.ParamBool, "try to compact function graph (i.e., function interpretations that are lookup tables)", "false", "model"); e != nil {
		return e
	}
	if e := catalogInsert(d, "completion", util.ParamBool, "enable/disable model completion", "false", "model"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectModelEvaluatorParams", catalogCollectModelEvaluatorParams)
	util.RegisterGlobalParamsCollector("catalogCollectModelParams", catalogCollectModelParams)
}
