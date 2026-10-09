// Source: src/util/nat_set.h:25-85.
package natset

import "math"

// Set is a fixed-domain set of natural numbers with constant-time reset.
type Set struct {
	current    uint32
	timestamps []uint32
}

// New creates an empty set with domain [0, domain).
func New(domain uint32) *Set {
	return &Set{timestamps: make([]uint32, int(domain))}
}

// SetDomain changes the domain to [0, domain). Existing entries inside the
// retained prefix keep their membership.
func (s *Set) SetDomain(domain uint32) {
	if int(domain) <= len(s.timestamps) {
		s.timestamps = s.timestamps[:int(domain)]
		return
	}
	next := make([]uint32, int(domain))
	copy(next, s.timestamps)
	s.timestamps = next
}

func (s *Set) Domain() uint32 { return uint32(len(s.timestamps)) }

func (s *Set) AssureDomain(value uint32) {
	if value >= s.Domain() {
		s.SetDomain(value + 1)
	}
}

func (s *Set) Contains(value uint32) bool { return s.timestamps[value] > s.current }
func (s *Set) Insert(value uint32)        { s.timestamps[value] = s.current + 1 }
func (s *Set) Remove(value uint32)        { s.timestamps[value] = s.current }

func (s *Set) Reset() {
	s.current++
	if s.current == math.MaxUint32 {
		clear(s.timestamps)
		s.current = 0
	}
}

func (s *Set) Empty() bool {
	for _, timestamp := range s.timestamps {
		if timestamp > s.current {
			return false
		}
	}
	return true
}
