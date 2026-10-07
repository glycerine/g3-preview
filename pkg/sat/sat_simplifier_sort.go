// Source: src/sat/sat_simplifier.cpp:83–96; std::stable_sort invocation. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2011.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// The merge workspace is actual charged Go physical storage. Its comparison
// schedule/malloc topology is not claimed identical to the C++ standard library.
// Moved output and untouched input ranges own clauses; stale scalar duplicates
// elsewhere are borrows, never an inferred backup clause registry.
type satSimplifierClauseSort struct {
	scratch                                               []StoredClauseRef
	bytes                                                 uint64
	n, left, mid, right, end, out, drainStage, drainIndex uint32
	learned, sourceScratch, active                        bool
}

func (s *satSimplifier) clauseVector(learned bool) []StoredClauseRef {
	if learned {
		return s.lifecycle.learned
	}
	return s.lifecycle.Original
}
func (s *satSimplifier) sortSource() []StoredClauseRef {
	if s.clauseSort.sourceScratch {
		return s.clauseSort.scratch
	}
	return s.clauseVector(s.clauseSort.learned)
}
func (s *satSimplifier) sortDestination() []StoredClauseRef {
	if s.clauseSort.sourceScratch {
		return s.clauseVector(s.clauseSort.learned)
	}
	return s.clauseSort.scratch
}
func (s *satSimplifier) sortClauses(learned bool) error {
	st := &s.clauseSort
	n := uint32(len(s.clauseVector(learned)))
	if n < 2 {
		return nil
	}
	if st.active {
		return arena.ErrClosed
	}
	bytes := uint64(n) * uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	if bytes > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := s.lifecycle.budget.Reserve(bytes); e != nil {
		return e
	}
	s.lifecycle.simplifierContinuation = s
	*st = satSimplifierClauseSort{scratch: make([]StoredClauseRef, n), bytes: bytes, n: n, learned: learned, active: true}
	for width := uint64(1); width < uint64(n); width *= 2 {
		st.out = 0
		for base := uint64(0); base < uint64(n); base += 2 * width {
			mid := base + width
			if mid > uint64(n) {
				mid = uint64(n)
			}
			end := base + 2*width
			if end > uint64(n) {
				end = uint64(n)
			}
			st.left = uint32(base)
			st.mid = uint32(mid)
			st.right = uint32(mid)
			st.end = uint32(end)
			for st.left < st.mid || st.right < st.end {
				right := false
				if st.left == st.mid {
					right = true
				} else if st.right < st.end {
					a, e := s.lifecycle.propagation.clauses.Size(s.sortSource()[st.right])
					if e != nil {
						return e
					}
					b, e := s.lifecycle.propagation.clauses.Size(s.sortSource()[st.left])
					if e != nil {
						return e
					}
					right = a > b
				}
				idx := st.left
				if right {
					idx = st.right
				}
				s.sortDestination()[st.out] = s.sortSource()[idx]
				st.out++
				if right {
					st.right++
				} else {
					st.left++
				}
			}
		}
		st.sourceScratch = !st.sourceScratch
	}
	if st.sourceScratch {
		copy(s.clauseVector(learned), st.scratch)
	}
	s.lifecycle.budget.Release(st.bytes)
	*st = satSimplifierClauseSort{}
	if s.lifecycle.simplifierContinuation == s && !s.cleanupInterrupted {
		s.lifecycle.simplifierContinuation = nil
	}
	return nil
}
func (s *satSimplifier) cleanupSort() error {
	st := &s.clauseSort
	if !st.active {
		return nil
	}
	for st.drainStage < 4 {
		var data []StoredClauseRef
		lo, hi := uint32(0), uint32(0)
		switch st.drainStage {
		case 0:
			data = s.sortDestination()
			hi = st.out
		case 1:
			data = s.sortSource()
			lo, hi = st.left, st.mid
		case 2:
			data = s.sortSource()
			lo, hi = st.right, st.end
		case 3:
			data = s.sortSource()
			lo, hi = st.end, st.n
		}
		if st.drainIndex < lo {
			st.drainIndex = lo
		}
		for st.drainIndex < hi {
			if uint64(st.drainIndex) >= uint64(len(data)) {
				return arena.ErrBounds
			}
			if e := s.lifecycle.propagation.clauses.Delete(data[st.drainIndex]); e != nil {
				return e
			}
			st.drainIndex++
		}
		st.drainStage++
		st.drainIndex = 0
	}
	if st.learned {
		s.lifecycle.learned = s.lifecycle.learned[:0]
	} else {
		s.lifecycle.Original = s.lifecycle.Original[:0]
	}
	s.lifecycle.budget.Release(st.bytes)
	*st = satSimplifierClauseSort{}
	if s.lifecycle.simplifierContinuation == s && !s.cleanupInterrupted {
		s.lifecycle.simplifierContinuation = nil
	}
	return nil
}
func (s *satSimplifier) registerClauses(learned bool) error {
	if e := s.sortClauses(learned); e != nil {
		return e
	}
	cs := s.clauseVector(learned)
	n := uint32(len(cs))
	for i := uint32(0); i < n; i++ {
		c := cs[i]
		f, e := s.lifecycle.propagation.clauses.Frozen(c)
		if e != nil {
			return e
		}
		if !f {
			if e = s.useList.Insert(c); e != nil {
				return e
			}
			b, e := s.lifecycle.propagation.clauses.Strengthened(c)
			if e != nil {
				return e
			}
			if b {
				if e = s.subTodo.Insert(c); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
