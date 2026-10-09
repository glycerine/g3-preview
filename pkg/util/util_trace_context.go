// Source: src/util/trace.cpp:25-62; src/util/trace.h:38-49.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrTraceContextOwner = errors.New("g3: invalid trace context owner")

// TraceContext owns enabled trace names as immutable Go strings.
type TraceContext struct {
	self        *TraceContext
	region      *arena.BytesRegion
	budget      *arena.Budget
	tags        map[string]struct{}
	ownerBytes  uint64
	all, closed bool
}

type TraceContextStats struct {
	OwnerBytes uint64
	Tags       uint32
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

func traceName(s string) string { return s[:cSymbolLength(s)] }

func (c *TraceContext) EnableTrace(name string) error {
	if e := c.open(); e != nil {
		return e
	}
	if c.tags == nil {
		c.tags = make(map[string]struct{})
	}
	c.tags[traceName(name)] = struct{}{}
	return nil
}

func (c *TraceContext) DisableTrace(name string) error {
	if e := c.open(); e != nil {
		return e
	}
	delete(c.tags, traceName(name))
	return nil
}

func (c *TraceContext) IsTraceEnabled(name string) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if c.all {
		return true, nil
	}
	_, ok := c.tags[traceName(name)]
	return ok, nil
}

func (c *TraceContext) EnableAllTrace(all bool) error {
	if e := c.open(); e != nil {
		return e
	}
	c.all = all
	return nil
}
func (c *TraceContext) FinalizeTrace() error {
	if e := c.open(); e != nil {
		return e
	}
	c.tags = nil
	return nil
}
func (c *TraceContext) Stats() TraceContextStats {
	if c.open() != nil {
		return TraceContextStats{}
	}
	return TraceContextStats{OwnerBytes: c.ownerBytes, Tags: uint32(len(c.tags))}
}
func (c *TraceContext) Close() error {
	if c == nil || c.self != c {
		return ErrTraceContextOwner
	}
	if c.closed {
		return nil
	}
	c.tags = nil
	c.budget.Release(c.ownerBytes)
	c.ownerBytes = 0
	c.closed = true
	return nil
}
func (c *TraceContext) IsTraceEnabledCString(name string) (bool, error) {
	return c.IsTraceEnabled(name)
}
