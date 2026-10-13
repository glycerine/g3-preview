// Source: src/math/subpaving/subpaving_t_def.h:802-875,1773-1786,
// concrete config_hwf retained unit inequalities and root assertion.
package subpaving

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrHardwareContextCanceled = errors.New("subpaving: hardware context canceled")

type hardwareUnitClause struct {
	atom  *HardwareInequality
	axiom bool
}

// AddUnitClause retains the same inequality pointer and axiom tag as the
// source's tagged unit vector.
func (c *HardwareContext) AddUnitClause(atom *HardwareInequality, axiom bool) error {
	if err := c.open(); err != nil {
		return err
	}
	if atom == nil || atom.context != c || atom.variable >= uint32(len(c.isInt)) || atom.refs >= (1<<30)-1 {
		return arena.ErrInvalidSlot
	}
	atom.refs++
	c.unitClauses = append(c.unitClauses, hardwareUnitClause{atom: atom, axiom: axiom})
	return nil
}

// AssertUnitClauses resumes at the first unasserted unit after cancellation.
// The source stops as soon as a published bound makes the root inconsistent.
func (c *HardwareContext) AssertUnitClauses(n *HardwareNode) error {
	if err := c.open(); err != nil {
		return err
	}
	if n == nil || n != c.root || n.context != c {
		return arena.ErrInvalidSlot
	}
	if err := n.open(); err != nil {
		return err
	}
	for c.unitHead < len(c.unitClauses) && n.conflictVar == ^uint32(0) {
		if !c.limit.Inc() {
			return ErrHardwareContextCanceled
		}
		unit := c.unitClauses[c.unitHead]
		reason := hardwareReasonAssumption
		if unit.axiom {
			reason = hardwareReasonAxiom
		}
		a := unit.atom
		if err := c.propagateBoundReason(n, a.variable, &a.value, a.lower, a.open, reason); err != nil {
			return err
		}
		c.unitHead++
	}
	return nil
}
