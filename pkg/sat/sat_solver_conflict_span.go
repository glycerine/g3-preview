// Source: src/sat/sat_solver.cpp; src/sat/sat_clause.h.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// A borrowed literal span over caller Go storage, native literal backing, or
// a retained Runtime column. Captured backing is never silently rebound.
type satSolverLiteralSpan struct {
	literals []Literal
	runtime  *util.Runtime
	block    util.Block
	store    *satSolverLiteralStore
	backing  *satLiteralBacking
}

func satConflictRuntimeSpan(c *RuntimeUintColumn) satSolverLiteralSpan {
	return satSolverLiteralSpan{runtime: c.Raw.Runtime, block: c.Raw.Block}
}

// Captures the actual c_ptr payload; later growth never silently rebinds it.
func satSimplifierCapturedSpan(s *satSolverLiteralStore, r satSolverLiteralListRef) (satSolverLiteralSpan, error) {
	rec, e := s.record(r)
	if e != nil {
		return satSolverLiteralSpan{}, e
	}
	return satSolverLiteralSpan{store: s, backing: rec.backing}, nil
}
func (s satSolverLiteralSpan) literal(i uint32) (Literal, error) {
	if s.runtime != nil {
		v, e := ReadRuntimeWord(s.runtime, s.block, 8+4*uint64(i))
		return Literal(v), e
	}
	if s.store != nil {
		if e := s.store.open(); e != nil {
			return 0, e
		}
		if s.backing == nil || !s.backing.alive {
			return 0, arena.ErrInvalidSlot
		}
		if uint64(i) >= uint64(len(s.backing.data)) {
			return 0, arena.ErrBounds
		}
		return s.backing.data[i], nil
	}
	if uint64(i) >= uint64(len(s.literals)) {
		return 0, arena.ErrBounds
	}
	return s.literals[i], nil
}

// Writes the actual captured pointer; it never rebinds to a relocated member.
func (s satSolverLiteralSpan) setLiteral(i uint32, l Literal) error {
	if s.runtime != nil {
		return s.runtime.WriteUint32(s.block, 8+4*uint64(i), uint32(l))
	}
	if s.store != nil {
		if e := s.store.open(); e != nil {
			return e
		}
		if s.backing == nil || !s.backing.alive {
			return arena.ErrInvalidSlot
		}
		if uint64(i) >= uint64(len(s.backing.data)) {
			return arena.ErrBounds
		}
		s.backing.data[i] = l
		return nil
	}
	if uint64(i) >= uint64(len(s.literals)) {
		return arena.ErrBounds
	}
	s.literals[i] = l
	return nil
}

// Concrete library sort adapter on a captured raw span. Errors occur only at
// actual reads/writes; the source library's comparison schedule is unspecified.
func satSortClauseSpan(s satSolverLiteralSpan, n uint32) error {
	if s.runtime == nil && s.store == nil {
		satSortClauseLiterals(s.literals[:n])
		return nil
	}
	swap := func(i, j uint32) error {
		a, e := s.literal(i)
		if e != nil {
			return e
		}
		b, e := s.literal(j)
		if e != nil {
			return e
		}
		if e = s.setLiteral(i, b); e != nil {
			return e
		}
		return s.setLiteral(j, a)
	}
	sift := func(root, end uint32) error {
		for root < end/2 {
			child := 2*root + 1
			if child+1 < end {
				a, e := s.literal(child)
				if e != nil {
					return e
				}
				b, e := s.literal(child + 1)
				if e != nil {
					return e
				}
				if a < b {
					child++
				}
			}
			a, e := s.literal(root)
			if e != nil {
				return e
			}
			b, e := s.literal(child)
			if e != nil {
				return e
			}
			if a >= b {
				return nil
			}
			if e = swap(root, child); e != nil {
				return e
			}
			root = child
		}
		return nil
	}
	for root := n / 2; root > 0; {
		root--
		if e := sift(root, n); e != nil {
			return e
		}
	}
	for end := n; end > 1; {
		end--
		if e := swap(0, end); e != nil {
			return e
		}
		if e := sift(0, end); e != nil {
			return e
		}
	}
	return nil
}
