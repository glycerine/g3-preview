// Licensed under the MIT license.
// Source: src/sat/sat_solver.cpp:367–465; sat_solver.h value/lvl accessors.
package g3

import "github.com/glycerine/g3/golang/arena"

// Borrowed views of the solver's literal-indexed assignment and variable-indexed
// level arrays. This cold view neither owns nor copies those arrays. The solver
// supplies its actual scope level on each call; it is not a search implementation.
type satSolverClauseState struct {
	variables  *satSolverVariableState
	assignment []int8
	levels     []uint32
}

func (s *satSolverClauseState) value(l satLiteral) (lbool, error) {
	if s == nil {
		return lUndef, ErrSATOwner
	}
	if s.variables != nil {
		return s.variables.valueLiteral(l)
	}
	if uint64(l.index()) >= uint64(len(s.assignment)) {
		return lUndef, arena.ErrBounds
	}
	return lbool(s.assignment[l.index()]), nil
}
func (s *satSolverClauseState) level(l satLiteral) (uint32, error) {
	if s == nil {
		return 0, ErrSATOwner
	}
	if s.variables != nil {
		return s.variables.levelVar(l.variable())
	}
	if uint64(l.variable()) >= uint64(len(s.levels)) {
		return 0, arena.ErrBounds
	}
	return s.levels[l.variable()], nil
}

// Concrete allocation-free library replacement for std::sort on scalar literal
// indices. Equal indices carry identical values, so sorting stability is not an
// observable choice. Heapsort preserves the library's O(n log n) work bound.
func satSortClauseLiterals(a []satLiteral) {
	sift := func(root, end int) {
		for root < end/2 {
			child := 2*root + 1
			if child+1 < end && a[child] < a[child+1] {
				child++
			}
			if a[root] >= a[child] {
				return
			}
			a[root], a[child] = a[child], a[root]
			root = child
		}
	}
	for root := len(a) / 2; root > 0; {
		root--
		sift(root, len(a))
	}
	for end := len(a) - 1; end > 0; end-- {
		a[0], a[end] = a[end], a[0]
		sift(0, end)
	}
}

func (s *satSolverClauseState) simplifyClause(scope uint32, n *uint32, lits []satLiteral) (bool, error) {
	if s == nil {
		return false, ErrSATOwner
	}
	if n == nil || uint64(*n) > uint64(len(lits)) {
		return false, arena.ErrBounds
	}
	satSortClauseLiterals(lits[:*n])
	prev := satDefaultLiteral()
	var j uint32
	for i := uint32(0); i < *n; i++ {
		curr := lits[i]
		val, e := s.value(curr)
		if e != nil {
			return false, e
		}
		if scope != 0 {
			lvl, e := s.level(curr)
			if e != nil {
				return false, e
			}
			if lvl > 0 {
				val = lUndef
			}
		}
		switch val {
		case lFalse:
		case lUndef:
			if curr == prev.negated() {
				return false, nil
			}
			if curr != prev {
				prev = curr
				if i != j {
					lits[j] = lits[i]
				}
				j++
			}
		case lTrue:
			return false, nil
		}
	}
	*n = j
	return true, nil
}

func (s *satSolverClauseState) selectWatchLiteral(a *satClauseAllocator, clause satStoredClauseRef, start uint32) (uint32, error) {
	if s == nil {
		return 0, ErrSATOwner
	}
	n, e := a.Size(clause)
	if e != nil {
		return 0, e
	}
	if satStorageSourceDebug && n < 2 {
		return 0, ErrSATAssertion
	}
	none := ^uint32(0)
	minTrue, maxFalse, unknown := none, none, none
	for i := start; i < n; i++ {
		l, e := a.Literal(clause, i)
		if e != nil {
			return 0, e
		}
		val, e := s.value(l)
		if e != nil {
			return 0, e
		}
		switch val {
		case lFalse:
			if maxFalse == none {
				maxFalse = i
			} else {
				current, e := s.level(l)
				if e != nil {
					return 0, e
				}
				previous, e := a.Literal(clause, maxFalse)
				if e != nil {
					return 0, e
				}
				level, e := s.level(previous)
				if e != nil {
					return 0, e
				}
				if current > level {
					maxFalse = i
				}
			}
		case lUndef:
			unknown = i
		case lTrue:
			if minTrue == none {
				minTrue = i
			} else {
				current, e := s.level(l)
				if e != nil {
					return 0, e
				}
				previous, e := a.Literal(clause, minTrue)
				if e != nil {
					return 0, e
				}
				level, e := s.level(previous)
				if e != nil {
					return 0, e
				}
				if current < level {
					minTrue = i
				}
			}
		}
	}
	if minTrue != none {
		return minTrue, nil
	}
	if unknown != none {
		return unknown, nil
	}
	if satStorageSourceDebug && maxFalse == none {
		return 0, ErrSATAssertion
	}
	return maxFalse, nil
}

func (s *satSolverClauseState) selectLearnedWatchLiteral(a *satClauseAllocator, clause satStoredClauseRef) (uint32, error) {
	if s == nil {
		return 0, ErrSATOwner
	}
	n, e := a.Size(clause)
	if e != nil {
		return 0, e
	}
	if satStorageSourceDebug && n < 2 {
		return 0, ErrSATAssertion
	}
	maxFalse := ^uint32(0)
	for i := uint32(1); i < n; i++ {
		l, e := a.Literal(clause, i)
		if e != nil {
			return 0, e
		}
		if satStorageSourceDebug {
			val, e := s.value(l)
			if e != nil {
				return 0, e
			}
			if val != lFalse {
				return 0, ErrSATAssertion
			}
		}
		if maxFalse == ^uint32(0) {
			maxFalse = i
		} else {
			current, e := s.level(l)
			if e != nil {
				return 0, e
			}
			previous, e := a.Literal(clause, maxFalse)
			if e != nil {
				return 0, e
			}
			level, e := s.level(previous)
			if e != nil {
				return 0, e
			}
			if current > level {
				maxFalse = i
			}
		}
	}
	return maxFalse, nil
}
