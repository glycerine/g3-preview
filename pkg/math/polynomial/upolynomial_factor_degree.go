// Licensed under the MIT license.
// Concrete factorization_degree_set and its actual bit_vector specialization.
package polynomial

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

type upolyFactorDegreeState struct {
	words []uint32
	bits  uint32
}

func upolyDegreeWords(bits uint32) uint32 { return (bits + 31) / 32 }
func (s *upolyFactorDegreeState) close(b *arena.Budget) {
	b.Release(uint64(cap(s.words)) * 4)
	*s = upolyFactorDegreeState{}
}
func (s *upolyFactorDegreeState) expand(b *arena.Budget, n uint32) error {
	if n < uint32(cap(s.words)) || uint64(n)*4 > math.MaxUint32 || uint64(n) > uint64(int(^uint(0)>>1))/4 {
		return arena.ErrBounds
	}
	bytes := uint64(n) * 4
	if e := b.Reserve(bytes); e != nil {
		return e
	}
	next := make([]uint32, n)
	copy(next, s.words)
	b.Release(uint64(cap(s.words)) * 4)
	s.words = next
	return nil
}
func (s *upolyFactorDegreeState) get(k uint32) (bool, error) {
	if polynomialSourceDebug && k >= s.bits {
		return false, ErrUPolynomialAssertion
	}
	if k/32 >= uint32(len(s.words)) {
		return false, arena.ErrBounds
	}
	return s.words[k/32]&(uint32(1)<<(k%32)) != 0, nil
}
func (s *upolyFactorDegreeState) set(k uint32, v bool) error {
	if polynomialSourceDebug && k >= s.bits {
		return ErrUPolynomialAssertion
	}
	if k/32 >= uint32(len(s.words)) {
		return arena.ErrBounds
	}
	mask := uint32(1) << (k % 32)
	if v {
		s.words[k/32] |= mask
	} else {
		s.words[k/32] &= ^mask
	}
	return nil
}
func (s *upolyFactorDegreeState) push(b *arena.Budget, value bool) error {
	idx := s.bits
	s.bits++
	// Check undefined wrapped num_words domain after the source size publication.
	if s.bits == 0 || s.bits > math.MaxUint32-31 {
		return arena.ErrBounds
	}
	if upolyDegreeWords(s.bits) > uint32(cap(s.words)) {
		n := uint32(2)
		if cap(s.words) != 0 {
			n = (uint32(cap(s.words))*3 + 1) >> 1
		}
		if e := s.expand(b, n); e != nil {
			return e
		}
	}
	return s.set(idx, value)
}
func (s *upolyFactorDegreeState) clone(b *arena.Budget, out *upolyFactorDegreeState) error {
	out.bits = s.bits
	if s.words == nil {
		return nil
	}
	if e := out.expand(b, uint32(cap(s.words))); e != nil {
		return e
	}
	copy(out.words, s.words)
	return nil
}
func (s *upolyFactorDegreeState) resize(b *arena.Budget, n uint32) error {
	if n <= s.bits {
		s.bits = n
		return nil
	}
	if n > math.MaxUint32-31 {
		return arena.ErrBounds
	}
	need := upolyDegreeWords(n)
	if need > uint32(cap(s.words)) {
		next := (need*3 + 1) >> 1
		if next < need {
			return arena.ErrBounds
		}
		if e := s.expand(b, next); e != nil {
			return e
		}
	}
	first, end := s.bits/32, need
	mask := uint32(1)<<(s.bits%32) - 1
	s.words[first] &= mask
	if first < end {
		for i := first + 1; i < end; i++ {
			s.words[i] = 0
		}
	}
	s.bits = n
	return nil
}
func (s *upolyFactorDegreeState) shift(b *arena.Budget, k uint32) error {
	if k == 0 {
		return nil
	}
	oldWords := upolyDegreeWords(s.bits)
	newBits := s.bits + k
	if newBits < s.bits {
		return arena.ErrBounds
	}
	if e := s.resize(b, newBits); e != nil {
		return e
	}
	shift, wordShift := k%32, k/32
	if wordShift > 0 {
		j, i := oldWords, oldWords+wordShift
		for j > 0 {
			j--
			i--
			s.words[i] = s.words[j]
		}
		for i > 0 {
			i--
			s.words[i] = 0
		}
	}
	if shift > 0 {
		if polynomialSourceDebug {
			for i := uint32(0); i < wordShift; i++ {
				if s.words[i] != 0 {
					return ErrUPolynomialAssertion
				}
			}
		}
		prev := uint32(0)
		for i := wordShift; i < upolyDegreeWords(newBits); i++ {
			next := s.words[i] >> (32 - shift)
			s.words[i] <<= shift
			s.words[i] |= prev
			prev = next
		}
	}
	return nil
}
func (s *upolyFactorDegreeState) or(b *arena.Budget, other *upolyFactorDegreeState) error {
	if s.bits < other.bits {
		if e := s.resize(b, other.bits); e != nil {
			return e
		}
	}
	n := upolyDegreeWords(other.bits)
	if n == 0 {
		return nil
	}
	rest := other.bits % 32
	for i := uint32(0); i < n; i++ {
		v := other.words[i]
		if rest != 0 && i == n-1 {
			v &= uint32(1)<<rest - 1
		}
		s.words[i] |= v
	}
	return nil
}
func (s *upolyFactorDegreeState) intersect(other *upolyFactorDegreeState) error {
	n1, n2 := upolyDegreeWords(s.bits), upolyDegreeWords(other.bits)
	if n1 == 0 {
		return nil
	}
	if n1 > uint32(len(s.words)) || n2 > uint32(len(other.words)) {
		return arena.ErrBounds
	}
	if n2 > n1 {
		for i := uint32(0); i < n1; i++ {
			s.words[i] &= other.words[i]
		}
		return nil
	}
	rest := other.bits % 32
	for i := uint32(0); i < n2; i++ {
		v := other.words[i]
		if rest != 0 && i == n2-1 {
			v &= uint32(1)<<rest - 1
		}
		s.words[i] &= v
	}
	for i := n2; i < n1; i++ {
		s.words[i] = 0
	}
	return nil
}
func (s *upolyFactorDegreeState) init(fs *UPolynomialFactors) error {
	m := fs.manager
	b := m.config.Integer.Budget()
	if e := s.push(b, true); e != nil {
		return e
	}
	for i := uint32(0); i < fs.count; i++ {
		degree := upolyStateDegree(&fs.factors[i])
		mult, e := fs.multiplicity(i)
		if e != nil {
			return e
		}
		for k := uint32(0); k < mult; k++ {
			if e = s.includeDegree(b, degree); e != nil {
				return e
			}
		}
	}
	if polynomialSourceDebug {
		zero, e := s.get(0)
		if e != nil {
			return e
		}
		if !zero {
			return ErrUPolynomialAssertion
		}
		total, e := s.get(fs.totalDegree)
		if e != nil {
			return e
		}
		if !total {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (s *upolyFactorDegreeState) includeDegree(b *arena.Budget, degree uint32) error {
	var tmp upolyFactorDegreeState
	defer tmp.close(b)
	if e := s.clone(b, &tmp); e != nil {
		return e
	}
	if e := s.shift(b, degree); e != nil {
		return e
	}
	return s.or(b, &tmp)
}
func (s *upolyFactorDegreeState) trivial() (bool, error) {
	// The actual source casts size to signed int before subtracting one.
	for i := int32(1); i < int32(s.bits)-1; i++ {
		yes, e := s.get(uint32(i))
		if e != nil {
			return false, e
		}
		if yes {
			return false, nil
		}
	}
	return true, nil
}

type UPolynomialDegreeSet struct {
	self   *UPolynomialDegreeSet
	budget *arena.Budget
	state  upolyFactorDegreeState
	header uint64
	closed bool
}
type UPolynomialDegreeSetStats struct {
	OwnerBytes, WordBytes uint64
	Bits, CapacityWords   uint32
}

func NewUPolynomialDegreeSet(fs *UPolynomialFactors) (*UPolynomialDegreeSet, error) {
	if e := fs.check(); e != nil {
		return nil, e
	}
	b := fs.manager.config.Integer.Budget()
	n := uint64(reflect.TypeOf(UPolynomialDegreeSet{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	d := &UPolynomialDegreeSet{budget: b, header: n}
	d.self = d
	if e := d.state.init(fs); e != nil {
		d.state.close(b)
		b.Release(n)
		return nil, e
	}
	return d, nil
}
func (d *UPolynomialDegreeSet) check() error {
	if d == nil || d.self != d {
		return ErrUPolynomialOwner
	}
	if d.closed {
		return arena.ErrClosed
	}
	return nil
}
func (d *UPolynomialDegreeSet) MaxDegree() (uint32, error) {
	if e := d.check(); e != nil {
		return 0, e
	}
	return d.state.bits - 1, nil
}
func (d *UPolynomialDegreeSet) InSet(k uint32) (bool, error) {
	if e := d.check(); e != nil {
		return false, e
	}
	return d.state.get(k)
}
func (d *UPolynomialDegreeSet) IsTrivial() (bool, error) {
	if e := d.check(); e != nil {
		return false, e
	}
	return d.state.trivial()
}
func (d *UPolynomialDegreeSet) Remove(k uint32) error {
	if e := d.check(); e != nil {
		return e
	}
	return d.state.set(k, false)
}
func (d *UPolynomialDegreeSet) Intersect(other *UPolynomialDegreeSet) error {
	if e := d.check(); e != nil {
		return e
	}
	if e := other.check(); e != nil {
		return e
	}
	return d.state.intersect(&other.state)
}
func (d *UPolynomialDegreeSet) Swap(other *UPolynomialDegreeSet) error {
	if e := d.check(); e != nil {
		return e
	}
	if e := other.check(); e != nil {
		return e
	}
	// Cold budget transfer only: reserve the positive physical-byte difference,
	// then refund the opposite owner before the infallible source payload swap.
	if d.budget != other.budget {
		a, b := uint64(cap(d.state.words))*4, uint64(cap(other.state.words))*4
		if b > a {
			if e := d.budget.Reserve(b - a); e != nil {
				return e
			}
			other.budget.Release(b - a)
		} else if a > b {
			if e := other.budget.Reserve(a - b); e != nil {
				return e
			}
			d.budget.Release(a - b)
		}
	}
	d.state, other.state = other.state, d.state
	return nil
}
func (d *UPolynomialDegreeSet) Close() error {
	if e := d.check(); e != nil {
		return e
	}
	d.state.close(d.budget)
	d.budget.Release(d.header)
	d.closed = true
	return nil
}
func (d *UPolynomialDegreeSet) Stats() UPolynomialDegreeSetStats {
	if d == nil || d.self != d || d.closed {
		return UPolynomialDegreeSetStats{}
	}
	return UPolynomialDegreeSetStats{OwnerBytes: d.header, WordBytes: uint64(cap(d.state.words)) * 4, Bits: d.state.bits, CapacityWords: uint32(cap(d.state.words))}
}
