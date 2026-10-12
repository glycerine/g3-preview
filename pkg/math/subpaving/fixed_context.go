// Source: src/math/subpaving/subpaving_t_def.h:408-514,528-589 and
// 1176-1315, concrete config_mpfx bound path.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrFixedContextTimestamp = errors.New("subpaving: bound timestamp exhausted")

// FixedContext owns the live bound-policy numerals and root node. The numeral
// manager and resource limit are borrowed from the caller.
type FixedContext struct {
	self             *FixedContext
	numbers          *numeral.FixedPointManager
	limit            *control.Reslimit
	isInt            []bool
	root             *FixedNode
	epsilon          *numeral.FixedPoint
	maxBound         *numeral.FixedPoint
	minusMaxBound    *numeral.FixedPoint
	rootPrecision    *numeral.FixedPoint
	zeroEpsilon      bool
	arithmeticFailed bool
	timestamp        uint64
	queue            []*FixedBound
	queueHead        int
	numVisited       uint64
	polynomials      map[uint32]*FixedPolynomial
	monomials        map[uint32]*FixedMonomial
	watchers         [][]uint32
	closed           bool
}

// NewFixedContext uses the source parameter values directly. The source
// defaults are epsilon=20, maxBoundPower=10 and rootPrecision=8192.
func NewFixedContext(m *numeral.FixedPointManager, limit *control.Reslimit, isInt []bool, epsilon, maxBoundPower, rootPrecision uint32) (_ *FixedContext, err error) {
	if m == nil || limit == nil || len(isInt) > math.MaxUint32 {
		return nil, arena.ErrInvalidSlot
	}
	c := &FixedContext{numbers: m, limit: limit, isInt: append([]bool(nil), isInt...), zeroEpsilon: epsilon == 0, polynomials: make(map[uint32]*FixedPolynomial), monomials: make(map[uint32]*FixedMonomial), watchers: make([][]uint32, len(isInt))}
	c.self = c
	defer func() {
		if err != nil {
			if c.root != nil {
				err = errors.Join(err, c.root.Close())
			}
			err = errors.Join(err, c.closeNumerals())
		}
	}()
	if c.epsilon, err = m.New(); err != nil {
		return nil, err
	}
	if c.maxBound, err = m.New(); err != nil {
		return nil, err
	}
	if c.minusMaxBound, err = m.New(); err != nil {
		return nil, err
	}
	if c.rootPrecision, err = m.New(); err != nil {
		return nil, err
	}
	if epsilon != 0 {
		if err = m.SetInt64(c.epsilon, int64(epsilon)); err != nil {
			return nil, err
		}
		if err = m.Inv(c.epsilon); err != nil {
			return nil, err
		}
	}
	if err = m.SetInt64(c.maxBound, 10); err != nil {
		return nil, err
	}
	if err = m.Power(c.maxBound, maxBoundPower, c.maxBound); err != nil {
		return nil, err
	}
	if err = m.Copy(c.minusMaxBound, c.maxBound); err != nil {
		return nil, err
	}
	if err = m.Neg(c.minusMaxBound); err != nil {
		return nil, err
	}
	if rootPrecision == 0 {
		rootPrecision = 1
	}
	if err = m.SetInt64(c.rootPrecision, int64(rootPrecision)); err != nil {
		return nil, err
	}
	if err = m.Inv(c.rootPrecision); err != nil {
		return nil, err
	}
	if c.root, err = NewFixedNode(m, uint32(len(isInt)), nil); err != nil {
		return nil, err
	}
	c.root.context = c
	return c, nil
}

func (c *FixedContext) open() error {
	if c == nil || c.self != c || c.numbers == nil || c.root == nil {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}

func (c *FixedContext) closeNumerals() (err error) {
	for _, v := range []*numeral.FixedPoint{c.epsilon, c.maxBound, c.minusMaxBound, c.rootPrecision} {
		if v != nil {
			err = errors.Join(err, v.Close())
		}
	}
	return err
}

func (c *FixedContext) Root() (*FixedNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.root, nil
}

func (c *FixedContext) NewChild(parent *FixedNode) (*FixedNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if parent == nil || parent.context != c {
		return nil, arena.ErrInvalidSlot
	}
	return NewFixedNode(c.numbers, uint32(len(c.isInt)), parent)
}

func (c *FixedContext) XnEqY(out, src *FixedInterval, degree uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	return out.XnEqY(src, degree, c.rootPrecision, interval.FixedRoot{Numbers: c.numbers, Limit: c.limit})
}

func (c *FixedContext) Close() error {
	if c == nil || c.self != c {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return nil
	}
	if c.root == nil {
		return arena.ErrInvalidSlot
	}
	if err := c.root.Close(); err != nil {
		return err
	}
	for x := range c.isInt {
		if p := c.polynomials[uint32(x)]; p != nil {
			if err := p.close(); err != nil {
				return err
			}
		}
	}
	if err := c.closeNumerals(); err != nil {
		return err
	}
	c.queue = nil
	c.queueHead = 0
	c.closed = true
	return nil
}
