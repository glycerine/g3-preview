// Source: src/util/vector.h:59-100,303-310,389-405.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

// RuntimeVectorGrowthOwner supplies caller-specific ownership tracking while
// the column preserves vector.h's allocation and publication order.
// Hooks are synchronous and borrowed for one expansion; the column stores none.
type RuntimeVectorGrowthOwner interface {
	MatchVectorGrowthColumn(*RuntimeVectorColumn) error
	StartVectorGrowth(*RuntimeVectorColumn) error
	RecordVectorGrowth(error, bool)
	EndVectorGrowth()
}

var ErrRuntimeVectorOwner = errors.New("sat: foreign or copied storage owner")
var ErrRuntimeVectorAssertion = errors.New("sat: source assertion failure")

// Actual source char_vector backing, including capacity/size uint32 headers.
// The borrowed Runtime is the sole owner of physical bytes/accounting. No Go
// shadow array or accounting ticket exists. Enclosing owner charges cold fields.
type RuntimeVectorColumn struct {
	Self                    *RuntimeVectorColumn
	Runtime                 *Runtime
	Block, Pending          Block
	PendingReplaces, Closed bool
}

func (c *RuntimeVectorColumn) Bind(r *Runtime) { c.Self = c; c.Runtime = r }
func (c *RuntimeVectorColumn) Open() error {
	if c == nil || c.Self != c || c.Runtime == nil {
		return ErrRuntimeVectorOwner
	}
	if c.Closed {
		return arena.ErrClosed
	}
	_, e := c.Runtime.AllocationSize()
	return e
}
func ReadRuntimeWord(r *Runtime, h Block, offset uint64) (uint32, error) {
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
func WriteRuntimeWord(r *Runtime, h Block, offset uint64, n uint32) error {
	for k := uint64(0); k < 4; k++ {
		if e := r.Write(h, offset+k, byte(n>>(8*k))); e != nil {
			return e
		}
	}
	return nil
}
func (c *RuntimeVectorColumn) Size() (uint32, error) {
	if e := c.Open(); e != nil {
		return 0, e
	}
	if c.Block == (Block{}) {
		return 0, nil
	}
	return ReadRuntimeWord(c.Runtime, c.Block, 4)
}
func (c *RuntimeVectorColumn) Capacity() (uint32, error) {
	if e := c.Open(); e != nil {
		return 0, e
	}
	if c.Block == (Block{}) {
		return 0, nil
	}
	return ReadRuntimeWord(c.Runtime, c.Block, 0)
}
func RuntimeByteColumnNext(old uint32) (uint32, uint32, error) {
	return RuntimeScalarColumnNext(old, 1)
}
func RuntimeScalarColumnNext(old, stride uint32) (uint32, uint32, error) {
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
func (c *RuntimeVectorColumn) Expand() error { return c.ExpandStride(1) }
func (c *RuntimeVectorColumn) ExpandStride(stride uint32) error {
	return c.ExpandStrideDebug(stride, nil)
}

// The default byte-column adapter remains assertion-enabled. The concrete general heap
// may borrow the actual global switch at this one reached source SASSERT site.
func (c *RuntimeVectorColumn) ExpandStrideDebug(stride uint32, debug *DebugContext) error {
	return c.ExpandStrideUnsignedOwner(stride, debug, nil)
}

// Concrete optional unsigned owner records actual lower allocation outcomes.
// The shared column layout and every legacy byte-column default route stay unchanged.
func (c *RuntimeVectorColumn) ExpandStrideUnsignedOwner(stride uint32, debug *DebugContext, owner *RuntimeUnsignedVector) error {
	return c.expandStrideConcreteOwners(stride, debug, owner, nil)
}
func (c *RuntimeVectorColumn) ExpandStrideOwned(stride uint32, debug *DebugContext, owner RuntimeVectorGrowthOwner) error {
	return c.expandStrideConcreteOwners(stride, debug, nil, owner)
}
func (c *RuntimeVectorColumn) expandStrideConcreteOwners(stride uint32, debug *DebugContext, owner *RuntimeUnsignedVector, growthOwner RuntimeVectorGrowthOwner) error {
	if growthOwner != nil {
		if e := growthOwner.MatchVectorGrowthColumn(c); e != nil {
			return e
		}
	}
	if owner != nil {
		if e := owner.matchGrowthColumn(c); e != nil {
			return e
		}
	}
	if c.Block != (Block{}) && sourceDebug {
		enabled := true
		if debug != nil {
			var e error
			enabled, e = debug.AssertionsEnabled()
			if e != nil {
				return e
			}
		}
		if enabled {
			capacity, e := c.Capacity()
			if e != nil {
				return e
			}
			if capacity == 0 {
				if e := DebugAssertionFailure(debug, "src/util/vector.h", 70, "capacity() > 0"); e != nil {
					return e
				}
			}
		}
	}
	old, e := c.Capacity()
	if e != nil {
		return e
	}
	// Non-null source m_data takes the geometric branch even when a raw
	// corrupted capacity is zero: DEBUG assertion, otherwise literal overflow.
	if c.Block != (Block{}) && old == 0 {
		return NewDefaultException("Overflow encountered when expanding vector")
	}
	n, request, e := RuntimeScalarColumnNext(old, stride)
	if e != nil {
		return e
	}
	if owner != nil {
		if e = owner.startUnsignedGrowth(c); e != nil {
			return e
		}
		defer owner.endUnsignedGrowth()
	}
	if growthOwner != nil {
		if e = growthOwner.StartVectorGrowth(c); e != nil {
			return e
		}
		defer growthOwner.EndVectorGrowth()
	}
	if c.Block == (Block{}) {
		h, e := c.Runtime.Allocate(uint64(request))
		if e != nil {
			if h != (Block{}) {
				c.Pending = h
			}
			if owner != nil {
				owner.recordUnsignedGrowth(e, h != (Block{}))
			}
			if growthOwner != nil {
				growthOwner.RecordVectorGrowth(e, h != (Block{}))
			}
			return e
		}
		// vector.h: capacity then size, then m_data publication.
		if e = WriteRuntimeWord(c.Runtime, h, 0, n); e == nil {
			e = WriteRuntimeWord(c.Runtime, h, 4, 0)
		}
		if e != nil {
			c.Pending = h
			if owner != nil {
				owner.recordUnsignedGrowth(e, true)
			}
			if growthOwner != nil {
				growthOwner.RecordVectorGrowth(e, true)
			}
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
		if owner != nil {
			owner.recordUnsignedGrowth(e, h != c.Block && h != (Block{}))
		}
		if growthOwner != nil {
			growthOwner.RecordVectorGrowth(e, h != c.Block && h != (Block{}))
		}
		return e
	}
	// Trivial realloc returns/publishes m_data before capacity overwrite.
	c.Block = h
	return WriteRuntimeWord(c.Runtime, c.Block, 0, n)
}
func (c *RuntimeVectorColumn) Push(v int8) error {
	if e := c.Open(); e != nil {
		return e
	}
	if c.Pending != (Block{}) {
		return ErrRuntimeVectorOwner
	}
	n, e := c.Size()
	if e != nil {
		return e
	}
	cap, e := c.Capacity()
	if e != nil {
		return e
	}
	if n == cap {
		if e = c.Expand(); e != nil {
			return e
		}
	}
	// Source push constructs element before incrementing header size.
	if e = c.Runtime.Write(c.Block, uint64(n)+8, byte(v)); e != nil {
		return e
	}
	return WriteRuntimeWord(c.Runtime, c.Block, 4, n+1)
}
func (c *RuntimeVectorColumn) Read(i uint32) (int8, error) {
	if e := c.Open(); e != nil {
		return 0, e
	}
	if sourceDebug {
		n, e := c.Size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrRuntimeVectorAssertion
		}
	}
	if c.Block == (Block{}) {
		return 0, arena.ErrBounds
	}
	b, e := c.Runtime.Read(c.Block, uint64(i)+8)
	return int8(b), e
}
func (c *RuntimeVectorColumn) Write(i uint32, v int8) error {
	if e := c.Open(); e != nil {
		return e
	}
	if sourceDebug {
		n, e := c.Size()
		if e != nil {
			return e
		}
		if i >= n {
			return ErrRuntimeVectorAssertion
		}
	}
	if c.Block == (Block{}) {
		return arena.ErrBounds
	}
	return c.Runtime.Write(c.Block, uint64(i)+8, byte(v))
}
func (c *RuntimeVectorColumn) Shrink(n uint32) error {
	if e := c.Open(); e != nil {
		return e
	}
	if c.Block == (Block{}) {
		if sourceDebug && n != 0 {
			return ErrRuntimeVectorAssertion
		}
		return nil
	}
	if sourceDebug {
		old, e := c.Size()
		if e != nil {
			return e
		}
		if n > old {
			return ErrRuntimeVectorAssertion
		}
	}
	// svector<char> has CallDestructors=false: release only writes the size.
	return WriteRuntimeWord(c.Runtime, c.Block, 4, n)
}
func (c *RuntimeVectorColumn) Release(h *Block) error {
	if *h == (Block{}) {
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
func (c *RuntimeVectorColumn) Finalize() error {
	if c == nil || c.Self != c {
		return ErrRuntimeVectorOwner
	}
	if c.Closed {
		return nil
	}
	if c.Pending != (Block{}) {
		if e := c.Release(&c.Pending); e != nil {
			return e
		}
		if c.PendingReplaces {
			c.Block = Block{}
			c.PendingReplaces = false
		}
	}
	if e := c.Release(&c.Block); e != nil {
		return e
	}
	c.Closed = true
	return nil
}

// runtimeVectorCopyRequest preserves source size_t arithmetic before allocation.
func runtimeVectorCopyRequest(capacity, stride uint32, bits uint8) uint64 {
	n := uint64(capacity)*uint64(stride) + 8
	if bits == 32 {
		return uint64(uint32(n))
	}
	return n
}
