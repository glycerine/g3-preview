// Source: src/util/nat_set.h:25-85.
package util

import "math"

// NatSet is a fixed-domain set of natural numbers with constant-time Reset.
// Call AssureDomain or SetDomain before accessing a value outside the current
// domain.
type NatSet struct {
	current    uint32
	timestamps []uint32
}

// NewNatSet creates an empty set with domain [0, domain).
func NewNatSet(domain uint32) *NatSet {
	return &NatSet{timestamps: make([]uint32, int(domain))}
}

// SetDomain changes the domain to [0, domain). Existing entries inside the
// retained prefix keep their membership.
func (s *NatSet) SetDomain(domain uint32) {
	if int(domain) <= len(s.timestamps) {
		s.timestamps = s.timestamps[:int(domain)]
		return
	}
	next := make([]uint32, int(domain))
	copy(next, s.timestamps)
	s.timestamps = next
}

// GetDomain returns the exclusive upper bound of values in the set's domain.
func (s *NatSet) GetDomain() uint32 {
	return uint32(len(s.timestamps))
}

// AssureDomain grows the domain just enough to include value.
func (s *NatSet) AssureDomain(value uint32) {
	if value >= s.GetDomain() {
		s.SetDomain(value + 1)
	}
}

// Contains reports whether value is present. Value must be in the domain.
func (s *NatSet) Contains(value uint32) bool {
	return s.timestamps[value] > s.current
}

// Insert adds value. Value must be in the domain.
func (s *NatSet) Insert(value uint32) {
	s.timestamps[value] = s.current + 1
}

// Remove removes value. Value must be in the domain.
func (s *NatSet) Remove(value uint32) {
	s.timestamps[value] = s.current
}

// Reset removes every value without scanning the domain, except when the
// timestamp wraps and the backing timestamps must be cleared.
func (s *NatSet) Reset() {
	s.current++
	if s.current == math.MaxUint32 {
		clear(s.timestamps)
		s.current = 0
	}
}

// Empty reports whether the set contains no values.
func (s *NatSet) Empty() bool {
	for _, timestamp := range s.timestamps {
		if timestamp > s.current {
			return false
		}
	}
	return true
}
