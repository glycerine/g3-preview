// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Licensed under the MIT license.
// Source: src/util/vector.h:59-100; src/math/polynomial/upolynomial.cpp:164-203.
// Concrete typed storage for actual svector<mpz> coefficient ownership.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// UPolynomial is an explicit scalar owner. Never copy it as another owning
// polynomial: Copy performs the source numeral copies through its manager.
type UPolynomial struct {
	origin           uint64
	slot, generation uint32
}

// UPolynomialView borrows actual physical coefficient storage. Swapping vector
// ownership preserves it; growth/free invalidates that physical binding.
type UPolynomialView struct {
	origin                            uint64
	backing, generation, start, count uint32
}

// Every hot coefficient is the actual named Int record. Variable backing
// slices stay exclusively in the charged cold physical-storage directory.
type upolyVectorState struct {
	origin                                           uint64
	backing, generation, size, capacity, initialized uint32
}
type upolyDescriptor struct {
	state      upolyVectorState
	generation uint32
	live       bool
}
type upolyDescriptorPage struct{ rows [32]upolyDescriptor }
type upolyBacking struct {
	values                  []Int
	generation, initialized uint32
	live                    bool
}

// The source vector SZ is uint32 and selected LP64 sizeof(mpz)=16, header8.
// Actual Go stride/host representability is checked independently; only the
// real typed allocation bytes are reserved, source header metadata is in its
// actual descriptor/cold ownership graph rather than a phantom allocation.
// upolySourceCapacity performs the literal source SZ arithmetic/check first.
func upolySourceCapacity(old uint32) (n, oldBytes, newBytes uint32, e error) {
	if old == 0 {
		return 2, 0, 40, nil
	}
	oldBytes = 16*old + 8
	n = (3*old + 1) >> 1
	newBytes = 16*n + 8
	if n <= old || newBytes <= oldBytes {
		e = arena.ErrBounds
	}
	return
}
func upolyNextCapacity(old, stride, hostMax uint64) (uint32, uint64, error) {
	if old > math.MaxUint32 || stride == 0 {
		return 0, 0, arena.ErrBounds
	}
	n, _, _, e := upolySourceCapacity(uint32(old))
	if e != nil {
		return 0, 0, e
	}
	// Separate undefined-source-allocation guard: wrapped byte allocations
	// cannot hold the actual source coefficient extent. Do not model OOB writes.
	if uint64(n)*16+8 > math.MaxUint32 || uint64(n) > hostMax/stride {
		return 0, 0, arena.ErrBounds
	}
	return n, uint64(n) * stride, nil
}

func upolyDirectoryNext(old, stride, host uint64) (uint64, uint64, error) {
	if stride == 0 || old > host/2 {
		return 0, 0, arena.ErrBounds
	}
	n := old * 2
	if n == 0 {
		n = 2
	}
	if n > host/stride {
		return 0, 0, arena.ErrBounds
	}
	return n, n * stride, nil
}

func (m *UPolynomialManager) values(s *upolyVectorState) []Int {
	if s.backing == 0 {
		return nil
	}
	return m.stateStore(s).backings[s.backing-1].values
}
func (m *UPolynomialManager) backingSlot() (uint32, error) {
	for i := range m.store.backings {
		b := &m.store.backings[i]
		if !b.live && b.generation != math.MaxUint32 {
			return uint32(i + 1), nil
		}
	}
	if uint64(len(m.store.backings)) >= math.MaxUint32 {
		return 0, arena.ErrSlotLimit
	}
	if len(m.store.backings) == cap(m.store.backings) {
		stride := uint64(reflect.TypeOf(upolyBacking{}).Size())
		next, bytes, e := upolyDirectoryNext(uint64(cap(m.store.backings)), stride, uint64(int(^uint(0)>>1)))
		if e != nil {
			return 0, e
		}
		n := int(next)
		if e := m.config.Integer.budget.Reserve(bytes); e != nil {
			return 0, e
		}
		a := make([]upolyBacking, len(m.store.backings), n)
		copy(a, m.store.backings)
		m.config.Integer.budget.Release(uint64(cap(m.store.backings)) * stride)
		m.store.backings = a
	}
	m.store.backings = append(m.store.backings, upolyBacking{generation: 1})
	return uint32(len(m.store.backings)), nil
}
func (m *UPolynomialManager) grow(s *upolyVectorState) error {
	return m.growWithFactorNumeralVectorReceipt(s, nil)
}
func (m *UPolynomialManager) growWithFactorNumeralVectorReceipt(s *upolyVectorState, r *polynomialFactorNumeralVectorReceipt) error {
	n, bytes, e := upolyNextCapacity(uint64(s.capacity), uint64(reflect.TypeOf(Int{}).Size()), uint64(int(^uint(0)>>1)))
	if e != nil {
		return e
	}
	if r != nil {
		if r.manager == nil || !r.manager.matchesFactorNumeralVector(r) || r.manager.upm != m || r.frame.output != s || !r.frame.active || !r.frame.executing || r.manager.core.callbackDepth != 0 {
			return ErrUPolynomialOwner
		}
		if e := r.manager.archiveFactorNumeralVector(); e != nil {
			return e
		}
	}
	slot, e := m.backingSlot()
	if e != nil {
		return e
	}
	if e = m.config.Integer.budget.Reserve(bytes); e != nil {
		return e
	}
	a := make([]Int, n)
	old := m.values(s)
	// Source nontrivial growth publishes the new data binding before the
	// ascending move loop; capacity publishes only after old storage deletion.
	oldSlot, oldCap := s.backing, s.capacity
	oldStore := m.stateStore(s)
	b := &m.store.backings[slot-1]
	b.values = a
	b.live = true
	s.origin = m.store.origin
	s.backing = slot
	s.generation = b.generation
	for i := uint32(0); i < s.size; i++ {
		a[i] = old[i]
		old[i].cell = 0
	}
	s.initialized = s.size
	b.initialized = s.size
	if oldSlot != 0 {
		oldB := &oldStore.backings[oldSlot-1]
		oldB.values = nil
		oldB.live = false
		if oldB.generation != math.MaxUint32 {
			oldB.generation++
		}
		m.config.Integer.budget.Release(uint64(oldCap) * uint64(reflect.TypeOf(Int{}).Size()))
	}
	s.capacity = n
	return nil
}
func (m *UPolynomialManager) reserve(s *upolyVectorState, n uint32) error {
	return m.reserveWithFactorNumeralVectorReceipt(s, n, nil)
}
func (m *UPolynomialManager) reserveWithFactorNumeralVectorReceipt(s *upolyVectorState, n uint32, r *polynomialFactorNumeralVectorReceipt) error {
	if n <= s.size {
		return nil
	}
	for n > s.capacity {
		if e := m.growWithFactorNumeralVectorReceipt(s, r); e != nil {
			return e
		}
	}
	old := s.size
	s.size = n // source resize publication precedes default construction
	a := m.values(s)
	for i := old; i < n; i++ {
		a[i] = Int{}
		if i+1 > m.stateStore(s).backings[s.backing-1].initialized {
			m.stateStore(s).backings[s.backing-1].initialized = i + 1
		}
	}
	if n > s.initialized {
		s.initialized = n
	}
	return nil
}
func (m *UPolynomialManager) freeBacking(s *upolyVectorState) {
	if s.backing != 0 {
		b := &m.stateStore(s).backings[s.backing-1]
		b.values = nil
		b.live = false
		if b.generation != math.MaxUint32 {
			b.generation++
		}
		m.config.Integer.budget.Release(uint64(s.capacity) * uint64(reflect.TypeOf(Int{}).Size()))
	}
	*s = upolyVectorState{}
}
func (m *UPolynomialManager) trim(s *upolyVectorState) error {
	a := m.values(s)
	n := s.size
	for n > 0 {
		zero, e := m.modular.IsZero(&a[n-1])
		if e != nil {
			return e
		}
		if !zero {
			break
		}
		if e = m.modular.Del(&a[n-1]); e != nil {
			return e
		}
		n--
	}
	s.size = n
	return nil
}
func (m *UPolynomialManager) setSize(s *upolyVectorState, n uint32) error {
	if modularSourceDebug && n > s.size {
		return ErrUPolynomialAssertion
	}
	if s.backing == 0 {
		return nil
	}
	if n > s.initialized {
		return arena.ErrBounds
	}
	a := m.values(s)
	for i := n; i < s.size; i++ {
		if e := m.modular.Del(&a[i]); e != nil {
			return e
		}
	}
	s.size = n
	return m.trim(s)
}
func (m *UPolynomialManager) reset(s *upolyVectorState) error { return m.setSize(s, 0) }
func (m *UPolynomialManager) Size(p UPolynomial) (uint32, error) {
	d, e := m.descriptor(p)
	if e != nil {
		return 0, e
	}
	return d.state.size, nil
}
func (m *UPolynomialManager) Capacity(p UPolynomial) (uint32, error) {
	d, e := m.descriptor(p)
	if e != nil {
		return 0, e
	}
	return d.state.capacity, nil
}
func (m *UPolynomialManager) Degree(p UPolynomial) (uint32, error) {
	n, e := m.Size(p)
	if e != nil {
		return 0, e
	}
	if n == 0 {
		return 0, nil
	}
	return n - 1, nil
}
func (m *UPolynomialManager) IsZero(p UPolynomial) (bool, error) {
	d, e := m.descriptor(p)
	if e != nil {
		return false, e
	}
	if e = m.trim(&d.state); e != nil {
		return false, e
	}
	return d.state.size == 0, nil
}
func (m *UPolynomialManager) IsConst(p UPolynomial) (bool, error) {
	d, e := m.descriptor(p)
	if e != nil {
		return false, e
	}
	if e = m.trim(&d.state); e != nil {
		return false, e
	}
	return d.state.size <= 1, nil
}
func (m *UPolynomialManager) Reset(p UPolynomial) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	return m.reset(&d.state)
}
func (m *UPolynomialManager) Trim(p UPolynomial) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	return m.trim(&d.state)
}
func (m *UPolynomialManager) SetSize(p UPolynomial, n uint32) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	return m.setSize(&d.state, n)
}
func (m *UPolynomialManager) Reserve(p UPolynomial, n uint32) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	return m.reserve(&d.state, n)
}
func (m *UPolynomialManager) Swap(p, q UPolynomial) error {
	a, e := m.descriptor(p)
	if e != nil {
		return e
	}
	b, e := m.descriptor(q)
	if e != nil {
		return e
	}
	a.state, b.state = b.state, a.state
	return nil
}
func (m *UPolynomialManager) View(p UPolynomial, start, count uint32) (UPolynomialView, error) {
	d, e := m.descriptor(p)
	if e != nil {
		return UPolynomialView{}, e
	}
	s := &d.state
	if uint64(start)+uint64(count) > uint64(s.size) {
		return UPolynomialView{}, arena.ErrBounds
	}
	return UPolynomialView{m.stateOrigin(s), s.backing, s.generation, start, count}, nil
}
func (m *UPolynomialManager) ReadView(v UPolynomialView, index uint32, out *Int) error {
	a, e := m.read(v, index)
	if e != nil {
		return e
	}
	return m.modular.Copy(out, a)
}
func (m *UPolynomialManager) ReadCoefficient(p UPolynomial, index uint32, out *Int) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	if index >= d.state.size {
		return arena.ErrBounds
	}
	return m.modular.Copy(out, &m.values(&d.state)[index])
}
func (m *UPolynomialManager) SetCoefficient(p UPolynomial, index uint32, n *Int) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	if index >= d.state.size {
		return arena.ErrBounds
	}
	return m.modular.Copy(&m.values(&d.state)[index], n)
}
func (m *UPolynomialManager) set(s *upolyVectorState, a []Int) error {
	if uint64(len(a)) > math.MaxUint32 {
		return arena.ErrBounds
	}
	n := uint32(len(a))
	if e := m.reserve(s, n); e != nil {
		return e
	}
	dst := m.values(s)
	for i := uint32(0); i < n; i++ {
		if e := m.modular.Copy(&dst[i], &a[i]); e != nil {
			return e
		}
	}
	return m.setSize(s, n)
}
func (m *UPolynomialManager) SetInts(p UPolynomial, a []Int) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	return m.set(&d.state, a)
}

// Borrowed input is checked at the actual p[i] access, after source reserve.
func (m *UPolynomialManager) SetView(p UPolynomial, v UPolynomialView) error {
	d, e := m.descriptor(p)
	if e != nil {
		return e
	}
	return m.setView(&d.state, v)
}
func (m *UPolynomialManager) alias(v UPolynomialView, s *upolyVectorState) bool {
	return s.backing != 0 && v.backing == s.backing && v.generation == s.generation && v.start == 0 && v.origin == m.stateOrigin(s)
}
func (m *UPolynomialManager) stateView(s *upolyVectorState) UPolynomialView {
	return UPolynomialView{m.stateOrigin(s), s.backing, s.generation, 0, s.size}
}
func (m *UPolynomialManager) read(v UPolynomialView, index uint32) (*Int, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	store := m.boundStore(v.origin)
	if store == nil || v.backing == 0 || uint64(v.backing) > uint64(len(store.backings)) {
		return nil, ErrUPolynomialOwner
	}
	b := &store.backings[v.backing-1]
	if !b.live || b.generation != v.generation {
		return nil, ErrUPolynomialOwner
	}
	if index >= v.count || uint64(v.start)+uint64(index) >= uint64(b.initialized) {
		return nil, arena.ErrBounds
	}
	return &b.values[v.start+index], nil
}
func (m *UPolynomialManager) setView(s *upolyVectorState, v UPolynomialView) error {
	return m.setViewWithHensel(s, v, nil)
}
func (m *UPolynomialManager) setViewWithHensel(s *upolyVectorState, v UPolynomialView, receipt *upolyHenselPairInvocationReceipt) error {
	context := m.modular.henselPairAllocationContext(receipt)
	if receipt != nil && context.henselPairReceipt(m.config.Integer) == nil {
		return ErrUPolynomialOwner
	}
	return m.setViewWithSourceContext(s, v, context)
}
func (m *UPolynomialManager) setViewWithSourceContext(s *upolyVectorState, v UPolynomialView, context integerAllocationInvocationContext) error {
	if m.alias(v, s) {
		if modularSourceDebug && s.size != v.count {
			return ErrUPolynomialAssertion
		}
		return nil
	}
	if e := m.reserve(s, v.count); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.copyWithSourceContext(&m.values(s)[i], a, context); e != nil {
			if q := context.removeOneHalfRootFrame(m.config.Integer); q != nil {
				q.recordFailedRow(s, i)
			}
			return e
		}
	}
	return m.setSize(s, v.count)
}
func (m *UPolynomialManager) Copy(dst, src UPolynomial) error {
	d, e := m.descriptor(src)
	if e != nil {
		return e
	}
	v, e := m.View(src, 0, d.state.size)
	if e != nil {
		return e
	}
	return m.SetView(dst, v)
}
