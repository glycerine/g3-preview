// Source: src/math/subpaving/subpaving_hwf.h:25-44 and
// src/math/subpaving/subpaving_t_def.h:408-514,1641, concrete config_hwf.
package subpaving

import (
	"math"

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
	maxDepth      uint32
	maxNodes      uint32
	maxMemory     uint64
	isInt         []bool
	polynomials   map[uint32]*HardwarePolynomial
	watchers      [][]uint32
	inequalities  []*HardwareInequality
	closed        bool
}

// HardwareContextPolicy contains the unsigned source subpaving parameters.
type HardwareContextPolicy struct {
	Epsilon, MaxBoundPower, RootPrecision uint32
	MaxDepth, MaxNodes, MaxMemoryMB       uint32
}

// NewHardwareContext applies the source epsilon, max_bound and
// nth_root_precision unsigned parameter values.
func NewHardwareContext(m *numeral.HardwareFloatNumeralManager, limit *control.Reslimit, epsilon, maxBoundPower, rootPrecision uint32) (*HardwareContext, error) {
	if m == nil || limit == nil {
		return nil, arena.ErrInvalidSlot
	}
	c := &HardwareContext{numbers: m, limit: limit}
	c.self = c
	if err := c.UpdateParams(HardwareContextPolicy{Epsilon: epsilon, MaxBoundPower: maxBoundPower, RootPrecision: rootPrecision, MaxDepth: 128, MaxNodes: 8192, MaxMemoryMB: math.MaxUint32}); err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateParams preserves the source's field mutation order. A failed numeral
// operation leaves the already reached prefix visible to the owning context.
func (c *HardwareContext) UpdateParams(p HardwareContextPolicy) error {
	if err := c.open(); err != nil {
		return err
	}
	if err := c.updateEpsilon(p.Epsilon); err != nil {
		return err
	}
	if err := c.updateMaxBound(p.MaxBoundPower); err != nil {
		return err
	}
	c.maxDepth = p.MaxDepth
	c.maxNodes = p.MaxNodes
	c.updateMaxMemory(p.MaxMemoryMB)
	return c.updateRootPrecision(p.RootPrecision)
}

func (c *HardwareContext) updateEpsilon(epsilon uint32) error {
	m := c.numbers
	if epsilon != 0 {
		if err := m.SetInt32(&c.epsilon, int32(epsilon)); err != nil {
			return err
		}
		if err := m.Inv(&c.epsilon); err != nil {
			return err
		}
		c.zeroEpsilon = false
	} else {
		if err := m.Reset(&c.epsilon); err != nil {
			return err
		}
		c.zeroEpsilon = true
	}
	return nil
}

func (c *HardwareContext) updateMaxBound(power uint32) error {
	m := c.numbers
	if err := m.SetInt32(&c.maxBound, 10); err != nil {
		return err
	}
	if err := m.Power(&c.maxBound, power, &c.maxBound); err != nil {
		return err
	}
	if err := m.Set(&c.minusMaxBound, &c.maxBound); err != nil {
		return err
	}
	if err := m.Neg(&c.minusMaxBound, &c.minusMaxBound); err != nil {
		return err
	}
	return nil
}

func (c *HardwareContext) updateMaxMemory(megabytes uint32) {
	if megabytes == math.MaxUint32 {
		c.maxMemory = math.MaxUint64
	} else {
		c.maxMemory = uint64(megabytes) * 1024 * 1024
	}
}

func (c *HardwareContext) updateRootPrecision(precision uint32) error {
	if precision == 0 {
		precision = 1
	}
	if err := c.numbers.SetInt32(&c.rootPrecision, int32(precision)); err != nil {
		return err
	}
	return c.numbers.Inv(&c.rootPrecision)
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
