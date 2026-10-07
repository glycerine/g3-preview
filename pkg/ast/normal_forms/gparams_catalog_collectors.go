// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package normal_forms

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/ast/normal_forms/nnf.h
func catalogCollectNnf(d *util.ParamDescrs) error { return catalogCollectNnfParams(d) }

// Source: src/ast/normal_forms/nnf_params.pyg
func catalogCollectNnfParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_memory", util.ParamUInt, "maximum amount of memory in megabytes", "4294967295", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "sk_hack", util.ParamBool, "hack for VCC", "false", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "mode", util.ParamSymbol, "NNF translation mode: skolem (skolem normal form), quantifiers (skolem normal form + quantifiers in NNF), full", "skolem", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_labels", util.ParamBool, "remove/ignore labels in the input formula, this option is ignored if proofs are enabled", "false", "nnf"); e != nil {
		return e
	}
	if e := catalogInsert(d, "skolemize", util.ParamBool, "skolemize (existential force) quantifiers", "true", "nnf"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectNnf", catalogCollectNnf)
	util.RegisterGlobalParamsCollector("catalogCollectNnfParams", catalogCollectNnfParams)
}
