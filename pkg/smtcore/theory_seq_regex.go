// Source: src/smt/theory_seq.cpp propagate_in_re. The retained Go sequence
// rewriter supplies the equivalent automaton membership formula.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
)

func (t *SMTSequenceTheory) addRegexMembershipAxiom(expression ast.ASTHandle) (result error) {
	state, err := t.State()
	if err != nil {
		return err
	}
	sequence, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	regex, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return err
	}
	rw, err := rewriter.NewSequenceRewriter(state.manager)
	if err != nil {
		return err
	}
	out, err := state.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, state.manager.DelRef(&out)) }()
	status, err := rw.RewriteStringInRegex(sequence, regex, out)
	if err != nil {
		return err
	}
	if status == rewriter.RewriteFailed {
		return nil
	}
	reduced, err := state.manager.RefGet(out)
	if err != nil {
		return err
	}
	if reduced == expression {
		return nil
	}
	equivalence, err := state.manager.MkEq(expression, reduced)
	if err != nil {
		return err
	}
	return t.axioms.Push(equivalence)
}
