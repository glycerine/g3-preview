//go:build g3debug

package ast

const ASTSourceDebug = true

func (m *ASTManager) registrationBefore(r astNodeRef) (bool, error) {
	_, contains, e := m.tableFind(r)
	if e != nil {
		return false, e
	}
	if !contains && m.debugNondeterminism {
		count := uint32(0)
		for _, curr := range m.table.entries.All() {
			equal, e := m.compareNodes(curr, r)
			if e != nil {
				return false, e
			}
			if equal {
				return false, ErrASTDomain
			}
			if m.nodeAt(curr.slot).kind == ASTApp && m.nodeAt(r.slot).kind == ASTApp {
				a, b := m.appAt(curr.slot), m.appAt(r.slot)
				if a.decl == b.decl && a.count == 0 && b.count == 0 {
					return false, ErrASTDomain
				}
			}
			count++
		}
		if count != m.table.size {
			return false, ErrASTDomain
		}
	}
	return contains, nil
}
func (m *ASTManager) registrationAfter(candidate, result astNodeRef, contains, unique bool) error {
	if m.nodeAt(candidate.slot).hash != m.nodeAt(result.slot).hash {
		return ErrASTDomain
	}
	if unique == contains {
		return ErrASTDomain
	}
	_, found, e := m.tableFind(candidate)
	if e != nil {
		return e
	}
	if !found {
		return ErrASTDomain
	}
	return nil
}
func (m *ASTManager) deletionBefore(r astNodeRef) error {
	_, yes, e := m.tableFind(r)
	if e != nil {
		return e
	}
	if !yes {
		return ErrASTDomain
	}
	return nil
}
func (m *ASTManager) deletionAfter(r astNodeRef) error {
	_, yes, e := m.tableFind(r)
	if e != nil {
		return e
	}
	if yes {
		return ErrASTDomain
	}
	return nil
}
