// Copyright (c) 2006, 2007 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/small_object_allocator.h/.cpp; explicit scalar pointer codec.
package g3

import (
	"errors"
	"math"
	"reflect"
	"sync/atomic"

	"github.com/glycerine/g3/golang/arena"
)

var ErrSmallObjectOwner = errors.New("g3: invalid small-object allocator owner or object")
var ErrSmallObjectAssertion = errors.New("g3: small-object allocator source assertion")
var ErrSmallObjectUnreachable = errors.New("g3: unreachable small-object placement delete")
var ErrSmallObjectPending = errors.New("g3: pending small-object allocation or cleanup")
var ErrSmallObjectNonterminating = errors.New("g3: cyclic small-object source traversal")
var smallObjectOrigins atomic.Uint64

type utilSmallObjectProfile struct {
	PointerBits    uint8
	Debug, Windows bool
}
type utilSmallObjectConfig struct {
	Runtime              *utilMemoryRuntime
	Profile              utilSmallObjectProfile
	ID                   string
	IDPresent, IDNull    bool
	BorrowedID           *arena.BytesRegion
	IDRef                GlobalParamCString
	Diagnostics, Verbose *NumericTextStream
	Verbosity            uint32
}
type utilSmallObjectRef struct {
	owner, address   uint64
	slot, generation uint32
}

func (r utilSmallObjectRef) LogicalAddress() uint64 { return r.address }

const (
	smallObjectChunk  uint8 = 1
	smallObjectPooled uint8 = 2
	smallObjectDirect uint8 = 3
)

// This concrete locator is pointer-free. Source links live in actual payload;
// this ledger provides checked identity, physical ownership and source addresses.
type smallObjectLocation struct {
	raw                               utilMemoryBlock
	address, requested                uint64
	offset, extent, chunk, generation uint32
	kind                              uint8
	live, free, retired               bool
}
type utilSmallObjectAllocator struct {
	self                                *utilSmallObjectAllocator
	config                              utilSmallObjectConfig
	origin, header, allocSize, sequence uint64
	locations                           []smallObjectLocation
	chunks, free                        [64]uint32
	pending                             utilMemoryBlock
	pendingPartial                      bool
	pendingScratch                      [2]utilMemoryBlock
	cleanupClass, cleanupCurrent        uint32
	cleanupMode                         uint8
	closed                              bool
	destructorDone                      bool
	cleanupChunksNeeded                 bool
}
type smallObjectStats struct {
	OwnerBytes, LocationBytes, TotalBytes uint64
	Chunks, DirectObjects, PendingBlocks  uint32
}

func newUtilSmallObjectAllocator(c utilSmallObjectConfig) (*utilSmallObjectAllocator, error) {
	if c.Runtime == nil {
		return nil, ErrSmallObjectOwner
	}
	if e := c.Runtime.open(); e != nil {
		return nil, e
	}
	if c.Profile.PointerBits == 0 {
		c.Profile.PointerBits = c.Runtime.config.SizeTBits
	}
	if (c.Profile.PointerBits != 32 && c.Profile.PointerBits != 64) || c.Profile.PointerBits != c.Runtime.config.SizeTBits {
		return nil, arena.ErrConfig
	}
	if e := arena.ValidateRecordType(reflect.TypeOf(smallObjectLocation{})); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(utilSmallObjectAllocator{}).Size())
	if e := c.Runtime.config.Budget.Reserve(n); e != nil {
		return nil, e
	}
	var id uint64
	for {
		old := smallObjectOrigins.Load()
		if old == math.MaxUint64 {
			c.Runtime.config.Budget.Release(n)
			return nil, arena.ErrSlotLimit
		}
		if smallObjectOrigins.CompareAndSwap(old, old+1) {
			id = old + 1
			break
		}
	}
	if c.ID == "" && !c.IDPresent {
		c.ID = "unknown"
	}
	a := &utilSmallObjectAllocator{config: c, origin: id, header: n}
	a.self = a
	return a, nil
}
func (a *utilSmallObjectAllocator) open() error {
	if a == nil || a.self != a {
		return ErrSmallObjectOwner
	}
	if a.closed || a.destructorDone {
		return arena.ErrClosed
	}
	return a.config.Runtime.open()
}
func (a *utilSmallObjectAllocator) pointerBytes() uint64 {
	return uint64(a.config.Profile.PointerBits / 8)
}
func (a *utilSmallObjectAllocator) slots() uint32 { return uint32(256 / a.pointerBytes()) }
func (a *utilSmallObjectAllocator) bypass() bool {
	return a.config.Profile.Debug && !a.config.Profile.Windows
}
func (a *utilSmallObjectAllocator) counter(n uint64) uint64 {
	if a.config.Profile.PointerBits == 32 {
		return uint64(uint32(n))
	}
	return n
}
func (a *utilSmallObjectAllocator) requireSize(n uint64) error {
	if a.config.Profile.PointerBits == 32 && n > math.MaxUint32 {
		return arena.ErrBounds
	}
	return nil
}
func (a *utilSmallObjectAllocator) reserveLocations(extra uint32) error {
	available := uint64(cap(a.locations) - len(a.locations))
	for i := range a.locations {
		r := &a.locations[i]
		if !r.live && !r.free && !r.retired && r.raw == (utilMemoryBlock{}) && r.generation < math.MaxUint32 {
			available++
		}
	}
	if available >= uint64(extra) {
		return nil
	}
	missing := uint64(extra) - available
	needed := uint64(cap(a.locations)) + missing
	if needed > math.MaxUint32 {
		return arena.ErrSlotLimit
	}
	n := uint64(cap(a.locations))
	if n == 0 {
		n = 2
	}
	for n < needed {
		next := (3*n + 1) >> 1
		if next <= n || next > math.MaxUint32 {
			return arena.ErrSlotLimit
		}
		n = next
	}
	stride := uint64(reflect.TypeOf(smallObjectLocation{}).Size())
	if n > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	bytes := n * stride
	if e := a.config.Runtime.config.Budget.Reserve(bytes); e != nil {
		return e
	}
	next := make([]smallObjectLocation, len(a.locations), int(n))
	copy(next, a.locations)
	a.config.Runtime.config.Budget.Release(uint64(cap(a.locations)) * stride)
	a.locations = next
	return nil
}
func (a *utilSmallObjectAllocator) installLocation(r smallObjectLocation) uint32 {
	for i := range a.locations {
		old := &a.locations[i]
		if !old.live && !old.free && !old.retired && old.raw == (utilMemoryBlock{}) && old.generation < math.MaxUint32 {
			r.generation = old.generation + 1
			*old = r
			return uint32(i + 1)
		}
	}
	r.generation = 1
	a.locations = append(a.locations, r)
	return uint32(len(a.locations))
}
func (a *utilSmallObjectAllocator) location(id uint32) (*smallObjectLocation, error) {
	if id == 0 || uint64(id) > uint64(len(a.locations)) {
		return nil, ErrSmallObjectOwner
	}
	return &a.locations[id-1], nil
}
func (a *utilSmallObjectAllocator) object(r utilSmallObjectRef) (*smallObjectLocation, error) {
	if r.owner != a.origin {
		return nil, ErrSmallObjectOwner
	}
	loc, e := a.location(r.slot)
	if e != nil {
		return nil, e
	}
	if !loc.live || loc.generation != r.generation || (loc.kind != smallObjectPooled && loc.kind != smallObjectDirect) {
		return nil, ErrSmallObjectOwner
	}
	return loc, nil
}
func (a *utilSmallObjectAllocator) ref(id uint32) utilSmallObjectRef {
	r := a.locations[id-1]
	return utilSmallObjectRef{a.origin, r.address, id, r.generation}
}
func (a *utilSmallObjectAllocator) nextAddress() (uint64, error) {
	if a.sequence >= math.MaxUint64/8192 {
		return 0, arena.ErrSlotLimit
	}
	a.sequence++
	return a.sequence * 8192, nil
}
func smallObjectReadWord(rt *utilMemoryRuntime, h utilMemoryBlock, offset, width uint64) (uint64, error) {
	var word uint64
	for i := uint64(0); i < width; i++ {
		v, e := rt.Read(h, offset+i)
		if e != nil {
			return 0, e
		}
		word |= uint64(v) << (8 * i)
	}
	return word, nil
}
func smallObjectWriteWord(rt *utilMemoryRuntime, h utilMemoryBlock, offset, width, v uint64) error {
	for i := uint64(0); i < width; i++ {
		if e := rt.Write(h, offset+i, byte(v>>(8*i))); e != nil {
			return e
		}
	}
	return nil
}
func (a *utilSmallObjectAllocator) rememberPending(h utilMemoryBlock) {
	if h == (utilMemoryBlock{}) {
		return
	}
	a.pending = h
	_, e := a.config.Runtime.Size(h)
	a.pendingPartial = e != nil
}
func (a *utilSmallObjectAllocator) Allocate(n uint64) (utilSmallObjectRef, error) {
	if e := a.open(); e != nil {
		return utilSmallObjectRef{}, e
	}
	if n == 0 {
		return utilSmallObjectRef{}, nil
	}
	if e := a.requireSize(n); e != nil {
		return utilSmallObjectRef{}, e
	}
	if a.hasPending() || a.cleanupMode != 0 || a.cleanupChunksNeeded {
		return utilSmallObjectRef{}, ErrSmallObjectPending
	}
	if !a.bypass() {
		a.allocSize = a.counter(a.allocSize + n)
	}
	width := a.pointerBytes()
	if a.bypass() || n >= 256-width {
		if e := a.reserveLocations(1); e != nil {
			return utilSmallObjectRef{}, e
		}
		address, e := a.nextAddress()
		if e != nil {
			return utilSmallObjectRef{}, e
		}
		h, e := a.config.Runtime.Allocate(n)
		if e != nil {
			a.rememberPending(h)
			return utilSmallObjectRef{}, e
		}
		id := a.installLocation(smallObjectLocation{raw: h, address: address, requested: n, kind: smallObjectDirect, live: true})
		return a.ref(id), nil
	}
	slot := uint32((n + width - 1) / width)
	if a.config.Profile.Debug && (slot == 0 || slot >= a.slots()) {
		return utilSmallObjectRef{}, ErrSmallObjectAssertion
	}
	if slot >= a.slots() {
		return utilSmallObjectRef{}, arena.ErrBounds
	}
	if a.free[slot] != 0 {
		id := a.free[slot]
		loc, e := a.location(id)
		if e != nil {
			return utilSmallObjectRef{}, e
		}
		next, e := smallObjectReadWord(a.config.Runtime, loc.raw, uint64(loc.offset), width)
		if e != nil {
			return utilSmallObjectRef{}, e
		}
		if next > math.MaxUint32 {
			return utilSmallObjectRef{}, ErrSmallObjectOwner
		}
		a.free[slot] = uint32(next)
		if loc.generation == math.MaxUint32 {
			loc.retired = true
			loc.free = false
			return utilSmallObjectRef{}, arena.ErrSlotLimit
		}
		loc.generation++
		loc.free = false
		loc.live = true
		loc.requested = n
		loc.extent = slot * uint32(width)
		return a.ref(id), nil
	}
	if e := a.reserveLocations(1); e != nil {
		return utilSmallObjectRef{}, e
	}
	size := uint64(slot) * width
	if a.chunks[slot] != 0 {
		chunk, e := a.location(a.chunks[slot])
		if e != nil {
			return utilSmallObjectRef{}, e
		}
		curr, e := smallObjectReadWord(a.config.Runtime, chunk.raw, width, width)
		if e != nil {
			return utilSmallObjectRef{}, e
		}
		if curr > 8192 {
			return utilSmallObjectRef{}, arena.ErrBounds
		}
		if curr+size < 8192 {
			if e = smallObjectWriteWord(a.config.Runtime, chunk.raw, width, width, curr+size); e != nil {
				return utilSmallObjectRef{}, e
			}
			id := a.installLocation(smallObjectLocation{raw: chunk.raw, address: chunk.address + curr, requested: n, offset: uint32(curr), extent: uint32(size), chunk: a.chunks[slot], kind: smallObjectPooled, live: true})
			return a.ref(id), nil
		}
	}
	if e := a.reserveLocations(2); e != nil {
		return utilSmallObjectRef{}, e
	}
	address, e := a.nextAddress()
	if e != nil {
		return utilSmallObjectRef{}, e
	}
	h, e := a.config.Runtime.Allocate(8192)
	if e != nil {
		a.rememberPending(h)
		return utilSmallObjectRef{}, e
	}
	// ctor sets curr first; source next and chain publication follow.
	if e = smallObjectWriteWord(a.config.Runtime, h, width, width, 2*width); e != nil {
		a.rememberPending(h)
		return utilSmallObjectRef{}, e
	}
	if e = smallObjectWriteWord(a.config.Runtime, h, 0, width, uint64(a.chunks[slot])); e != nil {
		a.rememberPending(h)
		return utilSmallObjectRef{}, e
	}
	chunkID := a.installLocation(smallObjectLocation{raw: h, address: address, kind: smallObjectChunk, live: true})
	a.chunks[slot] = chunkID
	if e = smallObjectWriteWord(a.config.Runtime, h, width, width, 2*width+size); e != nil {
		return utilSmallObjectRef{}, e
	}
	id := a.installLocation(smallObjectLocation{raw: h, address: address + 2*width, requested: n, offset: uint32(2 * width), extent: uint32(size), chunk: chunkID, kind: smallObjectPooled, live: true})
	return a.ref(id), nil
}
func (a *utilSmallObjectAllocator) Deallocate(n uint64, r utilSmallObjectRef) error {
	if e := a.open(); e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	if e := a.requireSize(n); e != nil {
		return e
	}
	if a.bypass() {
		loc, e := a.object(r)
		if e != nil {
			return e
		}
		h := loc.raw
		e = a.config.Runtime.Deallocate(&h)
		loc.raw = h
		if h == (utilMemoryBlock{}) {
			loc.live = false
		}
		return e
	}
	if a.config.Profile.Debug && (a.allocSize < n || r.slot == 0) {
		return ErrSmallObjectAssertion
	}
	a.allocSize = a.counter(a.allocSize - n)
	// Pooled free does not destroy the stored scalar object. A same-generation
	// duplicate retains the defined source subtraction/intrusive relink prefix.
	loc, e := a.location(r.slot)
	if e != nil {
		return e
	}
	if r.owner != a.origin || loc.generation != r.generation ||
		(!loc.live && !(loc.kind == smallObjectPooled && loc.free)) ||
		(loc.kind != smallObjectPooled && loc.kind != smallObjectDirect) {
		return ErrSmallObjectOwner
	}
	if n >= 256-a.pointerBytes() {
		if loc.offset != 0 {
			return arena.ErrBounds
		} // actual invalid interior raw header action
		h := loc.raw
		e = a.config.Runtime.Deallocate(&h)
		loc.raw = h
		if h == (utilMemoryBlock{}) {
			loc.live = false
		}
		return e
	}
	slot := uint32((n + a.pointerBytes() - 1) / a.pointerBytes())
	if a.config.Profile.Debug && (slot == 0 || slot >= a.slots()) {
		return ErrSmallObjectAssertion
	}
	if slot == 0 || slot >= a.slots() {
		return arena.ErrBounds
	}
	if e = smallObjectWriteWord(a.config.Runtime, loc.raw, uint64(loc.offset), a.pointerBytes(), uint64(a.free[slot])); e != nil {
		return e
	}
	a.free[slot] = r.slot
	loc.live = false
	loc.free = true
	return nil
}
func (a *utilSmallObjectAllocator) RawLocation(r utilSmallObjectRef) (utilMemoryBlock, uint64, error) {
	if e := a.open(); e != nil {
		return utilMemoryBlock{}, 0, e
	}
	loc, e := a.object(r)
	if e != nil {
		return utilMemoryBlock{}, 0, e
	}
	return loc.raw, uint64(loc.offset), nil
}
func (a *utilSmallObjectAllocator) Read(r utilSmallObjectRef, i uint64) (byte, error) {
	if e := a.open(); e != nil {
		return 0, e
	}
	loc, e := a.object(r)
	if e != nil {
		return 0, e
	}
	if i >= loc.requested {
		return 0, arena.ErrBounds
	}
	return a.config.Runtime.Read(loc.raw, uint64(loc.offset)+i)
}
func (a *utilSmallObjectAllocator) Write(r utilSmallObjectRef, i uint64, v byte) error {
	if e := a.open(); e != nil {
		return e
	}
	loc, e := a.object(r)
	if e != nil {
		return e
	}
	if i >= loc.requested {
		return arena.ErrBounds
	}
	return a.config.Runtime.Write(loc.raw, uint64(loc.offset)+i, v)
}
func (a *utilSmallObjectAllocator) AllocationSize() (uint64, error) {
	if e := a.open(); e != nil {
		return 0, e
	}
	return a.allocSize, nil
}
func (a *utilSmallObjectAllocator) freeMeasure(waste bool) (uint64, error) {
	if e := a.open(); e != nil {
		return 0, e
	}
	var result uint64
	for slot := uint32(0); slot < a.slots(); slot++ {
		id := a.free[slot]
		steps := 0
		for id != 0 {
			if steps >= len(a.locations) {
				return result, ErrSmallObjectNonterminating
			}
			steps++
			loc, e := a.location(id)
			if e != nil {
				return result, e
			}
			if waste {
				result = a.counter(result + uint64(slot)*a.pointerBytes())
			} else {
				result = a.counter(result + 1)
			}
			next, e := smallObjectReadWord(a.config.Runtime, loc.raw, uint64(loc.offset), a.pointerBytes())
			if e != nil {
				return result, e
			}
			if next > math.MaxUint32 {
				return result, ErrSmallObjectOwner
			}
			id = uint32(next)
		}
	}
	return result, nil
}
func (a *utilSmallObjectAllocator) WastedSize() (uint64, error)     { return a.freeMeasure(true) }
func (a *utilSmallObjectAllocator) NumFreeObjects() (uint64, error) { return a.freeMeasure(false) }
func (a *utilSmallObjectAllocator) invalidateChunk(id uint32) {
	for i := range a.locations {
		r := &a.locations[i]
		if uint32(i+1) == id || r.kind == smallObjectPooled && r.chunk == id {
			r.live = false
			r.free = false
			r.raw = utilMemoryBlock{}
			if r.generation == math.MaxUint32 {
				r.retired = true
			}
		}
	}
}

// The cold cursor retains real remaining roots if a raw deallocation transport
// reports an error after physical free. Retrying cleanup is a Go continuation.
func (a *utilSmallObjectAllocator) freeChunks(mode uint8) error {
	if a.cleanupChunksNeeded {
		return ErrSmallObjectPending
	}
	if a.cleanupMode != 0 && a.cleanupMode != mode {
		return ErrSmallObjectPending
	}
	if a.cleanupMode == 0 {
		a.cleanupMode = mode
		a.cleanupClass = 0
		a.cleanupCurrent = a.chunks[0]
	}
	for a.cleanupClass < a.slots() {
		if a.cleanupCurrent == 0 {
			a.chunks[a.cleanupClass] = 0
			a.free[a.cleanupClass] = 0
			a.cleanupClass++
			if a.cleanupClass < a.slots() {
				a.cleanupCurrent = a.chunks[a.cleanupClass]
			}
			continue
		}
		id := a.cleanupCurrent
		loc, e := a.location(id)
		if e != nil {
			return e
		}
		next, e := smallObjectReadWord(a.config.Runtime, loc.raw, 0, a.pointerBytes())
		if e != nil {
			return e
		}
		if next > math.MaxUint32 {
			return ErrSmallObjectOwner
		}
		h := loc.raw
		e = a.config.Runtime.Deallocate(&h)
		if h == (utilMemoryBlock{}) {
			a.invalidateChunk(id)
			a.cleanupCurrent = uint32(next)
		}
		if e != nil {
			return e
		}
	}
	a.cleanupMode = 0
	return nil
}
func (a *utilSmallObjectAllocator) Reset() error {
	if e := a.open(); e != nil {
		return e
	}
	if e := a.freeChunks(1); e != nil {
		return e
	}
	a.allocSize = 0
	return nil
}
func (a *utilSmallObjectAllocator) hasPending() bool {
	return a.pending != (utilMemoryBlock{}) || a.pendingScratch[0] != (utilMemoryBlock{}) || a.pendingScratch[1] != (utilMemoryBlock{})
}
func (a *utilSmallObjectAllocator) retainScratch(h utilMemoryBlock) {
	for i := range a.pendingScratch {
		if a.pendingScratch[i] == (utilMemoryBlock{}) {
			a.pendingScratch[i] = h
			return
		}
	}
	panic("small-object lexical root overflow") // exactly two lexical vectors
}
func (a *utilSmallObjectAllocator) CleanupPending() error {
	if a == nil || a.self != a {
		return ErrSmallObjectOwner
	}
	if a.pending != (utilMemoryBlock{}) {
		var e error
		if a.pendingPartial {
			e = a.config.Runtime.CleanupPartial(&a.pending)
		} else {
			e = a.config.Runtime.Deallocate(&a.pending)
		}
		if a.pending == (utilMemoryBlock{}) {
			a.pendingPartial = false
		}
		if e != nil {
			return e
		}
	}
	for i := range a.pendingScratch {
		if a.pendingScratch[i] != (utilMemoryBlock{}) {
			if e := a.config.Runtime.Deallocate(&a.pendingScratch[i]); e != nil {
				return e
			}
		}
	}
	return nil
}
func (a *utilSmallObjectAllocator) Close() error {
	if a == nil || a.self != a {
		return ErrSmallObjectOwner
	}
	if a.closed {
		return nil
	}
	if e := a.config.Runtime.open(); e != nil {
		return e
	}
	var e error
	if !a.destructorDone {
		if e = a.freeChunks(2); e != nil {
			return e
		}
		e = a.diagnostic()
		// Completed destructor actions, including any emitted diagnostic prefix,
		// are never replayed by owner-finalization retries.
		a.destructorDone = true
	}
	if a.hasPending() {
		if e != nil {
			return e
		}
		return ErrSmallObjectPending
	}
	a.config.Runtime.config.Budget.Release(a.header + uint64(cap(a.locations))*uint64(reflect.TypeOf(smallObjectLocation{}).Size()))
	a.locations = nil
	a.header = 0
	a.closed = true
	return e
}
func (a *utilSmallObjectAllocator) Stats() smallObjectStats {
	if a == nil || a.self != a {
		return smallObjectStats{}
	}
	s := smallObjectStats{OwnerBytes: a.header, LocationBytes: uint64(cap(a.locations)) * uint64(reflect.TypeOf(smallObjectLocation{}).Size())}
	s.TotalBytes = s.OwnerBytes + s.LocationBytes
	for _, r := range a.locations {
		if r.live && r.kind == smallObjectChunk {
			s.Chunks++
		}
		if r.live && r.kind == smallObjectDirect {
			s.DirectObjects++
		}
	}
	if a.pending != (utilMemoryBlock{}) {
		s.PendingBlocks = 1
	}
	for _, h := range a.pendingScratch {
		if h != (utilMemoryBlock{}) {
			s.PendingBlocks++
		}
	}
	return s
}
func (a *utilSmallObjectAllocator) PlacementDelete(utilSmallObjectRef) error {
	return ErrSmallObjectUnreachable
}
func (a *utilSmallObjectAllocator) PlacementDeleteArray(utilSmallObjectRef) error {
	return ErrSmallObjectUnreachable
}

// CleanupChunks is an explicit Go continuation after an interrupted source
// Consolidate. Source heads are left untouched at failure; retained cold
// locators keep every actual owning root accessible without raw traversal.
func (a *utilSmallObjectAllocator) CleanupChunks() error {
	if e := a.open(); e != nil {
		return e
	}
	if !a.cleanupChunksNeeded {
		return nil
	}
	for i := range a.locations {
		loc := &a.locations[i]
		if loc.kind != smallObjectChunk || !loc.live {
			continue
		}
		h := loc.raw
		e := a.config.Runtime.Deallocate(&h)
		if h == (utilMemoryBlock{}) {
			a.invalidateChunk(uint32(i + 1))
		}
		if e != nil {
			return e
		}
	}
	a.chunks = [64]uint32{}
	a.free = [64]uint32{}
	a.cleanupChunksNeeded = false
	return nil
}
