// Source: src/util/debug.cpp:28–79; src/util/debug.h:28–29,60–65,98. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2006.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
	"reflect"
)

var ErrDebugContextOwner = errors.New("g3: invalid debug context owner")

// DebugContext is the explicit cold owner of the source global assertion switch
// and lazy tag table. The caller owns immutable NUL-terminated tag bytes in the
// borrowed region; this owner never clones or frees them. This adapter is single
// goroutine state, not a process-wide singleton or a complete debugger service.
// Source macro consumers and notification/interactive debugger remain separate
// bindings. In particular source release CASSERT must not query this table.
type DebugContext struct {
	self             *DebugContext
	region           *arena.BytesRegion
	budget           *arena.Budget
	assertionStreams *MemoryStandardStreams
	assertionActions *DebugActionContext
	tags             *utilhash.StrHashtable
	assertions       bool
	ownerBytes       uint64
	closed           bool
}

func NewDebugContext(r *arena.BytesRegion, b *arena.Budget) (*DebugContext, error) {
	if r == nil {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(DebugContext{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	d := &DebugContext{region: r, budget: b, assertions: true, ownerBytes: n}
	d.self = d
	return d, nil
}
func (d *DebugContext) open() error {
	if d == nil || d.self != d {
		return ErrDebugContextOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return nil
}
func (d *DebugContext) EnableAssertions(enabled bool) error {
	if e := d.open(); e != nil {
		return e
	}
	d.assertions = enabled
	return nil
}
func (d *DebugContext) AssertionsEnabled() (bool, error) {
	if e := d.open(); e != nil {
		return false, e
	}
	return d.assertions, nil
}
func (d *DebugContext) initDebugTable() error {
	if e := d.open(); e != nil {
		return e
	}
	if d.tags != nil {
		return nil
	}
	table, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget})
	if e != nil {
		return e
	}
	d.tags = table
	return nil
}
func (d *DebugContext) EnableDebug(tag utilhash.StrKey) error {
	if e := d.initDebugTable(); e != nil {
		return e
	}
	return d.tags.Insert(tag)
}
func (d *DebugContext) DisableDebug(tag utilhash.StrKey) error {
	if e := d.initDebugTable(); e != nil {
		return e
	}
	return d.tags.Erase(tag)
}
func (d *DebugContext) IsDebugEnabled(tag utilhash.StrKey) (bool, error) {
	if e := d.initDebugTable(); e != nil {
		return false, e
	}
	return d.tags.Contains(tag)
}

// A source const char* lookup can borrow a Go string for the duration of this
// call. The existing table performs the same C-string hash/probe/equality and
// stops at the first NUL; the query owns no bytes and is never retained.
func (d *DebugContext) IsDebugEnabledCString(tag string) (bool, error) {
	if e := d.initDebugTable(); e != nil {
		return false, e
	}
	_, ok, e := d.tags.FindCString(tag)
	return ok, e
}

// Source dealloc is table destruction, not its distinct finalize/reset method.
// The assertion switch survives finalize_debug, and subsequent tag operations
// construct a fresh empty table through the original lazy path.
func (d *DebugContext) FinalizeDebug() error {
	if e := d.open(); e != nil {
		return e
	}
	if d.tags != nil {
		d.tags.Close()
		d.tags = nil
	}
	return nil
}
func (d *DebugContext) Close() error {
	if d == nil || d.self != d {
		return ErrDebugContextOwner
	}
	if d.closed {
		return nil
	}
	if e := d.FinalizeDebug(); e != nil {
		return e
	}
	d.budget.Release(d.ownerBytes)
	d.ownerBytes = 0
	d.closed = true
	return nil
}

// ConditionalAssertionEnabled implements only the short-circuit gate inside
// source CASSERT. The consumer must erase this entire call in a release build,
// and must evaluate its assertion body only after the gate returns true. Taking
// the body as an already evaluated bool would change the source read order.
func (d *DebugContext) ConditionalAssertionEnabled(tag string) (bool, error) {
	enabled, e := d.AssertionsEnabled()
	if e != nil || !enabled {
		return false, e
	}
	return d.IsDebugEnabledCString(tag)
}
