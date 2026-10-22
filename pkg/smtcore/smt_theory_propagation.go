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

func (s *SMTInternalizerState) PropagateTheoryAssignments() error {
	if err := s.open(); err != nil {
		return err
	}
	for s.atomPropagationHead < len(s.assignedLiterals) {
		literal := s.assignedLiterals[s.atomPropagationHead]
		variable := literal.Var()
		data, err := s.BooleanData(variable)
		if err != nil {
			return err
		}
		if data.IsTheoryAtom() {
			plugin := s.theories[data.Theory()]
			if plugin == nil {
				return ast.ErrASTDomain
			}
			if err = plugin.Assign(variable, !literal.Sign()); err != nil {
				return err
			}
		}
		s.atomPropagationHead++
	}
	return nil
}

func (s *SMTInternalizerState) propagateTheoryRelevancy() error {
	if s.relevancy == nil {
		return nil
	}
	if err := s.relevancy.Propagate(); err != nil {
		return err
	}
	for _, expression := range s.relevancy.DrainNewlyRelevant() {
		kind, err := s.manager.GetKind(expression)
		if err != nil {
			return err
		}
		if kind != ast.ASTApp {
			continue
		}
		notified := make(map[SMTTheoryID]struct{})
		family, err := s.manager.GetFamilyID(expression)
		if err != nil {
			return err
		}
		if family != s.manager.BasicFamilyID() {
			if plugin := s.theories[family]; plugin != nil {
				if err = plugin.Relevant(expression); err != nil {
					return err
				}
				notified[family] = struct{}{}
			}
		}
		if node := s.context.enodes[expression]; node != nil {
			for item := &node.theoryVariables; item != nil && item.theory != SMTNullTheoryID; item = item.next {
				if _, found := notified[item.theory]; found {
					continue
				}
				plugin := s.theories[item.theory]
				if plugin == nil {
					return ast.ErrASTDomain
				}
				if err = plugin.Relevant(expression); err != nil {
					return err
				}
				notified[item.theory] = struct{}{}
			}
		}
	}
	return nil
}

// PropagateCore runs the retained logical-context queues to a fixed point.
func (s *SMTInternalizerState) PropagateCore() (bool, error) {
	if err := s.open(); err != nil {
		return false, err
	}
	for {
		consistent, err := s.Propagate()
		if err != nil || !consistent {
			return consistent, err
		}
		if err = s.PropagateTheoryAssignments(); err != nil {
			return false, err
		}
		if err = s.propagateTheoryRelevancy(); err != nil {
			return false, err
		}
		if err = s.PropagateTheoryEvents(); err != nil {
			return false, err
		}
		theoryProgress := false
		for _, plugin := range s.theoryOrder {
			if plugin.CanPropagate() {
				theoryProgress = true
				if err = plugin.Propagate(); err != nil {
					return false, err
				}
			}
		}
		if s.inconsistent {
			return false, nil
		}
		relevancyDone := s.relevancy == nil || !s.relevancy.CanPropagate()
		if s.propagationHead == len(s.assignedLiterals) && s.atomPropagationHead == len(s.assignedLiterals) && s.theoryEqualityHead == len(s.theoryEqualities) && s.theoryDisequalityHead == len(s.theoryDisequalities) && relevancyDone && !theoryProgress {
			return true, nil
		}
	}
}
