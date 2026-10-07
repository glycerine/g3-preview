// Source: src/sat/sat_solver.cpp; src/sat/sat_clause.h. Licensed under the MIT license.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// A concrete borrowed literal pointer: either caller Go storage or a captured
// actual Runtime payload. No copied lemma, callback, or accounting ticket.
type satSolverLiteralSpan struct {
	literals []Literal
	runtime  *util.Runtime
	block    util.Block
	words    *arena.Words
	payload  arena.VectorRef
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
	return satSolverLiteralSpan{words: s.words, payload: rec.payload}, nil
}
func (s satSolverLiteralSpan) literal(i uint32) (Literal, error) {
	if s.runtime != nil {
		v, e := ReadRuntimeWord(s.runtime, s.block, 8+4*uint64(i))
		return Literal(v), e
	}
	if s.words != nil {
		v, e := s.words.Read(s.payload, i)
		return Literal(v), e
	}
	if uint64(i) >= uint64(cap(s.literals)) {
		return 0, arena.ErrBounds
	}
	return s.literals[:cap(s.literals)][i], nil
}

// Writes the actual captured pointer; it never rebinds to a relocated member.
func (s satSolverLiteralSpan) setLiteral(i uint32, l Literal) error {
	if s.runtime != nil {
		return s.runtime.WriteUint32(s.block, 8+4*uint64(i), uint32(l))
	}
	if s.words != nil {
		return s.words.Write(s.payload, i, uint32(l))
	}
	if uint64(i) >= uint64(cap(s.literals)) {
		return arena.ErrBounds
	}
	s.literals[:cap(s.literals)][i] = l
	return nil
}

// Concrete library sort adapter on a captured raw span. Errors occur only at
// actual reads/writes; the source library's comparison schedule is unspecified.
func satSortClauseSpan(s satSolverLiteralSpan, n uint32) error {
	if s.runtime == nil && s.words == nil {
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
