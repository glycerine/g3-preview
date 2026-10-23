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
	return true, t.axioms.Push(equivalence)
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
			unresolved = true
			continue
		}
		t.trackInternalTerms(expression)
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
