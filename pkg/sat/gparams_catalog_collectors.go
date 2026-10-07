// Parameter collectors remain with the C++ definitions they translate.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

func catalogInsert(d *util.ParamDescrs, key string, kind util.ParamKind, descr, def, module string) error {
	if module == "" {
		return d.InsertName(key, kind, &descr, &def, nil)
	}
	return d.InsertName(key, kind, &descr, &def, &module)
}

// Source: src/sat/sat_asymm_branch_params.pyg
func catalogCollectSatAsymmBranchParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "asymm_branch", util.ParamBool, "asymmetric branching", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "asymm_branch.rounds", util.ParamUInt, "maximum number of rounds of asymmetric branching", "32", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "asymm_branch.limit", util.ParamUInt, "approx. maximum number of literals visited during asymmetric branching", "100000000", "sat"); e != nil {
		return e
	}
	return nil
}

// Source: src/sat/sat_scc_params.pyg
func catalogCollectSatSccParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "scc", util.ParamBool, "eliminate Boolean variables by computing strongly connected components", "true", "sat"); e != nil {
		return e
	}
	return nil
}

// Source: src/sat/sat_simplifier_params.pyg
func catalogCollectSatSimplifierParams(d *util.ParamDescrs) error {
	if e := catalogInsert(d, "elim_blocked_clauses", util.ParamBool, "eliminate blocked clauses", "false", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_blocked_clauses_at", util.ParamUInt, "eliminate blocked clauses only once at the given simplification round", "2", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "blocked_clause_limit", util.ParamUInt, "maximum number of literals visited during blocked clause elimination", "100000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution", util.ParamBool, "eliminate boolean variables using resolution", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.limit", util.ParamUInt, "approx. maximum number of literals visited during variable elimination", "500000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff", util.ParamUInt, "first cutoff (on number of positive/negative occurrences) for Boolean variable elimination", "10", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff_range1", util.ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing less than res_cls_cutoff1 clauses", "8", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff_range2", util.ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing more than res_cls_cutoff1 and less than res_cls_cutoff2", "5", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.occ_cutoff_range3", util.ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing more than res_cls_cutoff2", "3", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.lit_cutoff_range1", util.ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing less than res_cls_cutoff1 clauses", "700", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.lit_cutoff_range2", util.ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing more than res_cls_cutoff1 and less than res_cls_cutoff2", "400", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.lit_cutoff_range3", util.ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing more than res_cls_cutoff2", "300", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.cls_cutoff1", util.ParamUInt, "limit1 - total number of problems clauses for the second cutoff of Boolean variable elimination", "100000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "resolution.cls_cutoff2", util.ParamUInt, "limit2 - total number of problems clauses for the second cutoff of Boolean variable elimination", "700000000", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "elim_vars", util.ParamBool, "enable variable elimination during simplification", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "subsumption", util.ParamBool, "eliminate subsumed clauses", "true", "sat"); e != nil {
		return e
	}
	if e := catalogInsert(d, "subsumption.limit", util.ParamUInt, "approx. maximum number of literals visited during subsumption (and subsumption resolution)", "100000000", "sat"); e != nil {
		return e
	}
	return nil
}

func init() {
	util.RegisterGlobalParamsCollector("catalogCollectSatAsymmBranchParams", catalogCollectSatAsymmBranchParams)
	util.RegisterGlobalParamsCollector("catalogCollectSatSccParams", catalogCollectSatSccParams)
	util.RegisterGlobalParamsCollector("catalogCollectSatSimplifierParams", catalogCollectSatSimplifierParams)
}
