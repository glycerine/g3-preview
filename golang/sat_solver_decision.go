// Source: src/sat/sat_solver.cpp:933–988,2770; src/sat/sat_solver.h:337–338; src/util/util.h. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2011.
package g3

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
)

// Actual decision methods share the conflict owner's source phase-cache flag
// and the variable owner's actual queue, assignments and scopes. This charged
// cold member grouping owns the sole source random_gen and decision statistic;
// it is not the complete solver constructor, parameter update or search loop.
type satSolverDecision struct {
	budget     *arena.Budget
	self       *satSolverDecision
	conflict   *satSolverConflictResolution
	rng        utilhash.Random
	decisions  uint32
	ownerBytes uint64
	closed     bool
}

func newSATSolverDecision(s *satSolverConflictResolution) (*satSolverDecision, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	cfg, e := s.analysis.propagation.config.State()
	if e != nil {
		return nil, e
	}
	d, e := newSATColdDecision(s.analysis.budget)
	if e != nil {
		return nil, e
	}
	d.conflict = s
	d.rng.SetSeed(cfg.RandomSeed)
	return d, nil
}

// Source random_gen default construction precedes dependent solver members.
// The explicit cold owner is real charged metadata, not a seeded shadow RNG.
func newSATColdDecision(b *arena.Budget) (*satSolverDecision, error) {
	n := uint64(reflect.TypeOf(satSolverDecision{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	d := &satSolverDecision{budget: b, ownerBytes: n}
	d.self = d
	return d, nil
}

func (d *satSolverDecision) constructionBorrow() error {
	if d == nil || d.self != d {
		return ErrSATOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return d.conflict.constructionBorrow()
}

func (d *satSolverDecision) open() error {
	if d == nil || d.self != d {
		return ErrSATOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return d.conflict.open()
}

func (d *satSolverDecision) nextVar() (satBoolVar, error) {
	if e := d.open(); e != nil {
		return satNullBoolVar, e
	}
	p := d.conflict.analysis.propagation
	v := p.variables
	cfg, e := p.config.State()
	if e != nil {
		return satNullBoolVar, e
	}
	draw := d.rng.Next()
	threshold := math.Trunc(cfg.RandomFreq * float64(utilhash.RandomMax))
	// The source floating-to-int conversion is undefined outside int32. This
	// guard is a Go domain adaptation, not a C++ malformed-prefix guarantee.
	if math.IsNaN(threshold) || threshold < math.MinInt32 || threshold > math.MaxInt32 {
		return satNullBoolVar, arena.ErrBounds
	}
	if int32(draw) < int32(threshold) {
		num, e := v.level.size()
		if e != nil {
			return satNullBoolVar, e
		}
		if num == 0 {
			return satNullBoolVar, nil
		}
		// num_vars is reread at the actual source modulo expression.
		draw := d.rng.Next()
		num, e = v.level.size()
		if e != nil {
			return satNullBoolVar, e
		}
		if num == 0 {
			return satNullBoolVar, arena.ErrBounds
		}
		next := satBoolVar(draw % num)
		value, e := v.valueVar(next)
		if e != nil {
			return satNullBoolVar, e
		}
		if value == lUndef {
			removed, e := v.wasEliminated(next)
			if e != nil {
				return satNullBoolVar, e
			}
			if !removed {
				return next, nil
			}
		}
	}
	for {
		empty, e := v.queue.emptyChecked()
		if e != nil {
			return satNullBoolVar, e
		}
		if empty {
			break
		}
		next, e := v.queue.NextVar()
		if e != nil {
			return satNullBoolVar, e
		}
		value, e := v.valueVar(next)
		if e != nil {
			return satNullBoolVar, e
		}
		if value == lUndef {
			removed, e := v.wasEliminated(next)
			if e != nil {
				return satNullBoolVar, e
			}
			if !removed {
				return next, nil
			}
		}
	}
	return satNullBoolVar, nil
}

func (d *satSolverDecision) decide() (bool, error) {
	next, e := d.nextVar()
	if e != nil {
		return false, e
	}
	if next == satNullBoolVar {
		return false, nil
	}
	l := d.conflict.dynamic.probing.lifecycle
	if e = l.push(); e != nil {
		return false, e
	}
	d.decisions++
	p := l.propagation
	if p.extensionPresent {
		return false, ErrSATMissingSourceExtBinding
	}
	phase := lUndef
	cfg, e := p.config.State()
	if e != nil {
		return false, e
	}
	switch cfg.Phase {
	case satPhaseAlwaysTrue:
		phase = lTrue
	case satPhaseAlwaysFalse:
		phase = lFalse
	case satPhaseCaching:
		phase = lFalse
		if d.conflict.phaseCacheOn {
			cached, e := p.variables.phase.read(uint32(next))
			if e != nil {
				return false, e
			}
			if cached != 2 {
				cached, e = p.variables.phase.read(uint32(next))
				if e != nil {
					return false, e
				}
				if cached == 0 {
					phase = lTrue
				}
			}
		}
	case satPhaseRandom:
		if d.rng.Next()%2 == 0 {
			phase = lTrue
		} else {
			phase = lFalse
		}
	default:
		return false, ErrSATAssertion
	}
	if satStorageSourceDebug && phase == lUndef {
		return false, ErrSATAssertion
	}
	if e = p.assign(satMakeLiteral(next, phase == lFalse), satJustification{}); e != nil {
		return false, e
	}
	return true, nil
}

func (d *satSolverDecision) Close() error {
	if d == nil || d.self != d {
		return ErrSATOwner
	}
	if d.closed {
		return nil
	}
	d.budget.Release(d.ownerBytes)
	d.ownerBytes = 0
	d.closed = true
	return nil
}
