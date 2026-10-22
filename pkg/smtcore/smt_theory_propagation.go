// Source: src/smt/smt_context.cpp theory equality and disequality queues.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (s *SMTInternalizerState) enqueueTheoryEquality(theory SMTTheoryID, left, right SMTTheoryVar) error {
	if theory == SMTNullTheoryID || left == SMTNullTheoryVar || right == SMTNullTheoryVar || left == right || s.theories[theory] == nil {
		return ast.ErrASTDomain
	}
	s.theoryEqualities = append(s.theoryEqualities, SMTTheoryPropagation{Theory: theory, Left: left, Right: right})
	return nil
}

func (s *SMTInternalizerState) enqueueTheoryDisequality(theory SMTTheoryID, left, right SMTTheoryVar) error {
	plugin := s.theories[theory]
	if plugin == nil || theory == SMTNullTheoryID || left == SMTNullTheoryVar || right == SMTNullTheoryVar || left == right {
		return ast.ErrASTDomain
	}
	s.theoryDisequalities = append(s.theoryDisequalities, SMTTheoryPropagation{Theory: theory, Left: left, Right: right})
	return nil
}

func (s *SMTInternalizerState) enqueueExistingDisequalities(root *SMTEnode, variable SMTTheoryVar, plugin SMTTheoryInternalizer) error {
	if !plugin.UseDisequalities() {
		return nil
	}
	for _, parent := range root.parents {
		if parent == nil || !parent.IsEquality() || len(parent.arguments) != 2 {
			continue
		}
		boolVariable, found := s.boolVar[parent.owner]
		if !found {
			continue
		}
		value, err := s.Assignment(NewSMTLiteral(boolVariable, false))
		if err != nil {
			return err
		}
		if value != lbool.False {
			continue
		}
		left, right := parent.arguments[0], parent.arguments[1]
		if right.root == root {
			left, right = right, left
		}
		if left.root != root || right.root == root {
			continue
		}
		other := right.root.TheoryVar(plugin.FamilyID())
		if other != SMTNullTheoryVar && other != variable {
			if err = s.enqueueTheoryDisequality(plugin.FamilyID(), variable, other); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SMTInternalizerState) PendingTheoryEqualities() []SMTTheoryPropagation {
	if s == nil {
		return nil
	}
	return append([]SMTTheoryPropagation(nil), s.theoryEqualities[s.theoryEqualityHead:]...)
}

func (s *SMTInternalizerState) PendingTheoryDisequalities() []SMTTheoryPropagation {
	if s == nil {
		return nil
	}
	return append([]SMTTheoryPropagation(nil), s.theoryDisequalities[s.theoryDisequalityHead:]...)
}

func (s *SMTInternalizerState) PropagateTheoryEvents() error {
	if err := s.open(); err != nil {
		return err
	}
	for s.theoryEqualityHead < len(s.theoryEqualities) {
		event := s.theoryEqualities[s.theoryEqualityHead]
		plugin := s.theories[event.Theory]
		if plugin == nil {
			return ast.ErrASTDomain
		}
		if err := plugin.NewEquality(event.Left, event.Right); err != nil {
			return err
		}
		s.theoryEqualityHead++
	}
	for s.theoryDisequalityHead < len(s.theoryDisequalities) {
		event := s.theoryDisequalities[s.theoryDisequalityHead]
		plugin := s.theories[event.Theory]
		if plugin == nil {
			return ast.ErrASTDomain
		}
		left, err := plugin.Enode(event.Left)
		if err != nil {
			return err
		}
		right, err := plugin.Enode(event.Right)
		if err != nil {
			return err
		}
		if left.IsInterpreted() && right.IsInterpreted() {
			s.theoryDisequalityHead++
			continue
		}
		if err := plugin.NewDisequality(event.Left, event.Right); err != nil {
			return err
		}
		s.theoryDisequalityHead++
	}
	return nil
}
