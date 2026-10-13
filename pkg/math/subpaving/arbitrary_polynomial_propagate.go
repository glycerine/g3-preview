// Source: src/math/subpaving/subpaving_t_def.h:1408-1496, concrete config_mpf
// polynomial propagation over live root-bound interval views.
package subpaving

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// PropagatePolynomial applies the source's single-unbounded-variable rule.
// The stored constant is intentionally absent from the source calculation.
func (c *ArbitraryContext) PropagatePolynomial(variable uint32, n *ArbitraryNode) error {
	if err := c.checkRoot(n, variable); err != nil {
		return err
	}
	if n.conflictVar != math.MaxUint32 {
		return arena.ErrInvalidSlot
	}
	p := c.polynomials[variable]
	if p == nil {
		return arena.ErrInvalidSlot
	}
	c.numVisited++
	p.visited = c.timestamp
	unbounded := uint32(math.MaxUint32)
	if n.lower[variable] == nil && n.upper[variable] == nil {
		unbounded = variable
	}
	for _, term := range p.terms {
		x := term.variable
		if n.lower[x] == nil && n.upper[x] == nil {
			if unbounded != math.MaxUint32 {
				return nil
			}
			unbounded = x
		}
	}
	if unbounded != math.MaxUint32 {
		return c.propagatePolynomialTarget(p, variable, unbounded, n)
	}
	if err := c.propagatePolynomialTarget(p, variable, variable, n); err != nil {
		return err
	}
	for _, term := range p.terms {
		if n.conflictVar != math.MaxUint32 {
			return nil
		}
		if err := c.propagatePolynomialTarget(p, variable, term.variable, n); err != nil {
			return err
		}
	}
	return nil
}

func (c *ArbitraryContext) propagatePolynomialTarget(p *ArbitraryPolynomial, variable, target uint32, n *ArbitraryNode) (err error) {
	r, err := NewArbitraryInterval(c.numbers)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.Close()) }()
	v, err := NewArbitraryInterval(c.numbers)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, v.Close()) }()
	av, err := NewArbitraryInterval(c.numbers)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, av.Close()) }()
	if target == variable {
		if len(p.terms) == 0 {
			return nil
		}
		for i := range p.terms {
			term := &p.terms[i]
			if err = v.SetConstant(n, term.variable); err != nil {
				return err
			}
			if err = av.MulScalar(&term.coefficient, v); err != nil {
				return err
			}
			if i == 0 {
				err = r.SetFrom(av)
			} else {
				err = r.Add(r, av)
			}
			if err != nil {
				return err
			}
		}
	} else {
		if err = v.SetConstant(n, variable); err != nil {
			return err
		}
		if err = r.SetFrom(v); err != nil {
			return err
		}
		var coefficient *arbitraryPolynomialTerm
		for i := range p.terms {
			term := &p.terms[i]
			if term.variable == target {
				coefficient = term
				continue
			}
			if err = v.SetConstant(n, term.variable); err != nil {
				return err
			}
			if err = av.MulScalar(&term.coefficient, v); err != nil {
				return err
			}
			if err = r.Sub(r, av); err != nil {
				return err
			}
		}
		if coefficient == nil {
			return arena.ErrInvalidSlot
		}
		if err = r.DivScalar(r, &coefficient.coefficient); err != nil {
			return err
		}
	}
	lower, upper, err := r.endpoints()
	if err != nil {
		return err
	}
	if !r.lowerInf {
		if _, err = c.propagateCandidateDefinition(n, target, lower, true, r.lowerOpen, variable); err != nil {
			return err
		}
		if n.conflictVar != math.MaxUint32 {
			return nil
		}
	}
	if !r.upperInf {
		_, err = c.propagateCandidateDefinition(n, target, upper, false, r.upperOpen, variable)
	}
	return err
}
