// Parameter collectors remain with the C++ definitions they translate.
package util

import "github.com/glycerine/g3/pkg/util"

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/parsers/util/parser_params.pyg
func catalogCollectParserParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "ignore_user_patterns", util.ParamBool, "ignore patterns provided by the user", "false", "parser"); e != nil {
		return e
	}
	if e := catalogInsert(d, "ignore_bad_patterns", util.ParamBool, "ignore malformed patterns", "true", "parser"); e != nil {
		return e
	}
	if e := catalogInsert(d, "error_for_visual_studio", util.ParamBool, "display error messages in Visual Studio format", "false", "parser"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectParserParams", catalogCollectParserParams)
}
