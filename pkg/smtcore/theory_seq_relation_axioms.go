// Source: src/smt/theory_seq.cpp assign_eh positive prefix, suffix and
// containment witnesses plus propagate_non_empty guards.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (t *SMTSequenceTheory) relationSkolem(name string, arguments []ast.ASTHandle, sort ast.ASTHandle) (ast.ASTHandle, error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	symbol, err := state.manager.Symbols.Intern(name)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return state.manager.MkSequenceSkolem(symbol, arguments, sort)
}

func (t *SMTSequenceTheory) addNegativeContainsStep(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	source, needle, matched, err := state.manager.SequenceBinaryOperationArguments(expression, 5)
	if err != nil || !matched {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	sort, err := state.manager.GetSort(source)
	if err != nil {
		return err
	}
	empty, err := state.manager.MkSequenceEmpty(sort)
	if err != nil {
		return err
	}
	sourceEmpty, err := state.manager.MkEq(source, empty)
	if err != nil {
		return err
	}
	if err = state.Internalize(sourceEmpty, false); err != nil {
		return err
	}
	literal, err := state.Literal(sourceEmpty)
	if err != nil {
		return err
	}
	value, err := state.Assignment(literal)
	if err != nil || value == lbool.True {
		return err
	}
	if value == lbool.Undef {
		if err = state.markRelevant(sourceEmpty); err != nil {
			return err
		}
		data, err := state.BooleanData(literal.Var())
		if err != nil {
			return err
		}
		// Source add_contains2contains forces the empty phase and retries
		// after that decision, before exposing the recursive tail atom.
		data.PhaseAvailable, data.Phase = true, !literal.Sign()
		return nil
	}
	if err = t.addSequenceCoherenceAxiom(source); err != nil {
		return err
	}
	zero, err := t.integer(0)
	if err != nil {
		return err
	}
	tail, err := t.relationSkolem("seq.tail", []ast.ASTHandle{source, zero}, sort)
	if err != nil {
		return err
	}
	tailContains, err := state.manager.MkSequenceContains(tail, needle)
	if err != nil {
		return err
	}
	notTailContains, err := state.manager.MkNot(tailContains)
	if err != nil {
		return err
	}
	step, err := state.manager.MkOr(sourceEmpty, notTailContains)
	if err != nil {
		return err
	}
	notContains, err := state.manager.MkNot(expression)
	if err != nil {
		return err
	}
	constraint, err := state.manager.MkImplies(notContains, step)
	if err != nil {
		return err
	}
	t.trackInternalTerms(tail, tailContains)
	return t.axioms.Push(constraint)
}

func (t *SMTSequenceTheory) addNegativePrefixStep(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	boundary, container, matched, err := state.manager.SequenceBinaryOperationArguments(expression, 3)
	if err != nil || !matched {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	if err = t.addSequenceCoherenceAxiom(boundary); err != nil {
		return err
	}
	if err = t.addSequenceCoherenceAxiom(container); err != nil {
		return err
	}
	sort, err := state.manager.GetSort(boundary)
	if err != nil {
		return err
	}
	elementSort, sequence, err := state.manager.SequenceSortElement(sort)
	if err != nil || !sequence {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	empty, err := state.manager.MkSequenceEmpty(sort)
	if err != nil {
		return err
	}
	containerEmpty, err := state.manager.MkEq(container, empty)
	if err != nil {
		return err
	}
	zero, err := t.integer(0)
	if err != nil {
		return err
	}
	boundaryHead, err := t.relationSkolem("seq.nth", []ast.ASTHandle{boundary, zero}, elementSort)
	if err != nil {
		return err
	}
	containerHead, err := t.relationSkolem("seq.nth", []ast.ASTHandle{container, zero}, elementSort)
	if err != nil {
		return err
	}
	boundaryTail, err := t.relationSkolem("seq.tail", []ast.ASTHandle{boundary, zero}, sort)
	if err != nil {
		return err
	}
	containerTail, err := t.relationSkolem("seq.tail", []ast.ASTHandle{container, zero}, sort)
	if err != nil {
		return err
	}
	headsEqual, err := state.manager.MkEq(boundaryHead, containerHead)
	if err != nil {
		return err
	}
	headsDistinct, err := state.manager.MkNot(headsEqual)
	if err != nil {
		return err
	}
	tailPrefix, err := state.manager.MkSequencePrefix(boundaryTail, containerTail)
	if err != nil {
		return err
	}
	notTailPrefix, err := state.manager.MkNot(tailPrefix)
	if err != nil {
		return err
	}
	step, err := state.manager.MkOrArray([]ast.ASTHandle{containerEmpty, headsDistinct, notTailPrefix})
	if err != nil {
		return err
	}
	notPrefix, err := state.manager.MkNot(expression)
	if err != nil {
		return err
	}
	constraint, err := state.manager.MkImplies(notPrefix, step)
	if err != nil {
		return err
	}
	t.trackInternalTerms(boundaryHead, containerHead, boundaryTail, containerTail, tailPrefix)
	return t.axioms.Push(constraint)
}

func (t *SMTSequenceTheory) sequenceInitialAndLast(expression ast.ASTHandle) (initial, last, empty ast.ASTHandle, result error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	sort, err := state.manager.GetSort(expression)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	elementSort, sequence, err := state.manager.SequenceSortElement(sort)
	if err != nil || !sequence {
		if err != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
		}
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, ast.ErrASTDomain
	}
	emptySequence, err := state.manager.MkSequenceEmpty(sort)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	empty, err = state.manager.MkEq(expression, emptySequence)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	initial, err = t.relationSkolem("seq.first", []ast.ASTHandle{expression}, sort)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	last, err = t.relationSkolem("seq.last", []ast.ASTHandle{expression}, elementSort)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	unit, err := state.manager.MkSequenceUnit(last)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	word, err := state.manager.MkSequenceConcat(initial, unit)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	definition, err := state.manager.MkEq(expression, word)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	coherence, err := state.manager.MkOr(empty, definition)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	t.trackInternalTerms(initial, last, unit, word)
	if err = t.axioms.Push(coherence); err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, ast.ASTHandle{}, err
	}
	return initial, last, empty, nil
}

func (t *SMTSequenceTheory) addNegativeSuffixStep(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	boundary, container, matched, err := state.manager.SequenceBinaryOperationArguments(expression, 4)
	if err != nil || !matched {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	boundaryInitial, boundaryLast, _, err := t.sequenceInitialAndLast(boundary)
	if err != nil {
		return err
	}
	containerInitial, containerLast, containerEmpty, err := t.sequenceInitialAndLast(container)
	if err != nil {
		return err
	}
	lastEqual, err := state.manager.MkEq(boundaryLast, containerLast)
	if err != nil {
		return err
	}
	lastDistinct, err := state.manager.MkNot(lastEqual)
	if err != nil {
		return err
	}
	initialSuffix, err := state.manager.MkSequenceSuffix(boundaryInitial, containerInitial)
	if err != nil {
		return err
	}
	notInitialSuffix, err := state.manager.MkNot(initialSuffix)
	if err != nil {
		return err
	}
	step, err := state.manager.MkOrArray([]ast.ASTHandle{containerEmpty, lastDistinct, notInitialSuffix})
	if err != nil {
		return err
	}
	notSuffix, err := state.manager.MkNot(expression)
	if err != nil {
		return err
	}
	constraint, err := state.manager.MkImplies(notSuffix, step)
	if err != nil {
		return err
	}
	t.trackInternalTerms(initialSuffix)
	return t.axioms.Push(constraint)
}

func (t *SMTSequenceTheory) addRelationAxioms(expression ast.ASTHandle, operation int32) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	left, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	right, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return err
	}
	sort, err := state.manager.GetSort(left)
	if err != nil {
		return err
	}
	arguments := []ast.ASTHandle{left, right}
	var definition ast.ASTHandle
	switch operation {
	case 3:
		remainder, makeErr := t.relationSkolem("seq.p.suffix", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		word, makeErr := state.manager.MkSequenceConcat(left, remainder)
		if makeErr != nil {
			return makeErr
		}
		definition, err = state.manager.MkEq(word, right)
	case 4:
		remainder, makeErr := t.relationSkolem("seq.s.prefix", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		word, makeErr := state.manager.MkSequenceConcat(remainder, left)
		if makeErr != nil {
			return makeErr
		}
		definition, err = state.manager.MkEq(word, right)
	case 5:
		prefix, makeErr := t.relationSkolem("seq.idx.left", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		suffix, makeErr := t.relationSkolem("seq.idx.right", arguments, sort)
		if makeErr != nil {
			return makeErr
		}
		tail, makeErr := state.manager.MkSequenceConcat(right, suffix)
		if makeErr != nil {
			return makeErr
		}
		word, makeErr := state.manager.MkSequenceConcat(prefix, tail)
		if makeErr != nil {
			return makeErr
		}
		definition, err = state.manager.MkEq(word, left)
	default:
		return ast.ErrASTDomain
	}
	if err != nil {
		return err
	}
	witness, err := state.manager.MkImplies(expression, definition)
	if err != nil {
		return err
	}
	if err = t.axioms.Push(witness); err != nil {
		return err
	}

	nonemptyArgument := left
	if operation == 5 {
		nonemptyArgument = right
	}
	empty, err := state.manager.MkSequenceEmpty(sort)
	if err != nil {
		return err
	}
	emptyEquality, err := state.manager.MkEq(nonemptyArgument, empty)
	if err != nil {
		return err
	}
	nonempty, err := state.manager.MkNot(emptyEquality)
	if err != nil {
		return err
	}
	notRelation, err := state.manager.MkNot(expression)
	if err != nil {
		return err
	}
	guard, err := state.manager.MkImplies(notRelation, nonempty)
	if err != nil {
		return err
	}
	if err = t.axioms.Push(guard); err != nil {
		return err
	}
	if operation == 3 || operation == 4 {
		return t.addNegativeBoundaryWitness(expression, left, right, operation == 4)
	}
	prefix, err := state.manager.MkSequencePrefix(right, left)
	if err != nil {
		return err
	}
	notPrefix, err := state.manager.MkNot(prefix)
	if err != nil {
		return err
	}
	if err = t.pushSequenceImplication([]ast.ASTHandle{notRelation}, notPrefix); err != nil {
		return err
	}
	suffix, err := state.manager.MkSequenceSuffix(right, left)
	if err != nil {
		return err
	}
	notSuffix, err := state.manager.MkNot(suffix)
	if err != nil {
		return err
	}
	return t.pushSequenceImplication([]ast.ASTHandle{notRelation}, notSuffix)
}
