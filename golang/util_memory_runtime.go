// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
	"sync/atomic"
)

var ErrMemoryRuntimeOwner = errors.New("g3: invalid raw memory owner or block")
var ErrMemoryRuntimeLive = errors.New("g3: live raw memory blocks")
var ErrMemoryRuntimeUninitialized = errors.New("g3: uninitialized raw memory byte")
var memoryRuntimeOrigins atomic.Uint64

type utilMemoryRuntimeConfig struct {
	Budget      *arena.Budget
	ThreadLocal bool
	SizeTBits   uint8
}
type utilMemoryBlock struct {
	origin           uint64
	slot, generation uint32
}

// Each entry owns one actual malloc backing, plus checked initialized-byte bits.
// Both slice headers are cold allocation metadata, never boxed hot records.
type memoryRuntimeBlock struct {
	data        []byte
	initialized []uint64
	generation  uint32
	live        bool
}
type utilMemoryRuntime struct {
	self           *utilMemoryRuntime
	config         utilMemoryRuntimeConfig
	origin, header uint64
	accounting     utilMemoryAccounting
	lane           utilMemoryLaneAccounting
	blocks         []memoryRuntimeBlock
	live           uint32
	closed         bool
}
type memoryRuntimeState struct {
	GlobalSize, LocalSize, GlobalCount, LocalCount, Peak int64
	OutOfMemory                                          bool
}
type memoryRuntimeStats struct {
	OwnerBytes, DirectoryBytes, PhysicalBytes, InitializationBytes, TotalBytes uint64
	LiveBlocks                                                                 uint32
}

// This is the explicit owner of the real raw allocate/reallocate/deallocate
// paths, not generated memory::initialize/finalize or runtime-wide typed hooks.
// One confined lane implements the selected TLS/non-TLS source body. Owners
// and their Budget require caller confinement; process-global wiring stays open.
func newUtilMemoryRuntime(c utilMemoryRuntimeConfig) (*utilMemoryRuntime, error) {
	if c.SizeTBits == 0 {
		c.SizeTBits = 64
	}
	if c.SizeTBits != 32 && c.SizeTBits != 64 {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(utilMemoryRuntime{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	var id uint64
	for {
		v := memoryRuntimeOrigins.Load()
		if v == math.MaxUint64 {
			c.Budget.Release(n)
			return nil, arena.ErrSlotLimit
		}
		if memoryRuntimeOrigins.CompareAndSwap(v, v+1) {
			id = v + 1
			break
		}
	}
	m := &utilMemoryRuntime{config: c, origin: id, header: n}
	m.self = m
	return m, nil
}
func (m *utilMemoryRuntime) open() error {
	if m == nil || m.self != m {
		return ErrMemoryRuntimeOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *utilMemoryRuntime) block(h utilMemoryBlock) (*memoryRuntimeBlock, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	if h.origin != m.origin || h.slot == 0 || uint64(h.slot) > uint64(len(m.blocks)) {
		return nil, ErrMemoryRuntimeOwner
	}
	b := &m.blocks[h.slot-1]
	if !b.live || b.generation != h.generation {
		return nil, ErrMemoryRuntimeOwner
	}
	return b, nil
}
func (m *utilMemoryRuntime) extent(n uint64) (uint64, error) {
	if m.config.SizeTBits == 32 {
		if n > math.MaxUint32 {
			return 0, arena.ErrBounds
		}
		return uint64(uint32(n) + 4), nil
	}
	return n + 8, nil
}
func (m *utilMemoryRuntime) transport(r memoryAccountingResult) error {
	switch r {
	case memoryAccountingOK:
		return nil
	case memoryAccountingOOM:
		return Z3Error{ErrorMemout}
	case memoryAccountingCountLimit:
		return Z3Error{ErrorAllocExceeded}
	default:
		return arena.ErrBounds
	}
}
func (m *utilMemoryRuntime) physicalFailure() error {
	m.accounting.outOfMemory = true
	return Z3Error{ErrorMemout}
}
func (m *utilMemoryRuntime) slot() (uint32, error) {
	for i := range m.blocks {
		b := &m.blocks[i]
		if !b.live && b.generation != math.MaxUint32 {
			return uint32(i + 1), nil
		}
	}
	if uint64(len(m.blocks)) >= math.MaxUint32 {
		return 0, arena.ErrSlotLimit
	}
	if len(m.blocks) == cap(m.blocks) {
		n := cap(m.blocks) * 2
		if n == 0 {
			n = 2
		}
		stride := uint64(reflect.TypeOf(memoryRuntimeBlock{}).Size())
		if n < cap(m.blocks) || uint64(n) > uint64(^uint(0)>>1)/stride {
			return 0, arena.ErrBounds
		}
		if e := m.config.Budget.Reserve(uint64(n) * stride); e != nil {
			return 0, m.physicalFailure()
		}
		next := make([]memoryRuntimeBlock, len(m.blocks), n)
		copy(next, m.blocks)
		m.config.Budget.Release(uint64(cap(m.blocks)) * stride)
		m.blocks = next
	}
	m.blocks = append(m.blocks, memoryRuntimeBlock{generation: 1})
	return uint32(len(m.blocks)), nil
}
func (m *utilMemoryRuntime) storage(n uint64) ([]byte, []uint64, error) {
	// Bound actual Go allocations separately from source SIZE_T arithmetic.
	// Too-short wrapped extents remain owned until the actual header-write guard.
	if n > uint64(^uint(0)>>1) {
		return nil, nil, arena.ErrBounds
	}
	words := (n + 63) / 64
	if words > math.MaxUint64/8 || n > math.MaxUint64-words*8 {
		return nil, nil, arena.ErrBounds
	}
	if e := m.config.Budget.Reserve(n + words*8); e != nil {
		return nil, nil, m.physicalFailure()
	}
	return make([]byte, int(n)), make([]uint64, int(words)), nil
}
func (m *utilMemoryRuntime) headerWrite(b *memoryRuntimeBlock, n uint64) error {
	if len(b.data) < int(m.config.SizeTBits/8) {
		return arena.ErrBounds
	}
	for i := uint8(0); i < m.config.SizeTBits/8; i++ {
		b.data[i] = byte(n >> (8 * i))
		b.initialized[0] |= uint64(1) << i
	}
	return nil
}
func (m *utilMemoryRuntime) headerRead(b *memoryRuntimeBlock) (uint64, error) {
	if len(b.data) < int(m.config.SizeTBits/8) {
		return 0, arena.ErrBounds
	}
	var n uint64
	for i := uint8(0); i < m.config.SizeTBits/8; i++ {
		n |= uint64(b.data[i]) << (8 * i)
	}
	return n, nil
}
func (m *utilMemoryRuntime) Allocate(n uint64) (utilMemoryBlock, error) {
	if e := m.open(); e != nil {
		return utilMemoryBlock{}, e
	}
	extent, e := m.extent(n)
	if e != nil {
		return utilMemoryBlock{}, e
	}
	if !m.config.ThreadLocal {
		if e = m.transport(m.accounting.accountExtentAllocation(&m.lane, extent, m.config.SizeTBits, false)); e != nil {
			return utilMemoryBlock{}, e
		}
	}
	slot, e := m.slot()
	if e != nil {
		return utilMemoryBlock{}, e
	}
	data, init, e := m.storage(extent)
	if e != nil {
		return utilMemoryBlock{}, e
	}
	b := &m.blocks[slot-1]
	b.data = data
	b.initialized = init
	b.live = true
	m.live++
	h := utilMemoryBlock{m.origin, slot, b.generation}
	if e = m.headerWrite(b, extent); e != nil {
		return h, e
	}
	if m.config.ThreadLocal {
		e = m.transport(m.accounting.accountExtentAllocation(&m.lane, extent, m.config.SizeTBits, true))
	}
	// TLS limit checks follow real allocation/header/counter publication. Return
	// the owned completed block even with an error; caller must Deallocate it.
	return h, e
}
func (m *utilMemoryRuntime) Reallocate(h *utilMemoryBlock, n uint64) error {
	if h == nil {
		return ErrMemoryRuntimeOwner
	}
	b, e := m.block(*h)
	if e != nil {
		return e
	}
	// Exhausted checked pointer generations cannot publish another realloc
	// binding. This explicit token-domain guard precedes source actions.
	if b.generation == math.MaxUint32 {
		return arena.ErrSlotLimit
	}
	old, e := m.headerRead(b)
	if e != nil {
		return e
	}
	extent, e := m.extent(n)
	if e != nil {
		return e
	}
	delta := extent - old
	if m.config.SizeTBits == 32 {
		delta = uint64(uint32(extent) - uint32(old))
	}
	if e = m.transport(m.accounting.accountExtentAllocation(&m.lane, delta, m.config.SizeTBits, m.config.ThreadLocal)); e != nil {
		return e
	}
	data, init, e := m.storage(extent)
	if e != nil {
		return e
	}
	copy(data, b.data)
	copy(init, b.initialized)
	// Byte-validity of newly grown storage must not copy unused high bits.
	if extent > old && old%64 != 0 {
		init[old/64] &= (uint64(1) << (old % 64)) - 1
	}
	m.config.Budget.Release(uint64(len(b.data)) + uint64(len(b.initialized))*8)
	b.data = data
	b.initialized = init
	// The checked descriptor tracks malloc ownership, not a raw pointer address.
	// A successful realloc invalidates old pointer tokens, including equal sizes.
	b.generation++
	h.generation = b.generation
	return m.headerWrite(b, extent)
}
func (m *utilMemoryRuntime) Deallocate(h *utilMemoryBlock) error {
	if h == nil {
		return ErrMemoryRuntimeOwner
	}
	b, e := m.block(*h)
	if e != nil {
		return e
	}
	n, e := m.headerRead(b)
	if e != nil {
		return e
	}
	if e = m.transport(m.accounting.beginExtentDeallocation(&m.lane, n, m.config.SizeTBits, m.config.ThreadLocal)); e != nil {
		return e
	}
	m.config.Budget.Release(uint64(len(b.data)) + uint64(len(b.initialized))*8)
	b.data = nil
	b.initialized = nil
	b.live = false
	m.live--
	if b.generation != math.MaxUint32 {
		b.generation++
	}
	*h = utilMemoryBlock{}
	return m.transport(m.accounting.finishDeallocation(&m.lane, m.config.ThreadLocal))
}
func (m *utilMemoryRuntime) Size(h utilMemoryBlock) (uint64, error) {
	b, e := m.block(h)
	if e != nil {
		return 0, e
	}
	n, e := m.headerRead(b)
	if e != nil {
		return 0, e
	}
	return n - uint64(m.config.SizeTBits/8), nil
}
func (m *utilMemoryRuntime) Read(h utilMemoryBlock, i uint64) (byte, error) {
	b, e := m.block(h)
	if e != nil {
		return 0, e
	}
	n, e := m.headerRead(b)
	if e != nil {
		return 0, e
	}
	offset := uint64(m.config.SizeTBits / 8)
	if i >= n-offset {
		return 0, arena.ErrBounds
	}
	i += offset
	if b.initialized[i/64]&(uint64(1)<<(i%64)) == 0 {
		return 0, ErrMemoryRuntimeUninitialized
	}
	return b.data[i], nil
}
func (m *utilMemoryRuntime) Write(h utilMemoryBlock, i uint64, v byte) error {
	b, e := m.block(h)
	if e != nil {
		return e
	}
	n, e := m.headerRead(b)
	if e != nil {
		return e
	}
	offset := uint64(m.config.SizeTBits / 8)
	if i >= n-offset {
		return arena.ErrBounds
	}
	i += offset
	b.data[i] = v
	b.initialized[i/64] |= uint64(1) << (i % 64)
	return nil
}
func (m *utilMemoryRuntime) SetMaxSize(n uint64) error {
	if e := m.open(); e != nil {
		return e
	}
	if m.config.SizeTBits == 32 && n > math.MaxUint32 {
		return arena.ErrBounds
	}
	m.accounting.maxSize = int64(n)
	return nil
}
func (m *utilMemoryRuntime) SetMaxCount(n uint64) error {
	if e := m.open(); e != nil {
		return e
	}
	if m.config.SizeTBits == 32 && n > math.MaxUint32 {
		return arena.ErrBounds
	}
	m.accounting.maxCount = int64(n)
	return nil
}
func (m *utilMemoryRuntime) SetWatermark(n uint64) error {
	if e := m.open(); e != nil {
		return e
	}
	if m.config.SizeTBits == 32 && n > math.MaxUint32 {
		return arena.ErrBounds
	}
	m.accounting.watermark = int64(n)
	return nil
}
func (m *utilMemoryRuntime) AboveWatermark() (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	return m.accounting.aboveWatermark(), nil
}
func (m *utilMemoryRuntime) State() memoryRuntimeState {
	if m == nil || m.self != m || m.closed {
		return memoryRuntimeState{}
	}
	a := &m.accounting
	l := &m.lane
	return memoryRuntimeState{a.size, l.size, a.count, l.count, a.peak, a.outOfMemory}
}
func (m *utilMemoryRuntime) Stats() memoryRuntimeStats {
	if m == nil || m.self != m || m.closed {
		return memoryRuntimeStats{}
	}
	s := memoryRuntimeStats{OwnerBytes: m.header, DirectoryBytes: uint64(cap(m.blocks)) * uint64(reflect.TypeOf(memoryRuntimeBlock{}).Size()), LiveBlocks: m.live}
	for i := range m.blocks {
		b := &m.blocks[i]
		if b.live {
			s.PhysicalBytes += uint64(len(b.data))
			s.InitializationBytes += uint64(len(b.initialized)) * 8
		}
	}
	s.TotalBytes = s.OwnerBytes + s.DirectoryBytes + s.PhysicalBytes + s.InitializationBytes
	return s
}
func (m *utilMemoryRuntime) Close() error {
	if m == nil || m.self != m {
		return ErrMemoryRuntimeOwner
	}
	if m.closed {
		return nil
	}
	if m.live != 0 {
		return ErrMemoryRuntimeLive
	}
	m.config.Budget.Release(m.header + uint64(cap(m.blocks))*uint64(reflect.TypeOf(memoryRuntimeBlock{}).Size()))
	m.blocks = nil
	m.header = 0
	m.closed = true
	return nil
}

func (m *utilMemoryRuntime) AllocationSize() (uint64, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.accounting.allocationSize(), nil
}
func (m *utilMemoryRuntime) MaximumUsed() (uint64, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.accounting.maximumUsed(), nil
}
func (m *utilMemoryRuntime) AllocationCount() (uint64, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.accounting.allocationCount(), nil
}

// CleanupPartial releases only a physical block whose source SIZE_T header
// cannot exist. It performs no fabricated source counter subtraction.
func (m *utilMemoryRuntime) CleanupPartial(h *utilMemoryBlock) error {
	if h == nil {
		return ErrMemoryRuntimeOwner
	}
	b, e := m.block(*h)
	if e != nil {
		return e
	}
	if len(b.data) >= int(m.config.SizeTBits/8) {
		return ErrMemoryRuntimeOwner
	}
	m.config.Budget.Release(uint64(len(b.data)) + uint64(len(b.initialized))*8)
	b.data = nil
	b.initialized = nil
	b.live = false
	m.live--
	if b.generation != math.MaxUint32 {
		b.generation++
	}
	*h = utilMemoryBlock{}
	return nil
}
