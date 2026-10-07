// Licensed under the MIT license.
// Parameter collectors remain with the C++ definitions they translate.
package ast

import (
	"github.com/glycerine/g3/pkg/util"
)

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/ast/pp_params.pyg
func catalogCollectPpParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "max_indent", util.ParamUInt, "max. indentation in pretty printer", "4294967295", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_num_lines", util.ParamUInt, "max. number of lines to be displayed in pretty printer", "4294967295", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_width", util.ParamUInt, "max. width in pretty printer", "80", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_ribbon", util.ParamUInt, "max. ribbon (width - indentation) in pretty printer", "80", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "max_depth", util.ParamUInt, "max. term depth (when pretty printing SMT2 terms/formulas)", "5", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "min_alias_size", util.ParamUInt, "min. size for creating an alias for a shared term (when pretty printing SMT2 terms/formulas)", "10", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "decimal", util.ParamBool, "pretty print real numbers using decimal notation (the output may be truncated). Z3 adds a ? if the value is not precise", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "decimal_precision", util.ParamUInt, "maximum number of decimal places to be used when pp.decimal=true", "10", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_literals", util.ParamBool, "use Bit-Vector literals (e.g, #x0F and #b0101) during pretty printing", "true", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "fp_real_literals", util.ParamBool, "use real-numbered floating point literals (e.g, +1.0p-1) during pretty printing", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bv_neg", util.ParamBool, "use bvneg when displaying Bit-Vector literals where the most significant bit is 1", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "flat_assoc", util.ParamBool, "flat associative operators (when pretty printing SMT2 terms/formulas)", "true", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "fixed_indent", util.ParamBool, "use a fixed indentation for applications", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "single_line", util.ParamBool, "ignore line breaks when true", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "bounded", util.ParamBool, "ignore characters exceeding max width", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "pretty_proof", util.ParamBool, "use slower, but prettier, printer for proofs", "false", "pp"); e != nil {
		return e
	}
	if e := catalogInsert(d, "simplify_implies", util.ParamBool, "simplify nested implications for pretty printing", "true", "pp"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectPpParams", catalogCollectPpParams)
}
