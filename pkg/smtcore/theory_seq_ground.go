// Source: src/smt/theory_seq.cpp add_length_axiom, add_indexof_axiom,
// add_replace_axiom, add_extract_axiom and add_at_axiom concrete cases.
package smtcore

import (
	"math/big"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *SMTSequenceTheory) groundInteger(expression ast.ASTHandle) (*big.Int, bool, error) {
	state, err := t.State()
	if err != nil {
		return nil, false, err
	}
	text, integer, matched, err := state.manager.ArithmeticNumeralText(expression)
	if err != nil || !matched || !integer {
		return nil, false, err
	}
	value, ok := new(big.Int).SetString(text, 10)
	if !ok {
		return nil, false, ast.ErrASTDomain
	}
	return value, true, nil
}

func (t *SMTSequenceTheory) pushGroundValue(expression, value ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	equality, err := state.manager.MkEq(expression, value)
	if err != nil {
		return false, err
	}
	t.trackInternalTerms(expression)
	return true, t.axioms.Push(equality)
}

func (t *SMTSequenceTheory) groundStringArgument(expression ast.ASTHandle, index uint32) (string, bool, error) {
	state, err := t.State()
	if err != nil {
		return "", false, err
	}
	argument, err := state.manager.GetArg(expression, index)
	if err != nil {
		return "", false, err
	}
	return t.syntacticGroundString(argument, make(map[ast.ASTHandle]struct{}))
}

func (t *SMTSequenceTheory) syntacticGroundString(expression ast.ASTHandle, visiting map[ast.ASTHandle]struct{}) (string, bool, error) {
	state, err := t.State()
	if err != nil {
		return "", false, err
	}
	if _, found := visiting[expression]; found {
		return "", false, nil
	}
	visiting[expression] = struct{}{}
	defer delete(visiting, expression)
	if text, matched, textErr := state.manager.SequenceStringValue(expression); textErr != nil || matched {
		return text, matched, textErr
	}
	if element, unit, unitErr := state.manager.SequenceUnitArgument(expression); unitErr != nil {
		return "", false, unitErr
	} else if unit {
		value, matched, valueErr := state.manager.SequenceUnitByteElementValue(element)
		if valueErr != nil || !matched {
			return "", false, valueErr
		}
		return string([]byte{value}), true, nil
	}
	left, right, concat, err := state.manager.SequenceConcatParts(expression)
	if err != nil || !concat {
		return "", false, err
	}
	leftText, leftConcrete, err := t.syntacticGroundString(left, visiting)
	if err != nil || !leftConcrete {
		return "", false, err
	}
	rightText, rightConcrete, err := t.syntacticGroundString(right, visiting)
	if err != nil || !rightConcrete {
		return "", false, err
	}
	return leftText + rightText, true, nil
}

func (t *SMTSequenceTheory) addGroundOperationAxiom(expression ast.ASTHandle, operation int32) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	if operation == ast.ASTSequenceOpUnit || operation == ast.ASTSequenceOpConcat {
		text, concrete, textErr := t.syntacticGroundString(expression, make(map[ast.ASTHandle]struct{}))
		if textErr != nil || !concrete {
			return false, textErr
		}
		value, makeErr := state.manager.MkString(text)
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, value)
	}
	if operation == 9 {
		text, concrete, textErr := t.groundStringArgument(expression, 0)
		if textErr != nil || !concrete {
			return false, textErr
		}
		value, makeErr := state.manager.ArithmeticIntegerNumeralText(new(big.Int).SetInt64(int64(len(text))).String())
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, value)
	}
	if operation >= 3 && operation <= 5 {
		left, leftConcrete, textErr := t.groundStringArgument(expression, 0)
		if textErr != nil || !leftConcrete {
			return false, textErr
		}
		right, rightConcrete, textErr := t.groundStringArgument(expression, 1)
		if textErr != nil || !rightConcrete {
			return false, textErr
		}
		value := false
		switch operation {
		case 3:
			value = strings.HasPrefix(right, left)
		case 4:
			value = strings.HasSuffix(right, left)
		case 5:
			value = strings.Contains(left, right)
		}
		boolean, makeErr := state.manager.MkFalse()
		if value {
			boolean, makeErr = state.manager.MkTrue()
		}
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, boolean)
	}
	if operation == 8 {
		text, concrete, textErr := t.groundStringArgument(expression, 0)
		if textErr != nil || !concrete {
			return false, textErr
		}
		indexExpression, argumentErr := state.manager.GetArg(expression, 1)
		if argumentErr != nil {
			return false, argumentErr
		}
		index, numeral, numeralErr := t.groundInteger(indexExpression)
		if numeralErr != nil || !numeral {
			return false, numeralErr
		}
		result := ""
		if index.Sign() >= 0 && index.IsInt64() && index.Int64() < int64(len(text)) {
			position := int(index.Int64())
			result = text[position : position+1]
		}
		value, makeErr := state.manager.MkString(result)
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, value)
	}
	if operation == 6 {
		text, concrete, textErr := t.groundStringArgument(expression, 0)
		if textErr != nil || !concrete {
			return false, textErr
		}
		offsetExpression, argumentErr := state.manager.GetArg(expression, 1)
		if argumentErr != nil {
			return false, argumentErr
		}
		lengthExpression, argumentErr := state.manager.GetArg(expression, 2)
		if argumentErr != nil {
			return false, argumentErr
		}
		offset, offsetNumeral, numeralErr := t.groundInteger(offsetExpression)
		if numeralErr != nil || !offsetNumeral {
			return false, numeralErr
		}
		length, lengthNumeral, numeralErr := t.groundInteger(lengthExpression)
		if numeralErr != nil || !lengthNumeral {
			return false, numeralErr
		}
		result := ""
		if offset.Sign() >= 0 && offset.IsInt64() && length.Sign() > 0 && offset.Int64() < int64(len(text)) {
			start := int(offset.Int64())
			end := len(text)
			remaining := int64(len(text) - start)
			if length.IsInt64() && length.Int64() < remaining {
				end = start + int(length.Int64())
			}
			result = text[start:end]
		}
		value, makeErr := state.manager.MkString(result)
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, value)
	}
	if operation == 10 {
		source, sourceConcrete, textErr := t.groundStringArgument(expression, 0)
		if textErr != nil || !sourceConcrete {
			return false, textErr
		}
		needle, needleConcrete, textErr := t.groundStringArgument(expression, 1)
		if textErr != nil || !needleConcrete {
			return false, textErr
		}
		offsetExpression, argumentErr := state.manager.GetArg(expression, 2)
		if argumentErr != nil {
			return false, argumentErr
		}
		offset, numeral, numeralErr := t.groundInteger(offsetExpression)
		if numeralErr != nil || !numeral {
			return false, numeralErr
		}
		index := int64(-1)
		if offset.Sign() >= 0 && offset.IsInt64() && offset.Int64() <= int64(len(source)) {
			start := int(offset.Int64())
			if found := strings.Index(source[start:], needle); found >= 0 {
				index = int64(start + found)
			}
		}
		value, makeErr := state.manager.ArithmeticIntegerNumeralText(new(big.Int).SetInt64(index).String())
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, value)
	}
	if operation == 7 {
		source, sourceConcrete, textErr := t.groundStringArgument(expression, 0)
		if textErr != nil || !sourceConcrete {
			return false, textErr
		}
		needle, needleConcrete, textErr := t.groundStringArgument(expression, 1)
		if textErr != nil || !needleConcrete {
			return false, textErr
		}
		replacement, replacementConcrete, textErr := t.groundStringArgument(expression, 2)
		if textErr != nil || !replacementConcrete {
			return false, textErr
		}
		value, makeErr := state.manager.MkString(strings.Replace(source, needle, replacement, 1))
		if makeErr != nil {
			return false, makeErr
		}
		return t.pushGroundValue(expression, value)
	}
	return false, nil
}
