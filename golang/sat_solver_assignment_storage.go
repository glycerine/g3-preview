// Source: src/util/vector.h:59-100,303-310,389-405.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Actual source char_vector backing, including capacity/size uint32 headers.
// The borrowed Runtime is the sole owner of physical bytes/accounting. No Go
// shadow array or accounting ticket exists. Enclosing owner charges cold fields.
type satSolverAssignmentColumn struct {
	self                    *satSolverAssignmentColumn
	runtime                 *memory.Runtime
	block, pending          memory.Block
	pendingReplaces, closed bool
}

func (c *satSolverAssignmentColumn) bind(r *memory.Runtime) { c.self = c; c.runtime = r }
func (c *satSolverAssignmentColumn) open() error {
	if c == nil || c.self != c || c.runtime == nil {
		return ErrSATOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	_, e := c.runtime.AllocationSize()
	return e
}
func satAssignmentReadWord(r *memory.Runtime, h memory.Block, offset uint64) (uint32, error) {
	var n uint32
	for k := uint64(0); k < 4; k++ {
		b, e := r.Read(h, offset+k)
		if e != nil {
			return 0, e
		}
		n |= uint32(b) << (8 * k)
	}
	return n, nil
}
func satAssignmentWriteWord(r *memory.Runtime, h memory.Block, offset uint64, n uint32) error {
	for k := uint64(0); k < 4; k++ {
		if e := r.Write(h, offset+k, byte(n>>(8*k))); e != nil {
			return e
		}
	}
	return nil
}
func (c *satSolverAssignmentColumn) size() (uint32, error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	if c.block == (memory.Block{}) {
		return 0, nil
	}
	return satAssignmentReadWord(c.runtime, c.block, 4)
}
func (c *satSolverAssignmentColumn) capacity() (uint32, error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	if c.block == (memory.Block{}) {
		return 0, nil
	}
	return satAssignmentReadWord(c.runtime, c.block, 0)
}
func satAssignmentNext(old uint32) (uint32, uint32, error) { return satScalarColumnNext(old, 1) }
func satScalarColumnNext(old, stride uint32) (uint32, uint32, error) {
	if old == 0 {
		return 2, 2*stride + 8, nil
	}
	n := (3*old + 1) >> 1
	before, after := old*stride+8, n*stride+8
	if n <= old || after <= before {
		return 0, 0, NewDefaultException("Overflow encountered when expanding vector")
	}
	return n, after, nil
}
func (c *satSolverAssignmentColumn) expand() error { return c.expandStride(1) }
func (c *satSolverAssignmentColumn) expandStride(stride uint32) error {
	return c.expandStrideDebug(stride, nil)
}

// The default SAT adapter remains assertion-enabled. The concrete general heap
// may borrow the actual global switch at this one reached source SASSERT site.
func (c *satSolverAssignmentColumn) expandStrideDebug(stride uint32, debug *DebugContext) error {
	return c.expandStrideUnsignedOwner(stride, debug, nil)
}

// Concrete optional unsigned owner records actual lower allocation outcomes.
// The shared column layout and every legacy/SAT default route stay unchanged.
func (c *satSolverAssignmentColumn) expandStrideUnsignedOwner(stride uint32, debug *DebugContext, owner *RuntimeUnsignedVector) error {
	if owner != nil {
		if e := owner.matchGrowthColumn(c); e != nil {
			return e
		}
	}
	if c.block != (memory.Block{}) && satStorageSourceDebug {
		enabled := true
		if debug != nil {
			var e error
			enabled, e = debug.AssertionsEnabled()
			if e != nil {
				return e
			}
		}
		if enabled {
			capacity, e := c.capacity()
			if e != nil {
				return e
			}
			if capacity == 0 {
				if e := debugAssertionFailure(debug, "src/util/vector.h", 70, "capacity() > 0"); e != nil {
					return e
				}
			}
		}
	}
	old, e := c.capacity()
	if e != nil {
		return e
	}
	// Non-null source m_data takes the geometric branch even when a raw
	// corrupted capacity is zero: DEBUG assertion, otherwise literal overflow.
	if c.block != (memory.Block{}) && old == 0 {
		return NewDefaultException("Overflow encountered when expanding vector")
	}
	n, request, e := satScalarColumnNext(old, stride)
	if e != nil {
		return e
	}
	if owner != nil {
		if e = owner.startUnsignedGrowth(c); e != nil {
			return e
		}
		defer owner.endUnsignedGrowth()
	}
	if c.block == (memory.Block{}) {
		h, e := c.runtime.Allocate(uint64(request))
		if e != nil {
			if h != (memory.Block{}) {
				c.pending = h
			}
			if owner != nil {
				owner.recordUnsignedGrowth(e, h != (memory.Block{}))
			}
			return e
		}
		// vector.h: capacity then size, then m_data publication.
		if e = satAssignmentWriteWord(c.runtime, h, 0, n); e == nil {
			e = satAssignmentWriteWord(c.runtime, h, 4, 0)
		}
		if e != nil {
			c.pending = h
			if owner != nil {
				owner.recordUnsignedGrowth(e, true)
			}
			return e
		}
		c.block = h
		return nil
	}
	h := c.block
	if e = c.runtime.Reallocate(&h, uint64(request)); e != nil {
		if h != c.block {
			c.pending = h
			c.pendingReplaces = true
		}
		if owner != nil {
			owner.recordUnsignedGrowth(e, h != c.block && h != (memory.Block{}))
		}
		return e
	}
	// Trivial realloc returns/publishes m_data before capacity overwrite.
	c.block = h
	return satAssignmentWriteWord(c.runtime, c.block, 0, n)
}
func (c *satSolverAssignmentColumn) push(v int8) error {
	if e := c.open(); e != nil {
		return e
	}
	if c.pending != (memory.Block{}) {
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
		if e = c.expand(); e != nil {
			return e
		}
	}
	// Source push constructs element before incrementing header size.
	if e = c.runtime.Write(c.block, uint64(n)+8, byte(v)); e != nil {
		return e
	}
	return satAssignmentWriteWord(c.runtime, c.block, 4, n+1)
}
func (c *satSolverAssignmentColumn) read(i uint32) (int8, error) {
	if e := c.open(); e != nil {
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
	if c.block == (memory.Block{}) {
		return 0, arena.ErrBounds
	}
	b, e := c.runtime.Read(c.block, uint64(i)+8)
	return int8(b), e
}
func (c *satSolverAssignmentColumn) write(i uint32, v int8) error {
	if e := c.open(); e != nil {
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
	if c.block == (memory.Block{}) {
		return arena.ErrBounds
	}
	return c.runtime.Write(c.block, uint64(i)+8, byte(v))
}
func (c *satSolverAssignmentColumn) shrink(n uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if c.block == (memory.Block{}) {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if satStorageSourceDebug {
		old, e := c.size()
		if e != nil {
			return e
		}
		if n > old {
			return ErrSATAssertion
		}
	}
	// svector<char> has CallDestructors=false: release only writes the size.
	return satAssignmentWriteWord(c.runtime, c.block, 4, n)
}
func (c *satSolverAssignmentColumn) release(h *memory.Block) error {
	if *h == (memory.Block{}) {
		return nil
	}
	if _, e := c.runtime.Size(*h); e != nil {
		if cleanup := c.runtime.CleanupPartial(h); cleanup != nil {
			return e
		}
		return nil
	}
	return c.runtime.Deallocate(h)
}
func (c *satSolverAssignmentColumn) finalize() error {
	if c == nil || c.self != c {
		return ErrSATOwner
	}
	if c.closed {
		return nil
	}
	if c.pending != (memory.Block{}) {
		if e := c.release(&c.pending); e != nil {
			return e
		}
		if c.pendingReplaces {
			c.block = memory.Block{}
			c.pendingReplaces = false
		}
	}
	if e := c.release(&c.block); e != nil {
		return e
	}
	c.closed = true
	return nil
}
