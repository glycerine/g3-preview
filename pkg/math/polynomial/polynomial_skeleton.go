// Source: src/math/polynomial/polynomial.cpp:2958-3068; src/util/buffer.h; src/util/vector.h.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The actual entry codec stores scalar Monomial16 then first/num uint32. All
// three vectors contain meaningful source fields; there are no allocation
// tickets alongside mirrored authoritative payloads.
type PolynomialSkeletonEntry struct {
	Monomial              Monomial
	FirstPower, NumPowers uint32
}
type skeletonMonomialBuffer struct {
	initial        [128]Monomial
	raw, pending   util.Block
	size, capacity uint32
}
type PolynomialSkeleton struct {
	self                                                                               *PolynomialSkeleton
	owner                                                                              *MultivariatePolynomialCore
	x, maxPowers                                                                       uint32
	entries, powers, originals                                                         polynomialVector
	ms                                                                                 skeletonMonomialBuffer
	header                                                                             uint64
	originalCause, cleanupCause, sourceCause                                           error
	completed, closing, closed, sourceStarted, sourceFinalized                         bool
	sourceStage, unwindStage                                                           uint8
	entryOwned, originalOwned, entryCursor, originalCursor                             uint32
	entryTransferred, originalTransferred                                              uint32
	entryDropping, originalDropping                                                    bool
	incomingEntry, incomingOriginal, generated                                         Monomial
	incomingEntryDropping, incomingOriginalDropping, generatedActive, calleeCreation   bool
	retiredEntries, retiredOriginals                                                   []Monomial
	retiredEntryCount, retiredOriginalCount, retiredEntryCursor, retiredOriginalCursor uint32
	retiredDropping                                                                    bool
}

func skeletonReadMonomial(r *util.Runtime, b util.Block, o uint64) (Monomial, error) {
	lo, e := r.ReadUint32(b, o)
	if e != nil {
		return Monomial{}, e
	}
	hi, e := r.ReadUint32(b, o+4)
	if e != nil {
		return Monomial{}, e
	}
	slot, e := r.ReadUint32(b, o+8)
	if e != nil {
		return Monomial{}, e
	}
	gen, e := r.ReadUint32(b, o+12)
	return Monomial{uint64(lo) | uint64(hi)<<32, slot, gen}, e
}
func skeletonWriteMonomial(r *util.Runtime, b util.Block, o uint64, a Monomial) error {
	for i, v := range [4]uint32{uint32(a.owner), uint32(a.owner >> 32), a.slot, a.generation} {
		if e := r.WriteUint32(b, o+uint64(i)*4, v); e != nil {
			return e
		}
	}
	return nil
}
func (b *skeletonMonomialBuffer) get(m *MultivariatePolynomialCore, i uint32) (Monomial, error) {
	if i >= b.size {
		return Monomial{}, arena.ErrBounds
	}
	if b.raw == (util.Block{}) {
		return b.initial[i], nil
	}
	return skeletonReadMonomial(m.memory, b.raw, uint64(i)*16)
}
func (b *skeletonMonomialBuffer) set(m *MultivariatePolynomialCore, i uint32, a Monomial) error {
	if i >= b.capacity {
		return arena.ErrBounds
	}
	if b.raw == (util.Block{}) {
		b.initial[i] = a
		return nil
	}
	return skeletonWriteMonomial(m.memory, b.raw, uint64(i)*16, a)
}
func (b *skeletonMonomialBuffer) push(m *MultivariatePolynomialCore, a Monomial) error {
	if b.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	if b.size == b.capacity {
		next := b.capacity << 1
		var e error
		b.pending, e = m.memory.Allocate(uint64(next) * 16)
		if e != nil {
			return e
		}
		for i := uint32(0); i < b.size; i++ {
			v, e := b.get(m, i)
			if e != nil {
				return e
			}
			if e = skeletonWriteMonomial(m.memory, b.pending, uint64(i)*16, v); e != nil {
				return e
			}
		}
		if b.raw != (util.Block{}) {
			if e = m.memory.Deallocate(&b.raw); e != nil {
				return e
			}
		}
		b.raw = b.pending
		b.pending = util.Block{}
		b.capacity = next
	}
	if e := b.set(m, b.size, a); e != nil {
		return e
	}
	b.size++
	return nil
}

// Source std::sort is translated to concrete heap sort. Canonical final order
// is preserved; platform comparison dispatch/refinement order is not claimed.
func (b *skeletonMonomialBuffer) sort(s *PolynomialSkeleton) error {
	less := func(i, j uint32) (bool, error) {
		a, e := b.get(s.owner, i)
		if e != nil {
			return false, e
		}
		c, e := b.get(s.owner, j)
		if e != nil {
			return false, e
		}
		v, e := s.owner.monomials.LexWithMinimum(a, c, s.x)
		return v < 0, e
	}
	swap := func(i, j uint32) error {
		a, e := b.get(s.owner, i)
		if e != nil {
			return e
		}
		c, e := b.get(s.owner, j)
		if e != nil {
			return e
		}
		if e = b.set(s.owner, i, c); e != nil {
			return e
		}
		return b.set(s.owner, j, a)
	}
	sift := func(start, end uint32) error {
		root := start
		for root < end/2 {
			child := root*2 + 1
			if child >= end {
				return nil
			}
			if child+1 < end {
				v, e := less(child, child+1)
				if e != nil {
					return e
				}
				if v {
					child++
				}
			}
			v, e := less(root, child)
			if e != nil {
				return e
			}
			if !v {
				return nil
			}
			if e = swap(root, child); e != nil {
				return e
			}
			root = child
		}
		return nil
	}
	for i := b.size / 2; i > 0; i-- {
		if e := sift(i-1, b.size); e != nil {
			return e
		}
	}
	for end := b.size; end > 1; end-- {
		if e := swap(0, end-1); e != nil {
			return e
		}
		if e := sift(0, end-1); e != nil {
			return e
		}
	}
	return nil
}
func skeletonReserveRetired(s *PolynomialSkeleton, a *[]Monomial, need uint32) error {
	if uint64(need) <= uint64(cap(*a)) {
		return nil
	}
	old := cap(*a)
	n := old + old/2 + 1
	if uint64(n) < uint64(need) {
		n = int(need)
	}
	stride := uint64(reflect.TypeOf(Monomial{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := s.owner.integer.Budget().Reserve(uint64(n) * stride); e != nil {
		return e
	}
	v := make([]Monomial, n)
	copy(v, *a)
	*a = v
	s.owner.integer.Budget().Release(uint64(old) * stride)
	return nil
}
func NewPolynomialSkeleton(m *MultivariatePolynomialCore, p Polynomial, x uint32) (*PolynomialSkeleton, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if _, e := m.polynomialLocation(p); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialSkeleton{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	s := &PolynomialSkeleton{owner: m, x: x, header: h}
	s.self = s
	s.entries.width = 24
	s.powers.width = 4
	s.originals.width = 16
	s.ms.capacity = 128
	fail := func(e error) (*PolynomialSkeleton, error) {
		s.originalCause = e
		s.cleanupCause = s.unwindConstructor()
		return s, s
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return fail(e)
	}
	for i := uint32(0); i < n; i++ {
		a, e := m.monomialCore(p, i)
		if e != nil {
			return fail(e)
		}
		if e = s.ms.push(m, a); e != nil {
			return fail(e)
		}
	}
	if e = s.ms.sort(s); e != nil {
		return fail(e)
	}
	var previous Monomial
	for i := uint32(0); i < n; i++ {
		orig, e := s.ms.get(m, i)
		if e != nil {
			return fail(e)
		}
		k, e := m.monomials.DegreeOf(orig, x)
		if e != nil {
			return fail(e)
		}
		a := orig
		if k > 0 {
			var outcome monomialFactoryOutcome
			a, e = m.monomials.removeVariableCreated(orig, x, &outcome)
			s.calleeCreation = outcome.started
			if e != nil {
				return fail(e)
			}
			s.calleeCreation = false
			if outcome.created {
				s.generated = a
				s.generatedActive = true
			}
		}
		if a == previous {
			count, e := s.entries.size(m)
			if e != nil {
				return fail(e)
			}
			if count == 0 {
				return fail(arena.ErrBounds)
			}
			o := uint64(8) + uint64(count-1)*24 + 20
			v, e := m.memory.ReadUint32(s.entries.raw, o)
			if e != nil {
				return fail(e)
			}
			v++
			if e = m.memory.WriteUint32(s.entries.raw, o, v); e != nil {
				return fail(e)
			}
			if v > s.maxPowers {
				s.maxPowers = v
			}
		} else {
			previous = a
			// This reservation is real cold Go ownership metadata, not a source
			// malloc/push. The source vector allocation still follows actual INC.
			if e = skeletonReserveRetired(s, &s.retiredEntries, s.entryOwned+1); e != nil {
				return fail(e)
			}
			if e = m.monomials.IncRef(a); e != nil {
				return fail(e)
			}
			s.incomingEntry = a
			s.generated = Monomial{}
			s.generatedActive = false
			index, e := s.entries.appendSlot(m)
			if e != nil {
				return fail(e)
			}
			if e = skeletonWriteMonomial(m.memory, s.entries.raw, 8+uint64(index)*24, a); e != nil {
				return fail(e)
			}
			first, e := s.powers.size(m)
			if e != nil {
				return fail(e)
			}
			if e = m.memory.WriteUint32(s.entries.raw, 24+uint64(index)*24, first); e != nil {
				return fail(e)
			}
			if e = m.memory.WriteUint32(s.entries.raw, 28+uint64(index)*24, 1); e != nil {
				return fail(e)
			}
			s.entryOwned = index + 1
			s.entries.initialized = index + 1
			s.incomingEntry = Monomial{}
			if e = s.entries.publishSize(m, index+1); e != nil {
				return fail(e)
			}
			if s.maxPowers == 0 {
				s.maxPowers = 1
			}
		}
		if e = skeletonReserveRetired(s, &s.retiredOriginals, s.originalOwned+1); e != nil {
			return fail(e)
		}
		if e = m.monomials.IncRef(orig); e != nil {
			return fail(e)
		}
		s.incomingOriginal = orig
		index, e := s.originals.appendSlot(m)
		if e != nil {
			return fail(e)
		}
		if e = s.originals.putMonomial(m, index, orig); e != nil {
			return fail(e)
		}
		s.originalOwned = index + 1
		s.originals.initialized = index + 1
		s.incomingOriginal = Monomial{}
		if e = s.originals.publishSize(m, index+1); e != nil {
			return fail(e)
		}
		index, e = s.powers.appendSlot(m)
		if e != nil {
			return fail(e)
		}
		if e = m.memory.WriteUint32(s.powers.raw, 8+uint64(index)*4, k); e != nil {
			return fail(e)
		}
		s.powers.initialized = index + 1
		if e = s.powers.publishSize(m, index+1); e != nil {
			return fail(e)
		}
	}
	if polynomialSourceDebug {
		n, e := s.entries.size(m)
		if e != nil {
			return fail(e)
		}
		for i := uint32(1); i < n; i++ {
			a, e := s.entryMonomialCore(i - 1)
			if e != nil {
				return fail(e)
			}
			b, e := s.entryMonomialCore(i)
			if e != nil {
				return fail(e)
			}
			v, e := m.monomials.Lex(a, b)
			if e != nil {
				return fail(e)
			}
			if v >= 0 {
				return fail(ErrPolynomialAssertion)
			}
		}
	}
	if s.ms.raw != (util.Block{}) {
		if e = m.memory.Deallocate(&s.ms.raw); e != nil {
			return fail(e)
		}
	}
	s.completed = true
	return s, nil
}
func (s *PolynomialSkeleton) Error() string {
	if s.originalCause != nil {
		return s.originalCause.Error()
	}
	if s.cleanupCause != nil {
		return s.cleanupCause.Error()
	}
	return ErrPolynomialPending.Error()
}
func (s *PolynomialSkeleton) Unwrap() error        { return s.originalCause }
func (s *PolynomialSkeleton) OriginalCause() error { return s.originalCause }
func (s *PolynomialSkeleton) CleanupCause() error  { return s.cleanupCause }
func (s *PolynomialSkeleton) ready() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	if s.closed || s.closing || s.sourceStarted || s.originalCause != nil {
		return arena.ErrClosed
	}
	return s.owner.check()
}
func (s *PolynomialSkeleton) entryIndexGuard(i uint32) error {
	if polynomialSourceDebug {
		n, e := s.entries.size(s.owner)
		if e != nil {
			return e
		}
		if i >= n {
			return ErrPolynomialAssertion
		}
	}
	return nil
}
func (s *PolynomialSkeleton) ownedEntryMonomial(i uint32) (Monomial, error) {
	return skeletonReadMonomial(s.owner.memory, s.entries.raw, 8+uint64(i)*24)
}
func (s *PolynomialSkeleton) entryMonomialCore(i uint32) (Monomial, error) {
	if polynomialSourceDebug {
		n, e := s.entries.size(s.owner)
		if e != nil {
			return Monomial{}, e
		}
		if i >= n {
			return Monomial{}, ErrPolynomialAssertion
		}
	}
	return skeletonReadMonomial(s.owner.memory, s.entries.raw, 8+uint64(i)*24)
}
func (s *PolynomialSkeleton) entryCore(i uint32) (PolynomialSkeletonEntry, error) {
	a, e := skeletonReadMonomial(s.owner.memory, s.entries.raw, 8+uint64(i)*24)
	if e != nil {
		return PolynomialSkeletonEntry{}, e
	}
	first, e := s.owner.memory.ReadUint32(s.entries.raw, 24+uint64(i)*24)
	if e != nil {
		return PolynomialSkeletonEntry{}, e
	}
	num, e := s.owner.memory.ReadUint32(s.entries.raw, 28+uint64(i)*24)
	return PolynomialSkeletonEntry{a, first, num}, e
}
func (s *PolynomialSkeleton) NumEntries() (uint32, error) {
	if e := s.ready(); e != nil {
		return 0, e
	}
	return s.entries.size(s.owner)
}
func (s *PolynomialSkeleton) MaxNumPowers() (uint32, error) {
	if e := s.ready(); e != nil {
		return 0, e
	}
	return s.maxPowers, nil
}
func (s *PolynomialSkeleton) Entry(i uint32) (PolynomialSkeletonEntry, error) {
	if e := s.ready(); e != nil {
		return PolynomialSkeletonEntry{}, e
	}
	return s.entryCore(i)
}
func (s *PolynomialSkeleton) EntryIndex(a Monomial) (uint32, error) {
	if e := s.ready(); e != nil {
		return 0, e
	}
	n, e := s.entries.size(s.owner)
	if e != nil {
		return 0, e
	}
	for i := uint32(0); i < n; i++ {
		entry, e := s.entryMonomialCore(i)
		if e != nil {
			return 0, e
		}
		if a == entry {
			return i, nil
		}
	}
	return ^uint32(0), nil
}
func (s *PolynomialSkeleton) Power(a PolynomialSkeletonEntry, i uint32) (uint32, error) {
	if e := s.ready(); e != nil {
		return 0, e
	}
	if polynomialSourceDebug && i >= a.NumPowers {
		return 0, ErrPolynomialAssertion
	}
	index := a.FirstPower + i
	if polynomialSourceDebug {
		n, e := s.powers.size(s.owner)
		if e != nil {
			return 0, e
		}
		if index >= n {
			return 0, ErrPolynomialAssertion
		}
	}
	return s.owner.memory.ReadUint32(s.powers.raw, 8+uint64(index)*4)
}
func (s *PolynomialSkeleton) Original(a PolynomialSkeletonEntry, i uint32) (Monomial, error) {
	if e := s.ready(); e != nil {
		return Monomial{}, e
	}
	if polynomialSourceDebug && i >= a.NumPowers {
		return Monomial{}, ErrPolynomialAssertion
	}
	index := a.FirstPower + i
	if polynomialSourceDebug {
		n, e := s.originals.size(s.owner)
		if e != nil {
			return Monomial{}, e
		}
		if index >= n {
			return Monomial{}, ErrPolynomialAssertion
		}
	}
	return s.originals.monomial(s.owner, index)
}

// The failed class constructor has no skeleton body destructor. Completed
// locals/members free only raw backing; actual leaked INC fields transfer into
// charged owning-abort records before their source backing is consumed.
func (s *PolynomialSkeleton) unwindConstructor() error {
	m := s.owner
	if s.unwindStage == 0 {
		if s.ms.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&s.ms.raw); e != nil {
				return e
			}
		}
		s.unwindStage++
	}
	if s.unwindStage == 1 {
		for s.originalTransferred < s.originalOwned {
			a, e := s.originals.monomial(m, s.originalTransferred)
			if e != nil {
				return e
			}
			s.retiredOriginals[s.retiredOriginalCount] = a
			s.retiredOriginalCount++
			s.originalTransferred++
		}
		if s.originals.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&s.originals.raw); e != nil {
				return e
			}
		}
		s.unwindStage++
	}
	if s.unwindStage == 2 {
		if s.powers.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&s.powers.raw); e != nil {
				return e
			}
		}
		s.unwindStage++
	}
	if s.unwindStage == 3 {
		for s.entryTransferred < s.entryOwned {
			a, e := s.ownedEntryMonomial(s.entryTransferred)
			if e != nil {
				return e
			}
			s.retiredEntries[s.retiredEntryCount] = a
			s.retiredEntryCount++
			s.entryTransferred++
		}
		if s.entries.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&s.entries.raw); e != nil {
				return e
			}
		}
		s.unwindStage++
	}
	return nil
}
func skeletonDrop(s *PolynomialSkeleton, a *Monomial, started *bool, owning bool) error {
	if *a == (Monomial{}) {
		return nil
	}
	if *started {
		if !owning {
			return s.sourceCause
		}
		return s.owner.monomials.CleanupValue(a)
	}
	e := s.owner.monomials.decRefCore(*a)
	if e != nil {
		l, x := s.owner.monomials.location(*a)
		if x == nil && l.stage != 0 {
			*started = true
		}
		return e
	}
	*a = Monomial{}
	return nil
}
func (s *PolynomialSkeleton) destroyEntries(owning bool) error {
	limit := s.entryOwned
	if !owning {
		n, e := s.entries.size(s.owner)
		if e != nil {
			return e
		}
		limit = n
	}
	for s.entryCursor < limit {
		if s.entryCursor < s.entryTransferred {
			s.entryCursor++
			continue
		}
		var entry Monomial
		var e error
		if owning {
			entry, e = s.ownedEntryMonomial(s.entryCursor)
		} else {
			entry, e = s.entryMonomialCore(s.entryCursor)
		}
		if e != nil {
			return e
		}
		if e = skeletonDrop(s, &entry, &s.entryDropping, owning); e != nil {
			return e
		}
		s.entryDropping = false
		s.entryCursor++
	}
	return nil
}
func (s *PolynomialSkeleton) destroyOriginals(owning bool) error {
	limit := s.originalOwned
	if !owning {
		n, e := s.originals.size(s.owner)
		if e != nil {
			return e
		}
		limit = n
	}
	for s.originalCursor < limit {
		if s.originalCursor < s.originalTransferred {
			s.originalCursor++
			continue
		}
		a, e := s.originals.monomial(s.owner, s.originalCursor)
		if e != nil {
			return e
		}
		if e = skeletonDrop(s, &a, &s.originalDropping, owning); e != nil {
			return e
		}
		s.originalDropping = false
		s.originalCursor++
	}
	return nil
}
func (s *PolynomialSkeleton) SourceFinalize() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	if s.closed || s.closing || !s.completed {
		return arena.ErrClosed
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if s.sourceFinalized {
		return nil
	}
	s.sourceStarted = true
	fail := func(e error) error { s.sourceCause = e; return e }
	if s.sourceStage == 0 {
		if e := s.destroyEntries(false); e != nil {
			return fail(e)
		}
		s.sourceStage++
	}
	if s.sourceStage == 1 {
		if e := s.destroyOriginals(false); e != nil {
			return fail(e)
		}
		s.sourceStage++
	}
	vectors := [3]*polynomialVector{&s.originals, &s.powers, &s.entries}
	for s.sourceStage >= 2 && s.sourceStage < 5 {
		v := vectors[s.sourceStage-2]
		if v.raw != (util.Block{}) {
			if e := s.owner.memory.Deallocate(&v.raw); e != nil {
				return fail(e)
			}
		}
		s.sourceStage++
	}
	s.sourceFinalized = true
	return nil
}
func (s *PolynomialSkeleton) CloseChecked() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	if s.closed {
		return nil
	}
	s.closing = true
	if e := s.destroyEntries(true); e != nil {
		return e
	}
	if e := s.destroyOriginals(true); e != nil {
		return e
	}
	for s.retiredEntryCursor < s.retiredEntryCount {
		if e := skeletonDrop(s, &s.retiredEntries[s.retiredEntryCursor], &s.retiredDropping, true); e != nil {
			return e
		}
		s.retiredDropping = false
		s.retiredEntryCursor++
	}
	for s.retiredOriginalCursor < s.retiredOriginalCount {
		if e := skeletonDrop(s, &s.retiredOriginals[s.retiredOriginalCursor], &s.retiredDropping, true); e != nil {
			return e
		}
		s.retiredDropping = false
		s.retiredOriginalCursor++
	}
	if e := skeletonDrop(s, &s.incomingEntry, &s.incomingEntryDropping, true); e != nil {
		return e
	}
	if e := skeletonDrop(s, &s.incomingOriginal, &s.incomingOriginalDropping, true); e != nil {
		return e
	}
	if s.generatedActive {
		if e := s.owner.monomials.CleanupValue(&s.generated); e != nil {
			return e
		}
		s.generatedActive = false
	}
	if s.calleeCreation {
		if e := s.owner.monomials.CleanupCreation(); e != nil {
			return e
		}
		s.calleeCreation = false
	}
	for _, b := range []*util.Block{&s.ms.raw, &s.ms.pending} {
		if *b != (util.Block{}) {
			if e := s.owner.memory.Deallocate(b); e != nil {
				return e
			}
		}
	}
	for _, v := range []*polynomialVector{&s.originals, &s.powers, &s.entries} {
		if e := v.close(s.owner); e != nil {
			return e
		}
	}
	b := s.owner.integer.Budget()
	b.Release(s.header + uint64(cap(s.retiredEntries)+cap(s.retiredOriginals))*uint64(reflect.TypeOf(Monomial{}).Size()))
	s.header = 0
	s.retiredEntries = nil
	s.retiredOriginals = nil
	s.closed = true
	return nil
}

// DisplayInto uses a borrowed cold transport. The entry is an actual reference
// represented by its scalar index, so loop tests/fields are re-read at reached
// accesses after output callbacks instead of cached snapshot values.
func (s *PolynomialSkeleton) DisplayInto(out MonomialTextSink) error {
	if e := s.ready(); e != nil {
		return e
	}
	n, e := s.entries.size(s.owner)
	if e != nil {
		return e
	}
	field := func(i uint32, o uint64) (uint32, error) {
		if e := s.ready(); e != nil {
			return 0, e
		}
		return s.owner.memory.ReadUint32(s.entries.raw, 8+uint64(i)*24+o)
	}
	for i := uint32(0); i < n; i++ {
		if e := s.entryIndexGuard(i); e != nil {
			return e
		}
		if i > 0 {
			if e = monomialSinkLiteral(out, " "); e != nil {
				return e
			}
		}
		if e = monomialSinkLiteral(out, "("); e != nil {
			return e
		}
		for j := uint32(0); ; j++ {
			count, e := field(i, 20)
			if e != nil {
				return e
			}
			if j >= count {
				break
			}
			if j > 0 {
				if e = monomialSinkLiteral(out, " "); e != nil {
					return e
				}
			}
			if e = monomialSinkLiteral(out, "x"); e != nil {
				return e
			}
			if e = s.ready(); e != nil {
				return e
			}
			if e = monomialSinkUnsigned(out, s.x); e != nil {
				return e
			}
			if e = monomialSinkLiteral(out, "^"); e != nil {
				return e
			}
			first, e := field(i, 16)
			if e != nil {
				return e
			}
			index := first + j
			if polynomialSourceDebug {
				size, e := s.powers.size(s.owner)
				if e != nil {
					return e
				}
				if index >= size {
					return ErrPolynomialAssertion
				}
			}
			k, e := s.owner.memory.ReadUint32(s.powers.raw, 8+uint64(index)*4)
			if e != nil {
				return e
			}
			if e = monomialSinkUnsigned(out, k); e != nil {
				return e
			}
		}
		if e = monomialSinkLiteral(out, ")*"); e != nil {
			return e
		}
		if e = s.ready(); e != nil {
			return e
		}
		a, e := s.ownedEntryMonomial(i)
		if e != nil {
			return e
		}
		if e = s.owner.monomials.DisplayInto(a, out, false); e != nil {
			return e
		}
	}
	return nil
}
