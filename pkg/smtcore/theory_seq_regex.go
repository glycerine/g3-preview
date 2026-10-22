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
