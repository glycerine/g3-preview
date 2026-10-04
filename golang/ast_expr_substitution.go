// Licensed under the MIT license.
// Concrete expr_substitution: scalar owners and actual encoded backing.
package g3

import "github.com/glycerine/g3/golang/arena"

type ASTExprSubstitution struct {
	origin uint64
	record arena.SmallBytesRef
}
type ASTScopedExprSubstitution struct {
	origin uint64
	record arena.SmallBytesRef
}
type astExprSubstTable struct {
	payload                                      arena.SmallBytesRef
	capacity, size, deleted, index, phase, flags uint32
}
type astExprSubstEntry struct {
	key, value   astNodeRef
	state, owned uint32
}

const astExprSubstBytes uint32 = 128
const astExprSubstMagic uint32 = 0x45535542
const astExprSubstEntryBytes uint32 = 24

func (m *ASTManager) substState(s ASTExprSubstitution) (uint32, error) {
	if s.origin != m.origin {
		return 0, ErrASTHandle
	}
	if size, e := m.allocator.Size(s.record); e != nil || size != astExprSubstBytes {
		return 0, ErrASTHandle
	}
	tag, e := m.readWord(s.record, 100)
	if e != nil {
		return 0, e
	}
	if tag != astExprSubstMagic {
		return 0, ErrASTHandle
	}
	return m.readWord(s.record, 96)
}
func (m *ASTManager) substTable(s ASTExprSubstitution, which uint32) (astExprSubstTable, error) {
	if _, e := m.substState(s); e != nil {
		return astExprSubstTable{}, e
	}
	var a [8]uint32
	for i := range a {
		x, e := m.readWord(s.record, which*32+uint32(i)*4)
		if e != nil {
			return astExprSubstTable{}, e
		}
		a[i] = x
	}
	return astExprSubstTable{arena.SmallBytesRef{Slot: arena.Slot(a[0]), Generation: arena.Generation(a[1])}, a[2], a[3], a[4], a[5], a[6], a[7]}, nil
}
func (m *ASTManager) substStoreTable(s ASTExprSubstitution, which uint32, t astExprSubstTable) error {
	for i, x := range [8]uint32{uint32(t.payload.Slot), uint32(t.payload.Generation), t.capacity, t.size, t.deleted, t.index, t.phase, t.flags} {
		if e := m.writeWord(s.record, which*32+uint32(i)*4, x); e != nil {
			return e
		}
	}
	return nil
}
func astExprSubstTableBytes(capacity uint32, hostMax uint64) (uint32, error) {
	n := uint64(capacity) * uint64(astExprSubstEntryBytes)
	if capacity == 0 || n > uint64(^uint32(0)) || n > hostMax {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
func (m *ASTManager) substAllocTable(capacity uint32) (arena.SmallBytesRef, error) {
	bytes, e := astExprSubstTableBytes(capacity, uint64(^uint(0)>>1))
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	p, e := m.vectorAllocator.Allocate(bytes)
	if e != nil {
		return p, e
	}
	// Source constructors initialize only key/free state. Scalar metadata is
	// initialized concretely without claiming uninitialized source values.
	for i := uint32(0); i < capacity; i++ {
		if e = m.substPutEntry(p, i, astExprSubstEntry{}); e != nil {
			m.vectorAllocator.Deallocate(bytes, p)
			return arena.SmallBytesRef{}, e
		}
	}
	return p, nil
}
func (m *ASTManager) newExprSubstitution(cores, proofs, explicit bool) (ASTExprSubstitution, error) {
	r, e := m.allocator.Allocate(astExprSubstBytes)
	if e != nil {
		return ASTExprSubstitution{}, e
	}
	s := ASTExprSubstitution{m.origin, r}
	for i := uint32(0); i < astExprSubstBytes; i += 4 {
		if e = m.writeWord(r, i, 0); e != nil {
			m.allocator.Deallocate(astExprSubstBytes, r)
			return ASTExprSubstitution{}, e
		}
	}
	flags := uint32(0)
	if proofs {
		flags |= 1
	}
	if cores {
		flags |= 2
	}
	m.writeWord(r, 96, flags)
	m.writeWord(r, 100, astExprSubstMagic)
	fail := func(err error) (ASTExprSubstitution, error) {
		for i := uint32(0); i < 3; i++ {
			t, _ := m.substTable(s, i)
			if t.payload != (arena.SmallBytesRef{}) {
				m.vectorAllocator.Deallocate(t.capacity*24, t.payload)
			}
		}
		m.allocator.Deallocate(astExprSubstBytes, r)
		return ASTExprSubstitution{}, err
	}
	p, e := m.substAllocTable(8)
	if e != nil {
		return fail(e)
	}
	m.substStoreTable(s, 0, astExprSubstTable{payload: p, capacity: 8})
	if explicit && astSourceDebug && proofs && m.proofMode == ASTProofDisabled {
		return fail(ErrASTDomain)
	}
	for i := uint32(1); i < 3; i++ {
		if flags&(1<<(i-1)) != 0 {
			p, e = m.substAllocTable(8)
			if e != nil {
				return fail(e)
			}
			m.substStoreTable(s, i, astExprSubstTable{payload: p, capacity: 8})
		}
	}
	return s, nil
}
func (m *ASTManager) NewExprSubstitution() (ASTExprSubstitution, error) {
	if e := m.lock(); e != nil {
		return ASTExprSubstitution{}, e
	}
	defer m.mu.Unlock()
	return m.newExprSubstitution(false, m.proofMode != ASTProofDisabled, false)
}
func (m *ASTManager) NewExprSubstitutionCores(cores bool) (ASTExprSubstitution, error) {
	if e := m.lock(); e != nil {
		return ASTExprSubstitution{}, e
	}
	defer m.mu.Unlock()
	return m.newExprSubstitution(cores, m.proofMode != ASTProofDisabled, false)
}
func (m *ASTManager) NewExprSubstitutionPolicy(cores, proofs bool) (ASTExprSubstitution, error) {
	if e := m.lock(); e != nil {
		return ASTExprSubstitution{}, e
	}
	defer m.mu.Unlock()
	return m.newExprSubstitution(cores, proofs, true)
}
func (m *ASTManager) ExprSubstitutionManager(s ASTExprSubstitution) (*ASTManager, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	if _, e := m.substState(s); e != nil {
		return nil, e
	}
	return m, nil
}
func (m *ASTManager) ExprSubstitutionProofsEnabled(s ASTExprSubstitution) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	f, e := m.substState(s)
	return f&1 != 0, e
}
func (m *ASTManager) ExprSubstitutionUnsatCoreEnabled(s ASTExprSubstitution) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	f, e := m.substState(s)
	return f&2 != 0, e
}
func (m *ASTManager) ExprSubstitutionEmpty(s ASTExprSubstitution) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	t, e := m.substTable(s, 0)
	return t.size == 0, e
}
