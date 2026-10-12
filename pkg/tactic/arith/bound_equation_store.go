// Source: src/tactic/arith/bound_propagator.cpp:70-106,140-199.
package arith

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// BoundEquationStore owns the linear constraints and variable watch lists used
// by the bound propagator. Propagation and bound trails belong to later paths.
type BoundEquationStore struct {
	self        *BoundEquationStore
	manager     *numeral.RationalManager
	equations   *LinearEquationManager
	constraints []equationConstraint
	variables   []equationVariable
	trail       []*equationBound
	conflict    uint32
	timestamp   uint32
	closed      bool
}

type equationConstraint struct {
	eq   *LinearEquation
	dead bool
}

type equationVariable struct {
	active  bool
	integer bool
	watches []uint32
	lower   *equationBound
	upper   *equationBound
}

func NewBoundEquationStore(m *numeral.RationalManager) (*BoundEquationStore, error) {
	equations, err := NewLinearEquationManager(m)
	if err != nil {
		return nil, err
	}
	s := &BoundEquationStore{manager: m, equations: equations, conflict: noEquationPosition}
	s.self = s
	return s, nil
}

func (s *BoundEquationStore) open() error {
	if s == nil || s.self != s || s.equations == nil {
		return arena.ErrInvalidSlot
	}
	if s.closed {
		return arena.ErrClosed
	}
	return s.equations.open()
}

// MkVar grows the variable table and reinitializes one source variable slot.
func (s *BoundEquationStore) MkVar(x uint32, integer bool) error {
	if err := s.open(); err != nil {
		return err
	}
	if x == noEquationPosition || uint64(x)+1 > uint64(int(^uint(0)>>1)) {
		return arena.ErrBounds
	}
	if int(x) >= len(s.variables) {
		s.variables = append(s.variables, make([]equationVariable, int(x)+1-len(s.variables))...)
	}
	v := &s.variables[x]
	if v.active {
		return arena.ErrInvalidSlot
	}
	v.active, v.integer, v.watches = true, integer, v.watches[:0]
	v.lower, v.upper = nil, nil
	return nil
}

// DelVar marks every watched equation dead as in the source trail setup.
func (s *BoundEquationStore) DelVar(x uint32) error {
	if err := s.open(); err != nil {
		return err
	}
	if uint64(x) >= uint64(len(s.variables)) || !s.variables[x].active {
		return arena.ErrBounds
	}
	v := &s.variables[x]
	v.active = false
	for _, index := range v.watches {
		s.constraints[index].dead = true
	}
	return nil
}

func (s *BoundEquationStore) MkIntEquation(as []numeral.Int, xs []uint32) (uint32, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	if err := s.checkVariables(xs); err != nil {
		return 0, err
	}
	eq, err := s.equations.MkInt(as, xs, false)
	if err != nil {
		return 0, err
	}
	return s.addEquation(eq)
}

func (s *BoundEquationStore) MkRationalEquation(as []numeral.Rational, xs []uint32) (uint32, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	if err := s.checkVariables(xs); err != nil {
		return 0, err
	}
	eq, err := s.equations.MkRational(as, xs, false)
	if err != nil {
		return 0, err
	}
	return s.addEquation(eq)
}

func (s *BoundEquationStore) checkVariables(xs []uint32) error {
	for _, x := range xs {
		if uint64(x) >= uint64(len(s.variables)) || !s.variables[x].active {
			return arena.ErrBounds
		}
	}
	return nil
}

func (s *BoundEquationStore) addEquation(eq *LinearEquation) (uint32, error) {
	if eq == nil {
		return noEquationPosition, nil
	}
	if uint64(len(s.constraints)) >= uint64(noEquationPosition) {
		return 0, errors.Join(arena.ErrBounds, eq.Close())
	}
	index := uint32(len(s.constraints))
	s.constraints = append(s.constraints, equationConstraint{eq: eq})
	for _, x := range eq.xs {
		s.variables[x].watches = append(s.variables[x].watches, index)
	}
	return index, nil
}

// Constraint returns the borrowed equation and dead mark for inspection.
func (s *BoundEquationStore) Constraint(index uint32) (*LinearEquation, bool, error) {
	if err := s.open(); err != nil {
		return nil, false, err
	}
	if uint64(index) >= uint64(len(s.constraints)) {
		return nil, false, arena.ErrBounds
	}
	c := &s.constraints[index]
	return c.eq, c.dead, nil
}

func (s *BoundEquationStore) Watches(x uint32) ([]uint32, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if uint64(x) >= uint64(len(s.variables)) {
		return nil, arena.ErrBounds
	}
	return append([]uint32(nil), s.variables[x].watches...), nil
}

// DeleteConstraints closes equations in source insertion order and clears all
// watch lists. The source allows this only at scope level zero; this owner has
// no scope/trail state yet.
func (s *BoundEquationStore) DeleteConstraints() error {
	if err := s.open(); err != nil {
		return err
	}
	for i := range s.constraints {
		if err := s.constraints[i].eq.Close(); err != nil {
			return err
		}
	}
	s.constraints = nil
	for i := range s.variables {
		s.variables[i].watches = nil
	}
	return nil
}

func (s *BoundEquationStore) Close() error {
	if s == nil || s.self != s {
		return arena.ErrInvalidSlot
	}
	if s.closed {
		return nil
	}
	for i := len(s.trail) - 1; i >= 0; i-- {
		if err := s.manager.Del(&s.trail[i].value); err != nil {
			return err
		}
	}
	s.trail = nil
	if err := s.DeleteConstraints(); err != nil {
		return err
	}
	if err := s.equations.Close(); err != nil {
		return err
	}
	s.variables = nil
	s.closed = true
	return nil
}
