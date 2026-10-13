// Source: src/math/subpaving/subpaving_hwf.h:25-44 and
// src/math/subpaving/subpaving_t_def.h:408-514,1641, concrete config_hwf.
package subpaving

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// HardwareContext owns the four scalar bound-policy numerals of context_hwf.
// Its manager and resource limit are borrowed. Node and search state follow
// as the retained concrete context is ported.
type HardwareContext struct {
	self          *HardwareContext
	numbers       *numeral.HardwareFloatNumeralManager
	limit         *control.Reslimit
	epsilon       numeral.HardwareFloat
	maxBound      numeral.HardwareFloat
	minusMaxBound numeral.HardwareFloat
	rootPrecision numeral.HardwareFloat
	zeroEpsilon   bool
	closed        bool
}

// NewHardwareContext applies the source epsilon, max_bound and
// nth_root_precision unsigned parameter values.
func NewHardwareContext(m *numeral.HardwareFloatNumeralManager, limit *control.Reslimit, epsilon, maxBoundPower, rootPrecision uint32) (*HardwareContext, error) {
	if m == nil || limit == nil {
		return nil, arena.ErrInvalidSlot
	}
	c := &HardwareContext{numbers: m, limit: limit, zeroEpsilon: epsilon == 0}
	c.self = c
	if epsilon != 0 {
		if err := m.SetInt32(&c.epsilon, int32(epsilon)); err != nil {
			return nil, err
		}
		if err := m.Inv(&c.epsilon); err != nil {
			return nil, err
		}
	} else if err := m.Reset(&c.epsilon); err != nil {
		return nil, err
	}
	if err := m.SetInt32(&c.maxBound, 10); err != nil {
		return nil, err
	}
	if err := m.Power(&c.maxBound, maxBoundPower, &c.maxBound); err != nil {
		return nil, err
	}
	if err := m.Set(&c.minusMaxBound, &c.maxBound); err != nil {
		return nil, err
	}
	if err := m.Neg(&c.minusMaxBound, &c.minusMaxBound); err != nil {
		return nil, err
	}
	if rootPrecision == 0 {
		rootPrecision = 1
	}
	if err := m.SetInt32(&c.rootPrecision, int32(rootPrecision)); err != nil {
		return nil, err
	}
	if err := m.Inv(&c.rootPrecision); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *HardwareContext) open() error {
	if c == nil || c.self != c || c.numbers == nil || c.limit == nil {
		return arena.ErrInvalidSlot
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}

func (c *HardwareContext) XnEqY(out, src *HardwareInterval, degree uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	return out.XnEqY(src, degree, &c.rootPrecision, interval.HardwareRoot{Numbers: c.numbers, Limit: c.limit})
}

func (c *HardwareContext) Close() error {
	if c == nil || c.self != c {
		return arena.ErrInvalidSlot
	}
	c.closed = true
	return nil
}
