package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

type ASTSharedOccurrencesConfig struct{ TrackAtomic, VisitQuantifiers, VisitPatterns bool }

func DefaultASTSharedOccurrencesConfig() ASTSharedOccurrencesConfig {
	return ASTSharedOccurrencesConfig{VisitQuantifiers: true}
}

type astSharedFrame struct {
	node astNodeRef
	next uint32
}
type ASTSharedOccurrences struct {
	owner         astMarkOwner
	table         astExprIdentityTable
	config        ASTSharedOccurrencesConfig
	stack         []astSharedFrame
	stackSize     uint32
	stackBytes    uint64
	releaseCursor uint32
	resetPhase    uint8
}

func NewASTSharedOccurrences(m *ASTManager) (*ASTSharedOccurrences, error) {
	return NewASTSharedOccurrencesWithConfig(m, DefaultASTSharedOccurrencesConfig())
}
func NewASTSharedOccurrencesWithConfig(m *ASTManager, c ASTSharedOccurrencesConfig) (*ASTSharedOccurrences, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	bytes := uint64(reflect.TypeOf(ASTSharedOccurrences{}).Size())
	id, e := prepareASTMarkOwner(m, bytes)
	if e != nil {
		return nil, e
	}
	s := &ASTSharedOccurrences{config: c}
	s.owner.bind(m, bytes, id)
	if e = s.table.init(m); e != nil {
		s.owner.finish()
		return nil, e
	}
	return s, nil
}
func (s *ASTSharedOccurrences) push(m *ASTManager, r astNodeRef) error {
	if s.stackSize == uint32(len(s.stack)) {
		n, e := astSharedFrameNextCapacity(uint64(len(s.stack)), uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		bytes := uint64(n) * uint64(reflect.TypeOf(astSharedFrame{}).Size())
		if e = m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		a := make([]astSharedFrame, n)
		copy(a, s.stack[:s.stackSize])
		m.BudgetField.Release(s.stackBytes)
		s.stack = a
		s.stackBytes = bytes
	}
	s.stack[s.stackSize] = astSharedFrame{node: r}
	s.stackSize++
	return nil
}
func (s *ASTSharedOccurrences) insert(m *ASTManager, r astNodeRef) error {
	fresh, e := s.table.insert(m, r)
	if e != nil {
		return e
	}
	if fresh {
		return m.incNode(r)
	}
	return nil
}
func (s *ASTSharedOccurrences) process(m *ASTManager, r astNodeRef, v *astSharedMarkState) (bool, error) {
	if e := m.validNode(r); e != nil {
		return false, e
	}
	n := m.nodeAt(r.slot)
	switch n.kind {
	case ASTApp:
		count := m.appAt(r.slot).count
		if n.rc > 1 && (s.config.TrackAtomic || count > 0) {
			if n.shared {
				return true, s.insert(m, r)
			}
			if e := v.mark(m, r, true); e != nil {
				return false, e
			}
		}
		if count == 0 {
			return true, nil
		}
		return false, s.push(m, r)
	case ASTVar:
		if s.config.TrackAtomic && n.rc > 1 {
			if n.shared {
				return true, s.insert(m, r)
			}
			return true, v.mark(m, r, true)
		}
		return true, nil
	case ASTQuantifier:
		if n.rc > 1 {
			if n.shared {
				return true, s.insert(m, r)
			}
			if e := v.mark(m, r, true); e != nil {
				return false, e
			}
		}
		if !s.config.VisitQuantifiers {
			return true, nil
		}
		return false, s.push(m, r)
	default:
		return false, ErrASTDomain
	}
}
func (s *ASTSharedOccurrences) collectLocked(m *ASTManager, r astNodeRef, v *astSharedMarkState) error {
	if ASTSourceDebug && s.stackSize != 0 {
		return ErrASTDomain
	}
	done, e := s.process(m, r, v)
	if e != nil || done {
		return e
	}
	for s.stackSize > 0 {
		idx := s.stackSize - 1
		fr := s.stack[idx]
		if e = m.validNode(fr.node); e != nil {
			return e
		}
		n := m.nodeAt(fr.node.slot)
		var count uint32
		var child astNodeRef
		switch n.kind {
		case ASTApp:
			a := m.appAt(fr.node.slot)
			count = a.count
			if fr.next < count {
				child, e = m.readInternal(a.args, fr.next)
			}
		case ASTQuantifier:
			if ASTSourceDebug && !s.config.VisitQuantifiers {
				return ErrASTDomain
			}
			q := m.quantifierAt(fr.node.slot)
			count = 1
			if s.config.VisitPatterns {
				count += q.patternCount + q.noPatternCount
			}
			if fr.next < count {
				if fr.next == 0 {
					child = q.body
				} else if fr.next <= q.patternCount {
					child, e = m.readInternal(q.patterns, fr.next-1)
				} else {
					child, e = m.readInternal(q.noPatterns, fr.next-q.patternCount-1)
				}
			}
		default:
			return ErrASTDomain
		}
		if e != nil {
			return e
		}
		if fr.next == count {
			s.stackSize--
			continue
		}
		s.stack[idx].next++
		if _, e = s.process(m, child, v); e != nil {
			return e
		}
	}
	return nil
}
func (s *ASTSharedOccurrences) resetLocked(m *ASTManager) error {
	if s.resetPhase == 0 {
		s.resetPhase = 1
		s.releaseCursor = 0
	}
	if s.resetPhase == 1 {
		for s.releaseCursor < uint32(len(s.table.entries)) {
			r := s.table.entries[s.releaseCursor]
			if !astExprUsed(r) {
				s.releaseCursor++
				continue
			}
			consumed, e := m.decNode(r)
			if consumed {
				s.releaseCursor++
			}
			if e != nil {
				return e
			}
			if !consumed {
				return ErrASTDomain
			}
		}
		s.resetPhase = 2
	}
	if s.resetPhase == 2 {
		if e := s.table.reset(m); e != nil {
			return e
		}
		s.resetPhase = 3
	}
	if e := s.table.reset(m); e != nil {
		return e
	}
	s.resetPhase = 0
	s.releaseCursor = 0
	return nil
}
func (s *ASTSharedOccurrences) Collect(h ASTHandle) (result error) {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	if ASTSourceDebug && s.stackSize != 0 {
		return ErrASTDomain
	}
	var v astSharedMarkState
	defer func() {
		if e := v.close(m); result == nil {
			result = e
		}
	}()
	if e = s.resetLocked(m); e != nil {
		return e
	}
	r, e := m.internal(h)
	if e != nil {
		return e
	}
	return s.collectLocked(m, r, &v)
}
func (s *ASTSharedOccurrences) CollectWithMark(h ASTHandle, v *ASTSharedOccurrencesMark) error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	if e = v.owner.check(); e != nil {
		return e
	}
	if v.owner.manager != m {
		return ErrASTHandle
	}
	if s.resetPhase != 0 {
		return arena.ErrBounds
	}
	r, e := m.internal(h)
	if e != nil {
		return e
	}
	return s.collectLocked(m, r, &v.state)
}
func (s *ASTSharedOccurrences) IsShared(h ASTHandle) (bool, error) {
	m, e := s.owner.lock()
	if e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return false, e
	}
	_, yes, e := s.table.find(m, r)
	return yes, e
}
func (s *ASTSharedOccurrences) NumShared() uint32 { return s.table.size }
func (s *ASTSharedOccurrences) Reset() error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	return s.resetLocked(m)
}
func (s *ASTSharedOccurrences) Cleanup() error {
	m, e := s.owner.lock()
	if e != nil {
		return e
	}
	defer m.mu.Unlock()
	if e = s.resetLocked(m); e != nil {
		return e
	}
	if e = s.table.finalize(m); e != nil {
		return e
	}
	m.BudgetField.Release(s.stackBytes)
	s.stack = nil
	s.stackBytes = 0
	s.stackSize = 0
	return nil
}
func (s *ASTSharedOccurrences) Close() error {
	if e := s.owner.check(); e != nil {
		return e
	}
	m := s.owner.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.Closed {
		if e := s.resetLocked(m); e != nil {
			return e
		}
	}
	m.BudgetField.Release(s.stackBytes)
	s.stack = nil
	s.stackBytes = 0
	s.table.close(m)
	s.owner.finish()
	return nil
}

// Source svector<pair<expr*,unsigned>> selects the LP64 reference extent16
// plus two unsigned32 header words. Actual Go scalar frames occupy12 bytes;
// charge that physical stride and check its host extent independently.
func astSharedFrameNextCapacity(old, hostMax uint64) (uint32, error) {
	if old > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	n := uint32(2)
	if old != 0 {
		n = (3*uint32(old) + 1) >> 1
	}
	sourceBytes := uint64(n)*16 + 8
	if uint64(n) <= old || sourceBytes > uint64(^uint32(0)) || uint64(n) > hostMax/uint64(reflect.TypeOf(astSharedFrame{}).Size()) {
		return 0, arena.ErrBounds
	}
	return n, nil
}
