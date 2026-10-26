// Source: src/smt/theory_lra.cpp:1303-1485,1733-1811. Live basic and LP propagation.
package smtcore

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/lp"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func (t *SMTLRATheory) setLRAConflict() error {
	evidence, err := t.numeric.infeasibilityEvidence()
	if err != nil {
		return err
	}
	var core []SMTLiteral
	for _, e := range evidence {
		core = append(core, e.literal)
	}
	return t.state.SetTheoryConflict(t.FamilyID(), core)
}

func (t *SMTLRATheory) impliedBoundLiteral(kind lp.LConstraintKind, value *big.Rat, b *smtLRABound) SMTLiteral {
	cmp := value.Cmp(&b.value)
	if (kind == lp.LConstraintLE || kind == lp.LConstraintLT) && b.upper && cmp <= 0 ||
		(kind == lp.LConstraintGE || kind == lp.LConstraintGT) && !b.upper && cmp >= 0 {
		return NewSMTLiteral(b.boolean, false)
	}
	if !b.upper && (kind == lp.LConstraintLE && cmp < 0 || kind == lp.LConstraintLT && cmp <= 0) ||
		b.upper && (kind == lp.LConstraintGE && cmp > 0 || kind == lp.LConstraintGT && cmp >= 0) {
		return NewSMTLiteral(b.boolean, true)
	}
	return SMTNullLiteral
}

// Numerical column and term indices occupy different source directories.
func (t *SMTLRATheory) variableForLPIndex(index uint32) SMTTheoryVar {
	vars := t.numeric.variables
	if t.numeric.solver.IsTerm(index) {
		vars = t.numeric.terms
		index -= lp.LARTermStartIndex
	}
	if uint64(index) >= uint64(len(vars)) {
		return SMTNullTheoryVar
	}
	return vars[index]
}

type smtLRABoundPropagation struct {
	owner *SMTLRATheory
	core  []SMTLiteral
}

func (p *smtLRABoundPropagation) BoundIsInteresting(index uint32, kind lp.LConstraintKind, value *numeral.Rational) (bool, error) {
	t := p.owner
	v := t.variableForLPIndex(index)
	if v == SMTNullTheoryVar || int(v) >= len(t.bounds) || t.unassigned[v] == 0 {
		return false, nil
	}
	r, err := t.numeric.rational(value)
	if err != nil {
		return false, err
	}
	for _, bound := range t.bounds[v] {
		assignment, err := t.state.Assignment(NewSMTLiteral(bound.boolean, false))
		if err != nil {
			return false, err
		}
		if assignment == lbool.Undef && t.impliedBoundLiteral(kind, r, bound) != SMTNullLiteral {
			return true, nil
		}
	}
	return false, nil
}

func (p *smtLRABoundPropagation) Consume(_ *numeral.Rational, witness uint32) error {
	// Native local_bound_propagator intentionally ignores the coefficient.
	// Definitions are hard constraints; immediate LRA has only inequality sources.
	if witness == smtLRANoIndex {
		return nil
	}
	s := p.owner.numeric
	if uint64(witness) >= uint64(len(s.inequalities)) || s.inequalities[witness] == SMTNullLiteral {
		return ast.ErrASTDomain
	}
	p.core = append(p.core, s.inequalities[witness])
	return nil
}

func (t *SMTLRATheory) propagateLPBounds() (err error) {
	if t.config.BoundProp == 0 {
		return nil
	}
	defer func() {
		if err != nil {
			t.numeric.stopped = true
		}
	}()
	hook := &smtLRABoundPropagation{owner: t}
	t.propagator = t.numeric.solver.NewBoundPropagator()
	t.propagator.Interest, t.propagator.Evidence = hook, hook
	if err = t.numeric.solver.PropagateBoundsForTouchedRows(t.propagator); err != nil {
		return err
	}
	canceled, err := t.state.manager.Canceled()
	if err != nil {
		return err
	}
	if !canceled {
		if t.numeric.solver.Bounds.Status.Value == lp.StatusInfeasible {
			if err = t.setLRAConflict(); err != nil {
				return err
			}
		} else {
			for _, implied := range t.propagator.FoundBounds {
				canceled, err = t.state.manager.Canceled()
				if err != nil {
					return err
				}
				if canceled || t.state.inconsistent {
					break
				}
				v := t.variableForLPIndex(implied.Column)
				if v == SMTNullTheoryVar || int(v) >= len(t.bounds) || t.unassigned[v] == 0 {
					continue
				}
				r, err := t.numeric.rational(&implied.Bound)
				if err != nil {
					return err
				}
				first := true
				for _, bound := range t.bounds[v] {
					assignment, err := t.state.Assignment(NewSMTLiteral(bound.boolean, false))
					if err != nil {
						return err
					}
					if assignment != lbool.Undef {
						continue
					}
					literal := t.impliedBoundLiteral(implied.Kind(), r, bound)
					if literal == SMTNullLiteral {
						continue
					}
					t.numeric.solver.Settings.Stats.ImpliedBounds++
					if first {
						first = false
						hook.core = nil
						if err = t.numeric.solver.ExplainImpliedBound(implied, t.propagator); err != nil {
							return err
						}
					}
					t.unassigned[v]--
					t.lpPropagations++
					if err = t.assignLRABound(literal, hook.core, nil); err != nil {
						return err
					}
				}
			}
		}
	}
	if err = t.propagator.Close(); err != nil {
		return err
	}
	t.propagator = nil
	return nil
}

func (t *SMTLRATheory) propagateBasicBounds() error {
	for _, bv := range t.boundChecks {
		b := t.boundByBoolean[bv]
		if b == nil {
			return ast.ErrASTDomain
		}
		assignment, err := t.state.Assignment(NewSMTLiteral(bv, false))
		if err != nil {
			return err
		}
		if err = t.propagateBasicBound(b, assignment == lbool.True); err != nil {
			return err
		}
		if t.state.inconsistent {
			break
		}
	}
	t.boundChecks = nil
	return nil
}

func (t *SMTLRATheory) propagateBasicBound(b *smtLRABound, truth bool) (result error) {
	if t.config.BoundProp == 0 || len(t.bounds[b.variable]) == 1 || t.unassigned[b.variable] == 0 {
		return nil
	}
	findLower := truth != b.upper
	var nearest *smtLRABound
	for _, other := range t.bounds[b.variable] {
		if other == b {
			continue
		}
		cmp := other.value.Cmp(&b.value)
		// False bounds have +/-epsilon; comparing a rational against that
		// infinitesimal reproduces these strict/equal threshold cases.
		eligible := cmp < 0 || !truth && cmp == 0
		if !findLower {
			eligible = cmp > 0 || !truth && cmp == 0
		}
		if !eligible {
			continue
		}
		if nearest == nil || findLower && other.value.Cmp(&nearest.value) > 0 || !findLower && other.value.Cmp(&nearest.value) < 0 {
			nearest = other
		}
	}
	if nearest == nil {
		return nil
	}
	literal := NewSMTLiteral(nearest.boolean, nearest.upper == findLower)
	t.unassigned[b.variable]--
	t.basicPropagations++
	proofs, err := t.state.manager.ProofsEnabled()
	if err != nil {
		return err
	}
	var parameters []ast.ASTParameter
	if proofs {
		antecedents := smtArithmeticAntecedents{literalCoefficients: []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)}}
		parameters, err = antecedents.parameters(t.state.manager, "farkas")
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, smtArithmeticReleaseParameters(t.state.manager, parameters)) }()
	}
	// Preserve the source quirk: m_core contains lit2, not lit1. In the
	// small-lemma path this creates a tautology, not a new implication.
	return t.assignLRABound(literal, []SMTLiteral{literal}, parameters)
}

func (t *SMTLRATheory) assignLRABound(literal SMTLiteral, core []SMTLiteral, parameters []ast.ASTParameter) error {
	s := t.state
	if uint64(len(core)) < uint64(t.config.SmallLemmaSize) {
		clause := make([]SMTLiteral, 0, len(core)+1)
		for _, premise := range core {
			clause = append(clause, premise.Negated())
		}
		clause = append(clause, literal)
		var justification *SMTTheoryLemmaJustification
		proofs, err := s.manager.ProofsEnabled()
		if err != nil {
			return err
		}
		if proofs {
			justification, err = NewSMTTheoryLemmaJustification(s.manager, t.FamilyID(), clause, s.boolExpr, parameters)
			if err != nil {
				return err
			}
		}
		var owner SMTJustification
		if justification != nil {
			owner = justification
		}
		beforeClauses, beforeOwners := len(s.clauses), len(s.ownedJustifications)
		err = s.addClauseKindWithJustificationAndDeleteHandler(clause, false, SMTClauseAuxLemma, ast.ASTHandle{}, owner, nil)
		if justification != nil && len(s.clauses) == beforeClauses && len(s.ownedJustifications) == beforeOwners {
			return errors.Join(err, justification.Release(s.manager))
		}
		return err
	}
	owner, err := NewSMTExtTheoryPropagationJustification(s.manager, t.FamilyID(), core, nil, literal, parameters)
	if err != nil {
		return err
	}
	justification, err := NewSMTBooleanJustification(owner)
	if err != nil {
		return errors.Join(err, owner.Release(s.manager))
	}
	if err = s.AssignLiteral(literal, justification); err != nil {
		return errors.Join(err, owner.Release(s.manager))
	}
	s.ownedJustifications = append(s.ownedJustifications, owner)
	return nil
}
