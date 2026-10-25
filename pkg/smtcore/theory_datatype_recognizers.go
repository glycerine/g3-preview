// Source: theory_datatype::assign_eh and propagate_recognizer.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Negative recognizers must activate the surviving constructor. In particular,
// a positive literal propagated by an exhaustiveness clause can still be
// irrelevant, so its reconstruction callback has not run yet.
func (t *SMTDatatypeTheory) propagateRecognizer(value ast.ASTHandle) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	node := s.context.enodes[value]
	if node == nil {
		return ast.ErrASTDomain
	}
	known, err := t.classConstructor(node)
	if err != nil || s.Inconsistent() {
		return err
	}
	sort, err := s.manager.GetSort(value)
	if err != nil {
		return err
	}
	constructors, err := s.manager.DatatypeConstructors(sort)
	if err != nil {
		return err
	}
	falseCases := make(map[ast.ASTHandle]SMTLiteral)
	trueCases := make(map[ast.ASTHandle]SMTLiteral)
	trueExpressions := make(map[ast.ASTHandle]ast.ASTHandle)
	guards := make([]SMTLiteral, 0)
	for variable, expression := range s.boolExpr {
		if expression.IsNull() {
			continue
		}
		op, own, err := t.operation(expression)
		if err != nil {
			return err
		}
		if !own || op != ast.ASTDatatypeRecognizer && op != ast.ASTDatatypeIs {
			continue
		}
		argument, err := s.manager.GetArg(expression, 0)
		if err != nil {
			return err
		}
		other := s.context.enodes[argument]
		if other == nil || other.root != node.root {
			continue
		}
		literal := NewSMTLiteral(int32(variable), false)
		assignment, err := s.Assignment(literal)
		if err != nil {
			return err
		}
		if assignment == lbool.Undef {
			continue
		}
		decl, err := s.manager.GetDecl(expression)
		if err != nil {
			return err
		}
		constructor, found, err := s.manager.DatatypeRecognizerConstructor(decl)
		if err != nil {
			return err
		}
		if !found {
			return ast.ErrASTDomain
		}
		if assignment == lbool.False && known != nil && known.declaration == constructor {
			return t.setDatatypeConflict([]SMTLiteral{literal.Negated()}, []SMTEnodePair{{known, other}})
		}
		if assignment == lbool.True {
			trueCases[constructor] = literal
			trueExpressions[constructor] = expression
		} else {
			falseCases[constructor] = literal
		}
		if argument != value {
			equality, err := t.MkEqualityLiteral(value, argument, true)
			if err != nil {
				return err
			}
			guards = append(guards, equality.Negated())
		}
	}
	prior := SMTNullLiteral
	var active ast.ASTHandle
	for _, constructor := range constructors {
		positive, found := trueCases[constructor]
		if !found {
			continue
		}
		if negative, conflict := falseCases[constructor]; conflict {
			clause := append([]SMTLiteral(nil), guards...)
			return s.addTheoryAxiom(t.FamilyID(), append(clause, positive.Negated(), negative))
		}
		if prior != SMTNullLiteral {
			clause := append([]SMTLiteral(nil), guards...)
			return s.addTheoryAxiom(t.FamilyID(), append(clause, prior.Negated(), positive.Negated()))
		}
		prior, active = positive, trueExpressions[constructor]
	}
	if !active.IsNull() {
		return s.markRelevant(active)
	}
	if known != nil {
		return nil
	}
	if len(falseCases) == 0 {
		return nil
	}
	var remaining ast.ASTHandle
	unassigned := 0
	clause := append([]SMTLiteral(nil), guards...)
	for _, constructor := range constructors {
		if literal, found := falseCases[constructor]; found {
			clause = append(clause, literal)
		} else {
			remaining = constructor
			unassigned++
		}
	}
	if unassigned > 1 {
		infinite, err := s.manager.SortIsInfinite(sort)
		if err != nil {
			return err
		}
		if t.lazySplits() == 0 || t.lazySplits() == 1 && !infinite {
			return t.split(value, sort)
		}
		return nil
	}
	if unassigned == 1 {
		literal, expression, err := t.recognizerLiteral(remaining, value)
		if err != nil {
			return err
		}
		if err = s.markRelevant(expression); err != nil {
			return err
		}
		clause = append(clause, literal)
	}
	return s.addTheoryAxiom(t.FamilyID(), clause)
}
