// Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Actual svector<unsigned>, fixed-width payload with source unsigned32 headers.
// The char owner's common header/lifetime methods hold the same raw allocation;
// no element slice, callback, or auxiliary accounting allocation exists.
type satSolverRuntimeUintColumn struct{ raw satSolverAssignmentColumn }

func (c *satSolverRuntimeUintColumn) bind(r *memory.Runtime)    { c.raw.bind(r) }
func (c *satSolverRuntimeUintColumn) size() (uint32, error)     { return c.raw.size() }
func (c *satSolverRuntimeUintColumn) capacity() (uint32, error) { return c.raw.capacity() }
func (c *satSolverRuntimeUintColumn) push(v uint32) error {
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
	cap, e := c.capacity()
	if e != nil {
		return e
	}
	if n == cap {
		if e = c.raw.expandStride(4); e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(c.raw.runtime, c.raw.block, 8+4*uint64(n), v); e != nil {
		return e
	}
	return satAssignmentWriteWord(c.raw.runtime, c.raw.block, 4, n+1)
}
func (c *satSolverRuntimeUintColumn) read(i uint32) (uint32, error) {
	if e := c.raw.open(); e != nil {
		return 0, e
	}
	if satStorageSourceDebug {
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
func (c *satSolverRuntimeUintColumn) write(i, v uint32) error {
	if e := c.raw.open(); e != nil {
		return e
	}
	if satStorageSourceDebug {
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
func (c *satSolverRuntimeUintColumn) shrink(n uint32) error { return c.raw.shrink(n) }
func (c *satSolverRuntimeUintColumn) finalize() error       { return c.raw.finalize() }
