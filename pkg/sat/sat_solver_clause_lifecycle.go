// Source: src/sat/sat_solver.cpp and .h.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Sole clause ownership lies in original/learned or the explicit interrupted
// raw-candidate root. Reinit/watch states are actual nonowning source borrowers.
// This owner is not a complete solver constructor or source debug destructor.
type satSolverClauseLifecycle struct {
	userContinuation                                           *Solver
	integrityGC                                                *satSolverGC
	simplifierContinuation                                     *satSimplifier
	asymmContinuation                                          *satAsymmBranch
	self                                                       *satSolverClauseLifecycle
	propagation                                                *satSolverPropagationState
	budget                                                     *arena.Budget
	Original, learned                                          []StoredClauseRef
	reinit                                                     []satClauseWrapper
	scopes                                                     []satSolverScope
	pendingCandidate                                           StoredClauseRef
	pendingDeclaredSize                                        uint32
	pendingUnconstructed                                       bool
	cleanerContinuation                                        *satCleaner
	gcContinuation                                             *satSolverGC
	eqContinuation                                             *satElimEqs
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
	if e := p.constructionBorrow(); e != nil {
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
func (s *satSolverClauseLifecycle) constructionBorrow() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	if s.userContinuation != nil && !s.userContinuation.userGC.running {
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
	if s.simplifierContinuation != nil && !s.simplifierContinuation.running {
		return arena.ErrClosed
	}
	return s.propagation.constructionBorrow()
}
func (s *satSolverClauseLifecycle) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	if s.userContinuation != nil && !s.userContinuation.userGC.running {
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
	if s.simplifierContinuation != nil && !s.simplifierContinuation.running {
		return arena.ErrClosed
	}
	return s.propagation.open()
}
func (s *satSolverClauseLifecycle) pushClause(learned bool, c StoredClauseRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if learned {
		if len(s.learned) == cap(s.learned) {
			n, bytes, e := satVariableNextCapacity(uint32(cap(s.learned)), uint64(reflect.TypeOf(StoredClauseRef{}).Size()), 8, uint64(^uint(0)>>1))
			if e != nil {
				return e
			}
			if e = s.budget.Reserve(bytes); e != nil {
				return e
			}
			next := make([]StoredClauseRef, len(s.learned), n)
			copy(next[:cap(s.learned)], s.learned[:cap(s.learned)])
			s.budget.Release(uint64(cap(s.learned)) * uint64(reflect.TypeOf(StoredClauseRef{}).Size()))
			s.learned = next
		}
		s.learned = append(s.learned, c)
	} else {
		if len(s.Original) == cap(s.Original) {
			n, bytes, e := satVariableNextCapacity(uint32(cap(s.Original)), uint64(reflect.TypeOf(StoredClauseRef{}).Size()), 8, uint64(^uint(0)>>1))
			if e != nil {
				return e
			}
			if e = s.budget.Reserve(bytes); e != nil {
				return e
			}
			next := make([]StoredClauseRef, len(s.Original), n)
			copy(next[:cap(s.Original)], s.Original[:cap(s.Original)])
			s.budget.Release(uint64(cap(s.Original)) * uint64(reflect.TypeOf(StoredClauseRef{}).Size()))
			s.Original = next
		}
		s.Original = append(s.Original, c)
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
	if s.pendingUnconstructed {
		return ErrSATIncompleteGrowth
	}
	c := s.pendingCandidate
	if c == (StoredClauseRef{}) {
		return arena.ErrInvalidSlot
	}
	if e := s.propagation.clauses.validate(c); e != nil {
		return e
	}
	if e := s.pushClause(learned, c); e != nil {
		return e
	}
	s.pendingCandidate = StoredClauseRef{}
	s.pendingDeclaredSize = 0
	s.pendingUnconstructed = false
	return nil
}
func (s *satSolverClauseLifecycle) delClause(c StoredClauseRef) error {
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
	if s.pendingCandidate == (StoredClauseRef{}) {
		return nil
	}
	if e := s.releasePendingCandidate(); e != nil {
		return e
	}
	s.pendingCandidate = StoredClauseRef{}
	s.pendingDeclaredSize = 0
	s.pendingUnconstructed = false
	return nil
}
func (s *satSolverClauseLifecycle) MkClauseCore(n uint32, lits []Literal, learned bool) (StoredClauseRef, error) {
	return s.mkClauseCoreSpan(n, satSolverLiteralSpan{literals: lits}, learned)
}
func (s *satSolverClauseLifecycle) mkClauseCoreSpan(n uint32, span satSolverLiteralSpan, learned bool) (StoredClauseRef, error) {
	if e := s.open(); e != nil {
		return StoredClauseRef{}, e
	}
	if s.pendingCandidate != (StoredClauseRef{}) {
		return StoredClauseRef{}, ErrSATOwner
	}
	p := s.propagation
	if !learned {
		v := p.variables.clauseState()
		keep, e := v.simplifyClauseSpan(p.variables.scopeLevel, &n, span)
		if e != nil {
			return StoredClauseRef{}, e
		}
		if !keep {
			return StoredClauseRef{}, nil
		}
		s.nonLearnedGeneration++
	}
	switch n {
	case 0:
		p.setConflict(satJustification{}, DefaultLiteral())
		return StoredClauseRef{}, nil
	case 1:
		if span.column == nil && span.store == nil && len(span.literals) == 0 {
			return StoredClauseRef{}, arena.ErrBounds
		}
		lit, e := span.literal(0)
		if e != nil {
			return StoredClauseRef{}, e
		}
		return StoredClauseRef{}, p.assign(lit, satJustification{})
	case 2:
		if span.column == nil && span.store == nil && len(span.literals) < 2 {
			return StoredClauseRef{}, arena.ErrBounds
		}
		a, e := span.literal(0)
		if e != nil {
			return StoredClauseRef{}, e
		}
		b, e := span.literal(1)
		if e != nil {
			return StoredClauseRef{}, e
		}
		return StoredClauseRef{}, s.mkBinClause(a, b, learned)
	case 3:
		return s.mkTerClauseSpan(span, learned)
	default:
		return s.mkNaryClauseSpan(n, span, learned)
	}
}
func (s *satSolverClauseLifecycle) mkTerClause(lits []Literal, learned bool) (StoredClauseRef, error) {
	return s.mkTerClauseSpan(satSolverLiteralSpan{literals: lits}, learned)
}
func (s *satSolverClauseLifecycle) mkTerClauseSpan(span satSolverLiteralSpan, learned bool) (StoredClauseRef, error) {
	s.mkTernary++
	var prefix satClauseConstructionPrefix
	c, e := s.propagation.clauses.mkClauseSpanPrefix(3, span, learned, &prefix)
	s.pendingDeclaredSize = 3
	s.pendingUnconstructed = prefix.unconstructed
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
func (s *satSolverClauseLifecycle) mkNaryClause(n uint32, lits []Literal, learned bool) (StoredClauseRef, error) {
	return s.mkNaryClauseSpan(n, satSolverLiteralSpan{literals: lits}, learned)
}
func (s *satSolverClauseLifecycle) mkNaryClauseSpan(n uint32, span satSolverLiteralSpan, learned bool) (StoredClauseRef, error) {
	s.mkNary++
	var prefix satClauseConstructionPrefix
	c, e := s.propagation.clauses.mkClauseSpanPrefix(n, span, learned, &prefix)
	s.pendingDeclaredSize = n
	s.pendingUnconstructed = prefix.unconstructed
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
	if e := s.cleanupOwningContinuations(); e != nil {
		return e
	}
	if e := s.deleteClausesBody(); e != nil {
		return e
	}
	if e := s.finalizeScopesReinit(); e != nil {
		return e
	}
	if e := s.finalizeClauseVectors(); e != nil {
		return e
	}
	return s.closeContext()
}

// Cold Go context acknowledgment after all actual members and (for a completed
// solver wrapper only) its source destructor body have already been drained.
// This is not a second invocation of solver::~solver's body.
func (s *satSolverClauseLifecycle) closeContext() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// Full facade destruction uses the declared-member stages independently.
// The standalone Close adapter below retains its existing combined order.
func (s *satSolverClauseLifecycle) cleanupOwningContinuations() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.userContinuation != nil {
		if e := s.userContinuation.cleanupUserGC(); e != nil {
			return e
		}
	}
	if s.simplifierContinuation != nil {
		if e := s.simplifierContinuation.cleanupStopped(); e != nil {
			return e
		}
	}
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
	return nil
}
func (s *satSolverClauseLifecycle) deleteClausesBody() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	a := s.propagation.clauses
	for s.closeOriginal < uint32(len(s.Original)) {
		if e := a.Delete(s.Original[s.closeOriginal]); e != nil {
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
	if s.pendingCandidate != (StoredClauseRef{}) {
		if e := s.releasePendingCandidate(); e != nil {
			return e
		}
		s.pendingCandidate = StoredClauseRef{}
		s.pendingDeclaredSize = 0
		s.pendingUnconstructed = false
	}
	return nil
}
func (s *satSolverClauseLifecycle) finalizeScopesReinit() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	// Adapter physical finalization after source clause deletion, reverse member order.
	s.budget.Release(uint64(cap(s.scopes)) * uint64(reflect.TypeOf(satSolverScope{}).Size()))
	s.scopes = nil
	s.budget.Release(uint64(cap(s.reinit)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size()))
	s.reinit = nil
	return nil
}
func (s *satSolverClauseLifecycle) finalizeClauseVectors() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	s.budget.Release(uint64(cap(s.learned)+cap(s.Original)) * uint64(reflect.TypeOf(StoredClauseRef{}).Size()))
	s.learned = nil
	s.Original = nil
	return nil
}

func (s *satSolverClauseLifecycle) releasePendingCandidate() error {
	if s.pendingUnconstructed {
		return s.propagation.clauses.cleanupAllocated(s.pendingCandidate, s.pendingDeclaredSize)
	}
	return s.propagation.clauses.Delete(s.pendingCandidate)
}
