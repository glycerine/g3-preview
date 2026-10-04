// Source: src/sat/sat_solver.cpp:161–176,723–802,993–1118,1191–1255,2751–2756.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package g3

import "errors"

var ErrSATMissingDimacsBinding = errors.New("sat: missing source DIMACS printer binding")

type satSolverAbort struct{}

func (satSolverAbort) Error() string { return "sat: abort_solver" }

func (s *satSolver) MkVar(external, decision bool) (satBoolVar, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.variables.mkVar(external, decision)
}
func (s *satSolver) MkClause(lits []satLiteral) error {
	if e := s.open(); e != nil {
		return e
	}
	s.variables.modelIsCurrent = false
	if satStorageSourceDebug {
		for _, l := range lits {
			elim, e := s.variables.wasEliminated(l.variable())
			if e != nil {
				return e
			}
			if elim {
				return ErrSATAssertion
			}
		}
	}
	n, e := s.conflict.userScopeLiterals.size()
	if e != nil {
		return e
	}
	if n == 0 {
		_, e = s.lifecycle.mkClauseCore(uint32(len(lits)), lits, false)
		return e
	}
	if e = s.auxLiterals.shrink(0); e != nil {
		return e
	}
	for _, l := range lits {
		if e = s.auxLiterals.push(uint32(l)); e != nil {
			return e
		}
	}
	for i := uint32(0); i < n; i++ {
		l, e := s.conflict.userScopeLiterals.read(i)
		if e != nil {
			return e
		}
		if e = s.auxLiterals.push(l); e != nil {
			return e
		}
	}
	size, e := s.auxLiterals.size()
	if e != nil {
		return e
	}
	_, e = s.lifecycle.mkClauseCoreSpan(size, satConflictRuntimeSpan(&s.auxLiterals), false)
	return e
}
func (s *satSolver) resetAssumptions() error {
	if e := s.conflict.assumptions.shrink(0); e != nil {
		return e
	}
	return s.conflict.assumptionSet.Reset()
}
func (s *satSolver) popToBase() error {
	if e := s.resetAssumptions(); e != nil {
		return e
	}
	return s.lifecycle.pop(s.variables.scopeLevel)
}
func (s *satSolver) initAssumptions(lits []satLiteral) error {
	return s.initAssumptionsSpan(uint32(len(lits)), satSolverLiteralSpan{literals: lits})
}
func (s *satSolver) initAssumptionsSpan(count uint32, span satSolverLiteralSpan) error {
	n, e := s.conflict.userScopeLiterals.size()
	if e != nil {
		return e
	}
	if count == 0 && n == 0 {
		return nil
	}
	if e = s.resetAssumptions(); e != nil {
		return e
	}
	if e = s.lifecycle.push(); e != nil {
		return e
	}
	if _, e = s.lifecycle.propagate(false, false, false); e != nil {
		return e
	}
	if s.propagation.inconsistent {
		return nil
	}
	for i := uint32(0); !s.propagation.inconsistent; i++ {
		n, e := s.conflict.userScopeLiterals.size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		l, e := s.conflict.userScopeLiterals.read(i)
		if e != nil {
			return e
		}
		if e = s.propagation.assign(satLiteral(l).negated(), satJustification{}); e != nil {
			return e
		}
	}
	for i := uint32(0); i < count && !s.propagation.inconsistent; i++ {
		l, e := span.literal(i)
		if e != nil {
			return e
		}
		if satStorageSourceDebug {
			yes, e := s.variables.isExternal(l.variable())
			if e != nil {
				return e
			}
			if !yes {
				return ErrSATAssertion
			}
		}
		if e = s.conflict.assumptionSet.Insert(l); e != nil {
			return e
		}
		if e = s.conflict.assumptions.push(uint32(l)); e != nil {
			return e
		}
		if e = s.propagation.assign(l, satJustification{}); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolver) checkInconsistent() (bool, error) {
	if !s.propagation.inconsistent {
		return false, nil
	}
	tracking, e := s.conflict.trackingAssumptions()
	if e != nil {
		return false, e
	}
	if tracking {
		if _, e = s.conflict.resolveConflict(); e != nil {
			return false, e
		}
	}
	return true, nil
}
func (s *satSolver) simplifyProblem() error {
	if s.conflict.conflicts < s.nextSimplify {
		return nil
	}
	if e := s.lifecycle.pop(s.variables.scopeLevel); e != nil {
		return e
	}
	if satStorageSourceDebug && s.variables.scopeLevel != 0 {
		return ErrSATAssertion
	}
	if _, e := s.cleaner.run(false); e != nil {
		return e
	}
	if _, e := s.scc.run(); e != nil {
		return e
	}
	if e := s.simplifier.Run(false); e != nil {
		return e
	}
	if len(s.lifecycle.learned) != 0 {
		if e := s.simplifier.Run(true); e != nil {
			return e
		}
	}
	for _, r := range s.variables.watches.lists {
		if e := s.variables.watches.store.SortByKind(r); e != nil {
			return e
		}
	}
	if _, e := s.probing.run(false); e != nil {
		return e
	}
	if e := s.asymm.run(false); e != nil {
		return e
	}
	if s.propagation.extensionPresent {
		return ErrSATMissingSourceExtBinding
	}
	if e := s.conflict.reinitAssumptions(); e != nil {
		return e
	}
	cfg, e := s.config.State()
	if e != nil {
		return e
	}
	if s.nextSimplify == 0 {
		s.nextSimplify = cfg.RestartInitial * cfg.SimplifyMult1
	} else {
		n, e := satGCUnsigned(float64(s.conflict.conflicts) * cfg.SimplifyMult2)
		if e != nil {
			return e
		}
		s.nextSimplify = n
		if s.nextSimplify > s.conflict.conflicts+cfg.SimplifyMax {
			s.nextSimplify = s.conflict.conflicts + cfg.SimplifyMax
		}
	}
	return nil
}
func (s *satSolver) backjumpStep() (lbool, bool, error) {
	done := true
	if _, e := s.lifecycle.propagate(true, false, false); e != nil {
		return lUndef, done, e
	}
	if !s.propagation.inconsistent {
		return lTrue, done, nil
	}
	yes, e := s.conflict.resolveConflict()
	if e != nil {
		return lUndef, done, e
	}
	if !yes {
		return lFalse, done, nil
	}
	cfg, e := s.config.State()
	if e != nil {
		return lUndef, done, e
	}
	if s.conflict.conflicts > cfg.MaxConflicts || s.conflict.conflictsSinceRestart > s.gc.restartThreshold {
		return lUndef, done, nil
	}
	if s.variables.scopeLevel == 0 {
		if e = s.cleaner.cleanup(); e != nil {
			return lUndef, done, e
		}
		if s.propagation.inconsistent {
			return lFalse, done, nil
		}
		if e = s.gc.gc(); e != nil {
			return lUndef, done, e
		}
	}
	return lTrue, false, nil
}
func (s *satSolver) boundedSearch() (lbool, error) {
	for {
		if e := s.propagation.checkpoint(); e != nil {
			return lUndef, e
		}
		done := false
		for !done {
			r, d, e := s.backjumpStep()
			if e != nil {
				return lUndef, e
			}
			if r != lTrue {
				return r, nil
			}
			done = d
		}
		if e := s.gc.gc(); e != nil {
			return lUndef, e
		}
		yes, e := s.decision.decide()
		if e != nil {
			return lUndef, e
		}
		if !yes {
			if s.propagation.extensionPresent {
				return lUndef, ErrSATMissingSourceExtBinding
			}
			if e = s.mkModel(); e != nil {
				return lUndef, e
			}
			return lTrue, nil
		}
	}
}
func (s *satSolver) Check(lits []satLiteral) (lbool, error) {
	return s.checkSpan(uint32(len(lits)), satSolverLiteralSpan{literals: lits})
}
func (s *satSolver) checkSpan(count uint32, span satSolverLiteralSpan) (lbool, error) {
	if e := s.open(); e != nil {
		return lUndef, e
	}
	if e := s.popToBase(); e != nil {
		return lUndef, e
	}
	if satStorageSourceDebug && s.variables.scopeLevel != 0 {
		return lUndef, ErrSATAssertion
	}
	cfg, e := s.config.State()
	if e != nil {
		return lUndef, e
	}
	if cfg.DimacsDisplay {
		if s.dimacsOutput == nil {
			return lUndef, ErrSATMissingDimacsBinding
		}
		if e = s.displayDIMACS(s.dimacsOutput); e != nil {
			return lUndef, e
		}
		for i := uint32(0); i < count; i++ {
			l, err := span.literal(i)
			if err != nil {
				return lUndef, err
			}
			if err = satDIMACSLiteralInto(s.dimacsOutput, l); err != nil {
				return lUndef, err
			}
			if err = s.dimacsOutput.AppendCString(" 0\n"); err != nil {
				return lUndef, err
			}
		}
		return lUndef, nil
	}
	if cfg.NumParallel > 1 && !s.conflict.parallelPresent {
		return lUndef, ErrSATMissingParallelBinding
	}
	r, e := s.checkCoreSpan(count, span)
	var abort satSolverAbort
	if errors.As(e, &abort) {
		return lUndef, nil
	}
	return r, e
}
func (s *satSolver) checkCore(lits []satLiteral) (lbool, error) {
	return s.checkCoreSpan(uint32(len(lits)), satSolverLiteralSpan{literals: lits})
}
func (s *satSolver) checkCoreSpan(count uint32, span satSolverLiteralSpan) (lbool, error) {
	if s.propagation.inconsistent {
		return lFalse, nil
	}
	if e := s.gc.initSearch(); e != nil {
		return lUndef, e
	}
	if _, e := s.lifecycle.propagate(false, false, false); e != nil {
		return lUndef, e
	}
	if s.propagation.inconsistent {
		return lFalse, nil
	}
	if e := s.initAssumptionsSpan(count, span); e != nil {
		return lUndef, e
	}
	if _, e := s.lifecycle.propagate(false, false, false); e != nil {
		return lUndef, e
	}
	yes, e := s.checkInconsistent()
	if e != nil {
		return lUndef, e
	}
	if yes {
		return lFalse, nil
	}
	if e = s.cleaner.cleanup(); e != nil {
		return lUndef, e
	}
	cfg, e := s.config.State()
	if e != nil {
		return lUndef, e
	}
	if cfg.MaxConflicts > 0 && cfg.BurstSearch > 0 {
		s.gc.restartThreshold = cfg.BurstSearch
		r, e := s.boundedSearch()
		if e != nil {
			return lUndef, e
		}
		if r != lUndef {
			return r, nil
		}
		if e = s.conflict.popReinit(s.variables.scopeLevel); e != nil {
			return lUndef, e
		}
		s.conflict.conflictsSinceRestart = 0
		s.gc.restartThreshold = cfg.RestartInitial
	}
	if e = s.simplifyProblem(); e != nil {
		return lUndef, e
	}
	yes, e = s.checkInconsistent()
	if e != nil {
		return lUndef, e
	}
	if yes {
		return lFalse, nil
	}
	if cfg.MaxConflicts == 0 {
		return lUndef, nil
	}
	for {
		if satStorageSourceDebug && s.propagation.inconsistent {
			return lUndef, ErrSATAssertion
		}
		r, e := s.boundedSearch()
		if e != nil {
			return lUndef, e
		}
		if r != lUndef {
			return r, nil
		}
		if s.conflict.conflicts > cfg.MaxConflicts {
			return lUndef, nil
		}
		if e = s.gc.restart(); e != nil {
			return lUndef, e
		}
		if e = s.simplifyProblem(); e != nil {
			return lUndef, e
		}
		yes, e = s.checkInconsistent()
		if e != nil {
			return lUndef, e
		}
		if yes {
			return lFalse, nil
		}
		if e = s.gc.gc(); e != nil {
			return lUndef, e
		}
		if cfg.RestartMax <= s.gc.restarts {
			return lUndef, nil
		}
	}
}
