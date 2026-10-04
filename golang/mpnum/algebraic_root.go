// Licensed under the MIT license.
// Selected algebraic_numbers.cpp cell/univariate bodies. This prerequisite
// borrows real services; it is NOT the missing multivariate manager constructor.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"sync/atomic"
)

var ErrAlgebraicOwner = errors.New("mpnum: invalid algebraic owner or value")
var ErrAlgebraicAssertion = errors.New("mpnum: algebraic source assertion")
var ErrAlgebraicLive = errors.New("mpnum: live caller algebraic owners")
var ErrAlgebraicPending = errors.New("mpnum: algebraic cleanup pending")
var algebraicOrigins atomic.Uint64

// Null is source rational zero. Owning nonzero values must not be shallow-copied.
type Algebraic struct {
	owner            uint64
	slot, generation uint32
}
type AlgebraicRootConfig struct {
	Rationals   *RationalManager
	Polynomials *UPolynomialManager
	Factors     *UPolynomialFactorizationService
	Binary      *BinaryRationalManager
	Intervals   *BinaryIntervalManager
	Allocator   *memory.SmallAllocator
	Checkpoint  func() bool
	Cooperation func() error
}

// Only checked locations live here. Numeral fields are encoded in real object
// bytes; coefficient arrays are actual owned UPolynomial typed backings.
type algebraicLocation struct {
	raw                                       memory.ObjectRef
	coefficientInitialized, coefficientCursor uint32
	sourceGuard                               bool
	generation                                uint32
	kind, stage                               uint8
	live, initialized, published              bool
}
type AlgebraicRootCore struct {
	predicate                                              algebraicPredicateScope
	isRationalTmp                                          Int
	rootSelection                                          *AlgebraicVector
	self                                                   *AlgebraicRootCore
	config                                                 AlgebraicRootConfig
	origin, header                                         uint64
	locations                                              []algebraicLocation
	closed, closing                                        bool
	factors                                                *UPolynomialFactors
	roots, lowers, uppers                                  *BinaryRationalVector
	tmp2, tmp3                                             UPolynomial
	pending                                                Algebraic
	minMagnitude                                           int32
	factor                                                 bool
	factorParams                                           UPolynomialFactorParams
	compareCheap, comparePoly, compareRefine, compareSturm uint32
	compareSequence                                        *UPolynomialSequence
	callers, vectors                                       uint32
	localQ                                                 Rational
	localActive                                            bool
	localOriginal, localCleanup                            error
	constructionStage                                      uint8
}

// Scalar codec is actual Go-adapter extent: basic 24, root 56 bytes. It is not
// C++ sizeof. No duplicate owning numerals exist in the locator directory.
type algebraicRecord struct {
	value       Rational
	poly        UPolynomial
	interval    BinaryInterval
	flags, size uint32
}

func NewAlgebraicRootCore(c AlgebraicRootConfig) (*AlgebraicRootCore, error) {
	if c.Rationals == nil || c.Polynomials == nil || c.Factors == nil || c.Binary == nil || c.Intervals == nil || c.Allocator == nil || c.Checkpoint == nil || c.Cooperation == nil {
		return nil, arena.ErrConfig
	}
	if e := c.Rationals.checkOwnerReady(); e != nil {
		return nil, e
	}
	z := c.Rationals.z
	if c.Polynomials.config.Integer != z || c.Factors.manager != c.Polynomials || c.Binary.z != z || c.Intervals.bounds != c.Binary {
		return nil, arena.ErrConfig
	}
	if e := c.Polynomials.check(); e != nil {
		return nil, e
	}
	if e := c.Factors.check(); e != nil {
		return nil, e
	}
	if e := c.Binary.open(); e != nil {
		return nil, e
	}
	if c.Allocator.RuntimeOwner() != z.memory {
		return nil, arena.ErrConfig
	}
	if _, e := c.Allocator.NumFreeObjects(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(AlgebraicRootCore{}).Size())
	if e := z.budget.Reserve(n); e != nil {
		return nil, e
	}
	var id uint64
	for {
		old := algebraicOrigins.Load()
		if old == math.MaxUint64 {
			z.budget.Release(n)
			return nil, arena.ErrSlotLimit
		}
		if algebraicOrigins.CompareAndSwap(old, old+1) {
			id = old + 1
			break
		}
	}
	m := &AlgebraicRootCore{config: c, origin: id, header: n, minMagnitude: -16, factor: true, factorParams: UPolynomialFactorParams{MaxPrime: 31, PrimeTrials: 1, MaxSearchSize: 5000}}
	m.self = m
	return m, nil
}
func (m *AlgebraicRootCore) check() error {
	if m == nil || m.self != m {
		return ErrAlgebraicOwner
	}
	if m.closed || m.closing {
		return arena.ErrClosed
	}
	return m.config.Rationals.checkOwnerReady()
}
func (m *AlgebraicRootCore) location(a *Algebraic) (*algebraicLocation, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return m.locationCleanup(a)
}
func (m *AlgebraicRootCore) locationCleanup(a *Algebraic) (*algebraicLocation, error) {
	if m == nil || m.self != m {
		return nil, ErrAlgebraicOwner
	}
	if m.closed {
		return nil, arena.ErrClosed
	}
	if a == nil {
		return nil, ErrAlgebraicOwner
	}
	if a.slot == 0 {
		if a.owner != 0 || a.generation != 0 {
			return nil, ErrAlgebraicOwner
		}
		return nil, nil
	}
	if a.owner != m.origin || uint64(a.slot) > uint64(len(m.locations)) {
		return nil, ErrAlgebraicOwner
	}
	l := &m.locations[a.slot-1]
	if !l.live || l.generation != a.generation {
		return nil, ErrAlgebraicOwner
	}
	return l, nil
}
func (m *AlgebraicRootCore) grow() error {
	old := cap(m.locations)
	n := 2
	if old != 0 {
		n = (3*old + 1) / 2
	}
	stride := uint64(reflect.TypeOf(algebraicLocation{}).Size())
	if n <= old || uint64(n) > uint64(int(^uint(0)>>1))/stride || uint64(n) > math.MaxUint32 {
		return arena.ErrBounds
	}
	b := m.config.Rationals.budget
	if e := b.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]algebraicLocation, len(m.locations), n)
	copy(next, m.locations)
	b.Release(uint64(old) * stride)
	m.locations = next
	return nil
}
func (m *AlgebraicRootCore) allocate(kind uint8) (Algebraic, error) {
	if e := m.check(); e != nil {
		return Algebraic{}, e
	}
	i := len(m.locations)
	for j := range m.locations {
		if !m.locations[j].live && m.locations[j].generation != math.MaxUint32 {
			i = j
			break
		}
	}
	if i == len(m.locations) {
		if len(m.locations) == cap(m.locations) {
			if e := m.grow(); e != nil {
				return Algebraic{}, e
			}
		}
		m.locations = append(m.locations, algebraicLocation{})
	}
	l := &m.locations[i]
	l.generation++
	l.kind = kind
	l.stage = 0
	l.live = true
	l.initialized = false
	l.coefficientInitialized = 0
	l.coefficientCursor = 0
	l.sourceGuard = false
	l.published = false
	a := Algebraic{m.origin, uint32(i + 1), l.generation}
	n := uint64(24)
	if kind == 1 {
		n = 56
	}
	raw, e := m.config.Allocator.Allocate(n)
	l.raw = raw
	if e != nil {
		if raw.LogicalAddress() == 0 {
			l.live = false
		}
		return a, e
	}
	r := algebraicRecord{value: NewRational()}
	if e = m.store(l, &r); e != nil {
		return a, e
	}
	l.initialized = true
	return a, nil
}
func algebraicIntWords(a Int) [3]uint32 {
	return [3]uint32{uint32(a.val), uint32(a.cell), uint32(uint64(a.cell) >> 32)}
}
func algebraicWordsInt(w []uint32) Int {
	return Int{val: int32(w[0]), cell: numericCellRef(uint64(w[1]) | uint64(w[2])<<32)}
}
func (m *AlgebraicRootCore) store(l *algebraicLocation, r *algebraicRecord) error {
	var w [14]uint32
	n := 6
	if l.kind == 0 {
		a, b := algebraicIntWords(r.value.num), algebraicIntWords(r.value.den)
		copy(w[:3], a[:])
		copy(w[3:6], b[:])
	} else {
		n = 14
		w[0] = uint32(r.poly.origin)
		w[1] = uint32(r.poly.origin >> 32)
		w[2] = r.poly.slot
		w[3] = r.poly.generation
		w[4] = r.flags
		w[5] = r.size
		a, b := algebraicIntWords(r.interval.lower.num), algebraicIntWords(r.interval.upper.num)
		copy(w[6:9], a[:])
		w[9] = r.interval.lower.k
		copy(w[10:13], b[:])
		w[13] = r.interval.upper.k
	}
	for i := 0; i < n; i++ {
		if e := m.config.Allocator.WriteUint32(l.raw, uint64(i)*4, w[i]); e != nil {
			return e
		}
	}
	return nil
}
func (m *AlgebraicRootCore) load(l *algebraicLocation) (algebraicRecord, error) {
	r := algebraicRecord{value: NewRational()}
	if !l.initialized {
		return r, memory.ErrRuntimeUninitialized
	}
	var w [14]uint32
	n := 6
	if l.kind == 1 {
		n = 14
	}
	for i := 0; i < n; i++ {
		v, e := m.config.Allocator.ReadUint32(l.raw, uint64(i)*4)
		if e != nil {
			return r, e
		}
		w[i] = v
	}
	if l.kind == 0 {
		r.value = Rational{algebraicWordsInt(w[:3]), algebraicWordsInt(w[3:6])}
	} else {
		r.poly = UPolynomial{uint64(w[0]) | uint64(w[1])<<32, w[2], w[3]}
		r.flags = w[4]
		r.size = w[5]
		r.interval = BinaryInterval{BinaryRational{algebraicWordsInt(w[6:9]), w[9]}, BinaryRational{algebraicWordsInt(w[10:13]), w[13]}}
	}
	return r, nil
}
func (m *AlgebraicRootCore) IsZero(a *Algebraic) (bool, error) {
	_, e := m.location(a)
	return a != nil && a.slot == 0, e
}
func (m *AlgebraicRootCore) Del(a *Algebraic) error {
	l, e := m.location(a)
	if e != nil || l == nil {
		return e
	}
	return m.destroyCell(a, l, false)
}

// Identity-only private destruction remains callable after owning Close stops
// ordinary operations; it retains source Del guards and their completed cursor.
func (m *AlgebraicRootCore) delClosing(a *Algebraic) error {
	l, e := m.locationCleanup(a)
	if e != nil || l == nil {
		return e
	}
	return m.destroyCell(a, l, false)
}

// The cursor is cold lifetime metadata for actual completed source actions.
// Owning abort skips only source-unconstructed slots; source Del preserves the
// endangered read and its guard history, without rebuilding an array.
func (m *AlgebraicRootCore) destroyCell(a *Algebraic, l *algebraicLocation, owning bool) error {
	if l.initialized {
		r, e := m.load(l)
		if e != nil {
			return e
		}
		z := m.config.Rationals.z
		if l.kind == 0 {
			for l.stage < 2 {
				n := &r.value.num
				if l.stage == 1 {
					n = &r.value.den
				}
				e = z.Del(n)
				if e = m.save(l, &r, e); e != nil {
					return e
				}
				l.stage++
			}
		} else {
			if l.stage == 0 {
				if e = m.destroyPolynomial(l, &r, owning); e != nil {
					return e
				}
				l.stage = 1
			}
			for l.stage < 3 {
				n := &r.interval.lower.num
				if l.stage == 2 {
					n = &r.interval.upper.num
				}
				e = z.Del(n)
				if e = m.save(l, &r, e); e != nil {
					return e
				}
				l.stage++
			}
		}
	}
	n := uint64(24)
	if l.kind == 1 {
		n = 56
	}
	if l.raw.LogicalAddress() != 0 {
		progress, e := m.config.Allocator.DeallocateTracked(n, l.raw)
		if progress.PhysicalReleased || progress.PooledLinked {
			l.raw = memory.ObjectRef{}
			l.initialized = false
		}
		if e != nil {
			return e
		}
	}
	l.live = false
	if l.published {
		m.callers--
		l.published = false
	}
	*a = Algebraic{}
	return nil
}
func (m *AlgebraicRootCore) deleteLocation(a *Algebraic, l *algebraicLocation) error {
	return m.destroyCell(a, l, true)
}
func (m *AlgebraicRootCore) destroyPolynomial(l *algebraicLocation, r *algebraicRecord, owning bool) error {
	p := m.config.Polynomials
	n := r.size
	if owning {
		n = l.coefficientInitialized
	}
	for l.coefficientCursor < n {
		if r.poly.slot == 0 || l.coefficientCursor >= l.coefficientInitialized {
			l.sourceGuard = true
			return arena.ErrBounds
		}
		v, e := m.sourcePolynomial(r)
		if e != nil {
			return e
		}
		a, e := p.read(v, l.coefficientCursor)
		if e != nil {
			l.sourceGuard = true
			return e
		}
		if e = p.config.Integer.Del(a); e != nil {
			return e
		}
		l.coefficientCursor++
	}
	if r.poly.slot != 0 {
		d, e := p.descriptor(r.poly)
		if e != nil {
			return e
		}
		if owning {
			values := p.values(&d.state)
			for i := l.coefficientCursor; i < d.state.initialized; i++ {
				if e = p.config.Integer.Del(&values[i]); e != nil {
					return e
				}
			}
		}
		p.freeBacking(&d.state)
		d.live = false
		if d.generation != math.MaxUint32 {
			d.generation++
		}
		p.live--
		r.poly = UPolynomial{}
	}
	r.size = 0
	l.coefficientCursor = 0
	l.coefficientInitialized = 0
	return m.store(l, r)
}
func (m *AlgebraicRootCore) beginQ() error {
	if m.localActive {
		return ErrAlgebraicPending
	}
	m.localQ = NewRational()
	m.localActive = true
	m.localOriginal = nil
	m.localCleanup = nil
	return nil
}
func (m *AlgebraicRootCore) finishQ(cause error) error {
	if !m.localActive {
		return cause
	}
	m.localOriginal = cause
	e := m.config.Rationals.Del(&m.localQ)
	if e != nil {
		m.localCleanup = e
		if cause != nil {
			return cause
		}
		return e
	}
	m.localActive = false
	m.localOriginal = nil
	m.localCleanup = nil
	return cause
}
func (m *AlgebraicRootCore) publish(a *Algebraic, next Algebraic) {
	*a = next
	l := &m.locations[next.slot-1]
	if !l.published {
		l.published = true
		m.callers++
	}
	if m.pending == next {
		m.pending = Algebraic{}
		m.constructionStage = 0
	}
}
func (m *AlgebraicRootCore) save(l *algebraicLocation, r *algebraicRecord, cause error) error {
	e := m.store(l, r)
	if e != nil {
		return e
	}
	return cause
}

// Const source overload makes the scoped copy BEFORE mutable setter actions.
func (m *AlgebraicRootCore) SetRational(a *Algebraic, q *Rational) (err error) {
	if _, err = m.location(a); err != nil {
		return err
	}
	if err = m.beginQ(); err != nil {
		return err
	}
	defer func() { err = m.finishQ(err) }()
	if err = m.config.Rationals.Copy(&m.localQ, q); err != nil {
		return err
	}
	return m.SetTake(a, &m.localQ)
}
func (m *AlgebraicRootCore) SetTake(a *Algebraic, q *Rational) error {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	zero, e := m.config.Rationals.IsZero(q)
	if e != nil {
		return e
	}
	if zero {
		return m.Del(a)
	}
	if l != nil && l.kind == 1 {
		if e = m.Del(a); e != nil {
			return e
		}
		l = nil
	}
	if l != nil {
		r, e := m.load(l)
		if e != nil {
			return e
		}
		e = m.config.Rationals.Copy(&r.value, q)
		return m.save(l, &r, e)
	}
	if m.pending.slot != 0 {
		return ErrAlgebraicPending
	}
	next, e := m.allocate(0)
	if next.slot != 0 && m.locations[next.slot-1].live {
		m.pending = next
	}
	if e != nil {
		return e
	}
	l = &m.locations[next.slot-1]
	r, e := m.load(l)
	if e != nil {
		return e
	}
	e = m.config.Rationals.Swap(&r.value, q)
	if e = m.save(l, &r, e); e != nil {
		return e
	}
	m.publish(a, next)
	return nil
}
func (m *AlgebraicRootCore) copyPolynomial(l *algebraicLocation, r *algebraicRecord, v UPolynomialView) error {
	r.size = v.count
	l.coefficientInitialized = 0
	l.coefficientCursor = 0
	if e := m.store(l, r); e != nil {
		return e
	}
	p := m.config.Polynomials
	q, e := p.NewPolynomial()
	if e != nil {
		return e
	}
	r.poly = q
	if e = m.store(l, r); e != nil {
		return e
	}
	d, e := p.descriptor(q)
	if e != nil {
		return e
	}
	if e = p.reserve(&d.state, v.count); e != nil {
		return e
	}
	// Typed backing has zeroed physical spare capacity; only this explicit
	// prefix denotes source placement construction, then ascending numeral copy.
	for i := uint32(0); i < v.count; i++ {
		l.coefficientInitialized = i + 1
		a, e := p.read(v, i)
		if e != nil {
			return e
		}
		if e = m.config.Rationals.z.Copy(&p.values(&d.state)[i], a); e != nil {
			return e
		}
	}
	return nil
}
func (m *AlgebraicRootCore) deletePolynomial(l *algebraicLocation, r *algebraicRecord) error {
	return m.destroyPolynomial(l, r, false)
}
func (m *AlgebraicRootCore) updateSign(l *algebraicLocation, r *algebraicRecord) error {
	v, e := m.config.Polynomials.wholeView(r.poly)
	if e != nil {
		return e
	}
	s, e := m.config.Polynomials.EvalSignBinary(v, &r.interval.lower)
	if e != nil {
		return e
	}
	if modularSourceDebug {
		if s == 0 {
			return ErrUPolynomialAssertion
		}
		u, e := m.config.Polynomials.EvalSignBinary(v, &r.interval.upper)
		if e != nil {
			return e
		}
		if u != -s {
			return ErrUPolynomialAssertion
		}
	}
	r.flags &^= 2
	if s < 0 {
		r.flags |= 2
	}
	return m.store(l, r)
}
func (m *AlgebraicRootCore) normalizeCoefficients(l *algebraicLocation, r *algebraicRecord) error {
	v, e := m.sourcePolynomial(r)
	if e != nil {
		return e
	}
	if e = m.config.Polynomials.NormalizeView(v); e != nil {
		return e
	}
	if r.size == 0 {
		return arena.ErrBounds
	}
	lead, e := m.config.Polynomials.read(v, r.size-1)
	if e != nil {
		return e
	}
	negative, e := m.config.Rationals.z.IsNeg(lead)
	if e != nil {
		return e
	}
	if negative {
		if e = m.config.Polynomials.NegViewInPlace(v); e != nil {
			return e
		}
		r.flags ^= 2
	}
	return m.store(l, r)
}
func (m *AlgebraicRootCore) SetRoot(a *Algebraic, p UPolynomialView, lower, upper *BinaryRational, minimal bool) (err error) {
	l, err := m.location(a)
	if err != nil {
		return err
	}
	if modularSourceDebug && p.count <= 1 {
		return ErrUPolynomialAssertion
	}
	if p.count < 2 {
		return arena.ErrBounds
	}
	if p.count == 2 {
		if err = m.beginQ(); err != nil {
			return err
		}
		defer func() { err = m.finishQ(err) }()
		x, e := m.config.Polynomials.read(p, 0)
		if e != nil {
			return e
		}
		y, e := m.config.Polynomials.read(p, 1)
		if e != nil {
			return e
		}
		if err = m.config.Rationals.SetParts(&m.localQ, x, y); err != nil {
			return err
		}
		if err = m.config.Rationals.Neg(&m.localQ); err != nil {
			return err
		}
		return m.SetTake(a, &m.localQ)
	}
	existing := l != nil && l.kind == 1
	if !existing {
		if err = m.Del(a); err != nil {
			return err
		}
		if m.pending.slot != 0 {
			return ErrAlgebraicPending
		}
		next, e := m.allocate(1)
		if next.slot != 0 && m.locations[next.slot-1].live {
			m.pending = next
		}
		if e != nil {
			return e
		}
		l = &m.locations[next.slot-1]
	}
	r, err := m.load(l)
	if err != nil {
		return err
	}
	if existing {
		if err = m.deletePolynomial(l, &r); err != nil {
			return err
		}
	}
	if err = m.copyPolynomial(l, &r, p); err != nil {
		return err
	}
	if err = m.config.Intervals.SetBounds(&r.interval, lower, upper); err != nil {
		return m.save(l, &r, err)
	}
	if err = m.store(l, &r); err != nil {
		return err
	}
	if existing {
		r.flags &= 2
		if minimal {
			r.flags |= 5
		}
		if err = m.store(l, &r); err != nil {
			return err
		}
	}
	if err = m.updateSign(l, &r); err != nil {
		return err
	}
	if !existing {
		r.flags &= 2
		if minimal {
			r.flags |= 5
		}
		if err = m.store(l, &r); err != nil {
			return err
		}
	}
	if err = m.normalizeCoefficients(l, &r); err != nil {
		return err
	}
	if !existing {
		m.publish(a, m.pending)
	}
	if modularSourceDebug {
		v, e := m.config.Polynomials.wholeView(r.poly)
		if e != nil {
			return e
		}
		s, e := m.config.Polynomials.EvalSignBinary(v, &r.interval.lower)
		if e != nil {
			return e
		}
		want := 1
		if r.flags&2 != 0 {
			want = -1
		}
		if s != want {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (m *AlgebraicRootCore) Copy(dst, src *Algebraic) error {
	sl, e := m.location(src)
	if e != nil {
		return e
	}
	dl, e := m.location(dst)
	if e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if sl == nil {
		return m.Del(dst)
	}
	r, e := m.load(sl)
	if e != nil {
		return e
	}
	if sl.kind == 0 {
		return m.SetRational(dst, &r.value)
	}
	if dl == nil || dl.kind == 0 {
		if e = m.Del(dst); e != nil {
			return e
		}
		next, e := m.allocate(1)
		if e != nil {
			if next.slot != 0 && m.locations[next.slot-1].live {
				m.pending = next
			}
			return e
		}
		m.publish(dst, next)
		dl = &m.locations[dst.slot-1]
	} else {
		old, e := m.load(dl)
		if e != nil {
			return e
		}
		if e = m.deletePolynomial(dl, &old); e != nil {
			return e
		}
		e = m.config.Intervals.Del(&old.interval)
		if e = m.save(dl, &old, e); e != nil {
			return e
		}
	}
	out, e := m.load(dl)
	if e != nil {
		return e
	}
	v, e := m.config.Polynomials.wholeView(r.poly)
	if e != nil {
		return e
	}
	if e = m.copyPolynomial(dl, &out, v); e != nil {
		return e
	}
	e = m.config.Intervals.Copy(&out.interval, &r.interval)
	if e = m.save(dl, &out, e); e != nil {
		return e
	}
	out.flags = r.flags
	return m.store(dl, &out)
}
func (m *AlgebraicRootCore) CleanupRetained() error {
	if m == nil || m.self != m || m.closed {
		return ErrAlgebraicOwner
	}
	if m.localActive {
		if e := m.finishQ(nil); e != nil {
			return e
		}
	}
	if m.pending.slot != 0 {
		l := &m.locations[m.pending.slot-1]
		if e := m.deleteLocation(&m.pending, l); e != nil {
			return e
		}
		m.constructionStage = 0
	}
	if e := m.finishPredicate(nil); e != nil {
		return e
	}
	return m.comparisonCleanup()
}
func (m *AlgebraicRootCore) Close() error {
	if m == nil || m.self != m {
		return ErrAlgebraicOwner
	}
	if m.closed {
		return nil
	}
	// Only named private selection elements are excluded from caller ownership.
	var privateCallers, privateVectors uint32
	if m.rootSelection != nil {
		privateVectors = 1
		for i := uint32(0); i < m.rootSelection.size; i++ {
			if m.rootSelection.values[i].slot != 0 {
				privateCallers++
			}
		}
	}
	if m.callers != privateCallers || m.vectors != privateVectors {
		return ErrAlgebraicLive
	}
	m.closing = true
	// The later lexical comparison sequence is destroyed before older owners.
	if e := m.comparisonCleanup(); e != nil {
		return e
	}
	if m.rootSelection != nil {
		if e := m.rootSelection.Close(); e != nil {
			return e
		}
		m.rootSelection = nil
	}
	if e := m.CleanupRetained(); e != nil {
		return e
	}
	// Reverse declaration order of the selected actual source members.
	for _, v := range []**BinaryRationalVector{&m.uppers, &m.lowers, &m.roots} {
		if *v != nil {
			if e := (*v).Close(); e != nil {
				return e
			}
			*v = nil
		}
	}
	if m.factors != nil {
		if e := m.factors.Close(); e != nil {
			return e
		}
		m.factors = nil
	}
	for _, p := range []*UPolynomial{&m.tmp3, &m.tmp2} {
		if p.slot != 0 {
			if e := m.config.Polynomials.DelPolynomial(p); e != nil {
				return e
			}
		}
	}
	if e := m.config.Rationals.z.Del(&m.isRationalTmp); e != nil {
		return e
	}
	b := m.config.Rationals.budget
	b.Release(uint64(cap(m.locations)) * uint64(reflect.TypeOf(algebraicLocation{}).Size()))
	b.Release(m.header)
	m.locations = nil
	m.header = 0
	m.closed = true
	return nil
}

func (s *RationalService) NewAlgebraicRootCore(c AlgebraicRootConfig) (*AlgebraicRootCore, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	defer s.mu.Unlock()
	if e := s.lexicalReady(); e != nil {
		return nil, e
	}
	if c.Rationals != nil && c.Rationals != s.manager {
		return nil, ErrAlgebraicOwner
	}
	c.Rationals = s.manager
	return NewAlgebraicRootCore(c)
}
func (m *AlgebraicRootCore) sourcePolynomial(r *algebraicRecord) (UPolynomialView, error) {
	v, e := m.config.Polynomials.wholeView(r.poly)
	if e != nil {
		return v, e
	}
	v.count = r.size
	return v, nil
}

// CleanupValue is the explicit owning abort of one caller-provided reachable
// value. It does not repair source metadata or discover roots from locations.
func (m *AlgebraicRootCore) CleanupValue(a *Algebraic) error {
	l, e := m.location(a)
	if e != nil || l == nil {
		return e
	}
	return m.deleteLocation(a, l)
}
