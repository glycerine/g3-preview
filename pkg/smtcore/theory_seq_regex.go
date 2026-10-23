// Source: src/smt/theory_seq.cpp propagate_in_re. The retained Go sequence
// rewriter supplies the equivalent automaton membership formula.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
)

func (t *SMTSequenceTheory) addRegexMembershipAxiom(expression ast.ASTHandle) (reducedExactly bool, result error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	sequence, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return false, err
	}
	regex, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return false, err
	}
	return t.addRegexMembershipAxiomForSequence(expression, sequence, regex)
}

func (t *SMTSequenceTheory) addRegexMembershipAxiomForSequence(expression, sequence, regex ast.ASTHandle) (reducedExactly bool, result error) {
	return t.addGuardedRegexMembershipAxiom(expression, sequence, regex, ast.ASTHandle{})
}

func (t *SMTSequenceTheory) addGuardedRegexMembershipAxiom(expression, sequence, regex, guard ast.ASTHandle) (reducedExactly bool, result error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	rw, err := rewriter.NewSequenceRewriter(state.manager)
	if err != nil {
		return false, err
	}
	out, err := state.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return false, err
	}
	defer func() { result = errors.Join(result, state.manager.DelRef(&out)) }()
	status, err := rw.RewriteStringInRegex(sequence, regex, out)
	if err != nil {
		return false, err
	}
	if status == rewriter.RewriteFailed {
		return false, nil
	}
	reduced, err := state.manager.RefGet(out)
	if err != nil {
		return false, err
	}
	if reduced == expression {
		return false, nil
	}
	equivalence, err := state.manager.MkEq(expression, reduced)
	if err != nil {
		return false, err
	}
	if !guard.IsNull() {
		equivalence, err = state.manager.MkImplies(guard, equivalence)
		if err != nil {
			return false, err
		}
	}
	return true, t.axioms.Push(equivalence)
}

func (t *SMTSequenceTheory) fixedLengthRegexWord(sequence ast.ASTHandle, count uint64) (word, guard ast.ASTHandle, internal []ast.ASTHandle, result error) {
	state, err := t.State()
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	sort, err := state.manager.GetSort(sequence)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	elementSort, isSequence, err := state.manager.SequenceSortElement(sort)
	if err != nil || !isSequence {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	parts := make([]ast.ASTHandle, 0, int(count))
	internal = make([]ast.ASTHandle, 0, int(count)*2+1)
	for index := uint64(0); index < count; index++ {
		position, makeErr := t.integer(index)
		if makeErr != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, nil, makeErr
		}
		element, makeErr := t.relationSkolem("seq.nth", []ast.ASTHandle{sequence, position}, elementSort)
		if makeErr != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, nil, makeErr
		}
		unit, makeErr := state.manager.MkSequenceUnit(element)
		if makeErr != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, nil, makeErr
		}
		parts = append(parts, unit)
		internal = append(internal, element, unit)
	}
	word, err = t.makeWord(parts, sort)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	length, err := state.manager.MkSequenceLength(sequence)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	lengthValue, err := t.integer(count)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	guard, err = state.manager.MkEq(length, lengthValue)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, nil, err
	}
	internal = append(internal, word)
	return word, guard, internal, nil
}

func (t *SMTSequenceTheory) reduceExpandedRegexMemberships() (progress, unresolved bool, result error) {
	state, err := t.State()
	if err != nil {
		return false, false, err
	}
	seen := make(map[ast.ASTHandle]struct{})
	for expression := range state.context.enodes {
		if _, found := seen[expression]; found {
			continue
		}
		seen[expression] = struct{}{}
		operation, own, operationErr := t.operation(expression)
		if operationErr != nil {
			return false, false, operationErr
		}
		if !own || operation != 12 {
			continue
		}
		sequence, argumentErr := state.manager.GetArg(expression, 0)
		if argumentErr != nil {
			return false, false, argumentErr
		}
		regex, argumentErr := state.manager.GetArg(expression, 1)
		if argumentErr != nil {
			return false, false, argumentErr
		}
		parts, _, expandErr := t.expandWord([]ast.ASTHandle{sequence}, nil)
		if expandErr != nil {
			return false, false, expandErr
		}
		sort, sortErr := state.manager.GetSort(sequence)
		if sortErr != nil {
			return false, false, sortErr
		}
		expanded, makeErr := t.makeWord(parts, sort)
		if makeErr != nil {
			return false, false, makeErr
		}
		before := len(t.axioms.set)
		reduced, reduceErr := t.addRegexMembershipAxiomForSequence(expression, expanded, regex)
		if reduceErr != nil {
			return false, false, reduceErr
		}
		if !reduced {
			count, exact, lengthErr := t.exactSequenceLength(sequence)
			if lengthErr != nil {
				return false, false, lengthErr
			}
			if exact && count <= smtSequenceMaximumFixedDecomposition {
				fixedWord, guard, internal, wordErr := t.fixedLengthRegexWord(sequence, count)
				if wordErr != nil {
					return false, false, wordErr
				}
				reduced, reduceErr = t.addGuardedRegexMembershipAxiom(expression, fixedWord, regex, guard)
				if reduceErr != nil {
					return false, false, reduceErr
				}
				if reduced {
					t.trackInternalTerms(internal...)
				}
			}
		}
		if !reduced {
			unresolved = true
			continue
		}
		t.trackInternalTerms(expression, expanded)
		t.trackInternalTerms(parts...)
		if treeErr := t.trackRegexTree(regex); treeErr != nil {
			return false, false, treeErr
		}
		if len(t.axioms.set) != before {
			progress = true
		}
	}
	return progress, unresolved, nil
}

func (t *SMTSequenceTheory) trackRegexTree(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	sort, err := state.manager.GetSort(expression)
	if err != nil {
		return err
	}
	_, regex, err := t.sequenceSort(sort)
	if err != nil {
		return err
	}
	if !regex {
		return nil
	}
	t.trackInternalTerms(expression)
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return err
	}
	for index := uint32(0); index < count; index++ {
		argument, argumentErr := state.manager.GetArg(expression, index)
		if argumentErr != nil {
			return argumentErr
		}
		if argumentErr = t.trackRegexTree(argument); argumentErr != nil {
			return argumentErr
		}
	}
	return nil
}
