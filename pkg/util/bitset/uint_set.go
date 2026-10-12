// Translation of src/util/uint_set.h, Leonardo de Moura, 2006-12-07.
package bitset

import (
	"math/bits"
	"strconv"
	"strings"
)

// UintSet owns packed uint32 words. Clone/Assign explicitly copy ownership;
// assigning the Go struct aliases storage. No Go map chooses set iteration order.
type UintSet struct{ words []uint32 }

func (s *UintSet) CapacityWords() uint32 { return uint32(cap(s.words)) }
func (s *UintSet) expand() {
	old := s.CapacityWords()
	next := uint32(2)
	if old != 0 {
		next = (3*old + 1) >> 1
		if next <= old || uint64(next)*4+8 > uint64(^uint32(0)) {
			panic("bitset: source vector capacity overflow")
		}
	}
	if uint64(next) > uint64(^uint(0)>>1)/4 {
		panic("bitset: vector capacity exceeds platform size")
	}
	words := make([]uint32, len(s.words), next)
	copy(words, s.words)
	s.words = words
}
func (s *UintSet) resize(n uint32) {
	// get_max_elem is a uint32 exclusive iteration bound. Source overflow at
	// 2^27 words would hide members from iteration/display; diagnose the size.
	if uint64(n)*32 > uint64(^uint32(0)) {
		panic("bitset: uint_set extent exceeds source uint32 range")
	}
	old := uint32(len(s.words))
	if n <= old {
		s.words = s.words[:n]
		return
	}
	for n > s.CapacityWords() {
		s.expand()
	}
	s.words = s.words[:n]
	clear(s.words[old:])
}
func (s *UintSet) pushWord(w uint32) {
	n := uint32(len(s.words))
	if n == s.CapacityWords() {
		s.expand()
	}
	s.words = s.words[:n+1]
	s.words[n] = w
}

// Clone follows the uint_set copy constructor's push_back loop, which derives
// capacity from logical source words rather than copying spare source capacity.
func (s *UintSet) Clone() *UintSet {
	r := new(UintSet)
	for _, w := range s.words {
		r.pushWord(w)
	}
	return r
}

// Assign translates implicitly generated uint_set assignment through vector's
// copy assignment, including allocation at the source's capacity.
func (s *UintSet) Assign(src *UintSet) {
	if s == src {
		return
	}
	if cap(src.words) == 0 {
		s.words = nil
		return
	}
	words := make([]uint32, len(src.words), cap(src.words))
	copy(words, src.words)
	s.words = words
}
func (s *UintSet) Swap(other *UintSet) { s.words, other.words = other.words, s.words }

// MaxElem is the source's exclusive word-length endpoint, not largest member.
func (s *UintSet) MaxElem() uint32 { return 32 * uint32(len(s.words)) }

// Reset preserves word capacity; subsequent resize default-initializes words.
func (s *UintSet) Reset() { s.words = s.words[:0] }

// Close releases owned backing storage, representing the source destructor.
func (s *UintSet) Close() { s.words = nil }
func (s *UintSet) Empty() bool {
	for _, w := range s.words {
		if w != 0 {
			return false
		}
	}
	return true
}
func (s *UintSet) Insert(v uint32) {
	i := v >> 5
	if i >= uint32(len(s.words)) {
		s.resize(i + 1)
	}
	s.words[i] |= uint32(1) << (v & 31)
}
func (s *UintSet) Remove(v uint32) {
	i := v >> 5
	if i < uint32(len(s.words)) {
		s.words[i] &= ^(uint32(1) << (v & 31))
	}
}
func (s *UintSet) Contains(v uint32) bool {
	i := v >> 5
	return i < uint32(len(s.words)) && s.words[i]&(uint32(1)<<(v&31)) != 0
}

// NumElems uses the Go equivalent of the source GCC __builtin_popcount.
func (s *UintSet) NumElems() uint32 {
	var n uint32
	for _, w := range s.words {
		n += uint32(bits.OnesCount32(w))
	}
	return n
}
func (s *UintSet) Union(src *UintSet) {
	n := uint32(len(src.words))
	if n > uint32(len(s.words)) {
		s.resize(n + 1)
	}
	for i := uint32(0); i < n; i++ {
		s.words[i] |= src.words[i]
	}
}
func (s *UintSet) Intersect(src *UintSet) {
	n := uint32(len(src.words))
	if n < uint32(len(s.words)) {
		s.resize(n)
	}
	for i := range s.words {
		s.words[i] &= src.words[i]
	}
}
func (s *UintSet) Equal(src *UintSet) bool {
	n := len(s.words)
	if len(src.words) < n {
		n = len(src.words)
	}
	for i := 0; i < n; i++ {
		if s.words[i] != src.words[i] {
			return false
		}
	}
	for i := n; i < len(s.words); i++ {
		if s.words[i] != 0 {
			return false
		}
	}
	for i := n; i < len(src.words); i++ {
		if src.words[i] != 0 {
			return false
		}
	}
	return true
}
func (s *UintSet) SubsetOf(src *UintSet) bool {
	n := len(s.words)
	if len(src.words) < n {
		n = len(src.words)
	}
	for i := 0; i < n; i++ {
		if s.words[i]&^src.words[i] != 0 {
			return false
		}
	}
	for i := n; i < len(s.words); i++ {
		if s.words[i] != 0 {
			return false
		}
	}
	return true
}

// Hash preserves uint_set's weighted-word sum and uint32 wraparound. This source
// container does not use string_hash/combine_hash from the hash utility header.
func (s *UintSet) Hash() uint32 {
	var h uint32
	for i, w := range s.words {
		h += (uint32(i) + 1) * w
	}
	return h
}

func (s *UintSet) String() string {
	var out strings.Builder
	out.WriteByte('{')
	first := true
	s.ForEach(func(i uint32) bool {
		if first {
			first = false
		} else {
			out.WriteString(", ")
		}
		out.WriteString(strconv.FormatUint(uint64(i), 10))
		return true
	})
	out.WriteByte('}')
	return out.String()
}

// ForEach visits members in ascending order. Returning false stops traversal.
func (s *UintSet) ForEach(yield func(uint32) bool) {
	for wi, word := range s.words {
		for word != 0 {
			bit := uint32(bits.TrailingZeros32(word))
			if !yield(uint32(wi)*32 + bit) {
				return
			}
			word &^= uint32(1) << bit
		}
	}
}
