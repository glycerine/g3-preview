// Source: src/math/subpaving/subpaving_t_def.h:291-342,525-569, concrete config_mpq.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// RationalContext owns the reached config_mpq variable and node-bound prefix.
// Constraint registration, propagation and search are still separate work.
type RationalContext struct {
	self         *RationalContext
	numbers      *numeral.RationalManager
	limit        *control.Reslimit
	runtime      *memory.Runtime
	process      *memory.ProcessAccounting
	maxMemory    uint64
	variables    []bool
	inequalities []*RationalInequality
	unitClauses  []rationalUnitClause
	unitHead     int
	root         *RationalNode
	nodes        []*RationalNode
	leafHead     *RationalNode
	leafTail     *RationalNode
	numBounds    uint64
	numConflicts uint64
	timestamp    uint64
	closed       bool
}

func NewRationalContext(m *numeral.RationalManager, limit *control.Reslimit) (*RationalContext, error) {
	if m == nil || limit == nil {
		return nil, arena.ErrInvalidSlot
	}
	c := &RationalContext{numbers: m, limit: limit, runtime: m.RuntimeOwner(), maxMemory: math.MaxUint64}
	c.self = c
	if c.runtime != nil {
		c.process = c.runtime.ProcessOwner()
	}
	return c, nil
}

func (c *RationalContext) open() error {
	if c == nil || c.self != c || c.numbers == nil || c.limit == nil {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}

func (c *RationalContext) NewVariable(integer bool) (uint32, error) {
	if err := c.open(); err != nil {
		return 0, err
	}
	if c.root != nil || len(c.variables) >= 1<<29 {
		return 0, arena.ErrInvalidSlot
	}
	x := uint32(len(c.variables))
	c.variables = append(c.variables, integer)
	return x, nil
}

func (c *RationalContext) Root() (*RationalNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if c.root != nil {
		if err := c.root.open(); err != nil {
			return nil, err
		}
		return c.root, nil
	}
	n, err := NewRationalNode(c.numbers, uint32(len(c.variables)), nil)
	if err != nil {
		return nil, err
	}
	n.context, n.registered = c, true
	c.root = n
	c.nodes = append(c.nodes, n)
	c.pushLeafFront(n)
	return n, nil
}

func (c *RationalContext) NewChild(parent *RationalNode) (*RationalNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if c.root == nil || parent == nil || parent.context != c || !parent.registered {
		return nil, arena.ErrInvalidSlot
	}
	n, err := NewRationalNode(c.numbers, uint32(len(c.variables)), parent)
	if err != nil {
		return nil, err
	}
	n.registered = true
	c.nodes = append(c.nodes, n)
	c.pushLeafFront(n)
	return n, nil
}

// PublishBound follows mk_bound's exact rational integer normalization before
// publishing the endpoint to this node's copied bound view.
func (c *RationalContext) PublishBound(n *RationalNode, variable uint32, value *numeral.Rational, lower, open bool) (err error) {
	return c.publishBoundReason(n, variable, value, lower, open, false)
}

func (c *RationalContext) publishBoundReason(n *RationalNode, variable uint32, value *numeral.Rational, lower, open, axiom bool) (err error) {
	if err = c.open(); err != nil {
		return err
	}
	if n == nil || n.context != c || !n.registered || variable >= uint32(len(c.variables)) || n.conflictVar != math.MaxUint32 {
		return arena.ErrInvalidSlot
	}
	if _, _, err = n.Bounds(variable); err != nil {
		return err
	}
	if value == nil {
		return arena.ErrInvalidSlot
	}
	if c.timestamp == math.MaxUint64 {
		return ErrRationalContextTimestamp
	}
	c.numBounds++
	if !c.variables[variable] {
		return c.publishNormalized(n, variable, value, lower, open, axiom)
	}
	adjusted := numeral.NewRational()
	defer func() { err = errors.Join(err, c.numbers.Del(&adjusted)) }()
	integer, err := c.numbers.IsInt(value)
	if err != nil {
		return err
	}
	if !integer {
		open = false
	}
	if lower {
		err = c.numbers.Ceil(value, &adjusted)
	} else {
		err = c.numbers.Floor(value, &adjusted)
	}
	if err != nil {
		return err
	}
	if open {
		open = false
		if lower {
			err = c.numbers.Inc(&adjusted)
		} else {
			err = c.numbers.Dec(&adjusted)
		}
		if err != nil {
			return err
		}
	}
	return c.publishNormalized(n, variable, &adjusted, lower, open, axiom)
}

func (c *RationalContext) publishNormalized(n *RationalNode, variable uint32, value *numeral.Rational, lower, open, axiom bool) error {
	if err := n.pushBoundAt(variable, value, lower, open, c.timestamp, axiom); err != nil {
		return err
	}
	conflict, err := n.ConflictingBounds(variable)
	if err != nil {
		return err
	}
	if conflict {
		n.conflictVar = variable
		c.numConflicts++
		c.removeLeaf(n)
	}
	c.timestamp++
	if c.timestamp == math.MaxUint64 {
		return ErrRationalContextTimestamp
	}
	return nil
}

func (c *RationalContext) Close() error {
	if err := c.open(); err != nil {
		if c != nil && c.self == c && c.closed {
			return nil
		}
		return err
	}
	for i := len(c.nodes) - 1; i >= 0; i-- {
		if err := c.nodes[i].Close(); err != nil {
			return err
		}
	}
	for i := len(c.inequalities) - 1; i >= 0; i-- {
		if !c.inequalities[i].closed {
			if err := c.numbers.Del(&c.inequalities[i].value); err != nil {
				return err
			}
			c.inequalities[i].closed = true
		}
	}
	c.nodes, c.root, c.variables = nil, nil, nil
	c.leafHead, c.leafTail = nil, nil
	c.inequalities, c.unitClauses = nil, nil
	c.closed = true
	return nil
}
