// Source: src/util/debug.cpp:44-68; src/util/hashtable.h:243-272.
// Source author: Leonardo de Moura, 2006-09-11.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"reflect"
)

// Each retained context was prepared before a reached constructor invocation.
// Only its actual returned table owns failed placement/source-prefix storage.
// Current publication owns one table directly and retires its completed node.
var ErrDebugTagConstructionExecuting = errors.New("g3: debug tag construction is executing")

type debugTagConstruction struct {
	next  *debugTagConstruction
	table *utilhash.StrHashtable
	bytes uint64
}

func (d *DebugContext) canBindTagRuntime(r *memory.Runtime) error {
	if e := d.open(); e != nil {
		return e
	}
	if d.tagConstructing {
		return ErrDebugTagConstructionExecuting
	}
	if r == nil || r.Budget() != d.budget {
		return arena.ErrConfig
	}
	if d.tagRuntime != nil && d.tagRuntime != r {
		return arena.ErrConfig
	}
	if d.tags != nil && d.tagRuntime == nil {
		return arena.ErrConfig
	}
	return nil
}

// Binding reads identities only. The table remains source-lazy; provider
// lifetime checks occur at its reached allocation/read/release leaves.
func (d *DebugContext) BindTagRuntime(r *memory.Runtime) error {
	if e := d.canBindTagRuntime(r); e != nil {
		return e
	}
	d.tagRuntime = r
	return nil
}
func (d *DebugContext) initRuntimeDebugTable() error {
	bytes := uint64(reflect.TypeOf(debugTagConstruction{}).Size())
	if e := d.budget.Reserve(bytes); e != nil {
		return e
	}
	node := &debugTagConstruction{bytes: bytes, next: d.failedTags}
	d.failedTags = node
	d.tagConstructing = true
	table, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: d.tagRuntime, Assertions: d})
	node.table = table
	d.tagConstructing = false
	if e != nil {
		if table == nil {
			d.failedTags = node.next
			d.budget.Release(node.bytes)
		}
		return e
	}
	d.tags = table
	node.table = nil
	d.failedTags = node.next
	node.next = nil
	d.currentTagConstruction = node
	return nil
}
