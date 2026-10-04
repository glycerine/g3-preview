// Source: src/sat/sat_probing.cpp; src/sat/sat_probing.h. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrSATMissingIntegrityBinding = errors.New("g3: missing source SAT integrity-checker binding")

// This is the actual probing member family, not solver construction/search.
// Borrowed lifecycle, propagation, parameters and Runtime must outlive it.
type satSolverProbing struct {
	self                                      *satSolverProbing
	lifecycle                                 *satSolverClauseLifecycle
	budget                                    *arena.Budget
	assigned                                  satProbingAssigned
	toAssert                                  satSolverRuntimeUintColumn
	cacheWork                                 satSolverRuntimeUintColumn // charged cold workspace; scalar tokens move back on return.
	cache                                     satProbingCache
	stoppedAt, limit, numAssigned             uint32
	cacheLimit, ownerBytes                    uint64
	enabled, cacheEnabled, binaryEnabled      bool
	debugProbing, debugPropagate, debugMissed bool
	constructed, closing, closed              bool
}

// The cold header and bound empty vectors are explicit Go ownership context;
// actual source parameter/statistics construction remains initializeSource.
func newSATColdProbing(l *satSolverClauseLifecycle, b *arena.Budget) (*satSolverProbing, error) {
	if e := l.constructionBorrow(); e != nil {
		return nil, e
	}
	if b != l.budget || b != l.propagation.memory.Budget() {
		return nil, ErrSATOwner
	}
	n := uint64(reflect.TypeOf(satSolverProbing{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &satSolverProbing{lifecycle: l, budget: b, ownerBytes: n}
	s.self = s
	s.toAssert.bind(l.propagation.memory)
	s.cacheWork.bind(l.propagation.memory)
	s.assigned.bind(l.propagation.memory)
	return s, nil
}
func (s *satSolverProbing) initializeSource(c *ParamsContext, v *ParamsRef) error {
	if e := s.constructionBorrow(); e != nil {
		return e
	}
	if s.constructed {
		return ErrSATOwner
	}
	if e := s.updateParams(c, v); e != nil {
		return e
	}
	s.resetStatistics()
	s.stoppedAt = 0
	s.propagation().probingCounter = 0
	s.constructed = true
	return nil
}
func newSATSolverProbing(l *satSolverClauseLifecycle, c *ParamsContext, v *ParamsRef, b *arena.Budget) (*satSolverProbing, error) {
	s, e := newSATColdProbing(l, b)
	if e != nil {
		return nil, e
	}
	if e = s.initializeSource(c, v); e != nil {
		if cleanup := s.Close(); cleanup != nil {
			return s, cleanup
		}
		return nil, e
	}
	return s, nil
}
func (s *satSolverProbing) propagation() *satSolverPropagationState { return s.lifecycle.propagation }
func (s *satSolverProbing) constructionBorrow() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	return s.lifecycle.constructionBorrow()
}
func (s *satSolverProbing) open() error {
	if e := s.constructionBorrow(); e != nil {
		return e
	}
	if !s.constructed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverProbing) updateParams(c *ParamsContext, p *ParamsRef) error {
	if e := s.constructionBorrow(); e != nil {
		return e
	}
	var e error
	k := "probing"
	s.enabled, e = c.GetBoolName(p, &k, true)
	if e != nil {
		return e
	}
	k = "probing_limit"
	s.limit, e = c.GetUIntName(p, &k, 5000000)
	if e != nil {
		return e
	}
	k = "probing_cache"
	s.cacheEnabled, e = c.GetBoolName(p, &k, true)
	if e != nil {
		return e
	}
	k = "probing_binary"
	s.binaryEnabled, e = c.GetBoolName(p, &k, true)
	if e != nil {
		return e
	}
	k = "probing_chache_limit"
	n, e := c.GetUIntName(p, &k, 1024)
	if e != nil {
		return e
	}
	s.cacheLimit = uint64(n) << 20
	return nil
}
func (s *satSolverProbing) resetStatistics() { s.numAssigned = 0 }

// key borrows the caller-provided static spelling "probing assigned"; this
// explicit byte-region binding replaces the source string-literal pointer.
func (s *satSolverProbing) collectStatistics(st *Statistics, key StatisticsKey) error {
	if e := s.open(); e != nil {
		return e
	}
	return st.UpdateUint(key, s.numAssigned)
}

// Source collect_param_descrs has an empty TODO body.
func satProbingCollectParamDescriptions(d *ParamDescrs) {}
func (s *satSolverProbing) dec(n uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	p := s.propagation()
	p.probingCounter = int32(uint32(p.probingCounter) - n)
	return nil
}
func (s *satSolverProbing) decrement() error {
	if s.propagation().probingCounter == -1<<31 {
		return ErrSATSignedCounterDomain
	}
	s.propagation().probingCounter--
	return nil
}
func (s *satSolverProbing) cacheBins(l satLiteral, old uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.cacheEnabled {
		return nil
	}
	allocation, e := s.propagation().memory.AllocationSize()
	if e != nil {
		return e
	}
	if allocation > s.cacheLimit {
		return nil
	}
	if e = s.cache.reserve(s, l.index()+1); e != nil {
		return e
	}
	if e = s.propagation().memory.Write(s.cache.block, 8+16*uint64(l.index()), 1); e != nil {
		return e
	}
	out, e := s.cache.workRow(s, l.index())
	if e != nil {
		return e
	}
	defer s.cache.saveWork(s, l.index())
	if e = out.shrink(0); e != nil {
		return e
	}
	v := s.propagation().variables
	n, e := v.trail.Size(v.trailRef)
	if e != nil {
		return e
	}
	for i := old; i < n; i++ {
		x, e := v.trailRead(i)
		if e != nil {
			return e
		}
		if e = out.push(uint32(x)); e != nil {
			return e
		}
	}
	return nil
}
func (s *satSolverProbing) cachedImpliedLits(l satLiteral) (satProbingCacheBorrow, bool, error) {
	if e := s.open(); e != nil {
		return satProbingCacheBorrow{}, false, e
	}
	if !s.cacheEnabled {
		return satProbingCacheBorrow{}, false, nil
	}
	n, e := s.cache.size(s)
	if e != nil {
		return satProbingCacheBorrow{}, false, e
	}
	if l.index() >= n {
		return satProbingCacheBorrow{}, false, nil
	}
	a, e := s.propagation().memory.Read(s.cache.block, 8+16*uint64(l.index()))
	if e != nil {
		return satProbingCacheBorrow{}, false, e
	}
	if a == 0 {
		return satProbingCacheBorrow{}, false, nil
	}
	return satProbingCacheBorrow{s, s.cache.block, l.index()}, true, nil
}
func (s *satSolverProbing) resetCache(l satLiteral) error {
	if e := s.open(); e != nil {
		return e
	}
	n, e := s.cache.size(s)
	if e != nil {
		return e
	}
	if l.index() >= n {
		return nil
	}
	if e = s.propagation().memory.Write(s.cache.block, 8+16*uint64(l.index()), 0); e != nil {
		return e
	}
	out, e := s.cache.workRow(s, l.index())
	if e != nil {
		return e
	}
	e = out.finalize()
	s.cache.saveWork(s, l.index())
	return e
}
func (s *satSolverProbing) finalize() error {
	if e := s.assigned.Finalize(); e != nil {
		return e
	}
	if e := s.toAssert.finalize(); e != nil {
		return e
	}
	s.toAssert.bind(s.propagation().memory)
	s.toAssert.raw.closed = false
	return s.cache.finalize(s)
}
func (s *satSolverProbing) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.closing = true
	// Reverse source member order: cached_bins, to_assert, assigned.
	if e := s.cache.finalize(s); e != nil {
		return e
	}
	if e := s.toAssert.finalize(); e != nil {
		return e
	}
	if e := s.assigned.Close(); e != nil {
		return e
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// Source check_invariant consumes real resource before constructing checker.
// Enabled diagnostic sites delegate to the actual shared lifecycle checker.
func (s *satSolverProbing) checkInvariant() error { return s.lifecycle.checkInvariant() }
func (s *satSolverProbing) propagate() error {
	_, e := s.lifecycle.propagate(false, s.debugPropagate, s.debugMissed)
	return e
}
func (s *satSolverProbing) assertReady(l satLiteral) error {
	if !satStorageSourceDebug {
		return nil
	}
	v := s.propagation().variables
	n, e := v.trail.Size(v.trailRef)
	if e != nil {
		return e
	}
	if v.qhead != n {
		return ErrSATAssertion
	}
	a, e := v.valueVar(l.variable())
	if e != nil {
		return e
	}
	if a != lUndef {
		return ErrSATAssertion
	}
	return nil
}
