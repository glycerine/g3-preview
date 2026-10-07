// Source: src/util/trace.cpp:25-62; src/util/trace.h:38-49.
// Licensed under the MIT license. See LICENSE.txt.
// Source author: Leonardo de Moura (leonardo), 2006.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrTraceContextOwner = errors.New("g3: invalid trace context owner")

// Explicit single-lane owner of the source tag globals. The table borrows
// CString storage; finalize destroys the table but retains the enable-all flag.
// The existing concrete table uses charged Go storage; source-global Runtime
// accounting, process singleton wiring and _TRACE ofstream remain open.
type TraceContext struct {
	self        *TraceContext
	region      *arena.BytesRegion
	budget      *arena.Budget
	tags        *StrHashtable
	ownerBytes  uint64
	all, closed bool
}
type TraceContextStats struct {
	OwnerBytes uint64
	Table      StrHashtableStats
}

func NewTraceContext(r *arena.BytesRegion, b *arena.Budget) (*TraceContext, error) {
	if r == nil {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(TraceContext{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	c := &TraceContext{region: r, budget: b, ownerBytes: n}
	c.self = c
	return c, nil
}
func (c *TraceContext) open() error {
	if c == nil || c.self != c {
		return ErrTraceContextOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}
func (c *TraceContext) initTable() error {
	if e := c.open(); e != nil {
		return e
	}
	if c.tags == nil {
		t, e := NewStrHashtable(c.region, StrHashtableConfig{Budget: c.budget})
		if e != nil {
			return e
		}
		c.tags = t
	}
	return nil
}
func (c *TraceContext) EnableTrace(k StrKey) error {
	if e := c.initTable(); e != nil {
		return e
	}
	return c.tags.Insert(k)
}
func (c *TraceContext) EnableTraceFrom(r *arena.BytesRegion, k StrKey) error {
	if e := c.initTable(); e != nil {
		return e
	}
	return c.tags.InsertFrom(r, k)
}
func (c *TraceContext) DisableTrace(k StrKey) error {
	if e := c.initTable(); e != nil {
		return e
	}
	return c.tags.Erase(k)
}
func (c *TraceContext) DisableTraceFrom(r *arena.BytesRegion, k StrKey) error {
	if e := c.initTable(); e != nil {
		return e
	}
	return c.tags.EraseFrom(r, k)
}
func (c *TraceContext) EnableAllTrace(all bool) error {
	if e := c.open(); e != nil {
		return e
	}
	c.all = all
	return nil
}
func (c *TraceContext) IsTraceEnabled(k StrKey) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if c.all {
		return true, nil
	}
	if c.tags == nil {
		return false, nil
	}
	return c.tags.Contains(k)
}
func (c *TraceContext) IsTraceEnabledFrom(r *arena.BytesRegion, k StrKey) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if c.all {
		return true, nil
	}
	if c.tags == nil {
		return false, nil
	}
	return c.tags.ContainsFrom(r, k)
}
func (c *TraceContext) FinalizeTrace() error {
	if e := c.open(); e != nil {
		return e
	}
	if c.tags != nil {
		c.tags.Close()
		c.tags = nil
	}
	return nil
}
func (c *TraceContext) Stats() TraceContextStats {
	if c.open() != nil {
		return TraceContextStats{}
	}
	s := TraceContextStats{OwnerBytes: c.ownerBytes}
	if c.tags != nil {
		s.Table = c.tags.Stats()
	}
	return s
}
func (c *TraceContext) Close() error {
	if c == nil || c.self != c {
		return ErrTraceContextOwner
	}
	if c.closed {
		return nil
	}
	if e := c.FinalizeTrace(); e != nil {
		return e
	}
	c.budget.Release(c.ownerBytes)
	c.closed = true
	return nil
}

// Borrowed source literal spelling, never retained or copied into the table.
func (c *TraceContext) IsTraceEnabledCString(tag string) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if c.all {
		return true, nil
	}
	if c.tags == nil {
		return false, nil
	}
	_, ok, e := c.tags.FindCString(tag)
	return ok, e
}
