// Licensed under the MIT license.
// Source delete_node ascending pushes/LIFO pops. Intrusive emergency metadata
// is a Go continuation adapter only when source worklist allocation would throw.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

// consumed reports whether the input reference was consumed, even when a
// deletion-worklist reserve error is returned after finishing its continuation.
func (m *ASTManager) decNode(r astNodeRef) (consumed bool, err error) {
	if r.slot == 0 {
		return true, nil
	}
	if e := m.validNode(r); e != nil {
		return false, e
	}
	n := m.nodeAt(r.slot)
	if n.rc == 0 {
		return false, ErrASTDomain
	}
	n.rc--
	if n.rc != 0 {
		return true, nil
	}
	return true, m.deleteNode(r)
}
func (m *ASTManager) deleteNode(first astNodeRef) error {
	var inline [16]astNodeRef
	stack := inline[:0]
	var overflowBytes uint64
	emergency := false
	var firstErr error
	parent, previousHead := m.deletionCurrent, m.deletionHead
	complete := false
	defer func() {
		if !complete {
			m.pendingDeletionCurrent = m.deletionCurrent
			m.pendingDeletionHead = m.deletionHead
		}
		m.budget.Release(overflowBytes)
		m.deletionCurrent = parent
		m.deletionHead = previousHead
	}()
	head := astNodeRef{}
	note := func(e error) {
		if e != nil && firstErr == nil {
			firstErr = e
		}
	}
	push := func(r astNodeRef) {
		n := m.nodeAt(r.slot)
		n.deleting = true
		n.deleteParent = parent
		n.deleteNext = head
		head = r
		m.deletionHead = head
		if m.deletionCurrent.slot != 0 && m.deletionCurrent != parent {
			m.nodeAt(m.deletionCurrent.slot).deleteNext = head
		}
		if emergency {
			return
		}
		if len(stack) == cap(stack) {
			count := uint64(cap(stack)) * 2
			if count > math.MaxUint32 || count > uint64(^uint(0)>>1)/8 {
				note(arena.ErrBounds)
				emergency = true
				return
			}
			bytes := count * 8
			if e := m.budget.Reserve(bytes); e != nil {
				note(e)
				emergency = true
				return
			}
			next := make([]astNodeRef, len(stack), int(count))
			copy(next, stack)
			m.budget.Release(overflowBytes)
			overflowBytes = bytes
			stack = next
		}
		stack = append(stack, r)
	}
	push(first)
	for head.slot != 0 {
		current := head
		n := m.nodeAt(current.slot)
		head = n.deleteNext
		m.deletionHead = head
		if !emergency {
			if astSourceDebug && stack[len(stack)-1] != current {
				note(ErrASTDomain)
			}
			stack = stack[:len(stack)-1]
		}
		n.deleteNext = head
		n.deleteParent = parent
		n.deletePhase = 0
		n.deleteIndex = 0
		m.deletionCurrent = current
		if e := m.deletionBefore(current); e != nil {
			return e
		}
		if e := m.tableEraseIdentity(current); e != nil {
			return e
		}
		if e := m.deletionAfter(current); e != nil {
			return e
		}
		n = m.nodeAt(current.slot)
		generator := &m.exprIDs
		if n.kind == ASTSort || n.kind == ASTFuncDecl {
			generator = &m.declIDs
		}
		// Publication preflight normally makes this infallible. The source OOM
		// callback still controls its real recycle skip at this exact action site.
		note(generator.recycle(n.id))
		n.deletePhase = 1
		slot, generation := m.infoForNode(current)
		if slot != 0 {
			info, e := m.info(slot, generation)
			if e != nil {
				return e
			}
			for n.deleteIndex < info.count {
				p, e := m.infoParameter(info, n.deleteIndex)
				if e != nil {
					return e
				}
				v := m.parameterAt(p.slot)
				if v.kind == ASTParameterAST {
					_, e = m.decNode(v.ast)
					note(e)
				}
				// All five source eager plugins inherit actual default external del(),
				// which has no action. A cleared registry also has no callback.
				n = m.nodeAt(current.slot)
				n.deleteIndex++
			}
			info.attached = false
			if e = m.dropInfoStorage(slot, generation); e != nil {
				return e
			}
		}
		n = m.nodeAt(current.slot)
		n.deletePhase = 2
		n.deleteIndex = 0
		count := m.nodeChildCount(current)
		for n.deleteIndex < count {
			child, e := m.nodeChild(current, n.deleteIndex)
			if e != nil {
				return e
			}
			if child.slot != 0 {
				if e = m.validNode(child); e != nil {
					return e
				}
				cn := m.nodeAt(child.slot)
				if cn.rc == 0 {
					return ErrASTDomain
				}
				cn.rc--
				if cn.rc == 0 {
					push(child)
				}
			}
			n = m.nodeAt(current.slot)
			n.deleteIndex++
		}
		n.deletePhase = 3
		if e := m.freeNodeStorage(current); e != nil {
			return e
		}
		m.deletionCurrent = parent
		m.deletionHead = head
	}
	complete = true
	return firstErr
}
