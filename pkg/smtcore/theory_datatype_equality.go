// Source: theory_datatype::assert_eq_axiom and dt_eq_justification.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (t *SMTDatatypeTheory) equalityAxiom(left, right ast.ASTHandle, antecedent SMTLiteral) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	proofs, err := state.manager.ProofsEnabled()
	if err != nil {
		return err
	}
	markLiteral := func(literal SMTLiteral) error {
		if literal == SMTTrueLiteral || literal == SMTFalseLiteral {
			return nil
		}
		expression, err := state.BoolExpression(literal.Var())
		if err != nil {
			return err
		}
		return state.markRelevant(expression)
	}
	clause := func(markAntecedent bool) error {
		literal, err := t.MkEqualityLiteral(left, right, true)
		if err != nil {
			return err
		}
		if err = markLiteral(literal); err != nil {
			return err
		}
		literals := []SMTLiteral{literal}
		if antecedent != SMTNullLiteral {
			if markAntecedent {
				if err = markLiteral(antecedent); err != nil {
					return err
				}
			}
			literals = append(literals, antecedent.Negated())
		}
		return state.addTheoryAxiom(t.FamilyID(), literals)
	}
	if proofs {
		return clause(false)
	}
	if err = state.Internalize(right, false); err != nil {
		return err
	}
	lhs, rhs := state.context.enodes[left], state.context.enodes[right]
	if lhs == nil || rhs == nil {
		return ast.ErrASTDomain
	}
	if antecedent == SMTNullLiteral {
		return state.addEquality(lhs, rhs, NewSMTEqualityAxiom())
	}
	assignment, err := state.Assignment(antecedent)
	if err != nil {
		return err
	}
	if assignment != lbool.True {
		return clause(true)
	}
	owner, err := NewSMTExtTheoryEqualityPropagationJustification(
		state.manager, t.FamilyID(), []SMTLiteral{antecedent}, nil, lhs, rhs, nil)
	if err != nil {
		return err
	}
	// Source dt_eq_justification::get_from_theory returns null: this
	// equality must notify the datatype theory that originated it.
	owner.returnToOwner = true
	// Transfer before merging: a late merge error can leave an equality
	// edge or conflict referencing this scoped owner.
	state.ownedJustifications = append(state.ownedJustifications, owner)
	return state.addEquality(lhs, rhs, NewSMTEqualityJustification(owner))
}
