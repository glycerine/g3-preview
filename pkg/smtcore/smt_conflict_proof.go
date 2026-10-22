// Source: Boolean and theory-literal portions of
// src/smt/smt_conflict_resolution.cpp mk_conflict_proof() and
// src/smt/smt_justification.cpp mk_proof methods.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
)

type smtConflictProofBuilder struct {
	state   *SMTInternalizerState
	marked  map[int32]struct{}
	literal map[SMTLiteral]ast.ASTHandle
	objects map[SMTJustification]ast.ASTHandle
}

func (b *smtConflictProofBuilder) literalExpression(literal SMTLiteral) (ast.ASTHandle, error) {
	if literal == SMTTrueLiteral {
		return b.state.manager.MkTrue()
	}
	if literal == SMTFalseLiteral {
		return b.state.manager.MkFalse()
	}
	expression, err := b.state.BoolExpression(literal.Var())
	if err != nil || !literal.Sign() {
		return expression, err
	}
	return b.state.manager.MkNot(expression)
}

func (b *smtConflictProofBuilder) clauseProof(clause *SMTClause) (ast.ASTHandle, error) {
	if clause == nil || clause.justification == nil {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return b.objectProof(clause.justification)
}

func (b *smtConflictProofBuilder) literalProof(literal SMTLiteral) (ast.ASTHandle, error) {
	if proof, found := b.literal[literal]; found {
		return proof, nil
	}
	expression, err := b.literalExpression(literal)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	variable := literal.Var()
	justification := b.state.booleanData[variable].Justification()
	if _, hypothesis := b.marked[variable]; hypothesis || justification.Kind() == SMTBooleanJustificationAxiom {
		proof, err := b.state.manager.MkProofHypothesis(expression)
		if err == nil {
			b.literal[literal] = proof
		}
		return proof, err
	}
	proof, err := b.booleanProof(literal, justification)
	if err == nil {
		b.literal[literal] = proof
	}
	return proof, err
}

func (b *smtConflictProofBuilder) booleanProof(consequent SMTLiteral, justification SMTBooleanJustification) (ast.ASTHandle, error) {
	parents := make([]ast.ASTHandle, 0)
	switch justification.Kind() {
	case SMTBooleanJustificationClause:
		clause, ok := justification.Clause()
		if !ok || clause == nil {
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
		proof, err := b.clauseProof(clause)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		parents = append(parents, proof)
		position := -1
		if consequent != SMTFalseLiteral {
			for i := 0; i < len(clause.literals) && i < 2; i++ {
				if clause.literals[i] == consequent {
					position = i
					break
				}
			}
			if position < 0 {
				return ast.ASTHandle{}, ast.ErrASTDomain
			}
		}
		for i, literal := range clause.literals {
			if i == position {
				continue
			}
			antecedent, err := b.literalProof(literal.Negated())
			if err != nil {
				return ast.ASTHandle{}, err
			}
			parents = append(parents, antecedent)
		}
	case SMTBooleanJustificationObject:
		object, ok := justification.Justification()
		if !ok || object == nil {
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
		return b.objectProof(object)
	case SMTBooleanJustificationAxiom:
		fact, err := b.literalExpression(consequent)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return b.state.manager.MkProofHypothesis(fact)
	default:
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	fact, err := b.literalExpression(consequent)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return b.state.manager.MkProofUnitResolutionFact(parents, fact)
}

func (b *smtConflictProofBuilder) literalProofs(literals []SMTLiteral) ([]ast.ASTHandle, error) {
	proofs := make([]ast.ASTHandle, len(literals))
	for i, literal := range literals {
		proof, err := b.literalProof(literal)
		if err != nil {
			return nil, err
		}
		proofs[i] = proof
	}
	return proofs, nil
}

func (b *smtConflictProofBuilder) theoryFact(literals []SMTLiteral) (ast.ASTHandle, error) {
	expressions := make([]ast.ASTHandle, len(literals))
	for i, literal := range literals {
		expression, err := b.literalExpression(literal)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		expressions[i] = expression
	}
	if len(expressions) == 1 {
		return expressions[0], nil
	}
	return b.state.manager.MkOrArgs(expressions)
}

func (b *smtConflictProofBuilder) objectProof(object SMTJustification) (ast.ASTHandle, error) {
	if proof, found := b.objects[object]; found {
		return proof, nil
	}
	var proof ast.ASTHandle
	var err error
	switch value := object.(type) {
	case *SMTJustificationProofWrapper:
		proof, err = value.Proof()
	case *SMTUnitResolutionJustification:
		if value.Antecedent == nil {
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
		first, e := b.objectProof(value.Antecedent)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		parents, e := b.literalProofs(value.Literals)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		proof, err = b.state.manager.MkProofUnitResolution(append([]ast.ASTHandle{first}, parents...))
	case *SMTTheoryAxiomJustification:
		fact, e := b.theoryFact(value.Literals)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		proof, err = b.state.manager.MkProofTheoryLemma(ast.FamilyID(value.theory), fact, nil, value.Parameters())
	case *SMTTheoryPropagationJustification:
		parents, e := b.literalProofs(value.Literals)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		fact, e := b.literalExpression(value.Consequent)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		proof, err = b.state.manager.MkProofTheoryLemma(ast.FamilyID(value.theory), fact, parents, value.Parameters())
	case *SMTTheoryConflictJustification:
		parents, e := b.literalProofs(value.Literals)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		fact, e := b.state.manager.MkFalse()
		if e != nil {
			return ast.ASTHandle{}, e
		}
		proof, err = b.state.manager.MkProofTheoryLemma(ast.FamilyID(value.theory), fact, parents, value.Parameters())
	case *SMTTheoryLemmaJustification:
		expressions, signs, e := value.Expressions()
		if e != nil || len(expressions) != len(signs) {
			if e != nil {
				return ast.ASTHandle{}, e
			}
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
		literals := make([]ast.ASTHandle, len(expressions))
		for i, expression := range expressions {
			literals[i] = expression
			if signs[i] {
				literals[i], e = b.state.manager.MkNot(expression)
				if e != nil {
					return ast.ASTHandle{}, e
				}
			}
		}
		var fact ast.ASTHandle
		if len(literals) == 1 {
			fact = literals[0]
		} else {
			fact, e = b.state.manager.MkOrArgs(literals)
			if e != nil {
				return ast.ASTHandle{}, e
			}
		}
		proof, err = b.state.manager.MkProofTheoryLemma(ast.FamilyID(value.theory), fact, nil, value.parameters.Parameters())
	default:
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	if err == nil {
		b.objects[object] = proof
	}
	return proof, err
}

func (r *SMTConflictResolver) conflictProof(conflict SMTBooleanJustification, notLiteral SMTLiteral, lemma []SMTLiteral) (ast.ASTHandle, error) {
	proofs, err := r.state.manager.ProofsEnabled()
	if err != nil || !proofs {
		return ast.ASTHandle{}, err
	}
	builder := &smtConflictProofBuilder{state: r.state, marked: make(map[int32]struct{}), literal: make(map[SMTLiteral]ast.ASTHandle), objects: make(map[SMTJustification]ast.ASTHandle)}
	for _, literal := range lemma {
		builder.marked[literal.Var()] = struct{}{}
	}
	consequent := SMTFalseLiteral
	if notLiteral != SMTNullLiteral {
		consequent = notLiteral.Negated()
	}
	conflictProof, err := builder.booleanProof(consequent, conflict)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if notLiteral != SMTNullLiteral {
		other, err := builder.literalProof(notLiteral)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		conflictProof, err = r.state.manager.MkProofUnitResolution([]ast.ASTHandle{other, conflictProof})
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	if len(lemma) == 0 {
		return conflictProof, nil
	}
	fact, err := builder.theoryFact(lemma)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return r.state.manager.MkProofLemma(conflictProof, fact)
}
