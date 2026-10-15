// Source: src/math/euclid/euclidean_solver.cpp:422-461,615-618.
package euclid

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// promoteSolvedEquation performs the source ownership transfer after a pivot
// equation has been normalized to coefficient -1 for x. The caller removes
// the old occurrences before this publication, as elim_unit does.
func (s *solverSetup) promoteSolvedEquation(eqIndex, x uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if eqIndex >= uint32(len(s.equations)) || x >= uint32(len(s.solved)) ||
		s.equations[eqIndex] == nil || s.solved[x] != nullEquationIndex {
		return arena.ErrInvalidSlot
	}
	eq := s.equations[eqIndex]
	p := position(eq.xs, x)
	if p == nullEquationIndex {
		return arena.ErrInvalidSlot
	}
	minus, e := s.manager.Integers().IsMinusOne(&eq.as[p])
	if e != nil {
		return e
	}
	if !minus {
		return arena.ErrInvalidSlot
	}
	s.solved[x] = uint32(len(s.solution))
	s.solution = append(s.solution, eq)
	s.equations[eqIndex] = nil
	return nil
}

// applySolutionOnce substitutes the equation solving x into target. It retains
// the source order: integer merge, constant update, integer raw swap, rational
// merge, rational raw swap. The integer coefficient a1 remains borrowed from
// the old target row after that row moves into tmpAs.
func (s *solverSetup) applySolutionOnce(x uint32, target *equation,
	updateOcc, updateQueue bool, eqIndex, exceptVar uint32, constant *numeral.Int) error {
	if e := s.open(); e != nil {
		return e
	}
	if target == nil || target.closed || x >= uint32(len(s.solved)) || s.solved[x] == nullEquationIndex ||
		s.solved[x] >= uint32(len(s.solution)) {
		return arena.ErrInvalidSlot
	}
	p := position(target.xs, x)
	if p == nullEquationIndex {
		return nil
	}
	a1 := &target.as[p]
	z := s.manager.Integers()
	zero, e := z.IsZero(a1)
	if e != nil {
		return e
	}
	if zero {
		return arena.ErrInvalidSlot
	}
	sol := s.solution[s.solved[x]]
	if sol == nil || sol.closed {
		return arena.ErrInvalidSlot
	}
	sp := position(sol.xs, x)
	if sp == nullEquationIndex {
		return arena.ErrInvalidSlot
	}
	minus, e := z.IsMinusOne(&sol.as[sp])
	if e != nil {
		return e
	}
	if !minus {
		return arena.ErrInvalidSlot
	}
	if e = s.addmulInt(target.as, target.xs, a1, sol.as, sol.xs,
		s.tmpAs, &s.tmpXs, updateOcc, updateQueue, eqIndex, exceptVar); e != nil {
		return e
	}
	if constant == nil {
		constant = &target.c
	}
	if e = z.AddMul(constant, a1, &sol.c, constant); e != nil {
		return e
	}
	if e = s.tmpAs.SwapRaw(&target.as); e != nil {
		return e
	}
	target.xs, s.tmpXs = s.tmpXs, target.xs
	if e = s.addmulRational(target.bs, target.js, a1, sol.bs, sol.js, s.tmpBs, &s.tmpXs); e != nil {
		return e
	}
	if e = s.tmpBs.SwapRaw(&target.bs); e != nil {
		return e
	}
	target.js, s.tmpXs = s.tmpXs, target.js
	return nil
}

// applySolutionToEquation follows the source reset/schedule/erase-min loop.
// Newly exposed solved variables are inserted by the integer row merge.
func (s *solverSetup) applySolutionToEquation(target *equation) error {
	return s.applySolutionToEquationWithConstant(target, nil)
}

// normalize passes its caller-owned c_prime directly: the source applies each
// substitution to that output before it finishes the coefficient vectors.
func (s *solverSetup) applySolutionToEquationWithConstant(target *equation, constant *numeral.Int) error {
	if e := s.open(); e != nil {
		return e
	}
	if target == nil || target.closed {
		return arena.ErrInvalidSlot
	}
	if e := s.queue.Reset(); e != nil {
		return e
	}
	if e := s.scheduleVarSubst(target.xs); e != nil {
		return e
	}
	for !s.queue.Empty() {
		x, e := s.queue.EraseMin()
		if e != nil {
			return e
		}
		if e := s.applySolutionOnce(uint32(x), target, false, true, nullEquationIndex, nullEquationIndex, constant); e != nil {
			return e
		}
	}
	return nil
}
