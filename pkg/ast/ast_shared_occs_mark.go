package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Inline storage has no self-referential slice, allowing the local source
// visited marker to remain on the stack. It borrows all listed nodes.
type astSharedMarkState struct {
	inline [16]astNodeRef
	heap   []astNodeRef
	size   uint32
	bytes  uint64
}

func (s *astSharedMarkState) at(i uint32) astNodeRef {
	if s.heap == nil {
		return s.inline[i]
	}
	return s.heap[i]
}
func (s *astSharedMarkState) mark(m *ASTManager, r astNodeRef, flag bool) error {
	if e := m.validNode(r); e != nil {
		return e
	}
	n := m.nodeAt(r.slot)
	if !flag {
		n.shared = false
		return nil
	}
	if n.shared {
		return nil
	}
	n.shared = true
	cap := uint32(16)
	if s.heap != nil {
		cap = uint32(len(s.heap))
	}
	if s.size == cap {
		next := uint64(cap) * 2
		bytes := next * 8
		if next > uint64(^uint32(0)) || bytes > uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		a := make([]astNodeRef, int(next))
		if s.heap == nil {
			copy(a, s.inline[:s.size])
		} else {
			copy(a, s.heap[:s.size])
		}
		m.BudgetField.Release(s.bytes)
		s.heap = a
		s.bytes = bytes
	}
	if s.heap == nil {
		s.inline[s.size] = r
	} else {
		s.heap[s.size] = r
	}
	s.size++
	return nil
}
func (s *astSharedMarkState) reset(m *ASTManager) error {
	for s.size > 0 { // source visits ASC; shift cursor only after a successful reset
		for i := uint32(0); i < s.size; i++ {
			r := s.at(i)
			if e := m.validNode(r); e != nil {
				return e
			}
			m.nodeAt(r.slot).shared = false
		}
		s.size = 0
	}
	return nil
}
func (s *astSharedMarkState) close(m *ASTManager) error {
	if !m.Closed {
		if e := s.reset(m); e != nil {
			return e
		}
	}
	m.BudgetField.Release(s.bytes)
	s.heap = nil
	s.bytes = 0
	s.size = 0
	return nil
}

type ASTSharedOccurrencesMark struct {
	owner astMarkOwner
	state astSharedMarkState
}

func NewASTSharedOccurrencesMark(m *ASTManager) (*ASTSharedOccurrencesMark, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	bytes := uint64(reflect.TypeOf(ASTSharedOccurrencesMark{}).Size())
	id, e := prepareASTMarkOwner(m, bytes)
	if e != nil {
		return nil, e
	}
	s := &ASTSharedOccurrencesMark{}
	s.owner.bind(m, bytes, id)
	return s, nil
}
func (s *ASTSharedOccurrencesMark) IsMarked(h ASTHandle) (bool, error) {
	m, e := s.owner.lock()
	if e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.internal(h)
	if e != nil {
		return false, e
	}
	return m.nodeAt(r.slot).shared, nil
}
func (s *ASTSharedOccurrencesMark) Mark(h ASTHandle, flag bool) error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.internal(h)
	if e != nil {
		return e
	}
	return s.state.mark(m, r, flag)
}
func (s *ASTSharedOccurrencesMark) ResetMark(h ASTHandle) error { return s.Mark(h, false) }
func (s *ASTSharedOccurrencesMark) Reset() error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	return s.state.reset(m)
}
func (s *ASTSharedOccurrencesMark) Close() error {
	if e := s.owner.check(); e != nil {
		return e
	}
	m := s.owner.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := s.state.close(m); e != nil {
		return e
	}
	s.owner.finish()
	return nil
}
