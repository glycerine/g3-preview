// Source: src/math/subpaving/subpaving_t_def.h:651-664,690-721,
// concrete config_mpf monomial definitions.
package subpaving

import (
	"math"
	"sort"

	"github.com/glycerine/g3/pkg/arena"
)

type ArbitraryPower struct {
	Variable uint32
	Degree   uint32
}

type ArbitraryMonomial struct {
	context  *ArbitraryContext
	variable uint32
	powers   []ArbitraryPower
}

// DefineMonomial sorts and combines factors in the context's reusable scratch,
// then copies the merged powers into the owned definition. The source marks a
// monomial integer when any factor variable is integer.
func (c *ArbitraryContext) DefineMonomial(powers []ArbitraryPower) (uint32, error) {
	if err := c.open(); err != nil {
		return 0, err
	}
	if c.root != nil || len(powers) == 0 || len(c.variables) >= math.MaxUint32 {
		return 0, arena.ErrInvalidSlot
	}
	for _, power := range powers {
		if power.Variable >= uint32(len(c.variables)) || power.Degree == 0 {
			return 0, arena.ErrInvalidSlot
		}
	}
	c.powersScratch = append(c.powersScratch[:0], powers...)
	sort.Slice(c.powersScratch, func(i, j int) bool {
		return c.powersScratch[i].Variable < c.powersScratch[j].Variable
	})
	write := 0
	for read := 1; read < len(c.powersScratch); read++ {
		if c.powersScratch[write].Variable == c.powersScratch[read].Variable {
			if math.MaxUint32-c.powersScratch[write].Degree < c.powersScratch[read].Degree {
				return 0, arena.ErrInvalidSlot
			}
			c.powersScratch[write].Degree += c.powersScratch[read].Degree
		} else {
			write++
			c.powersScratch[write] = c.powersScratch[read]
		}
	}
	merged := c.powersScratch[:write+1]
	integer := false
	for _, power := range merged {
		if c.variables[power.Variable] {
			integer = true
			break
		}
	}
	x, err := c.NewVariable(integer)
	if err != nil {
		return 0, err
	}
	m := &ArbitraryMonomial{context: c, variable: x, powers: append([]ArbitraryPower(nil), merged...)}
	if c.monomials == nil {
		c.monomials = make(map[uint32]*ArbitraryMonomial)
	}
	c.monomials[x] = m
	for _, power := range merged {
		c.watchers[power.Variable] = append(c.watchers[power.Variable], x)
	}
	return x, nil
}
