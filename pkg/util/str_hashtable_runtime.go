// Source: src/util/hashtable.h:94-110,142-147,222-238,243-272,287-322,375-645;
// src/util/memory_manager.h:103-120; src/util/debug.cpp:44-68.
// Source author: Leonardo de Moura, 2006-09-11.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// The selected Go representation uses an actual 24-byte object header and
// 32-byte entries. These are payload blocks, not native ABI-size claims.
// The cold tokens below name only allocations returned by reached leaves.
var ErrStrHashtableExecuting = errors.New("g3: string-table allocation is executing")

type strHashtableRaw struct {
	destructing        bool
	executing          bool
	inline             bool
	words              [6]uint32
	inlineStatistics   [2]uint32
	initialized        uint8
	lexical            *strHashtableScope
	legacyAcquisitions *strHashtableLegacyAllocation
	self               *StrHashtable
	object, table      Block
	acquisitions       *strHashtableAcquisition
	sourceCause        error
	destructorCause    error
	sourceStage        uint8
	complete           bool
}
type strHashtableAcquisition struct {
	next  *strHashtableAcquisition
	block Block
	bytes uint64
}
type StrHashtableRuntimeRoots struct {
	Object, Table    Block
	Acquisitions     []Block
	AcquisitionBytes uint64
}

func (h *StrHashtable) RuntimeRoots() (StrHashtableRuntimeRoots, error) {
	if h == nil || h.config.Runtime == nil || h.raw.self != h {
		return StrHashtableRuntimeRoots{}, arena.ErrConfig
	}
	out := StrHashtableRuntimeRoots{Object: h.raw.object, Table: h.raw.table}
	for n := h.raw.acquisitions; n != nil; n = n.next {
		if n.block != (Block{}) {
			out.Acquisitions = append(out.Acquisitions, n.block)
		}
		out.AcquisitionBytes += n.bytes
	}
	for frame := h.raw.lexical; frame != nil; frame = frame.next {
		out.AcquisitionBytes += frame.bytes
		if frame.copy != nil {
			roots, e := frame.copy.RuntimeRoots()
			if e != nil {
				return out, e
			}
			out.AcquisitionBytes += frame.copy.rootBytes + frame.copy.directoryBytes + roots.AcquisitionBytes
			for _, b := range append([]Block{roots.Object, roots.Table}, roots.Acquisitions...) {
				if b != (Block{}) {
					out.Acquisitions = append(out.Acquisitions, b)
				}
			}
		}
	}
	return out, nil
}
func (h *StrHashtable) allocateRuntime(n uint64) (*strHashtableAcquisition, error) {
	amount := uint64(reflect.TypeOf(strHashtableAcquisition{}).Size())
	if e := h.config.Budget.Reserve(amount); e != nil {
		return nil, e
	}
	a := &strHashtableAcquisition{bytes: amount, next: h.raw.acquisitions}
	h.raw.acquisitions = a
	h.raw.executing = true
	b, e := h.config.Runtime.Allocate(n)
	a.block = b
	h.raw.executing = false
	if e != nil && b == (Block{}) {
		h.transferRuntime(a)
		return nil, e
	}
	return a, e
}
func (h *StrHashtable) transferRuntime(a *strHashtableAcquisition) Block {
	b := a.block
	a.block = Block{}
	// The actual destination field takes ownership; discard only this completed
	// acquisition node, never other unjoined allocations.
	link := &h.raw.acquisitions
	for *link != a {
		link = &(*link).next
	}
	*link = a.next
	h.config.Budget.Release(a.bytes)
	a.bytes = 0
	return b
}
func (h *StrHashtable) writeRuntimeWord(off uint64, v uint32) error {
	if h.raw.inline {
		if off%4 != 0 || off >= h.runtimeObjectExtent() {
			return arena.ErrBounds
		}
		if off >= 24 {
			h.raw.inlineStatistics[(off-24)/4] = v
		} else {
			h.raw.words[off/4] = v
		}
		h.raw.initialized |= 1 << uint8(off/4)
		return nil
	}
	return h.config.Runtime.WriteUint32(h.raw.object, off, v)
}
func (h *StrHashtable) runtimeWord(off uint64) (uint32, error) {
	if h.raw.inline {
		if off%4 != 0 || off >= h.runtimeObjectExtent() {
			return 0, arena.ErrBounds
		}
		if h.raw.initialized&(1<<uint8(off/4)) == 0 {
			return 0, ErrRuntimeUninitialized
		}
		if off >= 24 {
			return h.raw.inlineStatistics[(off-24)/4], nil
		}
		return h.raw.words[off/4], nil
	}
	return h.config.Runtime.ReadUint32(h.raw.object, off)
}
func (h *StrHashtable) mustRuntimeWord(off uint64) uint32 {
	if h.raw.self != h {
		panic(arena.ErrConfig)
	}
	if h.closed || !h.raw.complete || h.raw.destructing {
		panic(arena.ErrClosed)
	}
	v, e := h.runtimeWord(off)
	if e != nil {
		panic(e)
	}
	return v
}
func (h *StrHashtable) publishRuntimeTable(b Block) error {
	if e := h.writeRuntimeWord(0, b.Slot()); e != nil {
		return e
	}
	if e := h.writeRuntimeWord(4, b.Generation()); e != nil {
		return e
	}
	h.raw.table = b
	return nil
}
func (h *StrHashtable) runtimeTable() (Block, error) {
	slot, e := h.runtimeWord(0)
	if e != nil {
		return Block{}, e
	}
	gen, e := h.runtimeWord(4)
	if e != nil {
		return Block{}, e
	}
	b := h.raw.table
	if slot != b.Slot() || gen != b.Generation() {
		return Block{}, ErrRuntimeOwner
	}
	return b, nil
}
func (h *StrHashtable) allocateRuntimeTable(n uint32) (*strHashtableAcquisition, error) {
	a, e := h.allocateRuntime(uint64(n) * strHashEntryBytes)
	if e != nil {
		return a, e
	}
	for i := uint32(0); i < n; i++ {
		if e = h.config.Runtime.WriteUint32(a.block, uint64(i)*strHashEntryBytes+4, strHashFree); e != nil {
			return a, e
		}
	}
	return a, nil
}
func (h *StrHashtable) constructRuntime(n uint32) error {
	h.raw.self = h
	a, e := h.allocateRuntime(h.runtimeObjectExtent())
	if e != nil {
		return e
	}
	h.raw.object = h.transferRuntime(a)
	on, e := h.assertionsEnabled()
	if e == nil && on && (n == 0 || n&(n-1) != 0) {
		e = h.assertionFailure(249, "is_power_of_two(initial_capacity)")
	}
	if e != nil {
		body := e
		d := h.config.Runtime.Deallocate(&h.raw.object)
		if d != nil {
			h.raw.sourceCause = d
		}
		return errors.Join(body, d)
	}
	a, e = h.allocateRuntimeTable(n)
	if e != nil {
		// The entry array's failed construction is not a completed member. The
		// enclosing new-expression releases its real object allocation.
		body := e
		d := h.config.Runtime.Deallocate(&h.raw.object)
		if d != nil {
			h.raw.sourceCause = d
		}
		return errors.Join(body, d)
	}
	b := a.block
	if e = h.publishRuntimeTable(b); e != nil {
		return e
	}
	h.transferRuntime(a)
	for _, w := range []struct {
		off uint64
		v   uint32
	}{{8, n}, {12, 0}, {16, 0}} {
		if e = h.writeRuntimeWord(w.off, w.v); e != nil {
			return e
		}
	}
	if h.config.Statistics {
		if e = h.writeCollision(0); e != nil {
			return e
		}
	}
	h.raw.complete = true
	return nil
}
func (h *StrHashtable) openRuntime() error {
	if h.raw.executing {
		return ErrStrHashtableExecuting
	}
	if h.raw.self != h {
		return arena.ErrConfig
	}
	if h.closed || h.raw.destructing {
		return arena.ErrClosed
	}
	if h.raw.sourceCause != nil {
		return h.raw.sourceCause
	}
	if !h.raw.complete {
		return arena.ErrClosed
	}
	return nil
}

// SourceFinalize stops at the first reached source destructor action. It never
// aborts unjoined allocation contexts or resumes a previously stopped action.
func (h *StrHashtable) SourceFinalize() error {
	if h == nil {
		return arena.ErrClosed
	}
	if h.raw.self != h {
		return arena.ErrConfig
	}
	if h.raw.executing {
		return ErrStrHashtableExecuting
	}
	if h.config.Runtime == nil {
		return h.sourceFinalizeLegacy()
	}
	if !h.raw.complete {
		if h.raw.sourceCause != nil {
			return h.raw.sourceCause
		}
		return arena.ErrClosed
	}
	if h.raw.destructorCause != nil {
		return h.raw.destructorCause
	}
	if h.closed {
		return arena.ErrClosed
	}
	if h.raw.sourceStage == 0 {
		h.raw.destructing = true
		b, e := h.runtimeTable()
		if e != nil {
			h.stopRuntimeDestructor(e)
			return e
		}
		if b != (Block{}) {
			e = h.config.Runtime.Deallocate(&b)
			h.raw.table = b
			if e != nil {
				h.stopRuntimeDestructor(e)
				return e
			}
		}
		if e = h.publishRuntimeTable(Block{}); e != nil {
			h.stopRuntimeDestructor(e)
			return e
		}
		h.raw.sourceStage = 1
	}
	if h.raw.sourceStage == 1 {
		var e error
		if !h.raw.inline {
			e = h.config.Runtime.Deallocate(&h.raw.object)
		}
		if e != nil {
			h.stopRuntimeDestructor(e)
			return e
		}
		h.raw.sourceStage = 2
	}
	return nil
}

// Preserve the first body history separately from the first actually reached
// destructor refusal. A completed object's source DT does not replay a body
// failure; incomplete constructors receive only separate owning abort.
func (h *StrHashtable) stopRuntimeDestructor(e error) {
	if h.raw.destructorCause == nil {
		h.raw.destructorCause = e
	}
	if h.raw.sourceCause == nil {
		h.raw.sourceCause = e
	}
}

// CloseChecked is explicit owning abort. Token-zero receipts acknowledge real
// consumed storage; nonzero tokens still reach the real Runtime free leaf.
func (h *StrHashtable) CloseChecked() error {
	if h == nil {
		return arena.ErrClosed
	}
	if h.raw.self != h {
		return arena.ErrConfig
	}
	if h.raw.executing {
		return ErrStrHashtableExecuting
	}
	if h.closed {
		return nil
	}
	if h.config.Runtime == nil {
		return h.closeLegacyChecked()
	}
	for h.raw.lexical != nil {
		frame := h.raw.lexical
		if frame.copy != nil {
			if e := frame.copy.CloseChecked(); e != nil {
				return e
			}
		}
		h.raw.lexical = frame.next
		h.config.Budget.Release(frame.bytes)
	}
	for _, b := range []*Block{&h.raw.table, &h.raw.object} {
		if *b != (Block{}) {
			if e := h.config.Runtime.Deallocate(b); e != nil {
				return e
			}
		}
	}
	for h.raw.acquisitions != nil {
		a := h.raw.acquisitions
		if a.block != (Block{}) {
			if _, e := h.config.Runtime.Size(a.block); e != nil {
				if !errors.Is(e, ErrRuntimeUninitialized) {
					return e
				}
				if e = h.config.Runtime.CleanupPartial(&a.block); e != nil {
					return e
				}
			} else if e = h.config.Runtime.Deallocate(&a.block); e != nil {
				return e
			}
		}
		h.raw.acquisitions = a.next
		h.config.Budget.Release(a.bytes)
	}
	h.config.Budget.Release(h.rootBytes + h.directoryBytes)
	h.rootBytes = 0
	h.directoryBytes = 0
	h.owners = nil
	h.closed = true
	return nil
}

func (h *StrHashtable) stateAt(b Block, i uint32) (uint32, error) {
	return h.config.Runtime.ReadUint32(b, uint64(i)*32+4)
}
func (h *StrHashtable) keyAt(b Block, i uint32) (StrKey, error) {
	var w [6]uint32
	for j := range w {
		v, e := h.config.Runtime.ReadUint32(b, uint64(i)*32+8+uint64(j)*4)
		if e != nil {
			return StrKey{}, e
		}
		w[j] = v
	}
	return StrKey{ref: arena.BytesRegionRef{Slot: arena.Slot(w[0]), Generation: arena.Generation(w[1])}, offset: w[2], length: w[3], origin: uint64(w[4]) | uint64(w[5])<<32}, nil
}
func (h *StrHashtable) putKeyAt(b Block, i uint32, k StrKey) error {
	w := [6]uint32{uint32(k.ref.Slot), uint32(k.ref.Generation), k.offset, k.length, uint32(k.origin), uint32(k.origin >> 32)}
	for j, v := range w {
		if e := h.config.Runtime.WriteUint32(b, uint64(i)*32+8+uint64(j)*4, v); e != nil {
			return e
		}
	}
	return h.config.Runtime.WriteUint32(b, uint64(i)*32+4, strHashUsed)
}
func (h *StrHashtable) probeEntry(i uint32) (strHashEntry, error) {
	if h.config.Runtime == nil {
		return h.table[i], nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return strHashEntry{}, e
	}
	state, e := h.stateAt(b, i)
	if e != nil {
		return strHashEntry{}, e
	}
	entry := strHashEntry{state: state}
	if state == strHashUsed {
		entry.hash, e = h.config.Runtime.ReadUint32(b, uint64(i)*32)
	}
	return entry, e
}
func (h *StrHashtable) entryKey(i uint32) (StrKey, error) {
	if h.config.Runtime == nil {
		return h.table[i].key, nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return StrKey{}, e
	}
	return h.keyAt(b, i)
}
func (h *StrHashtable) readEntry(i uint32) (strHashEntry, error) {
	entry, e := h.probeEntry(i)
	if e != nil {
		return entry, e
	}
	if entry.state == strHashUsed {
		entry.key, e = h.entryKey(i)
	}
	return entry, e
}
func (h *StrHashtable) moveRuntimeTable(old Block, oldcap uint32, next Block, n uint32) error {
	on, e := h.assertionsEnabled()
	if e != nil {
		return e
	}
	if on && n < oldcap {
		if e = h.assertionFailure(191, "target_capacity >= source_capacity"); e != nil {
			return e
		}
	}
	for i := uint32(0); i < oldcap; i++ {
		state, e := h.stateAt(old, i)
		if e != nil {
			return e
		}
		if state != strHashUsed {
			continue
		}
		hash, e := h.config.Runtime.ReadUint32(old, uint64(i)*32)
		if e != nil {
			return e
		}
		start := hash & (n - 1)
		j := start
		found := false
		for step := uint32(0); step < n; step++ {
			j = sourceProbeIndex(start, step, n)
			on, e = h.assertionsEnabled()
			if e != nil {
				return e
			}
			if on {
				state, e = h.stateAt(next, j)
				if e != nil {
					return e
				}
				if state == strHashDeleted {
					line := int32(202)
					if j < start {
						line = 209
					}
					if e = h.assertionFailure(line, "!target_curr->is_deleted()"); e != nil {
						return e
					}
				}
			}
			state, e = h.stateAt(next, j)
			if e != nil {
				return e
			}
			if state == strHashFree {
				found = true
				break
			}
		}
		if !found {
			if e = h.unreachable(215); e != nil {
				return e
			}
			continue
		}
		k, e := h.keyAt(old, i)
		if e != nil {
			return e
		}
		if e = h.putKeyAt(next, j, k); e != nil {
			return e
		}
		if e = h.config.Runtime.WriteUint32(next, uint64(j)*32, hash); e != nil {
			return e
		}
	}
	return nil
}
func (h *StrHashtable) rehashRuntime(n uint32) error {
	cap, e := h.runtimeWord(8)
	if e != nil {
		return e
	}
	old, e := h.runtimeTable()
	if e != nil {
		return e
	}
	a, e := h.allocateRuntimeTable(n)
	if e != nil {
		return e
	}
	if e = h.moveRuntimeTable(old, cap, a.block, n); e != nil {
		return e
	}
	// delete_table frees first, then publishes NULL, before the new pointer.
	b := old
	e = h.config.Runtime.Deallocate(&b)
	h.raw.table = b
	if e != nil {
		h.raw.sourceCause = e
		return e
	}
	if e = h.publishRuntimeTable(Block{}); e != nil {
		return e
	}
	if e = h.publishRuntimeTable(a.block); e != nil {
		return e
	}
	h.transferRuntime(a)
	if e = h.writeRuntimeWord(8, n); e != nil {
		return e
	}
	if e = h.writeRuntimeWord(16, 0); e != nil {
		return e
	}
	h.bump()
	return nil
}
func (h *StrHashtable) prepareInsertRuntime() error {
	if e := h.open(); e != nil {
		return e
	}
	size, e := h.runtimeWord(12)
	if e != nil {
		return e
	}
	deleted, e := h.runtimeWord(16)
	if e != nil {
		return e
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return e
	}
	if (size+deleted)*4 > cap*3 {
		return h.rehashRuntime(cap << 1)
	}
	return nil
}
func (h *StrHashtable) insertRuntime(region *arena.BytesRegion, key StrKey, overwrite bool) (bool, StrHashtableEntryRef, error) {
	if e := h.prepareInsertRuntime(); e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	hash, e := hashStrKey(region, key)
	if e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	b, e := h.runtimeTable()
	if e != nil {
		return false, StrHashtableEntryRef{}, e
	}
	i := hash & (cap - 1)
	deleted := ^uint32(0)
	for step := uint32(0); step < cap; step++ {
		state, e := h.stateAt(b, i)
		if e != nil {
			return false, StrHashtableEntryRef{}, e
		}
		switch state {
		case strHashUsed:
			cached, e := h.config.Runtime.ReadUint32(b, uint64(i)*32)
			if e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			if cached == hash {
				k, e := h.keyAt(b, i)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				owner, e := h.keyRegion(k)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				equal, e := equalStrKeys(owner, k, region, key)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				if equal {
					if overwrite {
						if e = h.adopt(region); e != nil {
							return false, StrHashtableEntryRef{}, e
						}
						if e = h.putKeyAt(b, i, key); e != nil {
							return false, StrHashtableEntryRef{}, e
						}
					}
					return false, h.ref(i), nil
				}
			}
			if e = h.incrementCollision(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
		default:
			on, err := h.assertionsEnabled()
			if err != nil {
				return false, StrHashtableEntryRef{}, err
			}
			if on {
				actual, err := h.stateAt(b, i)
				if err != nil {
					return false, StrHashtableEntryRef{}, err
				}
				if actual != strHashDeleted {
					line := int32(371)
					if !overwrite {
						line = 422
					}
					if err = h.assertionFailure(line, "curr->is_deleted()"); err != nil {
						return false, StrHashtableEntryRef{}, err
					}
				}
			}
			deleted = i
			if e = h.incrementCollision(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
		case strHashFree:
			if e = h.canBump(); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			if e = h.adopt(region); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
				count, e := h.runtimeWord(16)
				if e != nil {
					return false, StrHashtableEntryRef{}, e
				}
				if e = h.writeRuntimeWord(16, count-1); e != nil {
					return false, StrHashtableEntryRef{}, e
				}
			}
			if e = h.putKeyAt(b, slot, key); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			if e = h.config.Runtime.WriteUint32(b, uint64(slot)*32, hash); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			size, e := h.runtimeWord(12)
			if e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			if e = h.writeRuntimeWord(12, size+1); e != nil {
				return false, StrHashtableEntryRef{}, e
			}
			h.bump()
			return true, h.ref(slot), nil
		}
		i++
		if i == cap {
			i = 0
		}
	}
	line := int32(395)
	if !overwrite {
		line = 451
	}
	return false, StrHashtableEntryRef{}, h.unreachable(line)
}

func (h *StrHashtable) deleteRuntimeTable() error {
	b, e := h.runtimeTable()
	if e != nil {
		return e
	}
	if b != (Block{}) {
		e = h.config.Runtime.Deallocate(&b)
		h.raw.table = b
		if e != nil {
			h.raw.sourceCause = e
			return e
		}
	}
	return h.publishRuntimeTable(Block{})
}
func (h *StrHashtable) resetRuntime() error {
	if e := h.open(); e != nil {
		return e
	}
	size, e := h.runtimeWord(12)
	if e != nil {
		return e
	}
	deleted, e := h.runtimeWord(16)
	if e != nil {
		return e
	}
	if size == 0 && deleted == 0 {
		return nil
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return e
	}
	b, e := h.runtimeTable()
	if e != nil {
		return e
	}
	var overhead uint32
	for i := uint32(0); i < cap; i++ {
		state, e := h.stateAt(b, i)
		if e != nil {
			return e
		}
		if state != strHashFree {
			if e = h.config.Runtime.WriteUint32(b, uint64(i)*32+4, strHashFree); e != nil {
				return e
			}
		} else {
			overhead++
		}
	}
	h.bump()
	if cap > 16 && overhead<<2 > cap*3 {
		if e = h.deleteRuntimeTable(); e != nil {
			return e
		}
		for _, site := range []struct {
			line int32
			text string
		}{{300, "m_capacity > 16"}, {301, "is_power_of_two(m_capacity)"}} {
			on, err := h.assertionsEnabled()
			if err != nil {
				return err
			}
			if on {
				value, err := h.runtimeWord(8)
				if err != nil {
					return err
				}
				condition := value > 16
				if site.line == 301 {
					condition = value != 0 && value&(value-1) == 0
				}
				if !condition {
					if err = h.assertionFailure(site.line, site.text); err != nil {
						return err
					}
				}
			}
		}
		cap, e = h.runtimeWord(8)
		if e != nil {
			return e
		}
		cap >>= 1
		if e = h.writeRuntimeWord(8, cap); e != nil {
			return e
		}
		on, err := h.assertionsEnabled()
		if err != nil {
			return err
		}
		if on {
			value, err := h.runtimeWord(8)
			if err != nil {
				return err
			}
			if value == 0 || value&(value-1) != 0 {
				if err = h.assertionFailure(303, "is_power_of_two(m_capacity)"); err != nil {
					return err
				}
			}
		}
		cap, e = h.runtimeWord(8)
		if e != nil {
			return e
		}
		a, e := h.allocateRuntimeTable(cap)
		if e != nil {
			return e
		}
		if e = h.publishRuntimeTable(a.block); e != nil {
			return e
		}
		h.transferRuntime(a)
	}
	if e = h.writeRuntimeWord(12, 0); e != nil {
		return e
	}
	return h.writeRuntimeWord(16, 0)
}
func (h *StrHashtable) finalizeRuntime() error {
	if e := h.open(); e != nil {
		return e
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return e
	}
	if cap <= 64 {
		return h.resetRuntime()
	}
	if e = h.deleteRuntimeTable(); e != nil {
		return e
	}
	a, e := h.allocateRuntimeTable(64)
	if e != nil {
		return e
	}
	if e = h.publishRuntimeTable(a.block); e != nil {
		return e
	}
	h.transferRuntime(a)
	if e = h.writeRuntimeWord(8, 64); e != nil {
		return e
	}
	if e = h.writeRuntimeWord(12, 0); e != nil {
		return e
	}
	h.bump()
	return h.writeRuntimeWord(16, 0)
}
func (h *StrHashtable) removeRuntime(region *arena.BytesRegion, k StrKey) error {
	if e := h.open(); e != nil {
		return e
	}
	exhausted := false
	i, ok, e := h.findSlotFromSource(region, k, &exhausted)
	if e != nil {
		return e
	}
	if !ok {
		if exhausted {
			on, err := h.assertionsEnabled()
			if err != nil {
				return err
			}
			if on {
				found, err := h.ContainsFrom(region, k)
				if err != nil {
					return err
				}
				if found {
					if err = h.assertionFailure(559, "!contains(e)"); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return e
	}
	b, e := h.runtimeTable()
	if e != nil {
		return e
	}
	nextIndex := i + 1
	if nextIndex == cap {
		nextIndex = 0
	}
	next, e := h.stateAt(b, nextIndex)
	if e != nil {
		return e
	}
	if next == strHashFree {
		if e = h.config.Runtime.WriteUint32(b, uint64(i)*32+4, strHashFree); e != nil {
			return e
		}
		size, e := h.runtimeWord(12)
		if e != nil {
			return e
		}
		e = h.writeRuntimeWord(12, size-1)
		h.bump()
		return e
	}
	if e = h.config.Runtime.WriteUint32(b, uint64(i)*32+4, strHashDeleted); e != nil {
		return e
	}
	deleted, e := h.runtimeWord(16)
	if e != nil {
		return e
	}
	if e = h.writeRuntimeWord(16, deleted+1); e != nil {
		return e
	}
	size, e := h.runtimeWord(12)
	if e != nil {
		return e
	}
	if e = h.writeRuntimeWord(12, size-1); e != nil {
		return e
	}
	h.bump()
	deleted, e = h.runtimeWord(16)
	if e != nil {
		return e
	}
	size, e = h.runtimeWord(12)
	if e != nil {
		return e
	}
	if deleted > size && deleted > 64 {
		if e = h.config.Runtime.CheckOpen(); e != nil {
			return e
		}
		if !h.config.Runtime.State().OutOfMemory {
			return h.rehashRuntime(cap)
		}
	}
	return nil
}

func (h *StrHashtable) capacityChecked() (uint32, error) {
	if h.config.Runtime != nil {
		return h.runtimeWord(8)
	}
	return h.raw.words[2], nil
}
func (h *StrHashtable) sizeChecked() (uint32, error) {
	if h.config.Runtime != nil {
		return h.runtimeWord(12)
	}
	return h.size, nil
}
func (h *StrHashtable) cloneRuntime(heap bool) (*StrHashtable, error) {
	var e error
	root := uint64(reflect.TypeOf(StrHashtable{}).Size())
	if e = h.config.Budget.Reserve(root); e != nil {
		return nil, e
	}
	out := &StrHashtable{region: h.region, config: h.config, epoch: 1, rootBytes: root}
	out.raw.self = out
	out.raw.inline = !heap
	var a *strHashtableAcquisition
	if heap {
		a, e = out.allocateRuntime(out.runtimeObjectExtent())
		if e != nil {
			return retireEmptyFailedRuntime(out, e)
		}
		out.raw.object = out.transferRuntime(a)
	}
	failed := func(cause error) (*StrHashtable, error) {
		var d error
		if !out.raw.inline {
			d = out.config.Runtime.Deallocate(&out.raw.object)
		}
		if d != nil {
			out.raw.sourceCause = d
		}
		return retireEmptyFailedRuntime(out, errors.Join(cause, d))
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return failed(e)
	}
	if e = out.writeRuntimeWord(8, cap); e != nil {
		return failed(e)
	}
	a, e = out.allocateRuntimeTable(cap)
	if e != nil {
		return failed(e)
	}
	if e = out.publishRuntimeTable(a.block); e != nil {
		return failed(e)
	}
	out.transferRuntime(a)
	old, e := h.runtimeTable()
	if e != nil {
		return failed(e)
	}
	// copy_table captures both actual pointer arguments before its own assertions.
	target := out.raw.table
	// Deleted slots compress with ascending original bucket order.
	on, e := out.assertionsEnabled()
	if e != nil {
		return failed(e)
	}
	if on {
		sourcecap := cap
		if cap < sourcecap {
			if e = out.assertionFailure(159, "target_capacity >= source_capacity"); e != nil {
				return failed(e)
			}
		}
	}
	for i := uint32(0); i < cap; i++ {
		state, e := h.stateAt(old, i)
		if e != nil {
			return failed(e)
		}
		if state != strHashUsed {
			continue
		}
		hash, e := h.config.Runtime.ReadUint32(old, uint64(i)*32)
		if e != nil {
			return failed(e)
		}
		start := hash & (cap - 1)
		j := start
		found := false
		for step := uint32(0); step < cap; step++ {
			j = sourceProbeIndex(start, step, cap)
			on, e = out.assertionsEnabled()
			if e != nil {
				return failed(e)
			}
			if on {
				state, e = out.stateAt(target, j)
				if e != nil {
					return failed(e)
				}
				if state == strHashDeleted {
					line := int32(170)
					if j < start {
						line = 177
					}
					if e = out.assertionFailure(line, "!target_curr->is_deleted()"); e != nil {
						return failed(e)
					}
				}
			}
			state, e = out.stateAt(target, j)
			if e != nil {
				return failed(e)
			}
			if state == strHashFree {
				found = true
				break
			}
		}
		if !found {
			if e = out.unreachable(183); e != nil {
				return failed(e)
			}
			continue
		}
		k, e := h.keyAt(old, i)
		if e != nil {
			return failed(e)
		}
		owner, e := h.keyRegion(k)
		if e != nil {
			return failed(e)
		}
		if e = out.adopt(owner); e != nil {
			return failed(e)
		}
		if e = out.putKeyAt(target, j, k); e != nil {
			return failed(e)
		}
		if e = out.config.Runtime.WriteUint32(target, uint64(j)*32, hash); e != nil {
			return failed(e)
		}
	}
	size, e := h.runtimeWord(12)
	if e != nil {
		return failed(e)
	}
	if e = out.writeRuntimeWord(12, size); e != nil {
		return failed(e)
	}
	if e = out.writeRuntimeWord(16, 0); e != nil {
		return failed(e)
	}
	if out.config.Statistics {
		if e = out.writeCollision(0); e != nil {
			return failed(e)
		}
	}
	out.raw.complete = true
	return out, nil
}
func (h *StrHashtable) swapRuntime(other *StrHashtable) error {
	if h.config.Runtime != other.config.Runtime || h.config.Budget != other.config.Budget || h.config.Statistics != other.config.Statistics {
		return arena.ErrConfig
	}
	if e := h.bothOpen(other); e != nil {
		return e
	}
	if h == other {
		return nil
	}
	a, e := h.runtimeTable()
	if e != nil {
		return e
	}
	b, e := other.runtimeTable()
	if e != nil {
		return e
	}
	// Source swaps pointer first, then capacity, size and deleted fields.
	if e = h.publishRuntimeTable(b); e != nil {
		return e
	}
	if e = other.publishRuntimeTable(a); e != nil {
		return e
	}
	for _, off := range []uint64{8, 12, 16} {
		x, e := h.runtimeWord(off)
		if e != nil {
			return e
		}
		y, e := other.runtimeWord(off)
		if e != nil {
			return e
		}
		if e = h.writeRuntimeWord(off, y); e != nil {
			return e
		}
		if e = other.writeRuntimeWord(off, x); e != nil {
			return e
		}
	}
	if h.config.Statistics {
		left, e := h.readCollision()
		if e != nil {
			return e
		}
		right, e := other.readCollision()
		if e != nil {
			return e
		}
		if e = h.writeCollision(right); e != nil {
			return e
		}
		if e = other.writeCollision(left); e != nil {
			return e
		}
	}
	h.region, other.region = other.region, h.region
	h.owners, other.owners = other.owners, h.owners
	h.directoryBytes, other.directoryBytes = other.directoryBytes, h.directoryBytes
	h.bump()
	other.bump()
	return nil
}

func (h *StrHashtable) entryState(i uint32) (uint32, error) {
	if h.config.Runtime == nil {
		return h.table[i].state, nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return 0, e
	}
	return h.stateAt(b, i)
}
func (h *StrHashtable) CheckInvariantChecked() (bool, error) {
	if e := h.open(); e != nil {
		return false, e
	}
	cap, e := h.capacityChecked()
	if e != nil {
		return false, e
	}
	table := h.table
	var used, deleted uint32
	for i := uint32(0); i < cap; i++ {
		var state uint32
		if h.config.Runtime == nil {
			p, re := legacyEntryAt(table, i)
			if re != nil {
				e = re
			} else {
				state = p.state
			}
		} else {
			state, e = h.entryState(i)
		}
		if e != nil {
			return false, e
		}
		if state == strHashDeleted {
			deleted++
		}
		if state == strHashUsed {
			used++
		}
	}
	if strHashtableSourceDebug {
		on, e := h.assertionsEnabled()
		if e != nil {
			return false, e
		}
		if on {
			expected := h.deleted
			if h.config.Runtime != nil {
				expected, e = h.runtimeWord(16)
				if e != nil {
					return false, e
				}
			}
			if deleted != expected {
				if e = h.assertionFailure(643, "num_deleted == m_num_deleted"); e != nil {
					return false, e
				}
			}
		}
		on, e = h.assertionsEnabled()
		if e != nil {
			return false, e
		}
		if on {
			expected := h.size
			if h.config.Runtime != nil {
				expected, e = h.runtimeWord(12)
				if e != nil {
					return false, e
				}
			}
			if used != expected {
				if e = h.assertionFailure(644, "num_used == m_size"); e != nil {
					return false, e
				}
			}
		}
		return true, nil
	}
	size, e := h.sizeChecked()
	if e != nil {
		return false, e
	}
	var expected uint32
	if h.config.Runtime != nil {
		expected, e = h.runtimeWord(16)
	} else {
		expected = h.deleted
	}
	return used == size && deleted == expected, e
}

// HasRetainedRuntimeAcquisitions observes this owner's actual acquisition
// fields. It supplies no authority over Runtime directory records or callers.
func (h *StrHashtable) HasRetainedRuntimeAcquisitions() bool {
	if h == nil || h.config.Runtime == nil || h.raw.self != h {
		panic(arena.ErrConfig)
	}
	return h.raw.acquisitions != nil || h.raw.lexical != nil
}

// The lexical source copy has an inline object header, and owns only its real
// table allocation. The retained context is acquired before the copy action;
// it names a stopped copy/destructor without rebuilding or scanning storage.
type strHashtableScope struct {
	next     *strHashtableScope
	copy     *StrHashtable
	cause    error
	bytes    uint64
	complete bool
}

func (h *StrHashtable) retireScope(frame *strHashtableScope) {
	link := &h.raw.lexical
	for *link != frame {
		link = &(*link).next
	}
	*link = frame.next
	h.config.Budget.Release(frame.bytes)
}
func (h *StrHashtable) intersectRuntime(src *StrHashtable) (result error) {
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
	copy, e := h.cloneRuntime(false)
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
		e := copy.SourceFinalize()
		if e != nil {
			if frame.cause == nil {
				frame.cause = e
			}
			result = errors.Join(result, e)
			return
		}
		if copy.HasRetainedRuntimeAcquisitions() {
			return
		} // separate owning abort
		if e = copy.CloseChecked(); e != nil {
			if frame.cause == nil {
				frame.cause = e
			}
			result = errors.Join(result, e)
			return
		}
		frame.copy = nil
		h.retireScope(frame)
	}()
	cap, e := copy.capacityChecked()
	if e != nil {
		frame.cause = e
		return e
	}
	for i := uint32(0); i < cap; i++ {
		state, e := copy.entryState(i)
		if e != nil {
			frame.cause = e
			return e
		}
		if state != strHashUsed {
			continue
		}
		k, e := copy.entryKey(i)
		if e != nil {
			frame.cause = e
			return e
		}
		owner, e := copy.keyRegion(k)
		if e != nil {
			frame.cause = e
			return e
		}
		ok, e := src.ContainsFrom(owner, k)
		if e != nil {
			frame.cause = e
			return e
		}
		if !ok {
			if e = h.RemoveFrom(owner, k); e != nil {
				frame.cause = e
				return e
			}
		}
	}
	return nil
}

func retireEmptyFailedRuntime(h *StrHashtable, cause error) (*StrHashtable, error) {
	if h.raw.object == (Block{}) && h.raw.table == (Block{}) && h.raw.acquisitions == nil && h.raw.lexical == nil && h.raw.sourceCause == nil {
		if e := h.CloseChecked(); e != nil {
			return h, errors.Join(cause, e)
		}
		return nil, cause
	}
	return h, cause
}
