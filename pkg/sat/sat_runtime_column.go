// Source: src/util/vector.h:59-100,303-310,389-405.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// SAT-only backing for the remaining typed Runtime columns, with uint32 headers.
// The borrowed util.Runtime is the sole owner of physical bytes/accounting. No Go
// shadow array or accounting ticket exists. Enclosing owner charges cold fields.
type satRuntimeColumn struct {
	Self                    *satRuntimeColumn
	Runtime                 *util.Runtime
	Block, Pending          util.Block
	PendingReplaces, Closed bool
}

func (c *satRuntimeColumn) Bind(r *util.Runtime) { c.Self = c; c.Runtime = r }
func (c *satRuntimeColumn) Open() error {
	if c == nil || c.Self != c || c.Runtime == nil {
		return ErrSATOwner
	}
	if c.Closed {
		return arena.ErrClosed
	}
	_, e := c.Runtime.AllocationSize()
	return e
}
func ReadRuntimeWord(r *util.Runtime, h util.Block, offset uint64) (uint32, error) {
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
func WriteRuntimeWord(r *util.Runtime, h util.Block, offset uint64, n uint32) error {
	for k := uint64(0); k < 4; k++ {
		if e := r.Write(h, offset+k, byte(n>>(8*k))); e != nil {
			return e
		}
	}
	return nil
}
func (c *satRuntimeColumn) Size() (uint32, error) {
	if e := c.Open(); e != nil {
		return 0, e
	}
	if c.Block == (util.Block{}) {
		return 0, nil
	}
	return ReadRuntimeWord(c.Runtime, c.Block, 4)
}
func (c *satRuntimeColumn) Capacity() (uint32, error) {
	if e := c.Open(); e != nil {
		return 0, e
	}
	if c.Block == (util.Block{}) {
		return 0, nil
	}
	return ReadRuntimeWord(c.Runtime, c.Block, 0)
}
func RuntimeScalarColumnNext(old, stride uint32) (uint32, uint32, error) {
	if old == 0 {
		return 2, 2*stride + 8, nil
	}
	n := (3*old + 1) >> 1
	before, after := old*stride+8, n*stride+8
	if n <= old || after <= before {
		return 0, 0, z3exception.NewDefaultException("Overflow encountered when expanding vector")
	}
	return n, after, nil
}
func (c *satRuntimeColumn) ExpandStride(stride uint32) error {
	if c.Block != (util.Block{}) && satStorageSourceDebug {
		capacity, e := c.Capacity()
		if e != nil {
			return e
		}
		if capacity == 0 {
			return ErrSATAssertion
		}
	}
	old, e := c.Capacity()
	if e != nil {
		return e
	}
	// Non-null source m_data takes the geometric branch even when a raw
	// corrupted capacity is zero: DEBUG assertion, otherwise literal overflow.
	if c.Block != (util.Block{}) && old == 0 {
		return z3exception.NewDefaultException("Overflow encountered when expanding vector")
	}
	n, request, e := RuntimeScalarColumnNext(old, stride)
	if e != nil {
		return e
	}
	if c.Block == (util.Block{}) {
		h, e := c.Runtime.Allocate(uint64(request))
		if e != nil {
			if h != (util.Block{}) {
				c.Pending = h
			}
			return e
		}
		// vector.h: capacity then size, then m_data publication.
		if e = WriteRuntimeWord(c.Runtime, h, 0, n); e == nil {
			e = WriteRuntimeWord(c.Runtime, h, 4, 0)
		}
		if e != nil {
			c.Pending = h
			return e
		}
		c.Block = h
		return nil
	}
	h := c.Block
	if e = c.Runtime.Reallocate(&h, uint64(request)); e != nil {
		if h != c.Block {
			c.Pending = h
			c.PendingReplaces = true
		}
		return e
	}
	// Trivial realloc returns/publishes m_data before capacity overwrite.
	c.Block = h
	return WriteRuntimeWord(c.Runtime, c.Block, 0, n)
}
func (c *satRuntimeColumn) Shrink(n uint32) error {
	if e := c.Open(); e != nil {
		return e
	}
	if c.Block == (util.Block{}) {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if satStorageSourceDebug {
		old, e := c.Size()
		if e != nil {
			return e
		}
		if n > old {
			return ErrSATAssertion
		}
	}
	// Remaining scalar payloads have no element destructor; shrink writes size.
	return WriteRuntimeWord(c.Runtime, c.Block, 4, n)
}
func (c *satRuntimeColumn) Release(h *util.Block) error {
	if *h == (util.Block{}) {
		return nil
	}
	if _, e := c.Runtime.Size(*h); e != nil {
		if cleanup := c.Runtime.CleanupPartial(h); cleanup != nil {
			return e
		}
		return nil
	}
	return c.Runtime.Deallocate(h)
}
func (c *satRuntimeColumn) Finalize() error {
	if c == nil || c.Self != c {
		return ErrSATOwner
	}
	if c.Closed {
		return nil
	}
	if c.Pending != (util.Block{}) {
		if e := c.Release(&c.Pending); e != nil {
			return e
		}
		if c.PendingReplaces {
			c.Block = util.Block{}
			c.PendingReplaces = false
		}
	}
	if e := c.Release(&c.Block); e != nil {
		return e
	}
	c.Closed = true
	return nil
}
