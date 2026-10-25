// Source: theory_datatype::mk_split and datatype::util constructor preferences.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtDatatypeRecognizer struct {
	expression ast.ASTHandle
	literal    SMTLiteral
}

func (t *SMTDatatypeTheory) recognizerSlots(value ast.ASTHandle) (map[ast.ASTHandle]smtDatatypeRecognizer, error) {
	s, err := t.State()
	if err != nil {
		return nil, err
	}
	node := s.context.enodes[value]
	if node == nil {
		return nil, ast.ErrASTDomain
	}
	slots := make(map[ast.ASTHandle]smtDatatypeRecognizer)
	for variable, expression := range s.boolExpr {
		if expression.IsNull() {
			continue
		}
		op, own, err := t.operation(expression)
		if err != nil {
			return nil, err
		}
		if !own || op != ast.ASTDatatypeRecognizer && op != ast.ASTDatatypeIs {
			continue
		}
		argument, err := s.manager.GetArg(expression, 0)
		if err != nil {
			return nil, err
		}
		other := s.context.enodes[argument]
		if other == nil || other.root != node.root {
			continue
		}
		decl, err := s.manager.GetDecl(expression)
		if err != nil {
			return nil, err
		}
		constructor, found, err := s.manager.DatatypeRecognizerConstructor(decl)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, ast.ErrASTDomain
		}
		if _, exists := slots[constructor]; !exists {
			slots[constructor] = smtDatatypeRecognizer{expression, NewSMTLiteral(int32(variable), false)}
		}
	}
	return slots, nil
}

func (t *SMTDatatypeTheory) split(value, sort ast.ASTHandle) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	if t.utility == nil {
		t.utility, err = ast.NewDatatypeUtil(s.manager)
		if err != nil {
			return err
		}
	}
	preferred, found, err := t.utility.NonRecursiveConstructor(sort)
	if err != nil {
		return err
	}
	if !found {
		return ast.ErrASTDomain
	}
	slots, err := t.recognizerSlots(value)
	if err != nil {
		return err
	}
	t.statistics.splits++
	// Existing recognizers are activated or awaited without creating another
	// split. A false preferred recognizer advances in declaration order.
	activate := func(slot smtDatatypeRecognizer) (bool, error) {
		if !s.isRelevant(slot.expression) {
			return true, s.markRelevant(slot.expression)
		}
		assignment, err := s.Assignment(slot.literal)
		return assignment != lbool.False, err
	}
	constructor := preferred
	if slot, exists := slots[preferred]; exists {
		stop, err := activate(slot)
		if err != nil || stop {
			return err
		}
		constructors, err := s.manager.DatatypeConstructors(sort)
		if err != nil {
			return err
		}
		constructor = ast.ASTHandle{}
		for _, candidate := range constructors {
			slot, exists := slots[candidate]
			if !exists {
				constructor = candidate
				break
			}
			stop, err := activate(slot)
			if err != nil || stop {
				return err
			}
		}
		if constructor.IsNull() {
			return nil
		}
	}
	literal, expression, err := t.recognizerLiteral(constructor, value)
	if err != nil {
		return err
	}
	if _, err = s.setScopedTrueFirst(literal.Var()); err != nil {
		return err
	}
	return s.markRelevant(expression)
}

func (t *SMTDatatypeTheory) Close() error {
	if t == nil {
		return ast.ErrASTHandle
	}
	if t.utility != nil {
		if err := t.utility.Close(); err != nil {
			return err
		}
		t.utility = nil
	}
	return nil
}
