// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

func astExprArrayOffset(i uint32) (uint32, error) {
	n := uint64(astExprArrayHeaderBytes) + uint64(i)*8
	if n+8 > math.MaxUint32 {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
func (m *astExprArrayManager) rootSet(block arena.SmallBytesRef, i uint32, v astNodeRef) error {
	old, e := m.value(block, i)
	if e != nil {
		return e
	}
	if e = m.owner.incNode(v); e != nil {
		return e
	}
	consumed, err := m.owner.decNode(old)
	if !consumed {
		m.owner.decNode(v)
		return err
	}
	if e = m.setValue(block, i, v); e != nil {
		return e
	}
	return err
}
func (m *astExprArrayManager) rootPop(c *astExprArrayCell) error {
	if c.idx == 0 {
		return arena.ErrBounds
	}
	c.idx--
	v, e := m.value(c.link, c.idx)
	if e != nil {
		return e
	}
	_, e = m.owner.decNode(v)
	return e
}
func (m *astExprArrayManager) dropOwnedValues(block arena.SmallBytesRef, n uint32) error {
	var failure error
	for i := uint32(0); i < n; i++ {
		v, e := m.value(block, i)
		if e == nil {
			_, e = m.owner.decNode(v)
		}
		if failure == nil {
			failure = e
		}
	}
	m.freeValues(block)
	return failure
}
func (m *astExprArrayManager) closeAfterNodeTeardown() {
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
func (m *astExprArrayManager) decConsumed(r arena.SmallBytesRef) (bool, error) {
	if pArrayNull(r) {
		return true, nil
	}
	c, e := m.cell(r)
	if e != nil || c.rc == 0 {
		return false, ErrASTHandle
	}
	return true, m.dec(r)
}
