// Source: theory_pb process_antecedent/process_ineq/resolve_conflict and
// hoist_maximal_values/remove_from_lemma. All state belongs to one resolution.
package smtcore

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type smtPBResolution struct {
	theory      *SMTPseudoBooleanTheory
	base, level uint32
	marks       map[int32]int
	numMarks    uint32
	lemma       smtPBArgs
	guards      []SMTLiteral
}

func (r *smtPBResolution) remove(index int) {
	v := r.lemma.terms[index].literal.Var()
	last := len(r.lemma.terms) - 1
	if index != last {
		r.lemma.terms[index] = r.lemma.terms[last]
		r.marks[r.lemma.terms[index].literal.Var()] = index
	}
	r.lemma.terms = r.lemma.terms[:last]
	delete(r.marks, v)
	r.numMarks--
}

func (r *smtPBResolution) antecedent(literal SMTLiteral, coefficient *big.Rat) error {
	s := r.theory.state
	value, err := s.Assignment(literal)
	if err != nil {
		return err
	}
	v := literal.Var()
	level := s.booleanData[v].ScopeLevel
	if value != lbool.False {
		r.lemma.bound.Sub(&r.lemma.bound, coefficient)
		if index, marked := r.marks[v]; marked && r.theory.controls.LearnComplements {
			c := &r.lemma.terms[index].coeff
			c.Sub(c, coefficient)
			if c.Sign() <= 0 {
				r.remove(index)
			}
		}
	} else if level > r.base {
		if index, marked := r.marks[v]; marked {
			r.lemma.terms[index].coeff.Add(&r.lemma.terms[index].coeff, coefficient)
		} else {
			if level == r.level {
				r.numMarks++
			}
			r.marks[v] = len(r.lemma.terms)
			var c big.Rat
			c.Set(coefficient)
			r.lemma.terms = append(r.lemma.terms, smtPBTerm{literal, c})
		}
	}
	return nil
}

func (r *smtPBResolution) inequality(c *smtPBConstraint, consequent SMTLiteral, coefficient *big.Rat) error {
	other := new(big.Rat)
	if consequent == SMTNullLiteral {
		other.SetInt64(1)
	} else {
		for _, term := range c.current().terms {
			if term.literal == consequent {
				other.Set(&term.coeff)
				break
			}
		}
	}
	if other.Sign() <= 0 || coefficient.Sign() <= 0 || !other.IsInt() || !coefficient.IsInt() {
		return ast.ErrASTDomain
	}
	lcm := new(big.Rat).SetInt(smtPBLCM(coefficient.Num(), other.Num()))
	scale := new(big.Rat).Quo(lcm, coefficient)
	if scale.Cmp(big.NewRat(1, 1)) > 0 {
		for i := range r.lemma.terms {
			r.lemma.terms[i].coeff.Mul(&r.lemma.terms[i].coeff, scale)
		}
		r.lemma.bound.Mul(&r.lemma.bound, scale)
	}
	scale.Quo(lcm, other)
	r.lemma.bound.Add(&r.lemma.bound, new(big.Rat).Mul(scale, &c.current().bound))
	for _, term := range c.current().terms {
		if err := r.antecedent(term.literal, new(big.Rat).Mul(scale, &term.coeff)); err != nil {
			return err
		}
	}
	if r.theory.state.booleanData[c.lit().Var()].ScopeLevel > r.base {
		r.guards = append(r.guards, c.lit())
	}
	return nil
}

func (r *smtPBResolution) hoist() {
	for i := 0; i < len(r.lemma.terms); {
		if r.lemma.terms[i].coeff.Cmp(&r.lemma.bound) >= 0 {
			r.guards = append(r.guards, r.lemma.terms[i].literal.Negated())
			last := len(r.lemma.terms) - 1
			r.lemma.terms[i] = r.lemma.terms[last]
			r.lemma.terms = r.lemma.terms[:last]
		} else {
			i++
		}
	}
}

func (t *SMTPseudoBooleanTheory) resolutionExpression(a *smtPBArgs) (expression ast.ASTHandle, result error) {
	s := t.state
	service := s.manager.NumeralService()
	var values []numeral.OwnedRational
	defer func() {
		for i := range values {
			result = errors.Join(result, service.Del(&values[i]))
		}
	}()
	makeValue := func(q *big.Rat) (numeral.OwnedRational, error) {
		v, err := service.NewValue()
		if err != nil {
			return v, err
		}
		values = append(values, v)
		err = service.SetString(&values[len(values)-1], q.RatString())
		return values[len(values)-1], err
	}
	var coefficients []numeral.OwnedRational
	var arguments []ast.ASTHandle
	for _, term := range a.terms {
		q, err := makeValue(&term.coeff)
		if err != nil {
			return expression, err
		}
		coefficients = append(coefficients, q)
		h, err := s.BoolExpression(term.literal.Var())
		if err != nil {
			return expression, err
		}
		if term.literal.Sign() {
			h, err = s.manager.MkNot(h)
			if err != nil {
				return expression, err
			}
		}
		arguments = append(arguments, h)
	}
	k, err := makeValue(&a.bound)
	if err != nil {
		return expression, err
	}
	return s.manager.MkPseudoBooleanGE(coefficients, arguments, &k)
}

func (t *SMTPseudoBooleanTheory) resolutionClause(literals []SMTLiteral) error {
	var owner SMTJustification
	enabled, err := t.state.manager.ProofsEnabled()
	if err != nil {
		return err
	}
	if enabled {
		axiom, err := NewSMTTheoryAxiomJustification(t.state.manager, t.FamilyID(), literals, nil)
		if err != nil {
			return err
		}
		// Auxiliary lemmas own their Go justification through clause lifetime.
		// Keep the source theory-axiom type, including its resolution barrier.
		axiom.heap = true
		owner = axiom
	}
	return t.state.addClauseKindWithJustificationAndDeleteHandler(literals, false, SMTClauseAuxLemma, ast.ASTHandle{}, owner, nil)
}

func (t *SMTPseudoBooleanTheory) resolveConflict(c *smtPBConstraint) (bool, error) {
	if c.equality {
		return false, nil
	}
	s := t.state
	base := s.scopeLevel
	if s.searching {
		base = s.searchUserBaseLevel
	}
	r := smtPBResolution{theory: t, base: base, marks: make(map[int32]int)}
	for _, term := range c.current().terms {
		value, err := s.Assignment(term.literal)
		if err != nil {
			return false, err
		}
		if value == lbool.False && s.booleanData[term.literal.Var()].ScopeLevel > r.level {
			r.level = s.booleanData[term.literal.Var()].ScopeLevel
		}
	}
	if r.level < s.booleanData[c.lit().Var()].ScopeLevel || r.level == base {
		return false, nil
	}
	if err := r.inequality(c, SMTNullLiteral, big.NewRat(1, 1)); err != nil {
		return false, err
	}
	index := len(s.assignedLiterals) - 1
	for r.numMarks > 0 {
		truth := r.lemma.normalize(false)
		if truth == lbool.False {
			break
		}
		if truth == lbool.True {
			return false, nil
		}
		var consequent SMTLiteral
		position := -1
		for index >= 0 {
			consequent = s.assignedLiterals[index]
			index--
			if p, marked := r.marks[consequent.Var()]; marked {
				position = p
				break
			}
		}
		// Source refuses an asynchronous reason whose marked assignment
		// cannot be found earlier in the trail. Never index past the trail.
		if position < 0 {
			return false, nil
		}
		if r.lemma.terms[position].literal != consequent.Negated() {
			return false, ast.ErrASTDomain
		}
		coefficient := new(big.Rat).Set(&r.lemma.terms[position].coeff)
		r.remove(position)
		reason := s.booleanData[consequent.Var()].Justification()
		switch reason.Kind() {
		case SMTBooleanJustificationClause:
			clause, ok := reason.Clause()
			if !ok || clause == nil {
				return false, ast.ErrASTDomain
			}
			if owner := clause.Justification(); owner != nil {
				if _, proof := owner.(*SMTJustificationProofWrapper); !proof {
					r.guards = append(r.guards, consequent)
					continue
				}
			}
			if len(clause.literals) < 2 {
				return false, ast.ErrASTDomain
			}
			start := 1
			if clause.literals[0] != consequent {
				if clause.literals[1] != consequent {
					return false, ast.ErrASTDomain
				}
				if err := r.antecedent(clause.literals[0], coefficient); err != nil {
					return false, err
				}
				start = 2
			}
			for _, literal := range clause.literals[start:] {
				if err := r.antecedent(literal, coefficient); err != nil {
					return false, err
				}
			}
		case SMTBooleanJustificationBinaryClause:
			literal, ok := reason.Literal()
			if !ok {
				return false, ast.ErrASTDomain
			}
			if err := r.antecedent(literal.Negated(), coefficient); err != nil {
				return false, err
			}
		case SMTBooleanJustificationAxiom:
			if s.booleanData[consequent.Var()].ScopeLevel > base {
				r.guards = append(r.guards, consequent)
			}
		case SMTBooleanJustificationObject:
			owner, ok := reason.Justification()
			if !ok || owner == nil {
				return false, ast.ErrASTDomain
			}
			pb, owned := owner.(*smtPBJustification)
			if !consequent.Sign() && owner.FromTheory() == t.FamilyID() && owned && pb.constraint != nil && !pb.constraint.equality && pb.constraint.lit() != consequent {
				if err := r.inequality(pb.constraint, consequent, coefficient); err != nil {
					return false, err
				}
			} else {
				r.guards = append(r.guards, consequent)
			}
		default:
			return false, ast.ErrASTDomain
		}
	}
	r.hoist()
	truth := r.lemma.normalize(false)
	r.lemma.prune(false)
	switch truth {
	case lbool.True:
		return false, nil
	case lbool.False:
		c.propagations++
		t.statistics.conflicts++
		t.statistics.clauses++
		for i := range r.guards {
			r.guards[i] = r.guards[i].Negated()
		}
		return true, t.resolutionClause(r.guards)
	default:
		expression, err := t.resolutionExpression(&r.lemma)
		if err != nil {
			return false, err
		}
		if _, err = t.internalizeAtom(expression, true); err != nil {
			return false, err
		}
		if err = s.markRelevant(expression); err != nil {
			return false, err
		}
		literal, err := s.Literal(expression)
		if err != nil {
			return false, err
		}
		return true, t.addAssign(c, r.guards, literal)
	}
}
