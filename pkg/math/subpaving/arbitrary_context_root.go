// Source: src/math/subpaving/subpaving_mpf.h:23-43 and
// src/math/subpaving/subpaving_t_def.h:408-514,1641, concrete config_mpf.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// ArbitraryContext owns the retained config_mpf bound-policy numerals used by
// monomial inverse propagation. The numeral manager and limit are borrowed.
// Other context_t search and node state remains in the enclosing port.
type ArbitraryContext struct {
	self                                            *ArbitraryContext
	numbers                                         *numeral.ArbitraryFloatNumeralManager
	limit                                           *control.Reslimit
	owners                                          [4]*numeral.ScopedArbitraryFloatNumeral
	epsilon, maxBound, minusMaxBound, rootPrecision *numeral.ArbitraryFloat
	zeroEpsilon                                     bool
	closed                                          bool
}

// NewArbitraryContext takes the source epsilon, max_bound exponent and
// nth_root_precision parameter values.
func NewArbitraryContext(m *numeral.ArbitraryFloatNumeralManager, limit *control.Reslimit, epsilon, maxBoundPower, rootPrecision uint32) (_ *ArbitraryContext, err error) {
	if m == nil || limit == nil {
		return nil, arena.ErrInvalidSlot
	}
	c := &ArbitraryContext{numbers: m, limit: limit, zeroEpsilon: epsilon == 0}
	c.self = c
	defer func() {
		if err != nil {
			err = errors.Join(err, c.closeNumerals())
		}
	}()
	for i := range c.owners {
		c.owners[i], err = numeral.NewScopedArbitraryFloatNumeral(m)
		if err != nil {
			return nil, err
		}
	}
	if c.epsilon, err = c.owners[0].Get(); err != nil {
		return nil, err
	}
	if c.maxBound, err = c.owners[1].Get(); err != nil {
		return nil, err
	}
	if c.minusMaxBound, err = c.owners[2].Get(); err != nil {
		return nil, err
	}
	if c.rootPrecision, err = c.owners[3].Get(); err != nil {
		return nil, err
	}
	if epsilon != 0 {
		if err = m.SetInt32(c.epsilon, int32(epsilon)); err != nil {
			return nil, err
		}
		if err = m.Inv(c.epsilon); err != nil {
			return nil, err
		}
	} else if err = m.Reset(c.epsilon); err != nil {
		return nil, err
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
	if err = m.Neg(c.minusMaxBound, c.minusMaxBound); err != nil {
		return nil, err
	}
	if rootPrecision == 0 {
		rootPrecision = 1
	}
	if err = m.SetInt32(c.rootPrecision, int32(rootPrecision)); err != nil {
		return nil, err
	}
	if err = m.Inv(c.rootPrecision); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ArbitraryContext) open() error {
	if c == nil || c.self != c || c.numbers == nil || c.limit == nil {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}

func (c *ArbitraryContext) XnEqY(out, src *ArbitraryInterval, degree uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	return out.XnEqY(src, degree, c.rootPrecision, interval.ArbitraryRoot{Numbers: c.numbers, Limit: c.limit})
}

func (c *ArbitraryContext) closeNumerals() (err error) {
	for i := range c.owners {
		if c.owners[i] != nil {
			err = errors.Join(err, c.owners[i].Close())
		}
	}
	return err
}

func (c *ArbitraryContext) Close() error {
	if c == nil || c.self != c {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return nil
	}
	if err := c.closeNumerals(); err != nil {
		return err
	}
	c.closed = true
	return nil
}
