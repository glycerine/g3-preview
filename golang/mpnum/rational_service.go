// Copyright (c) 2006 Microsoft Corporation
// Concrete owner-local core of src/util/rational.h/.cpp and vector.h.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
)

var ErrRationalServiceOwner = errors.New("mpnum: rational service provenance mismatch")
var ErrRationalServiceCopy = errors.New("mpnum: shallow rational service copy")

// Only this scalar namespace is global; IDs never wrap or get reused.
var rationalServiceNextOwnerID uint64

// OwnedRational is a pointer-free owning token; its zero value is invalid.
// Never shallow-copy an owning token: use Copy/CopyFrom/Move/Swap. Provenance
// identifies the service domain, not individual numeral lifetimes. Service
// Close invalidates every outstanding value from that domain.
type OwnedRational struct {
	raw     Rational
	ownerID uint64
}
type RationalServiceConfig struct{ Integers IntegerConfig }
type RationalServiceStats struct {
	CacheLength, CacheCapacity uint32
	HeaderBytes, CacheBytes    uint64
	LiveCells, LivePayloads    uint32
	// LivePayloads includes actual numeric objects (including a pending cell)
	// and genuine Natural buffers. It is not a count of raw allocator blocks.
	LiveNumericObjects, PendingCellObjects, RetainedCellDescriptors, NaturalPayloads uint32
}

// RationalService owns the source synchronized manager, constants and cache.
// This mutex serializes all manager actions. A shared borrowed Budget still
// requires external owner confinement; independent service locks do not make
// the Budget safe for concurrent use. Never shallow-copy this owner.
type RationalService struct {
	mu                                             sync.Mutex
	manager                                        *RationalManager
	self                                           *RationalService
	ownerID                                        uint64
	zero, one, minusOne                            Rational
	infZero, infOne, infMinusOne                   OwnedInfRational
	infIntZero, infIntOne, infIntMinusOne          OwnedInfIntRational
	cache                                          []Rational
	cacheBytes, headerBytes                        uint64
	closed                                         bool
	originalCause, cleanupCause                    error
	cleanupStage                                   uint8
	cleanupCacheIndex                              uint32
	cleanupComponent                               uint8
	closing                                        bool
	sourceFinalizing, sourceFinalized, sourceGuard bool
	sourceCause                                    error
}

func nextRationalServiceID() (uint64, error) {
	for {
		old := atomic.LoadUint64(&rationalServiceNextOwnerID)
		if old == math.MaxUint64 {
			return 0, arena.ErrBounds
		}
		if atomic.CompareAndSwapUint64(&rationalServiceNextOwnerID, old, old+1) {
			return old + 1, nil
		}
	}
}
func NewRationalService(c RationalServiceConfig) (*RationalService, error) {
	if c.Integers.Cells.Budget != c.Integers.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	id, e := nextRationalServiceID()
	if e != nil {
		return nil, e
	}
	b := c.Integers.Cells.Budget
	header := uint64(reflect.TypeOf(RationalService{}).Size())
	if e = b.Reserve(header); e != nil {
		return nil, e
	}
	s := &RationalService{ownerID: id, zero: NewRational(), one: NewRational(), minusOne: NewRational(), headerBytes: header}
	s.self = s
	m, e := NewRationalManager(RationalConfig{Integers: c.Integers, Synchronized: true})
	if e != nil {
		b.Release(header)
		return nil, e
	}
	s.manager = m
	// Source initialization order: manager, zero, one, minus one.
	if e = m.SetInt64(&s.zero, 0); e == nil {
		e = m.SetInt64(&s.one, 1)
	}
	if e == nil {
		e = m.SetInt64(&s.minusOne, -1)
	}
	if e == nil {
		e = s.initInfRationalConstants()
	}
	if e == nil {
		e = s.initInfIntRationalConstants()
	}
	if e != nil {
		s.originalCause = e
		if cleanup := s.CloseChecked(); cleanup != nil {
			return nil, s
		}
		return nil, e
	}
	return s, nil
}
func (s *RationalService) OwnerID() uint64 {
	if s == nil {
		return 0
	}
	return s.ownerID
}
func (s *RationalService) lock() error {
	if s == nil || s.self != s {
		return ErrRationalServiceCopy
	}
	s.mu.Lock()
	if s.closed || s.closing || s.sourceFinalizing {
		s.mu.Unlock()
		return arena.ErrClosed
	}
	return nil
}
func (s *RationalService) check(v *OwnedRational) error {
	if v == nil || v.ownerID == 0 || v.ownerID != s.ownerID {
		return ErrRationalServiceOwner
	}
	return s.manager.check(&v.raw)
}
func (s *RationalService) NewValue() (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return OwnedRational{raw: NewRational(), ownerID: s.ownerID}, nil
}
func (s *RationalService) Del(v *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if v == nil || v.ownerID == 0 || v.ownerID != s.ownerID {
		return ErrRationalServiceOwner
	}
	if e := s.manager.checkCleanup(&v.raw); e != nil {
		return e
	}
	if e := s.manager.Del(&v.raw); e != nil {
		return e
	}
	*v = OwnedRational{}
	return nil
}
func (s *RationalService) Copy(dst, src *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(dst); e != nil {
		return e
	}
	if e := s.rationalScalarOwner(src); e != nil {
		return e
	}
	return s.manager.Copy(&dst.raw, &src.raw)
}
func (s *RationalService) CopyFrom(dst *OwnedRational, source *RationalService, src *OwnedRational) error {
	if s == nil || s.self != s || source == nil || source.self != source {
		return ErrRationalServiceCopy
	}
	if s == source {
		return s.Copy(dst, src)
	}
	first, second := s, source
	if first.ownerID > second.ownerID {
		first, second = second, first
	}
	first.mu.Lock()
	defer first.mu.Unlock()
	second.mu.Lock()
	defer second.mu.Unlock()
	if s.closed || source.closed {
		return arena.ErrClosed
	}
	if e := s.check(dst); e != nil {
		return e
	}
	if e := source.check(src); e != nil {
		return e
	}
	return s.manager.CopyFrom(&dst.raw, source.manager, &src.raw)
}
func (s *RationalService) Move(dst, src *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(dst); e != nil {
		return e
	}
	if e := s.check(src); e != nil {
		return e
	}
	return s.manager.Move(&dst.raw, &src.raw)
}
func (s *RationalService) Swap(a, b *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return e
	}
	if e := s.check(b); e != nil {
		return e
	}
	return s.manager.Swap(&a.raw, &b.raw)
}
func (s *RationalService) SetInt64(v *OwnedRational, n int64) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return e
	}
	return s.manager.SetInt64(&v.raw, n)
}

// SetUnsigned translates the source unsigned32 constructor overload, including
// INT_MAX inline. The source uint64 setter uses a distinct strict boundary.
func (s *RationalService) SetUnsigned(v *OwnedRational, n uint32) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return e
	}
	return s.manager.SetInt64(&v.raw, int64(n))
}
func (s *RationalService) SetUint64(v *OwnedRational, n uint64) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return e
	}
	return s.manager.SetUint64(&v.raw, n)
}
func (s *RationalService) SetFraction(v *OwnedRational, n int64, d uint64) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return e
	}
	return s.manager.SetFraction(&v.raw, n, d)
}
func (s *RationalService) SetFractionSigned(v *OwnedRational, n, d int32) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return e
	}
	return s.manager.SetFractionSigned(&v.raw, n, d)
}
func (s *RationalService) SetString(v *OwnedRational, text string) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return e
	}
	return s.manager.SetString(&v.raw, text)
}
func (s *RationalService) String(v *OwnedRational) (string, error) {
	if e := s.lock(); e != nil {
		return "", e
	}
	defer s.mu.Unlock()
	if e := s.check(v); e != nil {
		return "", e
	}
	return s.manager.String(&v.raw)
}
func (s *RationalService) Equal(a, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	return s.manager.Equal(&a.raw, &b.raw)
}
func (s *RationalService) copyOutput(raw *Rational) (OwnedRational, error) {
	v := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	if e := s.manager.Copy(&v.raw, raw); e != nil {
		s.manager.Del(&v.raw)
		return OwnedRational{}, e
	}
	return v, nil
}
func (s *RationalService) Zero() (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyOutput(&s.zero)
}
func (s *RationalService) One() (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyOutput(&s.one)
}
func (s *RationalService) MinusOne() (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyOutput(&s.minusOne)
}
func (s *RationalService) growCache() error {
	old := uint64(cap(s.cache))
	next := uint64(2)
	if old != 0 {
		next = (3*old + 1) / 2
	}
	bytes := next * uint64(reflect.TypeOf(Rational{}).Size())
	if next <= old || next > math.MaxUint32 || bytes > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := s.manager.budget.Reserve(bytes); e != nil {
		return e
	}
	data := make([]Rational, len(s.cache), int(next))
	// Source vector moves each element then destroys that moved-from wrapper.
	// Transfer the handles, retain old m_val history, then run its no-ref Del.
	for i := range s.cache {
		data[i] = s.cache[i]
		s.cache[i].num.cell = 0
		s.cache[i].den.cell = 0
		s.manager.Del(&s.cache[i])
	}
	s.cache = data
	s.manager.budget.Release(s.cacheBytes)
	s.cacheBytes = bytes
	return nil
}
func (s *RationalService) pushCache(v *Rational) error {
	if len(s.cache) == cap(s.cache) {
		if e := s.growCache(); e != nil {
			return e
		}
	}
	item := NewRational()
	if e := s.manager.Copy(&item, v); e != nil {
		s.manager.Del(&item)
		return e
	}
	// Placement-copy completes before source vector size is incremented.
	s.cache = append(s.cache, item)
	return nil
}
func (s *RationalService) fillPower(n uint32) error {
	if len(s.cache) == 0 {
		if e := s.pushCache(&s.one); e != nil {
			return e
		}
	}
	size := uint32(len(s.cache))
	curr, two := NewRational(), NewRational()
	defer s.manager.Del(&curr)
	defer s.manager.Del(&two)
	if e := s.manager.Copy(&curr, &s.cache[size-1]); e != nil {
		return e
	}
	if e := s.manager.SetInt64(&two, 2); e != nil {
		return e
	}
	for i := size; i <= n; i++ {
		if e := s.manager.Mul(&curr, &two, &curr); e != nil {
			return e
		}
		if e := s.pushCache(&curr); e != nil {
			return e
		}
	}
	return nil
}
func (s *RationalService) PowerOfTwo(k uint32) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	// Checked adapter for source k+1/loop and k+2 length overflow. Host backing
	// cardinality is checked before any numerical action or budget reservation.
	if k > math.MaxUint32-2 || uint64(k+2)*uint64(reflect.TypeOf(Rational{}).Size()) > uint64(^uint(0)>>1) {
		return OwnedRational{}, arena.ErrBounds
	}
	if uint64(k) >= uint64(len(s.cache)) {
		if e := s.fillPower(k + 1); e != nil {
			return OwnedRational{}, e
		}
	}
	return s.copyOutput(&s.cache[k])
}
func (s *RationalService) Stats() (RationalServiceStats, error) {
	if e := s.lock(); e != nil {
		return RationalServiceStats{}, e
	}
	defer s.mu.Unlock()
	z := s.manager.z
	stats := RationalServiceStats{CacheLength: uint32(len(s.cache)), CacheCapacity: uint32(cap(s.cache)), HeaderBytes: s.headerBytes, CacheBytes: s.cacheBytes, LiveCells: z.cells.Stats().Live, NaturalPayloads: z.words.Stats().Live + z.natural.overflowPayloads()}
	for n := uint64(1); n <= uint64(z.cells.Stats().Slots); n++ {
		c, e := z.cells.Read(arena.Slot(n))
		if e != nil {
			continue
		}
		if !c.released {
			stats.LiveNumericObjects++
		}
		if c.unpublished || c.cleanupPending {
			stats.RetainedCellDescriptors++
		}
	}
	if z.pendingCell.object != (memory.ObjectRef{}) {
		stats.PendingCellObjects = 1
	}
	stats.LivePayloads = stats.LiveNumericObjects + stats.PendingCellObjects + stats.NaturalPayloads
	return stats, nil
}

// destroyCache is the source vector::finalize traversal. Caller holds the owner
// lock; its separate cold helper permits observing release order before the
// manager itself is destroyed, without stored callbacks or per-record boxing.
func (s *RationalService) destroyCache() {
	for i := range s.cache {
		s.manager.Del(&s.cache[i])
	}
	s.cache = nil
	s.manager.budget.Release(s.cacheBytes)
	s.cacheBytes = 0
}

// The two source finalizer hooks run before the powers vector is destroyed.
// Caller holds the service lock (or the constructor has not published it).
func (s *RationalService) destroyInfinitesimalConstants() {
	s.destroyInfRationalConstants()
	s.destroyInfIntRationalConstants()
}
func (s *RationalService) Close() error { return s.CloseChecked() }
