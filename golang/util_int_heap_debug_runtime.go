// Source: src/util/heap.h:61-78,92-267; src/util/debug.h:60-65;
// src/util/vector.h:303-310,389-405. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2006-09-14.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// These gates are evaluated at the actual macro site, never as a constructor
// or operation-wide readiness preflight. Release erases the whole lookup.
func (h *IntHeap) assertionsRuntime() (bool, error) {
	if !satStorageSourceDebug {
		return false, nil
	}
	if h.debugContext == nil {
		return true, nil
	}
	return h.debugContext.AssertionsEnabled()
}
func (h *IntHeap) cassertRuntime() error {
	if !satStorageSourceDebug || h.debugContext == nil {
		return nil
	}
	on, e := h.debugContext.ConditionalAssertionEnabled("heap")
	if e != nil || !on {
		return e
	}
	ok, e := h.invariantRuntime()
	if e != nil {
		return e
	}
	if !ok {
		return ErrSATAssertion
	}
	return nil
}
func (q *satVarQueue) assertionsRuntime() (bool, error) {
	if q.generalSource != nil {
		return q.generalSource.assertionsRuntime()
	}
	return satStorageSourceDebug, nil
}
func (q *satVarQueue) cassertRuntime() error {
	if q.generalSource != nil {
		return q.generalSource.cassertRuntime()
	}
	return nil
}

// Only the source operator[] logical-size assertion is conditional. Runtime
// initialized-byte, actual physical extent and provenance guards always run.
func (h *IntHeap) readUintRuntime(c *satSolverRuntimeUintColumn, i uint32) (uint32, error) {
	if e := c.raw.open(); e != nil {
		return 0, e
	}
	on, e := h.assertionsRuntime()
	if e != nil {
		return 0, e
	}
	if on {
		n, e := c.size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrSATAssertion
		}
	}
	if c.raw.block == (memory.Block{}) {
		return 0, arena.ErrBounds
	}
	return satAssignmentReadWord(c.raw.runtime, c.raw.block, 8+4*uint64(i))
}
func (h *IntHeap) writeUintRuntime(c *satSolverRuntimeUintColumn, i, v uint32) error {
	if e := c.raw.open(); e != nil {
		return e
	}
	on, e := h.assertionsRuntime()
	if e != nil {
		return e
	}
	if on {
		n, e := c.size()
		if e != nil {
			return e
		}
		if i >= n {
			return ErrSATAssertion
		}
	}
	if c.raw.block == (memory.Block{}) {
		return arena.ErrBounds
	}
	return satAssignmentWriteWord(c.raw.runtime, c.raw.block, 8+4*uint64(i), v)
}
func (q *satVarQueue) readUintRuntime(c *satSolverRuntimeUintColumn, i uint32) (uint32, error) {
	if q.generalSource != nil {
		return q.generalSource.readUintRuntime(c, i)
	}
	return c.read(i)
}
func (q *satVarQueue) writeUintRuntime(c *satSolverRuntimeUintColumn, i, v uint32) error {
	if q.generalSource != nil {
		return q.generalSource.writeUintRuntime(c, i, v)
	}
	return c.write(i, v)
}
func (h *IntHeap) shrinkUintRuntime(c *satSolverRuntimeUintColumn, n uint32) error {
	if e := c.raw.open(); e != nil {
		return e
	}
	on, e := h.assertionsRuntime()
	if e != nil {
		return e
	}
	if c.raw.block == (memory.Block{}) {
		if on && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if on {
		old, e := c.size()
		if e != nil {
			return e
		}
		if n > old {
			return ErrSATAssertion
		}
	}
	return satAssignmentWriteWord(c.raw.runtime, c.raw.block, 4, n)
}

// The frame's first SIZE read is outside SASSERT, even in release. Every later
// assertion has its own short-circuit gate; recursive frames exist only when
// the corresponding source SASSERT actually evaluates the recursive call.
func (h *IntHeap) invariantRuntime() (bool, error) {
	if e := h.alive(); e != nil {
		return false, e
	}
	return h.invariantRuntimeAt(1)
}
func (h *IntHeap) invariantRuntimeAt(i int32) (bool, error) {
	q := &h.runtimeQueue
	n, e := q.runtimeValues.size()
	if e != nil {
		return false, e
	}
	if i >= int32(n) {
		return true, nil
	}
	on, e := h.assertionsRuntime()
	if e != nil {
		return false, e
	}
	if on {
		v, e := q.valueRuntime(i)
		if e != nil {
			return false, e
		}
		j, e := q.indexRuntime(v)
		if e != nil {
			return false, e
		}
		if j != i {
			return false, ErrSATAssertion
		}
	}
	on, e = h.assertionsRuntime()
	if e != nil {
		return false, e
	}
	if on && i>>1 != 0 {
		v, e := q.valueRuntime(i)
		if e != nil {
			return false, e
		}
		p, e := q.valueRuntime(i >> 1)
		if e != nil {
			return false, e
		}
		less, e := q.lessRuntime(v, p)
		if e != nil {
			return false, e
		}
		if less {
			return false, ErrSATAssertion
		}
	}
	on, e = h.assertionsRuntime()
	if e != nil {
		return false, e
	}
	if on {
		left, e := satHeapLeft(i)
		if e != nil {
			return false, e
		}
		ok, e := h.invariantRuntimeAt(left)
		if e != nil || !ok {
			return ok, e
		}
	}
	on, e = h.assertionsRuntime()
	if e != nil {
		return false, e
	}
	if on {
		left, e := satHeapLeft(i)
		if e != nil {
			return false, e
		}
		ok, e := h.invariantRuntimeAt(left + 1)
		if e != nil || !ok {
			return ok, e
		}
	}
	return true, nil
}

// Source placement new and memset are not operator[] expressions; no logical
// source assertion or tag lookup is inserted into their ascending stores.
func (h *IntHeap) pushUintRuntime(c *satSolverRuntimeUintColumn, v uint32) error {
	if e := c.raw.open(); e != nil {
		return e
	}
	if c.raw.pending != (memory.Block{}) {
		return ErrSATOwner
	}
	n, e := c.size()
	if e != nil {
		return e
	}
	capacity, e := c.capacity()
	if e != nil {
		return e
	}
	if c.raw.block == (memory.Block{}) || n == capacity {
		if e = c.raw.expandStrideDebug(4, h.debugContext); e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(c.raw.runtime, c.raw.block, 8+4*uint64(n), v); e != nil {
		return e
	}
	return satAssignmentWriteWord(c.raw.runtime, c.raw.block, 4, n+1)
}
func (q *satVarQueue) pushValuesRuntime(v uint32) error {
	if q.generalSource != nil {
		return q.generalSource.pushUintRuntime(&q.runtimeValues, v)
	}
	return q.runtimeValues.push(v)
}
func (q *satVarQueue) resetIndexRuntime(i uint32) error {
	if q.generalSource != nil {
		c := &q.runtimeIndices.raw
		return satAssignmentWriteWord(c.runtime, c.block, 8+4*uint64(i), 0)
	}
	return q.runtimeIndices.write(i, 0)
}
func (q *satVarQueue) resetValuesRuntime() error {
	if q.generalSource != nil {
		c := &q.runtimeValues.raw
		if c.block == (memory.Block{}) {
			return nil
		}
		return satAssignmentWriteWord(c.runtime, c.block, 4, 0)
	}
	return q.runtimeValues.shrink(0)
}

// back/pop each have their own source SASSERT; the unconditional back SIZE
// computation and pop header decrement retain their actual read positions.
func (h *IntHeap) backUintRuntime(c *satSolverRuntimeUintColumn) (uint32, error) {
	on, e := h.assertionsRuntime()
	if e != nil {
		return 0, e
	}
	if on {
		n, e := c.size()
		if e != nil {
			return 0, e
		}
		if n == 0 {
			return 0, ErrSATAssertion
		}
	}
	n, e := c.size()
	if e != nil {
		return 0, e
	}
	return h.readUintRuntime(c, n-1)
}
func (h *IntHeap) popUintRuntime(c *satSolverRuntimeUintColumn) error {
	on, e := h.assertionsRuntime()
	if e != nil {
		return e
	}
	if on {
		n, e := c.size()
		if e != nil {
			return e
		}
		if n == 0 {
			return ErrSATAssertion
		}
	}
	n, e := satAssignmentReadWord(c.raw.runtime, c.raw.block, 4)
	if e != nil {
		return e
	}
	return satAssignmentWriteWord(c.raw.runtime, c.raw.block, 4, n-1)
}
