// Source: src/sat/sat_solver.cpp:2781-2801,3716-3751; src/sat/sat_solver.h:48-70.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

// Borrowed static-spelling counterparts. Their actual byte owners outlive the
// Statistics output; no source counter or text payload is mirrored here.
type satSolverStatisticsKeys struct {
	main                [15]StatisticsKey
	cleaner             [2]StatisticsKey
	simplifier          [5]StatisticsKey
	scc, asymm, probing StatisticsKey
}

func (s *satSolver) collectMainStatistics(st *Statistics, keys [15]StatisticsKey) error {
	if e := st.UpdateUint(keys[0], s.variables.mkVarCount); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[1], s.lifecycle.mkBinary); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[2], s.lifecycle.mkTernary); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[3], s.lifecycle.mkNary); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[4], s.gc.gcClauseStat); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[5], s.lifecycle.deleted); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[6], s.conflict.conflictStat); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[7], s.propagation.clausePropagations); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[8], s.decision.decisions); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[9], s.propagation.binaryPropagations); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[10], s.propagation.ternaryPropagations); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[11], s.gc.restartStat); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[12], s.analysis.minimizedLits); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[13], s.dynamic.eliminated); e != nil {
		return e
	}
	if e := st.UpdateUint(keys[14], s.blockedCorrectionSets); e != nil {
		return e
	}
	return nil
}
func (s *satSolver) resetMainStatistics() {
	s.variables.mkVarCount = 0
	s.lifecycle.mkBinary = 0
	s.lifecycle.mkTernary = 0
	s.lifecycle.mkNary = 0
	s.conflict.conflictStat = 0
	s.propagation.clausePropagations = 0
	s.propagation.binaryPropagations = 0
	s.propagation.ternaryPropagations = 0
	s.decision.decisions = 0
	s.gc.restartStat = 0
	s.gc.gcClauseStat = 0
	s.lifecycle.deleted = 0
	s.analysis.minimizedLits = 0
	s.dynamic.eliminated = 0
	s.lifecycle.nonLearnedGeneration = 0
	s.blockedCorrectionSets = 0
}
func (s *satSolver) collectStatistics(st *Statistics, keys satSolverStatisticsKeys) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := s.collectMainStatistics(st, keys.main); e != nil {
		return e
	}
	if e := s.cleaner.collectStatistics(st, keys.cleaner[0], keys.cleaner[1]); e != nil {
		return e
	}
	if e := s.simplifier.collectStatistics(st, keys.simplifier); e != nil {
		return e
	}
	if e := s.scc.collectStatistics(st, keys.scc); e != nil {
		return e
	}
	if e := s.asymm.collectStatistics(st, keys.asymm); e != nil {
		return e
	}
	return s.probing.collectStatistics(st, keys.probing)
}
func (s *satSolver) resetStatistics() error {
	if e := s.open(); e != nil {
		return e
	}
	s.resetMainStatistics()
	s.cleaner.resetStatistics()
	s.simplifier.resetStatistics()
	// Source deliberately has no m_scc.reset_statistics() at this site.
	s.asymm.resetStatistics()
	s.probing.resetStatistics()
	return nil
}
