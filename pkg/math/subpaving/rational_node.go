// Source: src/math/subpaving/subpaving_t.h:146-316, concrete config_mpq bounds.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// RationalBound owns one exact endpoint. Child nodes borrow ancestor bounds.
type RationalBound struct {
	value numeral.Rational
	open  bool
}

// RationalNode owns bounds it publishes and copies its parent's bound views.
type RationalNode struct {
	self         *RationalNode
	numbers      *numeral.RationalManager
	parent       *RationalNode
	context      *RationalContext
	registered   bool
	lower, upper []*RationalBound
	owned        []*RationalBound
	children     uint32
	conflictVar  uint32
	closed       bool
}

func NewRationalNode(m *numeral.RationalManager, variables uint32, parent *RationalNode) (*RationalNode, error) {
	if m == nil {
		return nil, arena.ErrInvalidSlot
	}
	if parent != nil {
		if err := parent.open(); err != nil {
			return nil, err
		}
		if parent.numbers != m || uint32(len(parent.lower)) != variables || parent.children == math.MaxUint32 {
			return nil, arena.ErrInvalidSlot
		}
	}
	n := &RationalNode{numbers: m, parent: parent, lower: make([]*RationalBound, variables), upper: make([]*RationalBound, variables), conflictVar: math.MaxUint32}
	n.self = n
	if parent != nil {
		copy(n.lower, parent.lower)
		copy(n.upper, parent.upper)
		n.context = parent.context
		n.conflictVar = parent.conflictVar
		parent.children++
	}
	return n, nil
}

func (n *RationalNode) open() error {
	if n == nil || n.self != n || n.numbers == nil {
		return arena.ErrInvalidSlot
	}
	if n.closed {
		return arena.ErrClosed
	}
	return nil
}

func (n *RationalNode) Bounds(variable uint32) (*RationalBound, *RationalBound, error) {
	if err := n.open(); err != nil {
		return nil, nil, err
	}
	if variable >= uint32(len(n.lower)) {
		return nil, nil, arena.ErrInvalidSlot
	}
	return n.lower[variable], n.upper[variable], nil
}

// ConflictingBounds follows the source comparison after both bound views are
// available: reversed endpoints or an open singleton are inconsistent.
func (n *RationalNode) ConflictingBounds(variable uint32) (bool, error) {
	lo, hi, err := n.Bounds(variable)
	if err != nil || lo == nil || hi == nil {
		return false, err
	}
	cmp, err := n.numbers.Compare(&hi.value, &lo.value)
	if err != nil {
		return false, err
	}
	return cmp < 0 || cmp == 0 && (lo.open || hi.open), nil
}

func (n *RationalNode) PushBound(variable uint32, value *numeral.Rational, lower, open bool) error {
	if n != nil && n.context != nil {
		return arena.ErrInvalidSlot
	}
	return n.pushBound(variable, value, lower, open)
}

func (n *RationalNode) pushBound(variable uint32, value *numeral.Rational, lower, open bool) error {
	if _, _, err := n.Bounds(variable); err != nil {
		return err
	}
	b := &RationalBound{value: numeral.NewRational(), open: open}
	if err := n.numbers.Copy(&b.value, value); err != nil {
		return errors.Join(err, n.numbers.Del(&b.value))
	}
	n.owned = append(n.owned, b)
	if lower {
		n.lower[variable] = b
	} else {
		n.upper[variable] = b
	}
	return nil
}

func (n *RationalNode) Close() error {
	if err := n.open(); err != nil {
		if n != nil && n.self == n && n.closed {
			return nil
		}
		return err
	}
	if n.children != 0 {
		return arena.ErrInvalidSlot
	}
	for i := len(n.owned) - 1; i >= 0; i-- {
		if err := n.numbers.Del(&n.owned[i].value); err != nil {
			return err
		}
	}
	n.owned, n.lower, n.upper = nil, nil, nil
	n.closed = true
	if n.parent != nil {
		n.parent.children--
		n.parent = nil
	}
	return nil
}
