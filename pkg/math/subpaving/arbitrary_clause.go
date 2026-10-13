// Source: src/math/subpaving/subpaving_t_def.h:773-869, concrete config_mpf
// inequality references, ordinary watched clauses and axiom units.
package subpaving

import (
	"math"
	"sort"

	"github.com/glycerine/g3/pkg/arena"
)

type ArbitraryClause struct {
	context *ArbitraryContext
	atoms   []*ArbitraryInequality
}

type arbitraryUnitClause struct {
	atom  *ArbitraryInequality
	axiom bool
}

func (c *ArbitraryContext) checkInequality(a *ArbitraryInequality) error {
	if a == nil || a.context != c || a.variable >= uint32(len(c.variables)) {
		return arena.ErrInvalidSlot
	}
	return nil
}

// AddUnitClause keeps an inequality alive for root unit assertion.
func (c *ArbitraryContext) AddUnitClause(a *ArbitraryInequality, axiom bool) error {
	if err := c.open(); err != nil {
		return err
	}
	if err := c.checkInequality(a); err != nil {
		return err
	}
	if a.refs == math.MaxUint32 {
		return arena.ErrInvalidSlot
	}
	a.refs++
	c.unitClauses = append(c.unitClauses, arbitraryUnitClause{atom: a, axiom: axiom})
	return nil
}

// AddClause follows the wrapper's non-lemma watched-clause route. A singleton
// is an axiom unit; longer clauses retain input order within each variable.
func (c *ArbitraryContext) AddClause(atoms []*ArbitraryInequality) (*ArbitraryClause, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, arena.ErrInvalidSlot
	}
	counts := make(map[*ArbitraryInequality]uint32, len(atoms))
	for _, a := range atoms {
		if err := c.checkInequality(a); err != nil {
			return nil, err
		}
		counts[a]++
		if a.refs > math.MaxUint32-counts[a] {
			return nil, arena.ErrInvalidSlot
		}
	}
	if len(atoms) == 1 {
		return nil, c.AddUnitClause(atoms[0], true)
	}
	ordered := append([]*ArbitraryInequality(nil), atoms...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].variable < ordered[j].variable })
	clause := &ArbitraryClause{context: c, atoms: ordered}
	for _, a := range ordered {
		a.refs++
	}
	for i, a := range ordered {
		if i == 0 || a.variable != ordered[i-1].variable {
			c.clauseWatchers[a.variable] = append(c.clauseWatchers[a.variable], clause)
		}
	}
	c.clauses = append(c.clauses, clause)
	return clause, nil
}
