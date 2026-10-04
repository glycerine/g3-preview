// Source: src/sat/sat_solver.cpp; src/sat/sat_clause.h. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// A concrete borrowed literal pointer: either caller Go storage or a captured
// actual Runtime payload. No copied lemma, callback, or accounting ticket.
type satSolverLiteralSpan struct {
	literals []satLiteral
	runtime  *memory.Runtime
	block    memory.Block
	words    *arena.Words
	payload  arena.VectorRef
}

func satConflictRuntimeSpan(c *satSolverRuntimeUintColumn) satSolverLiteralSpan {
	return satSolverLiteralSpan{runtime: c.raw.runtime, block: c.raw.block}
}

// Captures the actual c_ptr payload; later growth never silently rebinds it.
func satSimplifierCapturedSpan(s *satSolverLiteralStore, r satSolverLiteralListRef) (satSolverLiteralSpan, error) {
	rec, e := s.record(r)
	if e != nil {
		return satSolverLiteralSpan{}, e
	}
	return satSolverLiteralSpan{words: s.words, payload: rec.payload}, nil
}
func (s satSolverLiteralSpan) literal(i uint32) (satLiteral, error) {
	if s.runtime != nil {
		v, e := satAssignmentReadWord(s.runtime, s.block, 8+4*uint64(i))
		return satLiteral(v), e
	}
	if s.words != nil {
		v, e := s.words.Read(s.payload, i)
		return satLiteral(v), e
	}
	if uint64(i) >= uint64(cap(s.literals)) {
		return 0, arena.ErrBounds
	}
	return s.literals[:cap(s.literals)][i], nil
}
