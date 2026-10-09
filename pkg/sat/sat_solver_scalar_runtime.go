// Source: src/util/vector.h.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// RuntimeUintColumn is SAT's concrete svector<unsigned> storage. It keeps the
// source unsigned32 header and payload layout in the shared checked Runtime.
type RuntimeUintColumn struct{ Raw util.RuntimeVectorColumn }

func (c *RuntimeUintColumn) Bind(r *util.Runtime)      { c.Raw.Bind(r) }
func (c *RuntimeUintColumn) Size() (uint32, error)     { return c.Raw.Size() }
func (c *RuntimeUintColumn) Capacity() (uint32, error) { return c.Raw.Capacity() }
func (c *RuntimeUintColumn) Shrink(n uint32) error     { return c.Raw.Shrink(n) }
func (c *RuntimeUintColumn) Finalize() error           { return c.Raw.Finalize() }
func (c *RuntimeUintColumn) Push(v uint32) error {
	if e := c.Raw.Open(); e != nil {
		return e
	}
	if c.Raw.Pending != (util.Block{}) {
		return ErrSATOwner
	}
	n, e := c.Size()
	if e != nil {
		return e
	}
	capacity, e := c.Capacity()
	if e != nil {
		return e
	}
	if n == capacity {
		if e = c.Raw.ExpandStride(4); e != nil {
			return e
		}
	}
	if e = util.WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 8+4*uint64(n), v); e != nil {
		return e
	}
	return util.WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 4, n+1)
}
func (c *RuntimeUintColumn) Read(i uint32) (uint32, error) {
	if e := c.Raw.Open(); e != nil {
		return 0, e
	}
	if satStorageSourceDebug {
		n, e := c.Size()
		if e != nil {
			return 0, e
		}
		if i >= n {
			return 0, ErrSATAssertion
		}
	}
	if c.Raw.Block == (util.Block{}) {
		return 0, arena.ErrBounds
	}
	return util.ReadRuntimeWord(c.Raw.Runtime, c.Raw.Block, 8+4*uint64(i))
}
func (c *RuntimeUintColumn) Write(i, v uint32) error {
	if e := c.Raw.Open(); e != nil {
		return e
	}
	if satStorageSourceDebug {
		n, e := c.Size()
		if e != nil {
			return e
		}
		if i >= n {
			return ErrSATAssertion
		}
	}
	if c.Raw.Block == (util.Block{}) {
		return arena.ErrBounds
	}
	return util.WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 8+4*uint64(i), v)
}

// copyRuntimeVectorColumn implements the source vector copy constructor.
// Callers bind dst to the same Runtime as src and retain failed-copy roots.
func copyRuntimeVectorColumn(dst, src *util.RuntimeVectorColumn, stride uint32) error {
	if src.Block == (util.Block{}) {
		dst.Block = util.Block{}
		return nil
	}
	n, e := src.Size()
	if e != nil {
		return e
	}
	capacity, e := src.Capacity()
	if e != nil {
		return e
	}
	h, e := src.Runtime.Allocate(satVectorCopyRequest(capacity, stride, src.Runtime.SizeTBits()))
	dst.Pending = h
	if e != nil {
		return e
	}
	if e = util.WriteRuntimeWord(src.Runtime, h, 0, capacity); e != nil {
		return e
	}
	if e = util.WriteRuntimeWord(src.Runtime, h, 4, n); e != nil {
		return e
	}
	dst.Block = h
	dst.Pending = util.Block{}
	for i := uint32(0); i < n; i++ {
		for k := uint32(0); k < stride; k++ {
			b, e := src.Runtime.Read(src.Block, 8+uint64(i)*uint64(stride)+uint64(k))
			if e != nil {
				return e
			}
			if e = dst.Runtime.Write(h, 8+uint64(i)*uint64(stride)+uint64(k), b); e != nil {
				return e
			}
		}
	}
	return nil
}

func satVectorCopyRequest(capacity, stride uint32, bits uint8) uint64 {
	n := uint64(capacity)*uint64(stride) + 8
	if bits == 32 {
		return uint64(uint32(n))
	}
	return n
}
