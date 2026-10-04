// Source: src/sat/sat_solver.h:539-542; src/util/vector.h:290-294.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

import "github.com/glycerine/g3/golang/arena"

// Captured begin/end pointers are represented by a borrowed backing and end.
// Existing pushClause replaces a backing ONLY with strictly larger capacity.
// Thus capacity equality proves identity within this owner lifetime; same-cap
// append/compaction remains visible. This does not implement source pointer8
// allocation ABI for the existing Go24 clause references.
type satSolverClauseIterator struct {
	owner      *satSolver
	backing    []satStoredClauseRef
	index, end uint32
	capacity   int
	learned    bool
}

func (s *satSolver) beginOriginalClauses() (satSolverClauseIterator, error) {
	return s.beginClauseIterator(false)
}
func (s *satSolver) beginLearnedClauses() (satSolverClauseIterator, error) {
	return s.beginClauseIterator(true)
}
func (s *satSolver) beginClauseIterator(learned bool) (satSolverClauseIterator, error) {
	if e := s.open(); e != nil {
		return satSolverClauseIterator{}, e
	}
	xs := s.lifecycle.original
	if learned {
		xs = s.lifecycle.learned
	}
	return satSolverClauseIterator{owner: s, backing: xs[:cap(xs)], end: uint32(len(xs)), capacity: cap(xs), learned: learned}, nil
}
func (it *satSolverClauseIterator) Next() (satStoredClauseRef, bool, error) {
	if it == nil {
		return satStoredClauseRef{}, false, ErrSATOwner
	}
	if e := it.owner.open(); e != nil {
		return satStoredClauseRef{}, false, e
	}
	if it.index >= it.end {
		return satStoredClauseRef{}, false, nil
	}
	xs := it.owner.lifecycle.original
	if it.learned {
		xs = it.owner.lifecycle.learned
	}
	if cap(xs) != it.capacity {
		return satStoredClauseRef{}, false, arena.ErrBounds
	}
	if uint64(it.index) >= uint64(len(it.backing)) {
		return satStoredClauseRef{}, false, arena.ErrBounds
	}
	c := it.backing[it.index]
	it.index++
	return c, true, nil
}
