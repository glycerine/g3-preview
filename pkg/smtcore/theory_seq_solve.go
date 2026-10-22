// Source: src/smt/theory_seq.cpp simplify_eq, solve_unit_eq, occurs,
// add_solution, solve_eqs, solve_eq, canonize and expand.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (t *SMTSequenceTheory) expandWordExpression(expression ast.ASTHandle, dependency *smtSequenceDependency, visiting map[ast.ASTHandle]struct{}) ([]ast.ASTHandle, *smtSequenceDependency, error) {
	state, err := t.State()
	if err != nil {
		return nil, dependency, err
	}
	if _, found := visiting[expression]; found {
		return []ast.ASTHandle{expression}, dependency, nil
	}
	visiting[expression] = struct{}{}
	defer delete(visiting, expression)
	representative, foundDependency, err := t.solutions.Find(expression)
	if err != nil {
		return nil, dependency, err
	}
	dependency = t.dependencies.Join(dependency, foundDependency)
	if representative != expression {
		return t.expandWordExpression(representative, dependency, visiting)
	}
	left, right, concat, err := state.manager.SequenceConcatParts(expression)
	if err != nil {
		return nil, dependency, err
	}
	if concat {
		leftParts, dependency, err := t.expandWordExpression(left, dependency, visiting)
		if err != nil {
			return nil, dependency, err
		}
		rightParts, dependency, err := t.expandWordExpression(right, dependency, visiting)
		return append(leftParts, rightParts...), dependency, err
	}
	empty, err := state.manager.IsSequenceEmpty(expression)
	if err != nil {
		return nil, dependency, err
	}
	if empty {
		return nil, dependency, nil
	}
	return []ast.ASTHandle{expression}, dependency, nil
}

func (t *SMTSequenceTheory) expandWord(parts []ast.ASTHandle, dependency *smtSequenceDependency) ([]ast.ASTHandle, *smtSequenceDependency, error) {
	result := make([]ast.ASTHandle, 0, len(parts))
	for _, part := range parts {
		expanded, updated, err := t.expandWordExpression(part, dependency, make(map[ast.ASTHandle]struct{}))
		if err != nil {
			return nil, dependency, err
		}
		dependency = updated
		result = append(result, expanded...)
	}
	result, err := t.mergeStringParts(result)
	return result, dependency, err
}

func (t *SMTSequenceTheory) mergeStringParts(parts []ast.ASTHandle) ([]ast.ASTHandle, error) {
	state, err := t.State()
	if err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, 0, len(parts))
	for _, part := range parts {
		text, stringValue, textErr := state.manager.SequenceStringValue(part)
		if textErr != nil {
			return nil, textErr
		}
		if stringValue && text == "" {
			continue
		}
		if stringValue && len(result) != 0 {
			previous, previousString, previousErr := state.manager.SequenceStringValue(result[len(result)-1])
			if previousErr != nil {
				return nil, previousErr
			}
			if previousString {
				combined, makeErr := state.manager.MkString(previous + text)
				if makeErr != nil {
					return nil, makeErr
				}
				result[len(result)-1] = combined
				continue
			}
		}
		result = append(result, part)
	}
	return result, nil
}

func (t *SMTSequenceTheory) sequenceVariable(expression ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	sort, err := state.manager.GetSort(expression)
	if err != nil {
		return false, err
	}
	_, sequence, err := state.manager.SequenceSortElement(sort)
	if err != nil || !sequence {
		return false, err
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil {
		return false, err
	}
	if family == t.FamilyID() {
		return false, nil
	}
	op, basic, err := state.basicOperation(expression)
	if err != nil {
		return false, err
	}
	return !basic || op != ast.ASTOpIte, nil
}

func (t *SMTSequenceTheory) occurs(variable ast.ASTHandle, parts []ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	stack := append([]ast.ASTHandle(nil), parts...)
	seen := make(map[ast.ASTHandle]struct{})
	for len(stack) != 0 {
		index := len(stack) - 1
		expression := stack[index]
		stack = stack[:index]
		if expression == variable {
			return true, nil
		}
		if _, found := seen[expression]; found {
			continue
		}
		seen[expression] = struct{}{}
		if op, basic, operationErr := state.basicOperation(expression); operationErr != nil {
			return false, operationErr
		} else if basic && op == ast.ASTOpIte {
			return true, nil
		}
		left, right, concat, concatErr := state.manager.SequenceConcatParts(expression)
		if concatErr != nil {
			return false, concatErr
		}
		if concat {
			stack = append(stack, left, right)
		}
	}
	return false, nil
}

func (t *SMTSequenceTheory) makeWord(parts []ast.ASTHandle, sort ast.ASTHandle) (ast.ASTHandle, error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if len(parts) == 0 {
		return state.manager.MkSequenceEmpty(sort)
	}
	result := parts[len(parts)-1]
	for index := len(parts) - 2; index >= 0; index-- {
		result, err = state.manager.MkSequenceConcat(parts[index], result)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return result, nil
}

func (t *SMTSequenceTheory) dependencyEvidence(dependency *smtSequenceDependency) ([]SMTLiteral, []SMTEnodePair) {
	assumptions := t.dependencies.Linearize(dependency)
	literals := make([]SMTLiteral, 0, len(assumptions))
	equalities := make([]SMTEnodePair, 0, len(assumptions))
	seenLiterals := make(map[SMTLiteral]struct{})
	seenEqualities := make(map[SMTEnodePair]struct{})
	for _, assumption := range assumptions {
		if assumption.literal != SMTNullLiteral {
			if _, found := seenLiterals[assumption.literal]; !found {
				seenLiterals[assumption.literal] = struct{}{}
				literals = append(literals, assumption.literal)
			}
			continue
		}
		if assumption.left == nil || assumption.right == nil {
			continue
		}
		pair := SMTEnodePair{First: assumption.left, Second: assumption.right}
		if _, found := seenEqualities[pair]; !found {
			seenEqualities[pair] = struct{}{}
			equalities = append(equalities, pair)
		}
	}
	return literals, equalities
}

func (t *SMTSequenceTheory) addSolution(variable, value ast.ASTHandle, dependency *smtSequenceDependency) (bool, error) {
	if variable == value {
		return false, nil
	}
	if err := t.solutions.Update(variable, value, dependency); err != nil {
		return false, err
	}
	state, err := t.State()
	if err != nil {
		return false, err
	}
	if err = state.Internalize(variable, false); err != nil {
		return false, err
	}
	if err = state.Internalize(value, false); err != nil {
		return false, err
	}
	left, right := state.context.enodes[variable], state.context.enodes[value]
	if left == nil || right == nil {
		return false, ast.ErrASTDomain
	}
	if left.Root() == right.Root() {
		return true, nil
	}
	literals, equalities := t.dependencyEvidence(dependency)
	justification, err := NewSMTExtTheoryEqualityPropagationJustification(state.manager, t.FamilyID(), literals, equalities, left, right, nil)
	if err != nil {
		return false, err
	}
	if err = state.addEquality(left, right, NewSMTEqualityJustification(justification)); err != nil {
		return false, err
	}
	return true, nil
}

func (t *SMTSequenceTheory) solveUnitWord(left, right []ast.ASTHandle, dependency *smtSequenceDependency) (bool, error) {
	for pass := 0; pass < 2; pass++ {
		if len(left) == 1 {
			variable, err := t.sequenceVariable(left[0])
			if err != nil {
				return false, err
			}
			if variable {
				cyclic, err := t.occurs(left[0], right)
				if err != nil {
					return false, err
				}
				if !cyclic {
					state, stateErr := t.State()
					if stateErr != nil {
						return false, stateErr
					}
					sort, sortErr := state.manager.GetSort(left[0])
					if sortErr != nil {
						return false, sortErr
					}
					value, makeErr := t.makeWord(right, sort)
					if makeErr != nil {
						return false, makeErr
					}
					return t.addSolution(left[0], value, dependency)
				}
			}
		}
		left, right = right, left
	}
	return false, nil
}

func (t *SMTSequenceTheory) equalWordParts(left, right ast.ASTHandle, dependency *smtSequenceDependency) (bool, *smtSequenceDependency) {
	if left == right {
		return true, dependency
	}
	state, err := t.State()
	if err != nil {
		return false, dependency
	}
	leftNode, rightNode := state.context.enodes[left], state.context.enodes[right]
	if leftNode == nil || rightNode == nil || leftNode.Root() != rightNode.Root() {
		return false, dependency
	}
	equality := t.dependencies.Leaf(smtSequenceAssumption{left: leftNode, right: rightNode})
	return true, t.dependencies.Join(dependency, equality)
}

func (t *SMTSequenceTheory) cancelWordBoundaries(left, right []ast.ASTHandle, dependency *smtSequenceDependency) ([]ast.ASTHandle, []ast.ASTHandle, *smtSequenceDependency, bool, bool, error) {
	state, err := t.State()
	if err != nil {
		return nil, nil, dependency, false, false, err
	}
	changed := false
	for len(left) != 0 && len(right) != 0 {
		equal, updated := t.equalWordParts(left[0], right[0], dependency)
		if !equal {
			break
		}
		dependency = updated
		left, right, changed = left[1:], right[1:], true
	}
	for len(left) != 0 && len(right) != 0 {
		equal, updated := t.equalWordParts(left[len(left)-1], right[len(right)-1], dependency)
		if !equal {
			break
		}
		dependency = updated
		left, right, changed = left[:len(left)-1], right[:len(right)-1], true
	}
	for _, front := range []bool{true, false} {
		if len(left) == 0 || len(right) == 0 {
			break
		}
		li, ri := 0, 0
		if !front {
			li, ri = len(left)-1, len(right)-1
		}
		leftText, leftString, textErr := state.manager.SequenceStringValue(left[li])
		if textErr != nil {
			return nil, nil, dependency, false, false, textErr
		}
		rightText, rightString, textErr := state.manager.SequenceStringValue(right[ri])
		if textErr != nil {
			return nil, nil, dependency, false, false, textErr
		}
		if !leftString || !rightString {
			continue
		}
		common := 0
		if front {
			for common < len(leftText) && common < len(rightText) && leftText[common] == rightText[common] {
				common++
			}
		} else {
			for common < len(leftText) && common < len(rightText) && leftText[len(leftText)-1-common] == rightText[len(rightText)-1-common] {
				common++
			}
		}
		if common == 0 {
			return left, right, dependency, changed, true, nil
		}
		changed = true
		if front {
			leftText, rightText = leftText[common:], rightText[common:]
		} else {
			leftText, rightText = leftText[:len(leftText)-common], rightText[:len(rightText)-common]
		}
		if leftText == "" {
			left = append(left[:li], left[li+1:]...)
		} else {
			left[li], err = state.manager.MkString(leftText)
			if err != nil {
				return nil, nil, dependency, false, false, err
			}
		}
		if rightText == "" {
			right = append(right[:ri], right[ri+1:]...)
		} else {
			right[ri], err = state.manager.MkString(rightText)
			if err != nil {
				return nil, nil, dependency, false, false, err
			}
		}
	}
	return left, right, dependency, changed, false, nil
}

func (t *SMTSequenceTheory) removeEquation(index int) error {
	last := t.equations.Len() - 1
	if index < 0 || index > last {
		return ast.ErrASTDomain
	}
	if index != last {
		id, left, right, dependency, err := t.equations.At(last)
		if err != nil {
			return err
		}
		if err = t.equations.Set(index, id, left, right, dependency); err != nil {
			return err
		}
	}
	return t.equations.PopBack()
}

func (t *SMTSequenceTheory) reduceEqualLengthBoundary(left, right []ast.ASTHandle, dependency *smtSequenceDependency) (bool, error) {
	if len(left) == 0 || len(right) == 0 {
		return false, nil
	}
	state, err := t.State()
	if err != nil {
		return false, err
	}
	for _, front := range []bool{true, false} {
		li, ri := 0, 0
		if !front {
			li, ri = len(left)-1, len(right)-1
		}
		if left[li] == right[ri] {
			continue
		}
		leftLength, makeErr := state.manager.MkSequenceLength(left[li])
		if makeErr != nil {
			return false, makeErr
		}
		rightLength, makeErr := state.manager.MkSequenceLength(right[ri])
		if makeErr != nil {
			return false, makeErr
		}
		leftNode, rightNode := state.context.enodes[leftLength], state.context.enodes[rightLength]
		if leftNode == nil || rightNode == nil || leftNode.Root() != rightNode.Root() {
			continue
		}
		lengthEquality := t.dependencies.Leaf(smtSequenceAssumption{left: leftNode, right: rightNode})
		return t.addSolution(left[li], right[ri], t.dependencies.Join(dependency, lengthEquality))
	}
	return false, nil
}

func (t *SMTSequenceTheory) solveWordEquations() (bool, error) {
	for index := 0; index < t.equations.Len(); index++ {
		id, left, right, dependency, err := t.equations.At(index)
		if err != nil {
			return false, err
		}
		left, leftDependency, err := t.expandWord(left, dependency)
		if err != nil {
			return false, err
		}
		right, rightDependency, err := t.expandWord(right, leftDependency)
		if err != nil {
			return false, err
		}
		dependency = rightDependency
		left, right, dependency, changed, conflict, err := t.cancelWordBoundaries(left, right, dependency)
		if err != nil {
			return false, err
		}
		if conflict || len(left) == 0 && len(right) != 0 && smtSequenceAllStrings(right, t) || len(right) == 0 && len(left) != 0 && smtSequenceAllStrings(left, t) {
			if err = t.setDependencyConflict(dependency); err != nil {
				return false, err
			}
			return true, nil
		}
		if len(left) == 0 && len(right) == 0 {
			return true, t.removeEquation(index)
		}
		reduced, err := t.reduceEqualLengthBoundary(left, right, dependency)
		if err != nil || reduced {
			return reduced, err
		}
		solved, err := t.solveUnitWord(left, right, dependency)
		if err != nil || solved {
			return solved, err
		}
		if changed {
			if err = t.equations.Set(index, id, left, right, dependency); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func smtSequenceAllStrings(parts []ast.ASTHandle, t *SMTSequenceTheory) bool {
	state, err := t.State()
	if err != nil {
		return false
	}
	for _, part := range parts {
		if _, matched, matchErr := state.manager.SequenceStringValue(part); matchErr != nil || !matched {
			return false
		}
	}
	return true
}
