// Source: src/sat/sat_simplifier.cpp:1503–1522; src/sat/sat_simplifier_params.pyg. Licensed under the MIT license.
package g3

// Keys borrow actual caller byte regions; they correspond, in source order, to
// subsumed, subsumption resolution, elim literals, elim bool vars, elim blocked clauses.
func (s *satSimplifier) collectStatistics(st *Statistics, keys [5]StatisticsKey) error {
	if e := s.open(); e != nil {
		return e
	}
	values := [5]uint32{s.numSubsumed, s.numSubRes, s.numElimLits, s.numElimVars, s.numBlocked}
	for i, v := range values {
		if e := st.UpdateUint(keys[i], v); e != nil {
			return e
		}
	}
	return nil
}
func collectSimplifierParamDescrs(d *ParamDescrs) error {
	for _, x := range [17]struct {
		name             string
		kind             ParamKind
		description, def string
	}{
		{"elim_blocked_clauses", ParamBool, "eliminate blocked clauses", "false"},
		{"elim_blocked_clauses_at", ParamUInt, "eliminate blocked clauses only once at the given simplification round", "2"},
		{"blocked_clause_limit", ParamUInt, "maximum number of literals visited during blocked clause elimination", "100000000"},
		{"resolution", ParamBool, "eliminate boolean variables using resolution", "true"},
		{"resolution.limit", ParamUInt, "approx. maximum number of literals visited during variable elimination", "500000000"},
		{"resolution.occ_cutoff", ParamUInt, "first cutoff (on number of positive/negative occurrences) for Boolean variable elimination", "10"},
		{"resolution.occ_cutoff_range1", ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing less than res_cls_cutoff1 clauses", "8"},
		{"resolution.occ_cutoff_range2", ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing more than res_cls_cutoff1 and less than res_cls_cutoff2", "5"},
		{"resolution.occ_cutoff_range3", ParamUInt, "second cutoff (number of positive/negative occurrences) for Boolean variable elimination, for problems containing more than res_cls_cutoff2", "3"},
		{"resolution.lit_cutoff_range1", ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing less than res_cls_cutoff1 clauses", "700"},
		{"resolution.lit_cutoff_range2", ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing more than res_cls_cutoff1 and less than res_cls_cutoff2", "400"},
		{"resolution.lit_cutoff_range3", ParamUInt, "second cutoff (total number of literals) for Boolean variable elimination, for problems containing more than res_cls_cutoff2", "300"},
		{"resolution.cls_cutoff1", ParamUInt, "limit1 - total number of problems clauses for the second cutoff of Boolean variable elimination", "100000000"},
		{"resolution.cls_cutoff2", ParamUInt, "limit2 - total number of problems clauses for the second cutoff of Boolean variable elimination", "700000000"},
		{"elim_vars", ParamBool, "enable variable elimination during simplification", "true"},
		{"subsumption", ParamBool, "eliminate subsumed clauses", "true"},
		{"subsumption.limit", ParamUInt, "approx. maximum number of literals visited during subsumption (and subsumption resolution)", "100000000"},
	} {
		if e := satConfigInsertDescriptor(d, x.name, x.kind, x.description, x.def); e != nil {
			return e
		}
	}
	return nil
}
