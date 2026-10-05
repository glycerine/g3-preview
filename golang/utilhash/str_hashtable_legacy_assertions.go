// Source: src/util/hashtable.h:159-215,249-256,371,395,422,451,559,643-644;
// src/util/debug.h:60-65,76-89.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

func (h *StrHashtable) constructLegacy(n uint32) error {
	h.raw.self = h
	on, e := h.assertionsEnabled()
	if e != nil {
		return e
	}
	if on && (n == 0 || n&(n-1) != 0) {
		if e = h.assertionFailure(249, "is_power_of_two(initial_capacity)"); e != nil {
			return e
		}
	}
	table, bytes, e := h.allocateTable(n)
	if e != nil {
		return e
	}
	h.table, h.tableBytes = table, bytes
	h.raw.words[2] = n
	if h.config.Statistics {
		if e = h.writeCollision(0); e != nil {
			return e
		}
	}
	h.raw.complete = true
	return nil
}

func (h *StrHashtable) copyLegacyEntries(owner *StrHashtable, source, target []strHashEntry, sourceCapacity, targetCapacity uint32, move bool) error {
	line := int32(159)
	if move {
		line = 191
	}
	on, e := h.assertionsEnabled()
	if e != nil {
		return e
	}
	if on && targetCapacity < sourceCapacity {
		if e = h.assertionFailure(line, "target_capacity >= source_capacity"); e != nil {
			return e
		}
	}
	for i := uint32(0); i < sourceCapacity; i++ {
		if uint64(i) >= uint64(len(source)) {
			return arena.ErrBounds
		}
		if source[i].state != strHashUsed {
			continue
		}
		hash := source[i].hash
		start := hash & (targetCapacity - 1)
		found := false
		var j uint32
		for step := uint32(0); step < targetCapacity; step++ {
			j = sourceProbeIndex(start, step, targetCapacity)
			on, e = h.assertionsEnabled()
			if e != nil {
				return e
			}
			if on {
				if uint64(j) >= uint64(len(target)) {
					return arena.ErrBounds
				}
				if target[j].state == strHashDeleted {
					line = int32(170)
					if j < start {
						line = 177
					}
					if move {
						line = 202
						if j < start {
							line = 209
						}
					}
					if e = h.assertionFailure(line, "!target_curr->is_deleted()"); e != nil {
						return e
					}
				}
			}
			if uint64(j) >= uint64(len(target)) {
				return arena.ErrBounds
			}
			if target[j].state == strHashFree {
				found = true
				break
			}
		}
		if !found {
			line = 183
			if move {
				line = 215
			}
			if e = h.unreachable(line); e != nil {
				return e
			}
			continue
		}
		// The scalar entry is the concrete source POD pointer/hash/state payload.
		entry := source[i]
		if !move {
			region, e := owner.keyRegion(entry.key)
			if e != nil {
				return e
			}
			if e = h.adopt(region); e != nil {
				return e
			}
		}
		target[j] = entry
	}
	return nil
}

// A concrete Go owning context for an unjoined source NEW array. The header is
// a representation charge; the source allocation is the actual table leaf.
// No provider directory or old-table snapshot supplies acquisition authority.
type strHashtableLegacyAllocation struct {
	next              *strHashtableLegacyAllocation
	table             []strHashEntry
	tableBytes, bytes uint64
	cause             error
}

func (h *StrHashtable) retireLegacyAllocation(a *strHashtableLegacyAllocation) {
	link := &h.raw.legacyAcquisitions
	for *link != a {
		link = &(*link).next
	}
	*link = a.next
	h.config.Budget.Release(a.bytes)
	a.bytes = 0
}
func (h *StrHashtable) rehashLegacy(n uint32) error {
	if e := h.canBump(); e != nil {
		return e
	}
	bytes := uint64(reflect.TypeOf(strHashtableLegacyAllocation{}).Size())
	if e := h.config.Budget.Reserve(bytes); e != nil {
		return e
	}
	a := &strHashtableLegacyAllocation{bytes: bytes, next: h.raw.legacyAcquisitions}
	h.raw.legacyAcquisitions = a
	h.raw.executing = true
	defer func() { h.raw.executing = false }()
	table, amount, e := h.allocateTable(n)
	if e != nil {
		h.retireLegacyAllocation(a)
		return e
	}
	a.table, a.tableBytes = table, amount
	old, capacity := h.table, h.raw.words[2]
	if e = h.copyLegacyEntries(h, old, a.table, capacity, n, true); e != nil {
		a.cause = e
		return e
	}
	h.deleteLegacyTable()
	h.table, h.tableBytes = a.table, a.tableBytes
	a.table, a.tableBytes = nil, 0
	h.raw.words[2] = n
	h.deleted = 0
	h.bump()
	h.retireLegacyAllocation(a)
	return nil
}

// One reached source pointer dereference in the concrete Go-array profile.
// A call-local captured slice borrows storage; it never acquires ownership.
func legacyEntryAt(table []strHashEntry, i uint32) (*strHashEntry, error) {
	if uint64(i) >= uint64(len(table)) {
		return nil, arena.ErrBounds
	}
	return &table[i], nil
}
