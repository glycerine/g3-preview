// Source: src/sat/sat_types.h:153-204,229-263; src/sat/sat_probing.cpp.
package sat

import "github.com/glycerine/g3/pkg/arena"

// satProbingAssigned is the concrete literal set shared by probing and
// assumption tracking. Values retain first insertion order. Go owns storage.
type satProbingAssigned struct {
	membership []bool
	values     []Literal
}

func (s *satProbingAssigned) Insert(l Literal) error {
	v := l.index()
	// Source v+1 wraps at the invalid all-ones literal index. Guard that
	// source-undefined access before allocating or publishing membership.
	if v == ^uint32(0) {
		return arena.ErrBounds
	}
	n := int(v) + 1
	if n > len(s.membership) {
		s.membership = append(s.membership, make([]bool, n-len(s.membership))...)
	}
	if !s.membership[v] {
		s.membership[v] = true
		s.values = append(s.values, l)
	}
	return nil
}
func (s *satProbingAssigned) contains(l Literal) bool {
	return uint64(l.index()) < uint64(len(s.membership)) && s.membership[l.index()]
}
func (s *satProbingAssigned) size() uint32 { return uint32(len(s.values)) }
func (s *satProbingAssigned) Reset() {
	s.membership = s.membership[:0]
	s.values = s.values[:0]
}
func (s *satProbingAssigned) Finalize() {
	s.membership = nil
	s.values = nil
}
