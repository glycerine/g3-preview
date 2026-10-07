// Parameter collectors remain with the C++ definitions they translate.
package fpa

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/ast/fpa/fpa2bv_rewriter_params.pyg
func catalogCollectFpa2bvRewriterParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "hi_fp_unspecified", util.ParamBool, "use the 'hardware interpretation' for unspecified values in fp.min, fp.max, fp.to_ubv, fp.to_sbv, and fp.to_real", "false", "rewriter"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectFpa2bvRewriterParams", catalogCollectFpa2bvRewriterParams)
}
