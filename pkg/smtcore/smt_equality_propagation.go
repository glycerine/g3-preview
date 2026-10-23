// Source: src/smt/smt_context.cpp add_eq, add_diseq, propagate_eqs, and
// congruence-table reinsertion. This is the retained single-goroutine core.
package smtcore

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func cloneSMTTheoryVariables(source SMTTheoryVarList) SMTTheoryVarList {
	result := SMTTheoryVarList{theory: source.theory, variable: source.variable}
	to := &result
	for from := source.next; from != nil; from = from.next {
		to.next = &SMTTheoryVarList{theory: from.theory, variable: from.variable}
		to = to.next
	}
	return result
}

func (s *SMTInternalizerState) equalitySnapshot() smtEqualitySnapshot {
	snapshot := smtEqualitySnapshot{
		nodes:         make([]smtEqualityNodeSnapshot, 0, len(s.enodeStack)),
		disequalities: make(map[smtEnodePairKey]struct{}, len(s.context.disequalities)),
		reasons:       make(map[smtEnodePairKey]SMTLiteral, len(s.disequalityReasons)),
	}
	for _, node := range s.enodeStack {
		snapshot.nodes = append(snapshot.nodes, smtEqualityNodeSnapshot{
			node: node, root: node.root, next: node.next, congruence: node.congruence,
			classSize: node.classSize, parents: append([]*SMTEnode(nil), node.parents...),
			theoryVariables: cloneSMTTheoryVariables(node.theoryVariables), trans: node.trans,
		})
	}
	for key := range s.context.disequalities {
		snapshot.disequalities[key] = struct{}{}
	}
	for key, literal := range s.disequalityReasons {
		snapshot.reasons[key] = literal
	}
	return snapshot
}

func (s *SMTInternalizerState) restoreEqualitySnapshot(snapshot smtEqualitySnapshot) error {
	for _, saved := range snapshot.nodes {
		saved.node.root, saved.node.next, saved.node.congruence = saved.root, saved.next, saved.congruence
		saved.node.classSize = saved.classSize
		saved.node.parents = append(saved.node.parents[:0], saved.parents...)
		saved.node.theoryVariables = cloneSMTTheoryVariables(saved.theoryVariables)
		saved.node.trans = saved.trans
	}
	s.context.disequalities = snapshot.disequalities
	s.disequalityReasons = snapshot.reasons
	return s.rebuildCongruence(false)
}

func smtInvertTrans(node *SMTEnode) {
	current := node.trans.Target
	previous := node
	justification := node.trans.Justification
	previous.trans = SMTTransJustification{Justification: NullSMTEqualityJustification()}
	for current != nil {
		next := current.trans.Target
		nextJustification := current.trans.Justification
		current.trans = SMTTransJustification{Target: previous, Justification: justification}
		previous, justification, current = current, nextJustification, next
	}
}

func (s *SMTInternalizerState) setEqualityConflict(justification SMTJustification, literal SMTLiteral) error {
	wrapper, err := NewSMTBooleanJustification(justification)
	if err != nil {
		return err
	}
	s.inconsistent = true
	s.conflict = wrapper
	s.conflictLiteral = literal
	s.statistics.NumConflicts++
	return nil
}

func (s *SMTInternalizerState) addDisequality(left, right *SMTEnode, literal SMTLiteral) error {
	if left == nil || right == nil || literal == SMTNullLiteral {
		return ast.ErrASTDomain
	}
	if left.root == right.root {
		justification, err := NewSMTEqualityPropagationJustification(left, right)
		if err != nil {
			return err
		}
		return s.setEqualityConflict(justification, literal)
	}
	key, err := smtCheckerPair(left, right)
	if err != nil {
		return err
	}
	s.context.disequalities[key] = struct{}{}
	s.disequalityReasons[key] = literal
	for item := &key.first.theoryVariables; item != nil && item.theory != SMTNullTheoryID; item = item.next {
		other := key.second.TheoryVar(item.theory)
		if other != SMTNullTheoryVar && other != item.variable && s.theories[item.theory] != nil && s.theories[item.theory].UseDisequalities() {
			if err = s.enqueueTheoryDisequality(item.theory, item.variable, other); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SMTInternalizerState) mergeTheoryVariables(from, to *SMTEnode) error {
	for item := &from.theoryVariables; item != nil && item.theory != SMTNullTheoryID; item = item.next {
		other := to.TheoryVar(item.theory)
		if other == SMTNullTheoryVar {
			if err := to.AddTheoryVar(item.theory, item.variable); err != nil {
				return err
			}
		} else if other != item.variable && s.theories[item.theory] != nil {
			if err := s.enqueueTheoryEquality(item.theory, other, item.variable); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SMTInternalizerState) remapDisequalities() error {
	rebuilt := make(map[smtEnodePairKey]struct{}, len(s.context.disequalities))
	reasons := make(map[smtEnodePairKey]SMTLiteral, len(s.disequalityReasons))
	for key := range s.context.disequalities {
		if key.first.root == key.second.root {
			proof, err := NewSMTEqualityPropagationJustification(key.first, key.second)
			if err != nil {
				return fmt.Errorf("make collapsed disequality proof: %w", err)
			}
			if err = s.setEqualityConflict(proof, s.disequalityReasons[key]); err != nil {
				return err
			}
			continue
		}
		mapped, err := smtCheckerPair(key.first, key.second)
		if err != nil {
			return fmt.Errorf("map disequality roots: %w", err)
		}
		rebuilt[mapped] = struct{}{}
		reasons[mapped] = s.disequalityReasons[key]
	}
	s.context.disequalities, s.disequalityReasons = rebuilt, reasons
	return nil
}

func (s *SMTInternalizerState) rebuildCongruence(enqueue bool) error {
	if err := s.congruence.Reset(); err != nil {
		return err
	}
	for _, node := range s.enodeStack {
		node.parents = nil
	}
	for _, parent := range s.enodeStack {
		seen := make(map[*SMTEnode]struct{})
		for _, argument := range parent.arguments {
			root := argument.root
			if _, found := seen[root]; found {
				continue
			}
			seen[root] = struct{}{}
			root.parents = append(root.parents, parent)
		}
	}
	for _, node := range s.enodeStack {
		if len(node.arguments) == 0 || !node.congruenceEnabled || node.IsTrueEquality() {
			node.congruence = node
			continue
		}
		inserted, err := s.congruence.Insert(node)
		if err != nil {
			return err
		}
		node.congruence = inserted.Node
		if enqueue && inserted.Node != node && inserted.Node.root != node.root {
			s.pending = append(s.pending, SMTInternalizerCongruence{Node: node, Existing: inserted.Node, Commuted: inserted.Commuted})
		}
	}
	return nil
}

func (s *SMTInternalizerState) propagateTrueEqualities() error {
	for _, node := range s.enodeStack {
		if !node.IsTrueEquality() {
			continue
		}
		variable, found := s.boolVar[node.owner]
		if !found {
			continue
		}
		justification, err := NewSMTEqualityPropagationJustification(node.arguments[0], node.arguments[1])
		if err != nil {
			return err
		}
		wrapped, err := NewSMTBooleanJustification(justification)
		if err != nil {
			return err
		}
		if err = s.AssignLiteral(NewSMTLiteral(variable, false), wrapped); err != nil {
			return err
		}
		if s.inconsistent {
			return nil
		}
	}
	return nil
}

func (s *SMTInternalizerState) propagateBooleanClass(root *SMTEnode) error {
	if root == nil || !root.boolean {
		return nil
	}
	var source *SMTEnode
	value := lbool.Undef
	for node := root; ; node = node.next {
		if s.manager.IsTrue(node.owner) || s.manager.IsFalse(node.owner) {
			source, value = node, lbool.False
			if s.manager.IsTrue(node.owner) {
				value = lbool.True
			}
			break
		}
		if variable, found := s.boolVar[node.owner]; found {
			assigned, err := s.Assignment(NewSMTLiteral(variable, false))
			if err != nil {
				return err
			}
			if assigned != lbool.Undef {
				source, value = node, assigned
				break
			}
		}
		if node.next == root {
			break
		}
	}
	if source == nil {
		return nil
	}
	for node := root; ; node = node.next {
		if variable, found := s.boolVar[node.owner]; found && node != source {
			literal := NewSMTLiteral(variable, value == lbool.False)
			var justification SMTJustification
			var err error
			if s.manager.IsTrue(source.owner) || s.manager.IsFalse(source.owner) {
				justification, err = NewSMTEqualityRootJustification(node)
			} else {
				justification, err = NewSMTModusPonensIffJustification(source, node)
			}
			if err != nil {
				return err
			}
			wrapped, err := NewSMTBooleanJustification(justification)
			if err != nil {
				return err
			}
			if err = s.AssignLiteral(literal, wrapped); err != nil {
				return err
			}
			if s.inconsistent {
				return nil
			}
		}
		if node.next == root {
			break
		}
	}
	return nil
}

// The source context creates true/false enodes at initialization. The concrete
// Go context creates them when a Boolean value first participates as a term.
func (s *SMTInternalizerState) booleanValueEnode(value bool) (*SMTEnode, error) {
	var expression ast.ASTHandle
	var err error
	if value {
		expression, err = s.manager.MkTrue()
	} else {
		expression, err = s.manager.MkFalse()
	}
	if err != nil {
		return nil, err
	}
	if node := s.context.enodes[expression]; node != nil {
		return node, nil
	}
	return s.MkEnode(expression, true, true, false)
}

// Source: context::propagate_bool_var_enode. This is also needed when a term
// enode is attached after its Boolean variable has already been assigned.
func (s *SMTInternalizerState) propagateBooleanEnode(variable int32, node *SMTEnode) error {
	if !node.mergeTruthValue {
		return nil
	}
	literal := NewSMTLiteral(variable, false)
	value, err := s.Assignment(literal)
	if err != nil || value == lbool.Undef {
		return err
	}
	constant, err := s.booleanValueEnode(value == lbool.True)
	if err != nil {
		return err
	}
	if value == lbool.False {
		literal = literal.Negated()
	}
	return s.addEquality(node, constant, NewSMTEqualityEquation(literal))
}

func (s *SMTInternalizerState) propagateEqualityClassRelevancy(first, second *SMTEnode) error {
	relevant := false
	for _, root := range []*SMTEnode{first, second} {
		for node := root; ; node = node.next {
			if s.context.relevant[node.owner] {
				relevant = true
				break
			}
			if node.next == root {
				break
			}
		}
		if relevant {
			break
		}
	}
	if !relevant {
		return nil
	}
	for _, root := range []*SMTEnode{first, second} {
		for node := root; ; node = node.next {
			if !s.context.relevant[node.owner] {
				if err := s.markRelevant(node.owner); err != nil {
					return err
				}
			}
			if node.next == root {
				break
			}
		}
	}
	return nil
}

func (s *SMTInternalizerState) addEquality(left, right *SMTEnode, justification SMTEqualityJustification) error {
	if left == nil || right == nil || left.manager != s.manager || right.manager != s.manager {
		return ast.ErrASTDomain
	}
	r1, r2 := left.root, right.root
	s.statistics.NumAddEq++
	if r1 == r2 {
		return nil
	}
	if s.context.isDisequal(r1, r2) {
		proof, err := NewSMTDisequalityConflictJustification(left, right, justification)
		if err != nil {
			return err
		}
		key, _ := smtCheckerPair(r1, r2)
		return s.setEqualityConflict(proof, s.disequalityReasons[key])
	}
	if r1.interpreted && r2.interpreted {
		proof, err := NewSMTEqualityConflictJustification(left, right, justification)
		if err != nil {
			return err
		}
		return s.setEqualityConflict(proof, SMTNullLiteral)
	}
	if (r1.classSize > r2.classSize && !r2.interpreted) || r1.interpreted {
		left, right, r1, r2 = right, left, r2, r1
	}
	s.equalityTrail = append(s.equalityTrail, s.equalitySnapshot())
	if err := s.propagateEqualityClassRelevancy(r1, r2); err != nil {
		return fmt.Errorf("propagate equality-class relevancy: %w", err)
	}
	if err := s.mergeTheoryVariables(r1, r2); err != nil {
		return fmt.Errorf("merge theory variables: %w", err)
	}
	smtInvertTrans(left)
	left.trans = SMTTransJustification{Target: right, Justification: justification}
	for current := r1; ; current = current.next {
		current.root = r2
		if current.next == r1 {
			break
		}
	}
	r1.next, r2.next = r2.next, r1.next
	r2.classSize += r1.classSize
	if s.quantifierManager != nil {
		s.quantifierManager.mam.AddEquality(r1, r2)
	}
	if err := s.propagateBooleanClass(r2); err != nil {
		return fmt.Errorf("propagate Boolean equality class: %w", err)
	}
	if s.inconsistent {
		return nil
	}
	if err := s.remapDisequalities(); err != nil {
		return fmt.Errorf("remap disequalities: %w", err)
	}
	if err := s.rebuildCongruence(true); err != nil {
		return fmt.Errorf("rebuild congruence: %w", err)
	}
	if err := s.propagateTrueEqualities(); err != nil {
		return fmt.Errorf("propagate true equalities: %w", err)
	}
	return nil
}

func (s *SMTInternalizerState) propagateCongruences() error {
	for s.pendingHead < len(s.pending) && !s.inconsistent {
		entry := s.pending[s.pendingHead]
		s.pendingHead++
		if entry.Node == nil || entry.Existing == nil || entry.Node.root == entry.Existing.root {
			continue
		}
		if err := s.addEquality(entry.Node, entry.Existing, NewSMTEqualityCongruence(entry.Commuted)); err != nil {
			return fmt.Errorf("merge pending nodes: %w", err)
		}
	}
	return nil
}
