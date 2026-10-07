// Source: src/sat/sat_solver.cpp:933–988,2770; src/sat/sat_solver.h:337–338; src/util/util.h.
package sat

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Actual decision methods share the conflict owner's source phase-cache flag
// and the variable owner's actual queue, assignments and scopes. This charged
// cold member grouping owns the sole source random_gen and decision statistic;
// it is not the complete solver constructor, parameter update or search loop.
type satSolverDecision struct {
	budget     *arena.Budget
	self       *satSolverDecision
	conflict   *satSolverConflictResolution
	rng        util.Random
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

func (d *satSolverDecision) nextVar() (BoolVar, error) {
	if e := d.open(); e != nil {
		return NullBoolVar, e
	}
	p := d.conflict.analysis.propagation
	v := p.variables
	cfg, e := p.config.State()
	if e != nil {
		return NullBoolVar, e
	}
	draw := d.rng.Next()
	threshold := math.Trunc(cfg.RandomFreq * float64(util.RandomMax))
	// The source floating-to-int conversion is undefined outside int32. This
	// guard is a Go domain adaptation, not a C++ malformed-prefix guarantee.
	if math.IsNaN(threshold) || threshold < math.MinInt32 || threshold > math.MaxInt32 {
		return NullBoolVar, arena.ErrBounds
	}
	if int32(draw) < int32(threshold) {
		num, e := v.level.Size()
		if e != nil {
			return NullBoolVar, e
		}
		if num == 0 {
			return NullBoolVar, nil
		}
		// num_vars is reread at the actual source modulo expression.
		draw := d.rng.Next()
		num, e = v.level.Size()
		if e != nil {
			return NullBoolVar, e
		}
		if num == 0 {
			return NullBoolVar, arena.ErrBounds
		}
		next := BoolVar(draw % num)
		value, e := v.valueVar(next)
		if e != nil {
			return NullBoolVar, e
		}
		if value == util.LUndef {
			removed, e := v.wasEliminated(next)
			if e != nil {
				return NullBoolVar, e
			}
			if !removed {
				return next, nil
			}
		}
	}
	for {
		empty, e := v.queue.emptyChecked()
		if e != nil {
			return NullBoolVar, e
		}
		if empty {
			break
		}
		next, e := v.queue.NextVar()
		if e != nil {
			return NullBoolVar, e
		}
		value, e := v.valueVar(next)
		if e != nil {
			return NullBoolVar, e
		}
		if value == util.LUndef {
			removed, e := v.wasEliminated(next)
			if e != nil {
				return NullBoolVar, e
			}
			if !removed {
				return next, nil
			}
		}
	}
	return NullBoolVar, nil
}

func (d *satSolverDecision) decide() (bool, error) {
	next, e := d.nextVar()
	if e != nil {
		return false, e
	}
	if next == NullBoolVar {
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
	phase := util.LUndef
	cfg, e := p.config.State()
	if e != nil {
		return false, e
	}
	switch cfg.Phase {
	case satPhaseAlwaysTrue:
		phase = util.LTrue
	case satPhaseAlwaysFalse:
		phase = util.LFalse
	case satPhaseCaching:
		phase = util.LFalse
		if d.conflict.phaseCacheOn {
			cached, e := p.variables.phase.Read(uint32(next))
			if e != nil {
				return false, e
			}
			if cached != 2 {
				cached, e = p.variables.phase.Read(uint32(next))
				if e != nil {
					return false, e
				}
				if cached == 0 {
					phase = util.LTrue
				}
			}
		}
	case satPhaseRandom:
		if d.rng.Next()%2 == 0 {
			phase = util.LTrue
		} else {
			phase = util.LFalse
		}
	default:
		return false, ErrSATAssertion
	}
	if satStorageSourceDebug && phase == util.LUndef {
		return false, ErrSATAssertion
	}
	if e = p.assign(MakeLiteral(next, phase == util.LFalse), satJustification{}); e != nil {
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
