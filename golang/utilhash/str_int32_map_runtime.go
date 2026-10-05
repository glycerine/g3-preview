// Concrete source specialization: src/util/map.h:24-196; src/util/hashtable.h:44-63.
// Source: src/util/hashtable.h:44-63,142-147,222-238,243-272,287-322,375-645;
// src/util/memory_manager.h:103-120; src/util/debug.cpp:44-68.
// Source author: Leonardo de Moura, 2006-09-11.
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// The selected Go representation uses an actual 24-byte object header and
// 32-byte entries. These are payload blocks, not native ABI-size claims.
// The cold tokens below name only allocations returned by reached leaves.
var ErrStrInt32MapExecuting = errors.New("g3: string-table allocation is executing")

type strInt32MapRaw struct {
	destructing        bool
	executing          bool
	inline             bool
	words              [6]uint32
	inlineStatistics   [2]uint32
	initialized        uint8
	lexical            *strInt32MapScope
	legacyAcquisitions *strInt32MapLegacyAllocation
	self               *StrInt32Map
	object, table      memory.Block
	acquisitions       *strInt32MapAcquisition
	sourceCause        error
	destructorCause    error
	sourceStage        uint8
	complete           bool
}
type strInt32MapAcquisition struct {
	next  *strInt32MapAcquisition
	block memory.Block
	bytes uint64
}
type StrInt32MapRuntimeRoots struct {
	Object, Table    memory.Block
	Acquisitions     []memory.Block
	AcquisitionBytes uint64
}

func (h *StrInt32Map) RuntimeRoots() (StrInt32MapRuntimeRoots, error) {
	if h == nil || h.config.Runtime == nil || h.raw.self != h {
		return StrInt32MapRuntimeRoots{}, arena.ErrConfig
	}
	out := StrInt32MapRuntimeRoots{Object: h.raw.object, Table: h.raw.table}
	for n := h.raw.acquisitions; n != nil; n = n.next {
		if n.block != (memory.Block{}) {
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
			for _, b := range append([]memory.Block{roots.Object, roots.Table}, roots.Acquisitions...) {
				if b != (memory.Block{}) {
					out.Acquisitions = append(out.Acquisitions, b)
				}
			}
		}
	}
	return out, nil
}
func (h *StrInt32Map) allocateRuntime(n uint64) (*strInt32MapAcquisition, error) {
	amount := uint64(reflect.TypeOf(strInt32MapAcquisition{}).Size())
	if e := h.config.Budget.Reserve(amount); e != nil {
		return nil, e
	}
	a := &strInt32MapAcquisition{bytes: amount, next: h.raw.acquisitions}
	h.raw.acquisitions = a
	h.raw.executing = true
	b, e := h.config.Runtime.Allocate(n)
	a.block = b
	h.raw.executing = false
	if e != nil && b == (memory.Block{}) {
		h.transferRuntime(a)
		return nil, e
	}
	return a, e
}
func (h *StrInt32Map) transferRuntime(a *strInt32MapAcquisition) memory.Block {
	b := a.block
	a.block = memory.Block{}
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
func (h *StrInt32Map) writeRuntimeWord(off uint64, v uint32) error {
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
func (h *StrInt32Map) runtimeWord(off uint64) (uint32, error) {
	if h.raw.inline {
		if off%4 != 0 || off >= h.runtimeObjectExtent() {
			return 0, arena.ErrBounds
		}
		if h.raw.initialized&(1<<uint8(off/4)) == 0 {
			return 0, memory.ErrRuntimeUninitialized
		}
		if off >= 24 {
			return h.raw.inlineStatistics[(off-24)/4], nil
		}
		return h.raw.words[off/4], nil
	}
	return h.config.Runtime.ReadUint32(h.raw.object, off)
}
func (h *StrInt32Map) mustRuntimeWord(off uint64) uint32 {
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
func (h *StrInt32Map) publishRuntimeTable(b memory.Block) error {
	if e := h.writeRuntimeWord(0, b.Slot()); e != nil {
		return e
	}
	if e := h.writeRuntimeWord(4, b.Generation()); e != nil {
		return e
	}
	h.raw.table = b
	return nil
}
func (h *StrInt32Map) runtimeTable() (memory.Block, error) {
	slot, e := h.runtimeWord(0)
	if e != nil {
		return memory.Block{}, e
	}
	gen, e := h.runtimeWord(4)
	if e != nil {
		return memory.Block{}, e
	}
	b := h.raw.table
	if slot != b.Slot() || gen != b.Generation() {
		return memory.Block{}, memory.ErrRuntimeOwner
	}
	return b, nil
}
func (h *StrInt32Map) allocateRuntimeTable(n uint32) (*strInt32MapAcquisition, error) {
	a, e := h.allocateRuntime(uint64(n) * strInt32EntryBytes)
	if e != nil {
		return a, e
	}
	for i := uint32(0); i < n; i++ {
		if e = h.config.Runtime.WriteUint32(a.block, uint64(i)*strInt32EntryBytes+4, strHashFree); e != nil {
			return a, e
		}
	}
	return a, nil
}
func (h *StrInt32Map) constructRuntime(n uint32) error {
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
func (h *StrInt32Map) openRuntime() error {
	if h.raw.executing {
		return ErrStrInt32MapExecuting
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
func (h *StrInt32Map) SourceFinalize() error {
	if h == nil {
		return arena.ErrClosed
	}
	if h.raw.self != h {
		return arena.ErrConfig
	}
	if h.raw.executing {
		return ErrStrInt32MapExecuting
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
		if b != (memory.Block{}) {
			e = h.config.Runtime.Deallocate(&b)
			h.raw.table = b
			if e != nil {
				h.stopRuntimeDestructor(e)
				return e
			}
		}
		if e = h.publishRuntimeTable(memory.Block{}); e != nil {
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
func (h *StrInt32Map) stopRuntimeDestructor(e error) {
	if h.raw.destructorCause == nil {
		h.raw.destructorCause = e
	}
	if h.raw.sourceCause == nil {
		h.raw.sourceCause = e
	}
}

// CloseChecked is explicit owning abort. Token-zero receipts acknowledge real
// consumed storage; nonzero tokens still reach the real Runtime free leaf.
func (h *StrInt32Map) CloseChecked() error {
	if h == nil {
		return arena.ErrClosed
	}
	if h.raw.self != h {
		return arena.ErrConfig
	}
	if h.raw.executing {
		return ErrStrInt32MapExecuting
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
	for _, b := range []*memory.Block{&h.raw.table, &h.raw.object} {
		if *b != (memory.Block{}) {
			if e := h.config.Runtime.Deallocate(b); e != nil {
				return e
			}
		}
	}
	for h.raw.acquisitions != nil {
		a := h.raw.acquisitions
		if a.block != (memory.Block{}) {
			if _, e := h.config.Runtime.Size(a.block); e != nil {
				if !errors.Is(e, memory.ErrRuntimeUninitialized) {
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

func (h *StrInt32Map) stateAt(b memory.Block, i uint32) (uint32, error) {
	return h.config.Runtime.ReadUint32(b, uint64(i)*40+4)
}
func (h *StrInt32Map) keyAt(b memory.Block, i uint32) (StrKey, error) {
	var w [6]uint32
	for j := range w {
		v, e := h.config.Runtime.ReadUint32(b, uint64(i)*40+8+uint64(j)*4)
		if e != nil {
			return StrKey{}, e
		}
		w[j] = v
	}
	return StrKey{ref: arena.BytesRegionRef{Slot: arena.Slot(w[0]), Generation: arena.Generation(w[1])}, offset: w[2], length: w[3], origin: uint64(w[4]) | uint64(w[5])<<32}, nil
}
func (h *StrInt32Map) writeKeyAt(b memory.Block, i uint32, k StrKey) error {
	w := [6]uint32{uint32(k.ref.Slot), uint32(k.ref.Generation), k.offset, k.length, uint32(k.origin), uint32(k.origin >> 32)}
	for j, v := range w {
		if e := h.config.Runtime.WriteUint32(b, uint64(i)*40+8+uint64(j)*4, v); e != nil {
			return e
		}
	}
	return nil
}

// default_hash_entry member assignment: hash, state, then _key_data key/value.
// Source and target are the captured actual array pointers from copy/move.
func (h *StrInt32Map) copyRuntimeEntry(source memory.Block, i uint32, target memory.Block, j uint32, sourceOwner *StrInt32Map) error {
	for _, off := range []uint64{0, 4} {
		v, e := h.config.Runtime.ReadUint32(source, uint64(i)*40+off)
		if e != nil {
			return e
		}
		if e = h.config.Runtime.WriteUint32(target, uint64(j)*40+off, v); e != nil {
			return e
		}
	}
	key, e := h.keyAt(source, i)
	if e != nil {
		return e
	}
	if sourceOwner != nil {
		owner, e := sourceOwner.keyRegion(key)
		if e != nil {
			return e
		}
		if e = h.adopt(owner); e != nil {
			return e
		}
	}
	if e = h.writeKeyAt(target, j, key); e != nil {
		return e
	}
	value, e := h.config.Runtime.ReadUint32(source, uint64(i)*40+32)
	if e != nil {
		return e
	}
	return h.config.Runtime.WriteUint32(target, uint64(j)*40+32, value)
}

func (h *StrInt32Map) putKeyAt(b memory.Block, i uint32, k StrKey) error {
	if e := h.writeKeyAt(b, i, k); e != nil {
		return e
	}
	return h.config.Runtime.WriteUint32(b, uint64(i)*40+4, strHashUsed)
}
func (h *StrInt32Map) putEntryAt(b memory.Block, i uint32, k StrKey, v int32) error {
	if e := h.writeKeyAt(b, i, k); e != nil {
		return e
	}
	if e := h.config.Runtime.WriteUint32(b, uint64(i)*40+32, uint32(v)); e != nil {
		return e
	}
	return h.config.Runtime.WriteUint32(b, uint64(i)*40+4, strHashUsed)
}
func (h *StrInt32Map) valueAt(i uint32) (int32, error) {
	if h.config.Runtime == nil {
		p, e := mapLegacyEntryAt(h.table, i)
		if e != nil {
			return 0, e
		}
		return p.value, nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return 0, e
	}
	v, e := h.config.Runtime.ReadUint32(b, uint64(i)*40+32)
	return int32(v), e
}
func (h *StrInt32Map) putValueAt(i uint32, v int32) error {
	if h.config.Runtime == nil {
		p, e := mapLegacyEntryAt(h.table, i)
		if e != nil {
			return e
		}
		p.value = v
		return nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return e
	}
	return h.config.Runtime.WriteUint32(b, uint64(i)*40+32, uint32(v))
}

func (h *StrInt32Map) probeEntry(i uint32) (strInt32Entry, error) {
	if h.config.Runtime == nil {
		return h.table[i], nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return strInt32Entry{}, e
	}
	state, e := h.stateAt(b, i)
	if e != nil {
		return strInt32Entry{}, e
	}
	entry := strInt32Entry{state: state}
	if state == strHashUsed {
		entry.hash, e = h.config.Runtime.ReadUint32(b, uint64(i)*40)
	}
	return entry, e
}
func (h *StrInt32Map) entryKey(i uint32) (StrKey, error) {
	if h.config.Runtime == nil {
		return h.table[i].key, nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return StrKey{}, e
	}
	return h.keyAt(b, i)
}
func (h *StrInt32Map) readEntry(i uint32) (strInt32Entry, error) {
	entry, e := h.probeEntry(i)
	if e != nil {
		return entry, e
	}
	if entry.state == strHashUsed {
		entry.key, e = h.entryKey(i)
	}
	return entry, e
}
func (h *StrInt32Map) moveRuntimeTable(old memory.Block, oldcap uint32, next memory.Block, n uint32) error {
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
		hash, e := h.config.Runtime.ReadUint32(old, uint64(i)*40)
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
		if e = h.copyRuntimeEntry(old, i, next, j, nil); e != nil {
			return e
		}
	}
	return nil
}
func (h *StrInt32Map) rehashRuntime(n uint32) error {
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
	if e = h.publishRuntimeTable(memory.Block{}); e != nil {
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
func (h *StrInt32Map) prepareInsertRuntime() error {
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
func (h *StrInt32Map) insertRuntime(region *arena.BytesRegion, key StrKey, value int32, overwrite bool) (bool, StrInt32MapEntryRef, error) {
	if e := h.prepareInsertRuntime(); e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	hash, e := hashStrKey(region, key)
	if e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	cap, e := h.runtimeWord(8)
	if e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	b, e := h.runtimeTable()
	if e != nil {
		return false, StrInt32MapEntryRef{}, e
	}
	i := hash & (cap - 1)
	deleted := ^uint32(0)
	for step := uint32(0); step < cap; step++ {
		state, e := h.stateAt(b, i)
		if e != nil {
			return false, StrInt32MapEntryRef{}, e
		}
		switch state {
		case strHashUsed:
			cached, e := h.config.Runtime.ReadUint32(b, uint64(i)*40)
			if e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			if cached == hash {
				k, e := h.keyAt(b, i)
				if e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
				owner, e := h.keyRegion(k)
				if e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
				equal, e := equalStrKeys(owner, k, region, key)
				if e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
				if equal {
					if overwrite {
						if e = h.adopt(region); e != nil {
							return false, StrInt32MapEntryRef{}, e
						}
						if e = h.putEntryAt(b, i, key, value); e != nil {
							return false, StrInt32MapEntryRef{}, e
						}
					}
					return false, h.ref(i), nil
				}
			}
			if e = h.incrementCollision(); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
		default:
			on, err := h.assertionsEnabled()
			if err != nil {
				return false, StrInt32MapEntryRef{}, err
			}
			if on {
				actual, err := h.stateAt(b, i)
				if err != nil {
					return false, StrInt32MapEntryRef{}, err
				}
				if actual != strHashDeleted {
					line := int32(371)
					if !overwrite {
						line = 422
					}
					if err = h.assertionFailure(line, "curr->is_deleted()"); err != nil {
						return false, StrInt32MapEntryRef{}, err
					}
				}
			}
			deleted = i
			if e = h.incrementCollision(); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
		case strHashFree:
			if e = h.canBump(); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			if e = h.adopt(region); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			slot := i
			if deleted != ^uint32(0) {
				slot = deleted
				count, e := h.runtimeWord(16)
				if e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
				if e = h.writeRuntimeWord(16, count-1); e != nil {
					return false, StrInt32MapEntryRef{}, e
				}
			}
			if e = h.putEntryAt(b, slot, key, value); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			if e = h.config.Runtime.WriteUint32(b, uint64(slot)*40, hash); e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			size, e := h.runtimeWord(12)
			if e != nil {
				return false, StrInt32MapEntryRef{}, e
			}
			if e = h.writeRuntimeWord(12, size+1); e != nil {
				return false, StrInt32MapEntryRef{}, e
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
	return false, StrInt32MapEntryRef{}, h.unreachable(line)
}

func (h *StrInt32Map) deleteRuntimeTable() error {
	b, e := h.runtimeTable()
	if e != nil {
		return e
	}
	if b != (memory.Block{}) {
		e = h.config.Runtime.Deallocate(&b)
		h.raw.table = b
		if e != nil {
			h.raw.sourceCause = e
			return e
		}
	}
	return h.publishRuntimeTable(memory.Block{})
}
func (h *StrInt32Map) resetRuntime() error {
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
			if e = h.config.Runtime.WriteUint32(b, uint64(i)*40+4, strHashFree); e != nil {
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
func (h *StrInt32Map) finalizeRuntime() error {
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
func (h *StrInt32Map) removeRuntime(region *arena.BytesRegion, k StrKey) error {
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
		if e = h.config.Runtime.WriteUint32(b, uint64(i)*40+4, strHashFree); e != nil {
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
	if e = h.config.Runtime.WriteUint32(b, uint64(i)*40+4, strHashDeleted); e != nil {
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

func (h *StrInt32Map) capacityChecked() (uint32, error) {
	if h.config.Runtime != nil {
		return h.runtimeWord(8)
	}
	return h.raw.words[2], nil
}
func (h *StrInt32Map) sizeChecked() (uint32, error) {
	if h.config.Runtime != nil {
		return h.runtimeWord(12)
	}
	return h.size, nil
}
func (h *StrInt32Map) cloneRuntime(heap bool) (*StrInt32Map, error) {
	var e error
	root := uint64(reflect.TypeOf(StrInt32Map{}).Size())
	if e = h.config.Budget.Reserve(root); e != nil {
		return nil, e
	}
	out := &StrInt32Map{region: h.region, config: h.config, epoch: 1, rootBytes: root}
	out.raw.self = out
	out.raw.inline = !heap
	var a *strInt32MapAcquisition
	if heap {
		a, e = out.allocateRuntime(out.runtimeObjectExtent())
		if e != nil {
			return retireEmptyFailedMapRuntime(out, e)
		}
		out.raw.object = out.transferRuntime(a)
	}
	failed := func(cause error) (*StrInt32Map, error) {
		var d error
		if !out.raw.inline {
			d = out.config.Runtime.Deallocate(&out.raw.object)
		}
		if d != nil {
			out.raw.sourceCause = d
		}
		return retireEmptyFailedMapRuntime(out, errors.Join(cause, d))
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
		hash, e := h.config.Runtime.ReadUint32(old, uint64(i)*40)
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
		if e = out.copyRuntimeEntry(old, i, target, j, h); e != nil {
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
func (h *StrInt32Map) swapRuntime(other *StrInt32Map) error {
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

func (h *StrInt32Map) entryState(i uint32) (uint32, error) {
	if h.config.Runtime == nil {
		return h.table[i].state, nil
	}
	b, e := h.runtimeTable()
	if e != nil {
		return 0, e
	}
	return h.stateAt(b, i)
}
func (h *StrInt32Map) CheckInvariantChecked() (bool, error) {
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
			p, re := mapLegacyEntryAt(table, i)
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
func (h *StrInt32Map) HasRetainedRuntimeAcquisitions() bool {
	if h == nil || h.config.Runtime == nil || h.raw.self != h {
		panic(arena.ErrConfig)
	}
	return h.raw.acquisitions != nil || h.raw.lexical != nil
}

// The lexical source copy has an inline object header, and owns only its real
// table allocation. The retained context is acquired before the copy action;
// it names a stopped copy/destructor without rebuilding or scanning storage.
type strInt32MapScope struct {
	next     *strInt32MapScope
	copy     *StrInt32Map
	cause    error
	bytes    uint64
	complete bool
}

func (h *StrInt32Map) retireScope(frame *strInt32MapScope) {
	link := &h.raw.lexical
	for *link != frame {
		link = &(*link).next
	}
	*link = frame.next
	h.config.Budget.Release(frame.bytes)
}
func (h *StrInt32Map) intersectRuntime(src *StrInt32Map) (result error) {
	if e := h.bothOpen(src); e != nil {
		return e
	}
	if h == src {
		return nil
	}
	bytes := uint64(reflect.TypeOf(strInt32MapScope{}).Size())
	if e := h.config.Budget.Reserve(bytes); e != nil {
		return e
	}
	frame := &strInt32MapScope{bytes: bytes, next: h.raw.lexical}
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

func retireEmptyFailedMapRuntime(h *StrInt32Map, cause error) (*StrInt32Map, error) {
	if h.raw.object == (memory.Block{}) && h.raw.table == (memory.Block{}) && h.raw.acquisitions == nil && h.raw.lexical == nil && h.raw.sourceCause == nil {
		if e := h.CloseChecked(); e != nil {
			return h, errors.Join(cause, e)
		}
		return nil, cause
	}
	return h, cause
}
