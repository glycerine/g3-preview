// Source: src/smt/smt_context.cpp is_shared and
// src/smt/theory_arith_aux.h arithmetic is_shared override.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (t *SMTLinearArithmeticTheory) sharedArithmeticNode(node *SMTEnode) (bool, error) {
	return smtSharedArithmeticNode(t.SMTTheoryBase, node, len(t.divisionTerms) != 0)
}

func smtSharedArithmeticNode(owner *SMTTheoryBase, node *SMTEnode, division bool) (bool, error) {
	root := node.root
	s := owner.state
	family, err := s.manager.GetFamilyID(root.owner)
	if err != nil {
		return false, err
	}
	kind, err := s.manager.GetDeclKind(root.owner)
	if err != nil {
		return false, err
	}
	if family == s.manager.BasicFamilyID() && kind == ast.ASTOpIte {
		return true, nil
	}
	count := root.NumTheoryVars()
	if count == 0 {
		return false, nil
	}
	if count > 1 {
		return true, nil
	}
	if q := s.quantifierManager; q != nil && len(q.activated) != 0 && q.mam != nil && q.mam.IsShared(root) {
		return true, nil
	}
	theory := root.theoryVariables.theory
	for _, parent := range root.parents {
		family, err := s.manager.GetFamilyID(parent.owner)
		if err != nil {
			return false, err
		}
		if family != theory && family != s.manager.BasicFamilyID() {
			return true, nil
		}
		if family == owner.FamilyID() && division {
			kind, err := s.manager.GetDeclKind(parent.owner)
			if err != nil {
				return false, err
			}
			if kind == smtArithOpDiv || kind == smtArithOpIDiv || kind == smtArithOpRem || kind == smtArithOpMod {
				return true, nil
			}
		}
	}
	return false, nil
}
