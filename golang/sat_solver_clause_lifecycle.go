// Source: src/sat/sat_solver.cpp and .h. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Sole clause ownership lies in original/learned or the explicit interrupted
// raw-candidate root. Reinit/watch states are actual nonowning source borrowers.
// This owner is not a complete solver constructor or source debug destructor.
type satSolverClauseLifecycle struct {
	self                                                       *satSolverClauseLifecycle
	propagation                                                *satSolverPropagationState
	budget                                                     *arena.Budget
	original, learned                                          []satStoredClauseRef
	reinit                                                     []satClauseWrapper
	scopes                                                     []satSolverScope
	pendingCandidate                                           satStoredClauseRef
	cleanerContinuation                                        *satCleaner
	gcContinuation                                             *satSolverGC
	eqContinuation                                             *satElimEqs
	asymmContinuation                                          *satAsymmBranch
	ownerBytes                                                 uint64
	mkBinary, mkTernary, mkNary, deleted, nonLearnedGeneration uint32
	reinitInitialized, scopesInitialized                       uint32
	closeOriginal, closeLearned                                uint32
	originalDone, learnedDone, closing, closed                 bool
}
type satSolverScope struct {
	trailLimit, reinitLimit uint32
	inconsistent            bool
}

func newSATSolverClauseLifecycle(p *satSolverPropagationState, b *arena.Budget) (*satSolverClauseLifecycle, error) {
	if e := p.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(satSolverClauseLifecycle{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &satSolverClauseLifecycle{propagation: p, budget: b, ownerBytes: n}
	s.self = s
	return s, nil
}
func (s *satSolverClauseLifecycle) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	if s.cleanerContinuation != nil && !s.cleanerContinuation.running {
		return arena.ErrClosed
	}
	if s.gcContinuation != nil && !s.gcContinuation.running {
		return arena.ErrClosed
	}
	if s.eqContinuation != nil && !s.eqContinuation.running {
		return arena.ErrClosed
	}
	if s.asymmContinuation != nil && !s.asymmContinuation.running {
		return arena.ErrClosed
	}
	return s.propagation.open()
}
func (s *satSolverClauseLifecycle) pushClause(learned bool, c satStoredClauseRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if learned {
		if len(s.learned) == cap(s.learned) {
			n, bytes, e := satVariableNextCapacity(uint32(cap(s.learned)), uint64(reflect.TypeOf(satStoredClauseRef{}).Size()), 8, uint64(^uint(0)>>1))
			if e != nil {
				return e
			}
			if e = s.budget.Reserve(bytes); e != nil {
				return e
			}
			next := make([]satStoredClauseRef, len(s.learned), n)
			copy(next[:cap(s.learned)], s.learned[:cap(s.learned)])
			s.budget.Release(uint64(cap(s.learned)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size()))
			s.learned = next
		}
		s.learned = append(s.learned, c)
	} else {
		if len(s.original) == cap(s.original) {
			n, bytes, e := satVariableNextCapacity(uint32(cap(s.original)), uint64(reflect.TypeOf(satStoredClauseRef{}).Size()), 8, uint64(^uint(0)>>1))
			if e != nil {
				return e
			}
			if e = s.budget.Reserve(bytes); e != nil {
				return e
			}
			next := make([]satStoredClauseRef, len(s.original), n)
			copy(next[:cap(s.original)], s.original[:cap(s.original)])
			s.budget.Release(uint64(cap(s.original)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size()))
			s.original = next
		}
		s.original = append(s.original, c)
	}
	return nil
}
func (s *satSolverClauseLifecycle) pushWrapper(w satClauseWrapper) error {
	if len(s.reinit) == cap(s.reinit) {
		n, bytes, e := satVariableNextCapacity(uint32(cap(s.reinit)), uint64(reflect.TypeOf(satClauseWrapper{}).Size()), 16, uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = s.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satClauseWrapper, len(s.reinit), n)
		copy(next[:cap(s.reinit)], s.reinit[:cap(s.reinit)])
		s.budget.Release(uint64(cap(s.reinit)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size()))
		s.reinit = next
	}
	s.reinit = append(s.reinit, w)
	if uint32(len(s.reinit)) > s.reinitInitialized {
		s.reinitInitialized = uint32(len(s.reinit))
	}
	return nil
}

// publishCandidate is an explicit Go continuation, not a source constructor retry.
// It publishes the same retained allocated object without replaying attachment.
func (s *satSolverClauseLifecycle) publishCandidate(learned bool) error {
	if e := s.open(); e != nil {
		return e
	}
	c := s.pendingCandidate
	if c == (satStoredClauseRef{}) {
		return arena.ErrInvalidSlot
	}
	if e := s.propagation.clauses.validate(c); e != nil {
		return e
	}
	if e := s.pushClause(learned, c); e != nil {
		return e
	}
	s.pendingCandidate = satStoredClauseRef{}
	return nil
}
func (s *satSolverClauseLifecycle) delClause(c satStoredClauseRef) error {
	if e := s.open(); e != nil {
		return e
	}
	a := s.propagation.clauses
	learned, e := a.Learned(c)
	if e != nil {
		return e
	}
	if !learned {
		s.nonLearnedGeneration++
	}
	if e = a.Delete(c); e != nil {
		return e
	}
	s.deleted++
	return nil
}

// Explicit interrupted-candidate teardown consumes ONLY its retained ownership.
// Source-completed watches/reinit remain, and their real stale accesses are guarded.
func (s *satSolverClauseLifecycle) cleanupCandidate() error {
	if e := s.open(); e != nil {
		return e
	}
	if s.pendingCandidate == (satStoredClauseRef{}) {
		return nil
	}
	if e := s.propagation.clauses.Delete(s.pendingCandidate); e != nil {
		return e
	}
	s.pendingCandidate = satStoredClauseRef{}
	return nil
}
func (s *satSolverClauseLifecycle) mkClauseCore(n uint32, lits []satLiteral, learned bool) (satStoredClauseRef, error) {
	return s.mkClauseCoreSpan(n, satSolverLiteralSpan{literals: lits}, learned)
}
func (s *satSolverClauseLifecycle) mkClauseCoreSpan(n uint32, span satSolverLiteralSpan, learned bool) (satStoredClauseRef, error) {
	if e := s.open(); e != nil {
		return satStoredClauseRef{}, e
	}
	if s.pendingCandidate != (satStoredClauseRef{}) {
		return satStoredClauseRef{}, ErrSATOwner
	}
	p := s.propagation
	if !learned {
		if span.runtime != nil {
			return satStoredClauseRef{}, ErrSATOwner
		} // private Runtime bridge is the actual learned-lemma consumer
		lits := span.literals
		v := p.variables.clauseState()
		keep, e := v.simplifyClause(p.variables.scopeLevel, &n, lits)
		if e != nil {
			return satStoredClauseRef{}, e
		}
		if !keep {
			return satStoredClauseRef{}, nil
		}
		s.nonLearnedGeneration++
	}
	switch n {
	case 0:
		p.setConflict(satJustification{}, satDefaultLiteral())
		return satStoredClauseRef{}, nil
	case 1:
		if span.runtime == nil && len(span.literals) == 0 {
			return satStoredClauseRef{}, arena.ErrBounds
		}
		lit, e := span.literal(0)
		if e != nil {
			return satStoredClauseRef{}, e
		}
		return satStoredClauseRef{}, p.assign(lit, satJustification{})
	case 2:
		if span.runtime == nil && len(span.literals) < 2 {
			return satStoredClauseRef{}, arena.ErrBounds
		}
		a, e := span.literal(0)
		if e != nil {
			return satStoredClauseRef{}, e
		}
		b, e := span.literal(1)
		if e != nil {
			return satStoredClauseRef{}, e
		}
		return satStoredClauseRef{}, s.mkBinClause(a, b, learned)
	case 3:
		return s.mkTerClauseSpan(span, learned)
	default:
		return s.mkNaryClauseSpan(n, span, learned)
	}
}
func (s *satSolverClauseLifecycle) mkTerClause(lits []satLiteral, learned bool) (satStoredClauseRef, error) {
	return s.mkTerClauseSpan(satSolverLiteralSpan{literals: lits}, learned)
}
func (s *satSolverClauseLifecycle) mkTerClauseSpan(span satSolverLiteralSpan, learned bool) (satStoredClauseRef, error) {
	s.mkTernary++
	c, e := s.propagation.clauses.mkClauseSpan(3, span, learned)
	// Retain the actual allocated candidate even if a guarded constructor read
	// fails. This is owning error continuation, not source exception rollback.
	s.pendingCandidate = c
	if e != nil {
		return c, e
	}
	reinit, e := s.attachTerClause(c)
	if e != nil {
		return c, e
	}
	if reinit && !learned {
		if e = s.pushReinitStack(c); e != nil {
			return c, e
		}
	}
	return c, s.publishCandidate(learned)
}
func (s *satSolverClauseLifecycle) mkNaryClause(n uint32, lits []satLiteral, learned bool) (satStoredClauseRef, error) {
	return s.mkNaryClauseSpan(n, satSolverLiteralSpan{literals: lits}, learned)
}
func (s *satSolverClauseLifecycle) mkNaryClauseSpan(n uint32, span satSolverLiteralSpan, learned bool) (satStoredClauseRef, error) {
	s.mkNary++
	c, e := s.propagation.clauses.mkClauseSpan(n, span, learned)
	// Retain the actual allocated candidate even if a guarded constructor read
	// fails. This is owning error continuation, not source exception rollback.
	s.pendingCandidate = c
	if e != nil {
		return c, e
	}
	if satStorageSourceDebug && learned {
		flag, e := s.propagation.clauses.Learned(c)
		if e != nil {
			return c, e
		}
		if !flag {
			return c, ErrSATAssertion
		}
	}
	reinit, e := s.attachNaryClause(c)
	if e != nil {
		return c, e
	}
	if reinit && !learned {
		if e = s.pushReinitStack(c); e != nil {
			return c, e
		}
	}
	return c, s.publishCandidate(learned)
}
func (s *satSolverClauseLifecycle) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.closing = true
	if s.asymmContinuation != nil {
		if e := s.asymmContinuation.cleanupInterrupted(); e != nil {
			return e
		}
	}
	if s.eqContinuation != nil {
		if e := s.eqContinuation.cleanupInterrupted(); e != nil {
			return e
		}
	}
	if s.cleanerContinuation != nil {
		if e := s.cleanerContinuation.cleanupInterrupted(); e != nil {
			return e
		}
	}
	if s.gcContinuation != nil {
		if e := s.gcContinuation.cleanupInterrupted(); e != nil {
			return e
		}
	}
	a := s.propagation.clauses
	for s.closeOriginal < uint32(len(s.original)) {
		if e := a.Delete(s.original[s.closeOriginal]); e != nil {
			return e
		}
		s.closeOriginal++
	}
	if !s.originalDone {
		s.nonLearnedGeneration++
		s.originalDone = true
	}
	for s.closeLearned < uint32(len(s.learned)) {
		if e := a.Delete(s.learned[s.closeLearned]); e != nil {
			return e
		}
		s.closeLearned++
	}
	if !s.learnedDone {
		s.nonLearnedGeneration++
		s.learnedDone = true
	}
	if s.pendingCandidate != (satStoredClauseRef{}) {
		if e := a.Delete(s.pendingCandidate); e != nil {
			return e
		}
		s.pendingCandidate = satStoredClauseRef{}
	}
	// Adapter physical finalization after source clause deletion, reverse member order.
	s.budget.Release(uint64(cap(s.scopes)) * uint64(reflect.TypeOf(satSolverScope{}).Size()))
	s.scopes = nil
	s.budget.Release(uint64(cap(s.reinit)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size()))
	s.reinit = nil
	s.budget.Release(uint64(cap(s.learned)+cap(s.original)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size()))
	s.learned = nil
	s.original = nil
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}
