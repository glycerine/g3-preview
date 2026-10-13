// Source: src/math/subpaving/subpaving.cpp:54 and
// src/math/subpaving/subpaving_t_def.h:473-501, concrete context_hwf.
package subpaving

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/params"
)

// UpdateParamsRef reads each retained parameter immediately before applying
// its source-ordered mutation. A later getter failure leaves the earlier
// context fields changed, as context_t::updt_params does.
func (c *HardwareContext) UpdateParamsRef(pc *params.ParamsContext, p *params.ParamsRef) error {
	if err := c.open(); err != nil {
		return err
	}
	if pc == nil || p == nil {
		return arena.ErrInvalidSlot
	}
	if err := pc.Check(p); err != nil {
		return err
	}
	epsilon := "epsilon"
	v, err := pc.GetUIntName(p, &epsilon, 20)
	if err != nil {
		return err
	}
	if err = c.updateEpsilon(v); err != nil {
		return err
	}
	maxBound := "max_bound"
	v, err = pc.GetUIntName(p, &maxBound, 10)
	if err != nil {
		return err
	}
	if err = c.updateMaxBound(v); err != nil {
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
	c.updateMaxMemory(v)
	rootPrecision := "nth_root_precision"
	v, err = pc.GetUIntName(p, &rootPrecision, 8192)
	if err != nil {
		return err
	}
	return c.updateRootPrecision(v)
}
