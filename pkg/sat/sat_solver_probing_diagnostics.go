// Source: src/sat/sat_probing.cpp; src/sat/sat_probing.h.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

// Actual source cpp3022-3074 conditional check producers, backed by real clause
// owners and assignment columns. They do not replace integrity_checker.

func (s *satSolverClauseLifecycle) clauseUnit(c StoredClauseRef) (bool, error) {
	p := s.propagation
	n, e := p.clauses.Size(c)
	if e != nil {
		return false, e
	}
	found := false
	for i := uint32(0); i < n; i++ {
		l, e := p.clauses.Literal(c, i)
		if e != nil {
			return false, e
		}
		v, e := p.variables.valueLiteral(l)
		if e != nil {
			return false, e
		}
		switch v {
		case util.LUndef:
			if found {
				return false, nil
			}
			found = true
		case util.LTrue:
			return false, nil
		case util.LFalse:
		}
	}
	return found, nil
}
func (s *satSolverClauseLifecycle) clauseEmpty(c StoredClauseRef) (bool, error) {
	p := s.propagation
	n, e := p.clauses.Size(c)
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		l, e := p.clauses.Literal(c, i)
		if e != nil {
			return false, e
		}
		v, e := p.variables.valueLiteral(l)
		if e != nil {
			return false, e
		}
		if v != util.LFalse {
			return false, nil
		}
	}
	return true, nil
}
func (s *satSolverClauseLifecycle) checkMissedClauseVector(cs []StoredClauseRef) error {
	// Source captures begin/end; concrete source calls have no owner mutation.
	for i := 0; i < len(cs); i++ {
		c := cs[i]
		frozen, e := s.propagation.clauses.Frozen(c)
		if e != nil {
			return e
		}
		if frozen {
			continue
		}
		empty, e := s.clauseEmpty(c)
		if e != nil {
			return e
		}
		unit := false
		if !empty {
			unit, e = s.clauseUnit(c)
			if e != nil {
				return e
			}
		}
		if empty || unit {
			return ErrSATAssertion
		} // source UNREACHABLE at actual dispatch.
		if satStorageSourceDebug {
			empty, e = s.clauseEmpty(c)
			if e != nil {
				return e
			}
			if empty {
				return ErrSATAssertion
			}
			unit, e = s.clauseUnit(c)
			if e != nil {
				return e
			}
			if unit {
				return ErrSATAssertion
			}
		}
	}
	return nil
}
func (s *satSolverClauseLifecycle) checkMissedPropagation() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.propagation.inconsistent {
		return nil
	}
	if e := s.checkMissedClauseVector(s.Original); e != nil {
		return e
	}
	return s.checkMissedClauseVector(s.learned)
}

func (s *satSolverProbing) clauseUnit(c StoredClauseRef) (bool, error) {
	return s.lifecycle.clauseUnit(c)
}
func (s *satSolverProbing) clauseEmpty(c StoredClauseRef) (bool, error) {
	return s.lifecycle.clauseEmpty(c)
}
func (s *satSolverProbing) checkMissedClauseVector(cs []StoredClauseRef) error {
	return s.lifecycle.checkMissedClauseVector(cs)
}
func (s *satSolverProbing) checkMissedPropagation() error {
	return s.lifecycle.checkMissedPropagation()
}
