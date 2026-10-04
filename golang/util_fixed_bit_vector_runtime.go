// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp; src/util/small_object_allocator.h.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"sync/atomic"
)

// RuntimeFixedBitVector is checked scalar provenance, not a source pointer encoding.
type RuntimeFixedBitVector struct {
	origin    uint64
	object    memory.ObjectRef
	rawBlock  memory.Block
	rawOffset uint64
	inline    bool
}

var fixedRuntimeOrigins atomic.Uint64

// The source m_0 is one inline uninitialized unsigned, retained across reset.
// The allocator cold owner and this reflected Go owner are separate physical charges.
type RuntimeFixedBitVectorManager struct {
	self                                    *RuntimeFixedBitVectorManager
	runtime                                 *memory.Runtime
	budget                                  *arena.Budget
	allocator                               *memory.SmallAllocator
	profile                                 memory.SmallAllocatorProfile
	origin, ownerBytes                      uint64
	numBits, numWords, numBytes, mask, zero uint32
	zeroInitialized                         uint8
	closed, sourceDone                      bool
	sourceStop                              error
}

func NewRuntimeFixedBitVectorManager(n uint32, c memory.SmallAllocatorConfig) (*RuntimeFixedBitVectorManager, error) {
	if c.Runtime == nil {
		return nil, arena.ErrConfig
	}
	if e := c.Runtime.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(RuntimeFixedBitVectorManager{}).Size())
	b := c.Runtime.Budget()
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	m := &RuntimeFixedBitVectorManager{runtime: c.Runtime, budget: b, profile: c.Profile, ownerBytes: bytes}
	m.self = m
	if m.profile.PointerBits == 0 {
		m.profile.PointerBits = c.Runtime.SizeTBits()
	}
	a, e := memory.NewSmallAllocator(c)
	if e != nil {
		b.Release(bytes)
		return nil, e
	}
	m.allocator = a
	for {
		old := fixedRuntimeOrigins.Load()
		if old == math.MaxUint64 {
			m.sourceStop = arena.ErrSlotLimit
			return m, m.sourceStop
		}
		if fixedRuntimeOrigins.CompareAndSwap(old, old+1) {
			m.origin = old + 1
			break
		}
	}
	m.numBits = n
	m.numWords = (n + 31) / 32
	m.numBytes = m.numWords * 4
	m.mask = (uint32(1) << (n % 32)) - 1
	if m.mask == 0 {
		m.mask = ^uint32(0)
	}
	return m, nil
}
func (m *RuntimeFixedBitVectorManager) identity() error {
	if m == nil || m.self != m {
		return ErrSATOwner
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) open() error {
	if e := m.identity(); e != nil {
		return e
	}
	if m.closed || m.sourceDone || m.sourceStop != nil {
		return arena.ErrClosed
	}
	return nil
}
func (m *RuntimeFixedBitVectorManager) NumBits() (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.numBits, nil
}
func (m *RuntimeFixedBitVectorManager) NumWords() (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.numWords, nil
}
func (m *RuntimeFixedBitVectorManager) NumBytes() (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.numBytes, nil
}
func (m *RuntimeFixedBitVectorManager) Mask() (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	return m.mask, nil
}
func (m *RuntimeFixedBitVectorManager) Allocate() (RuntimeFixedBitVector, error) {
	if e := m.open(); e != nil {
		return RuntimeFixedBitVector{}, e
	}
	v := RuntimeFixedBitVector{origin: m.origin}
	if m.numBytes == 0 {
		v.inline = true
		return v, nil
	}
	ref, e := m.allocator.Allocate(uint64(m.numBytes))
	v.object = ref
	if e == nil {
		v.rawBlock, v.rawOffset, e = m.allocator.RawLocation(ref)
	}
	return v, e
}
func (m *RuntimeFixedBitVectorManager) Allocate0() (RuntimeFixedBitVector, error) {
	v, e := m.Allocate()
	if e == nil {
		e = m.Fill0(v)
	}
	return v, e
}
func (m *RuntimeFixedBitVectorManager) Allocate1() (RuntimeFixedBitVector, error) {
	v, e := m.Allocate()
	if e == nil {
		e = m.Fill1(v)
	}
	return v, e
}
func (m *RuntimeFixedBitVectorManager) AllocateCopy(src RuntimeFixedBitVector) (RuntimeFixedBitVector, error) {
	return m.AllocateCopyFrom(m, src)
}
func (m *RuntimeFixedBitVectorManager) AllocateCopyFrom(s *RuntimeFixedBitVectorManager, src RuntimeFixedBitVector) (RuntimeFixedBitVector, error) {
	v, e := m.Allocate()
	if e == nil {
		e = m.CopyFrom(v, s, src)
	}
	return v, e
}
func (m *RuntimeFixedBitVectorManager) Deallocate(v RuntimeFixedBitVector) error {
	if e := m.open(); e != nil {
		return e
	}
	if m.numBytes == 0 {
		return nil
	}
	if v.origin != m.origin || v.inline {
		return ErrSATOwner
	}
	return m.allocator.Deallocate(uint64(m.numBytes), v.object)
}
func (m *RuntimeFixedBitVectorManager) Reset() error {
	if e := m.open(); e != nil {
		return e
	}
	return m.allocator.Reset()
}
func (m *RuntimeFixedBitVectorManager) readByte(v RuntimeFixedBitVector, i uint64) (byte, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	if v.origin != m.origin {
		return 0, ErrSATOwner
	}
	if v.inline {
		if i >= 4 {
			return 0, arena.ErrBounds
		}
		if m.zeroInitialized&(1<<i) == 0 {
			return 0, memory.ErrRuntimeUninitialized
		}
		return byte(m.zero >> (8 * i)), nil
	}
	return m.allocator.Read(v.object, i)
}
func (m *RuntimeFixedBitVectorManager) writeByte(v RuntimeFixedBitVector, i uint64, b byte) error {
	if e := m.open(); e != nil {
		return e
	}
	if v.origin != m.origin {
		return ErrSATOwner
	}
	if v.inline {
		if i >= 4 {
			return arena.ErrBounds
		}
		m.zero = (m.zero &^ (uint32(255) << (8 * i))) | uint32(b)<<(8*i)
		m.zeroInitialized |= 1 << i
		return nil
	}
	return m.allocator.Write(v.object, i, b)
}
func (m *RuntimeFixedBitVectorManager) Word(v RuntimeFixedBitVector, i uint32) (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	if v.origin != m.origin {
		return 0, ErrSATOwner
	}
	if v.inline {
		if i != 0 {
			return 0, arena.ErrBounds
		}
		if m.zeroInitialized != 0xF {
			return 0, memory.ErrRuntimeUninitialized
		}
		return m.zero, nil
	}
	return m.allocator.ReadUint32(v.object, uint64(i)*4)
}
func (m *RuntimeFixedBitVectorManager) putWord(v RuntimeFixedBitVector, i, n uint32) error {
	if e := m.open(); e != nil {
		return e
	}
	if v.origin != m.origin {
		return ErrSATOwner
	}
	if v.inline {
		if i != 0 {
			return arena.ErrBounds
		}
		m.zero = n
		m.zeroInitialized = 0xF
		return nil
	}
	return m.allocator.WriteUint32(v.object, uint64(i)*4, n)
}
func (m *RuntimeFixedBitVectorManager) Get(v RuntimeFixedBitVector, i uint32) (bool, error) {
	w, e := m.Word(v, i/32)
	return w&(uint32(1)<<(i%32)) != 0, e
}

// A captured span is a source raw-pointer domain. It checks actual bytes, not
// later ObjectRef generation; pooled frees/reuse therefore have native lifetime
// preconditions. Only genuine direct allocations may be released through it.
type RuntimeFixedBitVectorSpan struct {
	Runtime        *memory.Runtime
	Block          memory.Block
	Offset, Extent uint64
	Profile        memory.SmallAllocatorProfile
	Direct         bool
}

func (m *RuntimeFixedBitVectorManager) Capture(v RuntimeFixedBitVector) (RuntimeFixedBitVectorSpan, error) {
	if e := m.open(); e != nil {
		return RuntimeFixedBitVectorSpan{}, e
	}
	if v.origin != m.origin || v.inline {
		return RuntimeFixedBitVectorSpan{}, ErrSATOwner
	}
	h, off, e := m.allocator.RawLocation(v.object)
	if e != nil {
		return RuntimeFixedBitVectorSpan{}, e
	}
	direct := m.profile.Debug && !m.profile.Windows || uint64(m.numBytes) >= 256-uint64(m.profile.PointerBits/8)
	return RuntimeFixedBitVectorSpan{m.runtime, h, off, uint64(m.numBytes), m.profile, direct}, nil
}
func (s RuntimeFixedBitVectorSpan) Word(i uint32) (uint32, error) {
	off := uint64(i) * 4
	if off > s.Extent || s.Extent-off < 4 {
		return 0, arena.ErrBounds
	}
	if s.Runtime == nil {
		return 0, ErrSATOwner
	}
	return s.Runtime.ReadUint32(s.Block, s.Offset+off)
}
func (s *RuntimeFixedBitVectorSpan) ReleaseDirect() error {
	if s == nil || s.Runtime == nil || !s.Direct || s.Offset != 0 {
		return ErrSATOwner
	}
	return s.Runtime.Deallocate(&s.Block)
}
func (m *RuntimeFixedBitVectorManager) SourceFinalize() error {
	if e := m.identity(); e != nil {
		return e
	}
	if m.sourceStop != nil {
		return m.sourceStop
	}
	if m.sourceDone {
		return nil
	}
	e := m.allocator.Close()
	if e != nil {
		m.sourceStop = e
		return e
	}
	m.sourceDone = true
	return nil
}
func (m *RuntimeFixedBitVectorManager) Close() error {
	if e := m.identity(); e != nil {
		return e
	}
	if m.closed {
		return nil
	}
	if e := m.SourceFinalize(); e != nil {
		return e
	}
	m.budget.Release(m.ownerBytes)
	m.ownerBytes = 0
	m.closed = true
	return nil
}

// CleanupStopped is an explicit owning abort, not source destructor replay or
// registry cleanup. The allocator holds the actual source action cursors.
func (m *RuntimeFixedBitVectorManager) CleanupStopped() error {
	if e := m.identity(); e != nil {
		return e
	}
	if m.closed {
		return nil
	}
	// A stopped raw invocation borrows its caller's pointer. Acknowledge only
	// this receiver's metadata, preserving source count/stage/cause and donor.
	if _, _, e := m.allocator.AbortRawInvocation(); e != nil {
		return errors.Join(m.sourceStop, e)
	}
	// Abort only the actual retained pending fields before asking the child for
	// its idempotent completed destructor receipt. CleanupPending is identity-only
	// and remains defined after source destructorDone; CleanupChunks is not.
	if e := m.allocator.CleanupPending(); e != nil {
		return errors.Join(m.sourceStop, e)
	}
	e := m.allocator.Close()
	if e != nil {
		if m.sourceStop == nil {
			m.sourceStop = e
		}
		if !errors.Is(e, memory.ErrAllocatorPending) {
			return errors.Join(m.sourceStop, e)
		}
		if e = m.allocator.CleanupChunks(); e != nil {
			return errors.Join(m.sourceStop, e)
		}
		if e = m.allocator.Close(); e != nil {
			return errors.Join(m.sourceStop, e)
		}
	}
	m.sourceDone = true
	m.budget.Release(m.ownerBytes)
	m.ownerBytes = 0
	m.closed = true
	return nil
}
