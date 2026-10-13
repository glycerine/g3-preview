// Source: src/math/subpaving/subpaving_t_def.h:473-513, concrete config_mpf.
package subpaving

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/params"
)

// UpdateParamsRef reads a source parameter immediately before applying its
// mutation. If a later step fails, completed earlier fields remain changed.
func (c *ArbitraryContext) UpdateParamsRef(pc *params.ParamsContext, p *params.ParamsRef) error {
	if err := c.open(); err != nil {
		return err
	}
	if pc == nil || p == nil {
		return arena.ErrInvalidSlot
	}
	if err := pc.Check(p); err != nil {
		return err
	}
	m := c.numbers
	epsilon := "epsilon"
	v, err := pc.GetUIntName(p, &epsilon, 20)
	if err != nil {
		return err
	}
	if v != 0 {
		if err = m.SetInt32(c.epsilon, int32(v)); err != nil {
			return err
		}
		if err = m.Inv(c.epsilon); err != nil {
			return err
		}
		c.zeroEpsilon = false
	} else {
		if err = m.Reset(c.epsilon); err != nil {
			return err
		}
		c.zeroEpsilon = true
	}
	maxBound := "max_bound"
	v, err = pc.GetUIntName(p, &maxBound, 10)
	if err != nil {
		return err
	}
	if err = m.SetInt32(c.maxBound, 10); err != nil {
		return err
	}
	if err = m.Power(c.maxBound, v, c.maxBound); err != nil {
		return err
	}
	if err = m.Copy(c.minusMaxBound, c.maxBound); err != nil {
		return err
	}
	if err = m.Neg(c.minusMaxBound, c.minusMaxBound); err != nil {
		return err
	}
	maxDepth := "max_depth"
	v, err = pc.GetUIntName(p, &maxDepth, 128)
	if err != nil {
		return err
	}
	c.maxDepth = v
	maxNodes := "max_nodes"
	v, err = pc.GetUIntName(p, &maxNodes, 8192)
	if err != nil {
		return err
	}
	c.maxNodes = v
	maxMemory := "max_memory"
	v, err = pc.GetUIntName(p, &maxMemory, math.MaxUint32)
	if err != nil {
		return err
	}
	if v == math.MaxUint32 {
		c.maxMemory = math.MaxUint64
	} else {
		c.maxMemory = uint64(v) * 1024 * 1024
	}
	rootPrecision := "nth_root_precision"
	v, err = pc.GetUIntName(p, &rootPrecision, 8192)
	if err != nil {
		return err
	}
	if v == 0 {
		v = 1
	}
	if err = m.SetInt32(c.rootPrecision, int32(v)); err != nil {
		return err
	}
	return m.Inv(c.rootPrecision)
}

// The option declarations are shared by the two retained concrete context
// instantiations of the same C++ template.
func CollectArbitraryParamDescrs(d *params.ParamDescrs) error {
	return CollectHardwareParamDescrs(d)
}
