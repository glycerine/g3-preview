package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

func astDependencyArrayOffset(i uint32) (uint32, error) {
	n := uint64(astDependencyArrayHeaderBytes) + uint64(i)*8
	if n+8 > math.MaxUint32 {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
func (m *astDependencyArrayManager) rootSet(block arena.SmallBytesRef, i uint32, v arena.SmallBytesRef) error {
	if e := m.owner.dependencies.inc(v); e != nil {
		return e
	}
	old, e := m.value(block, i)
	if e != nil {
		return e
	} // source INC prefix survives invalid physical access.
	consumed, err := m.valueDec(old)
	if !consumed {
		return err // retain the actual source incoming INC prefix.
	}
	if e = m.setValue(block, i, v); e != nil {
		return e
	}
	return err
}
func (m *astDependencyArrayManager) rootPop(c *astDependencyArrayCell) error {
	if c.idx == 0 {
		return arena.ErrBounds
	}
	c.idx--
	v, e := m.value(c.link, c.idx)
	if e != nil {
		return e
	}
	_, e = m.valueDec(v)
	return e
}
func (m *astDependencyArrayManager) dropOwnedValues(block arena.SmallBytesRef, n uint32) error {
	var failure error
	for i := uint32(0); i < n; i++ {
		v, e := m.value(block, i)
		if e == nil {
			_, e = m.valueDec(v)
		}
		if failure == nil {
			failure = e
		}
	}
	m.freeValues(block)
	return failure
}
func (m *astDependencyArrayManager) closeAfterNodeTeardown() {
	// Source ast_manager destroys its nodes before allocator/member teardown.
	// Escaped array tokens therefore cannot invoke their AST callbacks again.
	// The enclosing allocator reclaims all remaining encoded records and values.
	m.budget.Release(uint64(cap(m.getTmp)+cap(m.rerootTmp)) * 8)
	m.getTmp = nil
	m.rerootTmp = nil
	m.liveCells = 0
	m.liveBlocks = 0
	m.valueBytes = 0
	m.closed = true
}

// A valid cell's reference is consumed before any AST domain destructor can
// report an emergency-worklist error. Callers must publish/clear their token
// after that error rather than retaining a dangling source union pointer.
func (m *astDependencyArrayManager) decConsumed(r arena.SmallBytesRef) (bool, error) {
	if astPArrayNull(r) {
		return true, nil
	}
	c, e := m.cell(r)
	if e != nil {
		return false, e
	}
	if ASTSourceDebug && c.rc == 0 {
		return false, ErrASTDomain
	}
	return true, m.dec(r)
}

func (m *astDependencyArrayManager) valueDec(r arena.SmallBytesRef) (bool, error) {
	if dependencyNull(r) {
		return true, nil
	}
	c, e := m.owner.dependencies.node(r)
	if e != nil {
		return false, e
	}
	if ASTSourceDebug && c.rc == 0 {
		return false, ErrASTDomain
	}
	return true, m.owner.dependencies.dec(r)
}
