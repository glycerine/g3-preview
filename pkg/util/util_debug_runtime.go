// Source: src/util/debug.cpp:44-68; src/util/hashtable.h:243-272.
// Source author: Leonardo de Moura, 2006-09-11.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Each retained context was prepared before a reached constructor invocation.
// Only its actual returned table owns failed placement/source-prefix storage.
// Current publication owns one table directly and retires its completed node.
var ErrDebugTagConstructionExecuting = errors.New("g3: debug tag construction is executing")

type debugTagConstruction struct {
	Next  *debugTagConstruction
	Table *StrHashtable
	bytes uint64
}

func (d *DebugContext) canBindTagRuntime(r *Runtime) error {
	if e := d.open(); e != nil {
		return e
	}
	if d.tagConstructing {
		return ErrDebugTagConstructionExecuting
	}
	if r == nil || r.Budget() != d.BudgetField {
		return arena.ErrConfig
	}
	if d.TagRuntime != nil && d.TagRuntime != r {
		return arena.ErrConfig
	}
	if d.Tags != nil && d.TagRuntime == nil {
		return arena.ErrConfig
	}
	return nil
}

// Binding reads identities only. The table remains source-lazy; provider
// lifetime checks occur at its reached allocation/read/release leaves.
func (d *DebugContext) BindTagRuntime(r *Runtime) error {
	if e := d.canBindTagRuntime(r); e != nil {
		return e
	}
	d.TagRuntime = r
	return nil
}
func (d *DebugContext) initRuntimeDebugTable() error {
	bytes := uint64(reflect.TypeOf(debugTagConstruction{}).Size())
	if e := d.BudgetField.Reserve(bytes); e != nil {
		return e
	}
	node := &debugTagConstruction{bytes: bytes, Next: d.FailedTags}
	d.FailedTags = node
	d.tagConstructing = true
	table, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, Runtime: d.TagRuntime, Assertions: d})
	node.Table = table
	d.tagConstructing = false
	if e != nil {
		if table == nil {
			d.FailedTags = node.Next
			d.BudgetField.Release(node.bytes)
		}
		return e
	}
	d.Tags = table
	node.Table = nil
	d.FailedTags = node.Next
	node.Next = nil
	d.currentTagConstruction = node
	return nil
}
