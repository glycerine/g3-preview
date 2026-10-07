package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func (s *satSolverClauseLifecycle) push() error {
	if e := s.open(); e != nil {
		return e
	}
	p := s.propagation
	v := p.variables
	if satStorageSourceDebug {
		if p.inconsistent {
			return ErrSATAssertion
		}
		n, e := v.trail.Size(v.trailRef)
		if e != nil {
			return e
		}
		if v.qhead != n {
			return ErrSATAssertion
		}
	}
	if len(s.scopes) == cap(s.scopes) {
		next, bytes, e := satVariableNextCapacity(uint32(cap(s.scopes)), uint64(reflect.TypeOf(satSolverScope{}).Size()), 12, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = s.budget.Reserve(bytes); e != nil {
			return e
		}
		buf := make([]satSolverScope, len(s.scopes), next)
		copy(buf[:cap(s.scopes)], s.scopes[:cap(s.scopes)])
		s.budget.Release(uint64(cap(s.scopes)) * uint64(reflect.TypeOf(satSolverScope{}).Size()))
		s.scopes = buf
	}
	s.scopes = append(s.scopes, satSolverScope{})
	if uint32(len(s.scopes)) > s.scopesInitialized {
		s.scopesInitialized = uint32(len(s.scopes))
	}
	v.scopeLevel++
	n, e := v.trail.Size(v.trailRef)
	if e != nil {
		return e
	}
	s.scopes[len(s.scopes)-1].trailLimit = n
	s.scopes[len(s.scopes)-1].reinitLimit = uint32(len(s.reinit))
	s.scopes[len(s.scopes)-1].inconsistent = p.inconsistent
	if p.extensionPresent {
		return ErrSATMissingSourceExtBinding
	}
	return nil
}
func (s *satSolverClauseLifecycle) pop(n uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	p := s.propagation
	v := p.variables
	if p.extensionPresent {
		return ErrSATMissingSourceExtBinding
	}
	if satStorageSourceDebug && n > v.scopeLevel {
		return ErrSATAssertion
	}
	newLevel := v.scopeLevel - n
	if newLevel >= s.scopesInitialized || uint64(newLevel) >= uint64(cap(s.scopes)) {
		return arena.ErrBounds
	}
	scope := s.scopes[:cap(s.scopes)][newLevel]
	p.inconsistent = false
	if e := s.unassignVars(scope.trailLimit); e != nil {
		return e
	}
	v.scopeLevel -= n
	s.scopes = s.scopes[:newLevel]
	return s.reinitClauses(scope.reinitLimit)
}
func (s *satSolverClauseLifecycle) unassignVars(old uint32) error {
	v := s.propagation.variables
	n, e := v.trail.Size(v.trailRef)
	if e != nil {
		return e
	}
	if satStorageSourceDebug && old > n {
		return ErrSATAssertion
	}
	for n != old {
		n--
		l, e := v.trailRead(n)
		if e != nil {
			return e
		}
		if e = v.assignment.Write(l.index(), int8(util.LUndef)); e != nil {
			return e
		}
		if e = v.assignment.Write(l.Negated().index(), int8(util.LUndef)); e != nil {
			return e
		}
		if satStorageSourceDebug {
			value, e := v.valueVar(l.variable())
			if e != nil {
				return e
			}
			if value != util.LUndef {
				return ErrSATAssertion
			}
		}
		if e = v.queue.UnassignVar(l.variable()); e != nil {
			return e
		}
	}
	if e = v.trail.Shrink(v.trailRef, old); e != nil {
		return e
	}
	v.qhead = old
	if satStorageSourceDebug {
		n, e := v.trail.Size(v.trailRef)
		if e != nil {
			return e
		}
		if v.qhead != n {
			return ErrSATAssertion
		}
	}
	return nil
}
func (s *satSolverClauseLifecycle) reinitClauses(old uint32) error {
	n := uint32(len(s.reinit))
	if satStorageSourceDebug && old > n {
		return ErrSATAssertion
	}
	out := old
	for i := old; i < n; i++ {
		w := s.reinit[i]
		reinit := false
		if w.Binary() {
			prop, e := s.propagateBinClause(w.first, w.second)
			if e != nil {
				return e
			}
			if prop && s.propagation.variables.scopeLevel > 0 {
				s.reinit[out] = w
				out++
			}
		} else {
			if e := s.detachClause(w.ref); e != nil {
				return e
			}
			flag, e := s.attach(w.ref)
			if e != nil {
				return e
			}
			reinit = flag
			if s.propagation.variables.scopeLevel > 0 && reinit {
				s.reinit[out] = w
				out++
			} else {
				if e = s.propagation.clauses.SetReinitStack(w.ref, false); e != nil {
					return e
				}
			}
		}
	}
	if out > s.reinitInitialized || uint64(out) > uint64(cap(s.reinit)) {
		return arena.ErrBounds
	}
	s.reinit = s.reinit[:out]
	return nil
}
