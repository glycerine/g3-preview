// Source: src/util/hashtable.h:259-283,287-322,613-623.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

func (h *StrHashtable) deleteLegacyTable() {
	h.config.Budget.Release(h.tableBytes)
	h.table = nil
	h.tableBytes = 0
}
func (h *StrHashtable) stopLegacy(e error) error {
	if e != nil && h.raw.sourceCause == nil {
		h.raw.sourceCause = e
	}
	return e
}
func (h *StrHashtable) resetLegacy() error {
	if h != nil && h.raw.self != h {
		return arena.ErrConfig
	}
	// src/util/hashtable.h:288-289 reads only these scalar members. A prior
	// replacement refusal does not make this defined no-op read the table.
	if h == nil || h.closed || h.raw.destructing {
		return arena.ErrClosed
	}
	if h.size == 0 && h.deleted == 0 {
		return nil
	}
	if e := h.open(); e != nil {
		return e
	}
	if e := h.canBump(); e != nil {
		return e
	}
	var overhead uint32
	for i := range h.table {
		if h.table[i].state != strHashFree {
			h.table[i].state = strHashFree
		} else {
			overhead++
		}
	}
	h.bump()
	capacity := h.raw.words[2]
	if capacity > 16 && overhead<<2 > capacity*3 {
		h.deleteLegacyTable()
		for _, site := range []struct {
			line int32
			text string
		}{{300, "m_capacity > 16"}, {301, "is_power_of_two(m_capacity)"}} {
			on, e := h.assertionsEnabled()
			if e != nil {
				return h.stopLegacy(e)
			}
			if on {
				value := h.raw.words[2]
				condition := value > 16
				if site.line == 301 {
					condition = value != 0 && value&(value-1) == 0
				}
				if !condition {
					if e = h.assertionFailure(site.line, site.text); e != nil {
						return h.stopLegacy(e)
					}
				}
			}
		}
		h.raw.words[2] >>= 1
		on, e := h.assertionsEnabled()
		if e != nil {
			return h.stopLegacy(e)
		}
		if on {
			value := h.raw.words[2]
			if value == 0 || value&(value-1) != 0 {
				if e = h.assertionFailure(303, "is_power_of_two(m_capacity)"); e != nil {
					return h.stopLegacy(e)
				}
			}
		}
		table, bytes, e := h.allocateTable(h.raw.words[2])
		if e != nil {
			return h.stopLegacy(e)
		}
		h.table, h.tableBytes = table, bytes
	}
	h.size, h.deleted = 0, 0
	return nil
}
func (h *StrHashtable) finalizeLegacy() error {
	if e := h.open(); e != nil {
		return e
	}
	if h.raw.words[2] <= 64 {
		return h.resetLegacy()
	}
	if e := h.canBump(); e != nil {
		return e
	}
	h.deleteLegacyTable()
	h.bump()
	table, bytes, e := h.allocateTable(64)
	if e != nil {
		return h.stopLegacy(e)
	}
	h.table, h.tableBytes = table, bytes
	h.raw.words[2] = 64
	h.size, h.deleted = 0, 0
	return nil
}

func (h *StrHashtable) sourceFinalizeLegacy() error {
	if !h.raw.complete {
		// A failed copy constructor has no source core destructor.
		if h.raw.sourceCause != nil {
			return h.raw.sourceCause
		}
		return arena.ErrClosed
	}
	if h.raw.sourceStage == 2 {
		return nil
	}
	h.raw.destructing = true
	h.deleteLegacyTable()
	h.raw.sourceStage = 2
	return nil
}
func (h *StrHashtable) closeLegacyChecked() error {
	if h.closed {
		return nil
	}
	for a := h.raw.legacyAcquisitions; a != nil; a = h.raw.legacyAcquisitions {
		h.config.Budget.Release(a.tableBytes)
		a.table, a.tableBytes = nil, 0
		h.retireLegacyAllocation(a)
	}
	for h.raw.lexical != nil {
		frame := h.raw.lexical
		if frame.copy != nil {
			if e := frame.copy.CloseChecked(); e != nil {
				return e
			}
		}
		h.retireScope(frame)
	}
	h.deleteLegacyTable()
	h.config.Budget.Release(h.rootBytes + h.directoryBytes)
	h.rootBytes, h.directoryBytes = 0, 0
	h.owners = nil
	h.region = nil
	h.size, h.deleted = 0, 0
	h.raw.words[2] = 0
	h.closed = true
	if h.epoch != ^uint64(0) {
		h.bump()
	}
	return nil
}
func (h *StrHashtable) legacyRetainedBytes() uint64 {
	if h.config.Runtime != nil {
		return 0
	}
	var total uint64
	for a := h.raw.legacyAcquisitions; a != nil; a = a.next {
		total += a.bytes + a.tableBytes
	}
	for frame := h.raw.lexical; frame != nil; frame = frame.next {
		total += frame.bytes
		if frame.copy != nil {
			copy := frame.copy
			total += copy.rootBytes + copy.tableBytes + copy.directoryBytes + copy.legacyRetainedBytes()
		}
	}
	return total
}
func (h *StrHashtable) cloneLegacy() (*StrHashtable, error) {
	root := uint64(reflect.TypeOf(StrHashtable{}).Size())
	if e := h.config.Budget.Reserve(root); e != nil {
		return nil, e
	}
	out := &StrHashtable{region: h.region, config: h.config, epoch: 1, rootBytes: root}
	out.raw.self = out
	out.raw.words[2] = h.raw.words[2]
	table, bytes, e := out.allocateTable(out.raw.words[2])
	if e != nil {
		if cleanup := out.CloseChecked(); cleanup != nil {
			return out, errors.Join(e, cleanup)
		}
		return nil, e
	}
	out.table, out.tableBytes = table, bytes
	if e = out.copyLegacyEntries(h, h.table, out.table, h.raw.words[2], out.raw.words[2], false); e != nil {
		out.raw.sourceCause = e
		return out, e
	}
	out.size = h.size
	if out.config.Statistics {
		if e = out.writeCollision(0); e != nil {
			return out, e
		}
	}
	out.raw.complete = true
	return out, nil
}
func (h *StrHashtable) intersectLegacy(src *StrHashtable) (result error) {
	if e := h.bothOpen(src); e != nil {
		return e
	}
	if h == src {
		return nil
	}
	bytes := uint64(reflect.TypeOf(strHashtableScope{}).Size())
	if e := h.config.Budget.Reserve(bytes); e != nil {
		return e
	}
	frame := &strHashtableScope{bytes: bytes, next: h.raw.lexical}
	h.raw.lexical = frame
	h.raw.executing = true
	copy, e := h.cloneLegacy()
	frame.copy = copy
	h.raw.executing = false
	if e != nil {
		frame.cause = e
		if copy == nil {
			h.retireScope(frame)
		}
		return e
	}
	frame.complete = true
	defer func() {
		if e := copy.SourceFinalize(); e != nil {
			if frame.cause == nil {
				frame.cause = e
			}
			result = errors.Join(result, e)
			return
		}
		if e := copy.CloseChecked(); e != nil {
			if frame.cause == nil {
				frame.cause = e
			}
			result = errors.Join(result, e)
			return
		}
		frame.copy = nil
		h.retireScope(frame)
	}()
	table := copy.table
	for _, entry := range table {
		if entry.state != strHashUsed {
			continue
		}
		owner, e := copy.keyRegion(entry.key)
		if e != nil {
			return e
		}
		contains, e := src.ContainsFrom(owner, entry.key)
		if e != nil {
			return e
		}
		if !contains {
			if e = h.Remove(entry.key); e != nil {
				return e
			}
		}
	}
	return nil
}
