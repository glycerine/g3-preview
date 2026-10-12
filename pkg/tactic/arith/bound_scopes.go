// Source: src/tactic/arith/bound_propagator.cpp:202-263,741-759.
package arith

import "github.com/glycerine/g3/pkg/arena"

type equationScope struct {
	trailLimit  int
	qheadOld    int
	reinitLimit int
	timestamp   uint32
	inConflict  bool
}

func (s *BoundEquationStore) ScopeLevel() (uint32, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	return uint32(len(s.scopes)), nil
}

func (s *BoundEquationStore) Push() error {
	if err := s.open(); err != nil {
		return err
	}
	s.scopes = append(s.scopes, equationScope{
		trailLimit: len(s.trail), qheadOld: s.qhead,
		reinitLimit: len(s.reinitStack), timestamp: s.timestamp,
		inConflict: s.conflict != noEquationPosition,
	})
	return nil
}

func (s *BoundEquationStore) undoTrail(limit int) error {
	if limit < 0 || limit > len(s.trail) {
		return arena.ErrBounds
	}
	for len(s.trail) > limit {
		last := len(s.trail) - 1
		b := s.trail[last]
		v := &s.variables[b.x]
		if b.lower {
			if v.lower == b {
				v.lower = b.prev
			}
		} else {
			if v.upper == b {
				v.upper = b.prev
			}
		}
		if err := s.manager.Del(&b.value); err != nil {
			return err
		}
		s.trail = s.trail[:last]
	}
	return nil
}

// Pop restores the oldest popped scope, then rechecks constraints that first
// propagated during any of the removed levels.
func (s *BoundEquationStore) Pop(numScopes uint32) error {
	if err := s.open(); err != nil {
		return err
	}
	if numScopes == 0 || uint64(numScopes) > uint64(len(s.scopes)) {
		return arena.ErrBounds
	}
	newLevel := len(s.scopes) - int(numScopes)
	scope := s.scopes[newLevel]
	if err := s.undoTrail(scope.trailLimit); err != nil {
		return err
	}
	s.timestamp = scope.timestamp
	s.qhead = scope.qheadOld
	if !scope.inConflict {
		s.conflict = noEquationPosition
	}
	s.scopes = s.scopes[:newLevel]
	end, write := len(s.reinitStack), scope.reinitLimit
	for read := scope.reinitLimit; read < end; read++ {
		index := s.reinitStack[read]
		propagated, err := s.propagateEquation(index)
		if err != nil {
			return err
		}
		if newLevel > 0 && propagated {
			s.reinitStack[write] = index
			write++
		}
	}
	s.reinitStack = s.reinitStack[:write]
	return nil
}
