// Source: src/util/debug.cpp:28–79; src/util/debug.h:28–29,60–65,98. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2006.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrDebugContextOwner = errors.New("g3: invalid debug context owner")

// DebugContext is the explicit cold owner of the source global assertion switch
// and lazy tag table. The caller owns immutable NUL-terminated tag bytes in the
// borrowed region; this owner never clones or frees them. This adapter is single
// goroutine state, not a process-wide singleton or a complete debugger service.
// Source macro consumers and notification/interactive debugger remain separate
// bindings. In particular source release CASSERT must not query this table.
type DebugContext struct {
	Self                   *DebugContext
	region                 *arena.BytesRegion
	BudgetField            *arena.Budget
	assertionStreams       *MemoryStandardStreams
	assertionActions       *DebugActionContext
	Tags                   *StrHashtable
	TagRuntime             *Runtime
	FailedTags             *debugTagConstruction
	currentTagConstruction *debugTagConstruction
	tagConstructing        bool
	assertions             bool
	ownerBytes             uint64
	closed                 bool
}

func NewDebugContext(r *arena.BytesRegion, b *arena.Budget) (*DebugContext, error) {
	if r == nil {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(DebugContext{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	d := &DebugContext{region: r, BudgetField: b, assertions: true, ownerBytes: n}
	d.Self = d
	return d, nil
}
func (d *DebugContext) open() error {
	if d == nil || d.Self != d {
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
	if d.tagConstructing {
		return ErrDebugTagConstructionExecuting
	}
	if d.Tags != nil {
		return nil
	}
	if d.TagRuntime != nil {
		return d.initRuntimeDebugTable()
	}
	table, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField})
	if e != nil {
		return e
	}
	d.Tags = table
	return nil
}
func (d *DebugContext) EnableDebug(tag StrKey) error {
	if e := d.initDebugTable(); e != nil {
		return e
	}
	return d.Tags.Insert(tag)
}
func (d *DebugContext) DisableDebug(tag StrKey) error {
	if e := d.initDebugTable(); e != nil {
		return e
	}
	return d.Tags.Erase(tag)
}
func (d *DebugContext) IsDebugEnabled(tag StrKey) (bool, error) {
	if e := d.initDebugTable(); e != nil {
		return false, e
	}
	return d.Tags.Contains(tag)
}

// A source const char* lookup can borrow a Go string for the duration of this
// call. The existing table performs the same C-string hash/probe/equality and
// stops at the first NUL; the query owns no bytes and is never retained.
func (d *DebugContext) IsDebugEnabledCString(tag string) (bool, error) {
	if e := d.initDebugTable(); e != nil {
		return false, e
	}
	_, ok, e := d.Tags.FindCString(tag)
	return ok, e
}

// Source dealloc is table destruction, not its distinct finalize/reset method.
// The assertion switch survives finalize_debug, and subsequent tag operations
// construct a fresh empty table through the original lazy path.
func (d *DebugContext) FinalizeDebug() error {
	if e := d.open(); e != nil {
		return e
	}
	if d.Tags != nil {
		if e := d.Tags.SourceFinalize(); e != nil {
			return e
		}
		table := d.Tags
		d.Tags = nil // actual source g_debug_tags = NULL after completed destruction
		if d.currentTagConstruction != nil {
			node := d.currentTagConstruction
			d.currentTagConstruction = nil
			if table.HasRetainedRuntimeAcquisitions() {
				node.Table = table
				node.Next = d.FailedTags
				d.FailedTags = node
			} else {
				if e := table.CloseChecked(); e != nil {
					return e
				}
				d.BudgetField.Release(node.bytes)
			}
		} else if e := table.CloseChecked(); e != nil {
			return e
		}
	}
	return nil
}
func (d *DebugContext) Close() error {
	if d == nil || d.Self != d {
		return ErrDebugContextOwner
	}
	if d.closed {
		return nil
	}
	if d.tagConstructing {
		return ErrDebugTagConstructionExecuting
	}
	if d.Tags != nil {
		if e := d.Tags.CloseChecked(); e != nil {
			return e
		}
		d.Tags = nil
		if d.currentTagConstruction != nil {
			d.BudgetField.Release(d.currentTagConstruction.bytes)
			d.currentTagConstruction = nil
		}
	}
	for d.FailedTags != nil {
		node := d.FailedTags
		if node.Table != nil {
			if e := node.Table.CloseChecked(); e != nil {
				return e
			}
		}
		d.FailedTags = node.Next
		d.BudgetField.Release(node.bytes)
	}
	d.BudgetField.Release(d.ownerBytes)
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
