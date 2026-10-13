// Source: src/math/subpaving/subpaving_t_def.h:408-514,528-589,
// concrete config_mpff bound context.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

var ErrFastContextTimestamp = errors.New("subpaving: fast-float bound timestamp exhausted")

// FastContext owns its bound-policy numerals and root node. The numeral
// manager and resource limit remain caller-owned.
type FastContext struct {
	self             *FastContext
	numbers          *numeral.FastFloatManager
	limit            *control.Reslimit
	isInt            []bool
	root             *FastNode
	owned            [4]*numeral.ScopedFastFloat
	epsilon          *numeral.FastFloat
	maxBound         *numeral.FastFloat
	minusMaxBound    *numeral.FastFloat
	rootPrecision    *numeral.FastFloat
	zeroEpsilon      bool
	arithmeticFailed bool
	timestamp        uint64
	queue            []*FastBound
	closed           bool
}

// NewFastContext accepts the source's epsilon, max_bound exponent and
// nth_root_precision parameter values. The caller owns the manager and limit.
func NewFastContext(m *numeral.FastFloatManager, limit *control.Reslimit, isInt []bool, epsilon, maxBoundPower, rootPrecision uint32) (_ *FastContext, err error) {
	if m == nil || limit == nil || len(isInt) > math.MaxUint32 {
		return nil, arena.ErrInvalidSlot
	}
	c := &FastContext{numbers: m, limit: limit, isInt: append([]bool(nil), isInt...), zeroEpsilon: epsilon == 0}
	c.self = c
	defer func() {
		if err != nil {
			if c.root != nil {
				err = errors.Join(err, c.root.Close())
			}
			err = errors.Join(err, c.closeNumerals())
		}
	}()
	for i := range c.owned {
		c.owned[i], err = numeral.NewScopedFastFloat(m)
		if err != nil {
			return nil, err
		}
	}
	if c.epsilon, err = c.owned[0].Get(); err != nil {
		return nil, err
	}
	if c.maxBound, err = c.owned[1].Get(); err != nil {
		return nil, err
	}
	if c.minusMaxBound, err = c.owned[2].Get(); err != nil {
		return nil, err
	}
	if c.rootPrecision, err = c.owned[3].Get(); err != nil {
		return nil, err
	}
	if epsilon != 0 {
		if err = m.SetInt32(c.epsilon, int32(epsilon)); err != nil {
			return nil, err
		}
		if err = m.InvInPlace(c.epsilon); err != nil {
			return nil, err
		}
	}
	if err = m.SetInt32(c.maxBound, 10); err != nil {
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
	if err = m.SetInt32(c.rootPrecision, int32(rootPrecision)); err != nil {
		return nil, err
	}
	if err = m.InvInPlace(c.rootPrecision); err != nil {
		return nil, err
	}
	if c.root, err = NewFastNode(m, uint32(len(isInt)), nil); err != nil {
		return nil, err
	}
	c.root.context = c
	return c, nil
}

func (c *FastContext) open() error {
	if c == nil || c.self != c || c.numbers == nil || c.root == nil {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}

func (c *FastContext) closeNumerals() (err error) {
	for _, owner := range c.owned {
		if owner != nil {
			err = errors.Join(err, owner.Close())
		}
	}
	return err
}

func (c *FastContext) Root() (*FastNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.root, nil
}

func (c *FastContext) NewChild(parent *FastNode) (*FastNode, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if parent == nil || parent.context != c {
		return nil, arena.ErrInvalidSlot
	}
	return NewFastNode(c.numbers, uint32(len(c.isInt)), parent)
}

func (c *FastContext) Close() error {
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
	if err := c.closeNumerals(); err != nil {
		return err
	}
	c.queue = nil
	c.closed = true
	return nil
}
