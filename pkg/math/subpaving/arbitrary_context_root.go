// Source: src/math/subpaving/subpaving_mpf.h:23-43 and
// src/math/subpaving/subpaving_t_def.h:408-514,1641, concrete config_mpf.
package subpaving

import (
	"errors"
	"math"

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
	variables                                       []bool
	watchers                                        [][]uint32
	clauseWatchers                                  [][]*ArbitraryClause
	watchOrder                                      [][]arbitraryWatch
	polynomials                                     map[uint32]*ArbitraryPolynomial
	polynomialOwners                                []*ArbitraryPolynomial
	monomials                                       map[uint32]*ArbitraryMonomial
	powersScratch                                   []ArbitraryPower
	numBuffer                                       []numeral.ArbitraryFloat
	inequalities                                    []*ArbitraryInequality
	clauses                                         []*ArbitraryClause
	unitClauses                                     []arbitraryUnitClause
	unitHead                                        int
	initStage                                       uint8
	initDefHead                                     uint32
	root                                            *ArbitraryNode
	nodes                                           []*ArbitraryNode
	leafHead, leafTail                              *ArbitraryNode
	bounds                                          []*ArbitraryBound
	queue                                           []*ArbitraryBound
	queueHead                                       int
	timestamp                                       uint64
	numVisited                                      uint64
	arithmeticFailed                                bool
	candidateScratch                                numeral.ArbitraryFloat
	relevanceScratch                                [3]numeral.ArbitraryFloat
	owners                                          [4]*numeral.ScopedArbitraryFloatNumeral
	epsilon, maxBound, minusMaxBound, rootPrecision *numeral.ArbitraryFloat
	zeroEpsilon                                     bool
	closed                                          bool
}

// ArbitraryInequality owns the directed MPF scalar copied by mk_ineq.
type ArbitraryInequality struct {
	context  *ArbitraryContext
	variable uint32
	value    numeral.ArbitraryFloat
	lower    bool
	open     bool
	refs     uint32
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

// NewVariable is the concrete config_mpf mk_var prefix reached by the tactic.
func (c *ArbitraryContext) NewVariable(integer bool) (uint32, error) {
	if err := c.open(); err != nil {
		return 0, err
	}
	if c.root != nil || len(c.variables) >= math.MaxUint32 {
		return 0, arena.ErrInvalidSlot
	}
	x := uint32(len(c.variables))
	c.variables = append(c.variables, integer)
	c.watchers = append(c.watchers, nil)
	c.clauseWatchers = append(c.clauseWatchers, nil)
	c.watchOrder = append(c.watchOrder, nil)
	return x, nil
}

// NewInequality copies the wrapper's rounded scalar into context ownership.
// A stopped copy remains in the context for explicit cleanup.
func (c *ArbitraryContext) NewInequality(variable uint32, value *numeral.ArbitraryFloat, lower, open bool) (*ArbitraryInequality, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if variable >= uint32(len(c.variables)) || value == nil {
		return nil, arena.ErrInvalidSlot
	}
	a := &ArbitraryInequality{context: c, variable: variable, lower: lower, open: open}
	c.inequalities = append(c.inequalities, a)
	if err := c.numbers.Copy(&a.value, value); err != nil {
		return nil, err
	}
	return a, nil
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
	if err := c.numbers.Del(&c.candidateScratch); err != nil {
		return err
	}
	for i := range c.relevanceScratch {
		if err := c.numbers.Del(&c.relevanceScratch[i]); err != nil {
			return err
		}
	}
	c.queue = nil
	c.queueHead = 0
	for len(c.bounds) != 0 {
		b := c.bounds[0]
		if err := c.numbers.Del(&b.value); err != nil {
			return err
		}
		c.bounds[0] = nil
		c.bounds = c.bounds[1:]
	}
	c.bounds = nil
	c.root = nil
	c.nodes = nil
	c.leafHead, c.leafTail = nil, nil
	for len(c.unitClauses) != 0 {
		a := c.unitClauses[0].atom
		if a.refs == 0 {
			return arena.ErrInvalidSlot
		}
		a.refs--
		c.unitClauses = c.unitClauses[1:]
	}
	for len(c.clauses) != 0 {
		clause := c.clauses[0]
		for _, a := range clause.atoms {
			if a.refs == 0 {
				return arena.ErrInvalidSlot
			}
			a.refs--
		}
		c.clauses[0] = nil
		c.clauses = c.clauses[1:]
	}
	c.unitClauses = nil
	c.clauses = nil
	c.clauseWatchers = nil
	c.watchOrder = nil
	for len(c.inequalities) != 0 {
		a := c.inequalities[0]
		if err := c.numbers.Del(&a.value); err != nil {
			return err
		}
		c.inequalities[0] = nil
		c.inequalities = c.inequalities[1:]
	}
	c.inequalities = nil
	for len(c.polynomialOwners) != 0 {
		p := c.polynomialOwners[0]
		for p.closeIndex < len(p.terms) {
			if err := c.numbers.Del(&p.terms[p.closeIndex].coefficient); err != nil {
				return err
			}
			p.closeIndex++
		}
		if err := c.numbers.Del(&p.constant); err != nil {
			return err
		}
		c.polynomialOwners[0] = nil
		c.polynomialOwners = c.polynomialOwners[1:]
	}
	for len(c.numBuffer) != 0 {
		if err := c.numbers.Del(&c.numBuffer[0]); err != nil {
			return err
		}
		c.numBuffer = c.numBuffer[1:]
	}
	c.polynomialOwners = nil
	c.numBuffer = nil
	c.polynomials = nil
	c.monomials = nil
	c.powersScratch = nil
	c.watchers = nil
	c.variables = nil
	c.closed = true
	return nil
}
