// Source: src/sat/sat_cleaner.h and .cpp:26–229; src/sat/sat_solver.cpp:3082–3087.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Actual source cleaner member. Dependencies are borrowed and must outlive it.
// The embedded interruption ledger owns only source-compaction ranges, never
// all allocator-live objects. It is an explicit guarded exception adapter.
type satCleaner struct {
	self                                                           *satCleaner
	lifecycle                                                      *satSolverClauseLifecycle
	lastNumUnits, elimClauses, elimLiterals                        uint32
	ownerBytes                                                     uint64
	cursor, out, end, drainPrefix, drainSuffix                     uint32
	interrupted, learned, closing, closed, debugInvariant, running bool
}

func newSATCleaner(l *satSolverClauseLifecycle, b *arena.Budget) (*satCleaner, error) {
	if e := l.constructionBorrow(); e != nil {
		return nil, e
	}
	if b != l.budget || b != l.propagation.memory.Budget() {
		return nil, ErrSATOwner
	}
	n := uint64(reflect.TypeOf(satCleaner{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	c := &satCleaner{lifecycle: l, ownerBytes: n}
	c.self = c
	l.propagation.cleanerCounter = 0
	c.resetStatistics()
	return c, nil
}
func (c *satCleaner) identity() error {
	if c == nil || c.self != c {
		return ErrSATOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}
func (c *satCleaner) constructionBorrow() error {
	if e := c.identity(); e != nil {
		return e
	}
	if c.closing || c.interrupted {
		return arena.ErrClosed
	}
	return c.lifecycle.constructionBorrow()
}
func (c *satCleaner) open() error {
	if e := c.identity(); e != nil {
		return e
	}
	if c.closing || c.interrupted {
		return arena.ErrClosed
	}
	return c.lifecycle.open()
}
func (c *satCleaner) resetStatistics() { c.elimClauses = 0; c.elimLiterals = 0 }

// Keys are explicit caller-borrowed static spelling counterparts.
func (c *satCleaner) collectStatistics(s *util.Statistics, clauses, literals util.StatisticsKey) error {
	if e := c.open(); e != nil {
		return e
	}
	if e := s.UpdateUint(clauses, c.elimClauses); e != nil {
		return e
	}
	return s.UpdateUint(literals, c.elimLiterals)
}
func (c *satCleaner) invariant() error {
	if satStorageSourceDebug && c.debugInvariant {
		return c.lifecycle.checkInvariant()
	}
	return nil
}
func (c *satCleaner) run(force bool) (done bool, err error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if e := c.invariant(); e != nil {
		return false, e
	}
	p := c.lifecycle.propagation
	n, e := p.variables.trail.Size(p.variables.trailRef)
	if e != nil {
		return false, e
	}
	if _, e = p.propagateCore(false); e != nil {
		return false, e
	}
	if p.inconsistent {
		return false, nil
	}
	if c.lastNumUnits == n {
		return false, nil
	}
	if !force && p.cleanerCounter > 0 {
		return false, nil
	}
	// Source report starts only after all early exits; original error wins on Stop.
	var report util.Stopwatch
	if e = report.Start(); e != nil {
		return false, e
	}
	defer func() {
		if e := report.Stop(); err == nil && e != nil {
			err = e
		}
	}()
	c.lastNumUnits = n
	p.cleanerCounter = 0
	if e = c.cleanupWatches(); e != nil {
		return false, e
	}
	if e = c.cleanupClauses(false); e != nil {
		return false, e
	}
	if e = c.cleanupClauses(true); e != nil {
		return false, e
	}
	if _, e = p.propagateCore(false); e != nil {
		return false, e
	}
	if e = c.invariant(); e != nil {
		return false, e
	}
	return true, nil
}
func (c *satCleaner) cleanup() error {
	if e := c.open(); e != nil {
		return e
	}
	p := c.lifecycle.propagation
	if p.variables.scopeLevel > 0 || p.inconsistent {
		return nil
	}
	done, e := c.run(false)
	if e != nil {
		return e
	}
	if done && p.extensionPresent {
		return ErrSATMissingSourceExtBinding
	}
	return nil
}
func (c *satCleaner) cleanupWatches() error {
	if e := c.open(); e != nil {
		return e
	}
	p := c.lifecycle.propagation
	v := p.variables
	// Source captures outer vector end; no owner callback grows that vector here.
	n := uint32(len(v.watches.lists))
	for idx := uint32(0); idx < n; idx++ {
		list, e := v.watches.Get(idx)
		if e != nil {
			return e
		}
		value, e := v.valueLiteral(Literal(idx))
		if e != nil {
			return e
		}
		if value != util.LUndef {
			if e = list.Finalize(); e != nil {
				return e
			}
			if satStorageSourceDebug {
				sz, e := list.Size()
				if e != nil {
					return e
				}
				if sz != 0 {
					return ErrSATAssertion
				}
			}
			continue
		}
		ref, e := list.ref()
		if e != nil {
			return e
		}
		it, e := v.watches.store.Iter(ref)
		if e != nil {
			return e
		}
		out := uint32(0)
		for pos := uint32(0); pos < it.end; pos++ {
			w, e := satPropagationCursorRead(&it, pos)
			if e != nil {
				return e
			}
			keep := false
			switch w.Kind() {
			case 0:
				l, e := w.Literal()
				if e != nil {
					return e
				}
				if satStorageSourceDebug {
					a, e := v.valueLiteral(l)
					if e != nil {
						return e
					}
					if a != util.LTrue {
						a, e = v.valueLiteral(l)
						if e != nil {
							return e
						}
						if a != util.LUndef {
							return ErrSATAssertion
						}
					}
				}
				a, e := v.valueLiteral(l)
				if e != nil {
					return e
				}
				keep = a == util.LUndef
			case 1, 2:
			case 3:
				keep = true
			default:
				return ErrSATAssertion
			}
			if keep {
				if e = satPropagationCursorWrite(&it, out, w); e != nil {
					return e
				}
				out++
			}
		}
		if e = p.finishPropagationList(&it, list, out); e != nil {
			return e
		}
	}
	return nil
}
func (c *satCleaner) Close() error {
	if c == nil || c.self != c {
		return ErrSATOwner
	}
	if c.closed {
		return nil
	}
	c.closing = true
	if e := c.cleanupInterrupted(); e != nil {
		return e
	}
	c.lifecycle.budget.Release(c.ownerBytes)
	c.ownerBytes = 0
	c.closed = true
	return nil
}
