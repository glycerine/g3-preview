// Source: theory_datatype::sign_recognizer_conflict and merge_eh.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *SMTDatatypeTheory) setDatatypeConflict(literals []SMTLiteral, equalities []SMTEnodePair) error {
	s, err := t.State()
	if err != nil || s.Inconsistent() {
		return err
	}
	owner, err := NewSMTExtTheoryConflictJustification(s.manager, t.FamilyID(), literals, equalities, nil)
	if err != nil {
		return err
	}
	wrapped, err := NewSMTBooleanJustification(owner)
	if err != nil {
		return errors.Join(err, owner.Release(s.manager))
	}
	s.inconsistent = true
	s.conflict = wrapped
	s.conflictLiteral = SMTNullLiteral
	s.statistics.NumConflicts++
	return nil
}

// Source var_data stores this constructor at the datatype union-find root.
// The Go context already owns the scoped equivalence class, so inspect that
// class directly and retain its actual constructor enodes as evidence.
func (t *SMTDatatypeTheory) classConstructor(node *SMTEnode) (*SMTEnode, error) {
	if node == nil {
		return nil, ast.ErrASTDomain
	}
	s, err := t.State()
	if err != nil {
		return nil, err
	}
	var constructor *SMTEnode
	root := node.root
	for member := root; ; member = member.next {
		_, found, err := s.manager.DatatypeConstructorApp(member.owner)
		if err != nil {
			return nil, err
		}
		if found {
			if constructor != nil && constructor.declaration != member.declaration {
				return constructor, t.setDatatypeConflict(nil, []SMTEnodePair{{constructor, member}})
			}
			constructor = member
		}
		if member.next == root {
			break
		}
	}
	return constructor, nil
}
