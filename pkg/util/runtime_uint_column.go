package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Actual svector<unsigned>, fixed-width payload with source unsigned32 headers.
// The char owner's common header/lifetime methods hold the same raw allocation;
// no element slice, callback, or auxiliary accounting allocation exists.

type RuntimeUintColumn struct{ Raw RuntimeVectorColumn }

func (c *RuntimeUintColumn) Bind(r *Runtime)           { c.Raw.Bind(r) }
func (c *RuntimeUintColumn) Size() (uint32, error)     { return c.Raw.Size() }
func (c *RuntimeUintColumn) Capacity() (uint32, error) { return c.Raw.Capacity() }
func (c *RuntimeUintColumn) Push(v uint32) error {
	if e := c.Raw.Open(); e != nil {
		return e
	}
	if c.Raw.Pending != (Block{}) {
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
		if e = c.Raw.ExpandStride(4); e != nil {
			return e
		}
	}
	if e = WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 8+4*uint64(n), v); e != nil {
		return e
	}
	return WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 4, n+1)
}
func (c *RuntimeUintColumn) Read(i uint32) (uint32, error) {
	if e := c.Raw.Open(); e != nil {
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
	if c.Raw.Block == (Block{}) {
		return 0, arena.ErrBounds
	}
	return ReadRuntimeWord(c.Raw.Runtime, c.Raw.Block, 8+4*uint64(i))
}
func (c *RuntimeUintColumn) Write(i, v uint32) error {
	if e := c.Raw.Open(); e != nil {
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
	if c.Raw.Block == (Block{}) {
		return arena.ErrBounds
	}
	return WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 8+4*uint64(i), v)
}
func (c *RuntimeUintColumn) Shrink(n uint32) error { return c.Raw.Shrink(n) }
func (c *RuntimeUintColumn) Finalize() error       { return c.Raw.Finalize() }
