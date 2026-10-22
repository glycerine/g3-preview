// Source: src/smt/theory_seq.cpp check_length_coherence, mk_decompose and
// propagate_length_coherence.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

// addSequenceCoherenceAxiom gives every symbolic sequence its source-level
// empty-or-cons decomposition.  The two Skolems are stable ASTs, so later
// arithmetic bounds and word-equation rules can refer to the same head and
// tail witnesses.
func (t *SMTSequenceTheory) addSequenceCoherenceAxiom(expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	sort, err := state.manager.GetSort(expression)
	if err != nil {
		return err
	}
	elementSort, sequence, err := state.manager.SequenceSortElement(sort)
	if err != nil {
		return err
	}
	if !sequence {
		return ast.ErrASTDomain
	}
	empty, err := state.manager.MkSequenceEmpty(sort)
	if err != nil {
		return err
	}
	emptyEquality, err := state.manager.MkEq(expression, empty)
	if err != nil {
		return err
	}
	zero, err := t.integer(0)
	if err != nil {
		return err
	}
	head, err := t.relationSkolem("seq.nth", []ast.ASTHandle{expression, zero}, elementSort)
	if err != nil {
		return err
	}
	tail, err := t.relationSkolem("seq.tail", []ast.ASTHandle{expression, zero}, sort)
	if err != nil {
		return err
	}
	unit, err := state.manager.MkSequenceUnit(head)
	if err != nil {
		return err
	}
	word, err := state.manager.MkSequenceConcat(unit, tail)
	if err != nil {
		return err
	}
	decomposition, err := state.manager.MkEq(expression, word)
	if err != nil {
		return err
	}
	coherence, err := state.manager.MkOr(emptyEquality, decomposition)
	if err != nil {
		return err
	}
	for _, internal := range []ast.ASTHandle{head, tail, unit, word} {
		if _, found := t.coherence[internal]; !found {
			t.coherence[internal] = struct{}{}
			t.coherenceLog = append(t.coherenceLog, internal)
		}
	}
	return t.axioms.Push(coherence)
}
