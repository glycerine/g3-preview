// Licensed under the MIT license.
package g3

func (m *ASTManager) nodeMark(r astNodeRef, plane uint8, owner uint64, op uint8, flag bool) (bool, error) {
	if e := m.validNode(r); e != nil {
		return false, e
	}
	n := m.nodeAt(r.slot)
	oldOwner := n.markOwner1
	bit := n.mark1
	if plane == 2 {
		oldOwner = n.markOwner2
		bit = n.mark2
	}
	if astSourceDebug && oldOwner != 0 && oldOwner != owner {
		return false, ErrASTDomain
	}
	if op == 0 {
		return bit, nil
	}
	if op == 2 {
		flag = false
		owner = 0
	}
	if plane == 1 {
		n.mark1 = flag
		if astSourceDebug {
			n.markOwner1 = owner
		}
	} else {
		n.mark2 = flag
		if astSourceDebug {
			n.markOwner2 = owner
		}
	}
	return flag, nil
}
func (m *ASTManager) publicNodeMark(h ASTHandle, plane uint8, owner uint64, op uint8, flag bool) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.internal(h)
	if e != nil {
		return false, e
	}
	return m.nodeMark(r, plane, owner, op, flag)
}
func (m *ASTManager) MarkNode1(h ASTHandle, flag bool, owner uint64) error {
	_, e := m.publicNodeMark(h, 1, owner, 1, flag)
	return e
}
func (m *ASTManager) MarkNode2(h ASTHandle, flag bool, owner uint64) error {
	_, e := m.publicNodeMark(h, 2, owner, 1, flag)
	return e
}
func (m *ASTManager) ResetNodeMark1(h ASTHandle, owner uint64) error {
	_, e := m.publicNodeMark(h, 1, owner, 2, false)
	return e
}
func (m *ASTManager) ResetNodeMark2(h ASTHandle, owner uint64) error {
	_, e := m.publicNodeMark(h, 2, owner, 2, false)
	return e
}
func (m *ASTManager) IsNodeMarked1(h ASTHandle, owner uint64) (bool, error) {
	return m.publicNodeMark(h, 1, owner, 0, false)
}
func (m *ASTManager) IsNodeMarked2(h ASTHandle, owner uint64) (bool, error) {
	return m.publicNodeMark(h, 2, owner, 0, false)
}
