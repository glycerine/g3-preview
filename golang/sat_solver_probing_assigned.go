// Source: src/sat/sat_types.h; src/sat/sat_probing.cpp. Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/memory"

// The probing-used sat_types.h uint_set/literal_set operations. Both actual
// source member buffers share the solver Runtime; enclosing header owns cold
// metadata once. This is not a replacement of the general set facade.
type satProbingAssigned struct {
	membership satSolverAssignmentColumn
	values     satSolverRuntimeUintColumn
}

func (s *satProbingAssigned) bind(r *memory.Runtime) { s.membership.bind(r); s.values.bind(r) }
func (s *satProbingAssigned) reserve(n uint32) error {
	old, e := s.membership.size()
	if e != nil {
		return e
	}
	if n <= old {
		return nil
	}
	for {
		capacity, e := s.membership.capacity()
		if e != nil {
			return e
		}
		if capacity >= n {
			break
		}
		if e = s.membership.expand(); e != nil {
			return e
		}
	}
	// resize(n,false) publishes size before ascending element construction.
	if e = satAssignmentWriteWord(s.membership.runtime, s.membership.block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = s.membership.write(i, 0); e != nil {
			return e
		}
	}
	return nil
}
func (s *satProbingAssigned) Insert(l satLiteral) error {
	v := l.index()
	if e := s.reserve(v + 1); e != nil {
		return e
	}
	member, e := s.membership.read(v)
	if e != nil {
		return e
	}
	if member != 0 {
		return nil
	}
	if e = s.membership.write(v, 1); e != nil {
		return e
	} // source mark BEFORE list push.
	return s.values.push(v)
}
func (s *satProbingAssigned) contains(l satLiteral) (bool, error) {
	n, e := s.membership.size()
	if e != nil {
		return false, e
	}
	if l.index() >= n {
		return false, nil
	}
	v, e := s.membership.read(l.index())
	return v != 0, e
}
func (s *satProbingAssigned) size() (uint32, error) { return s.values.size() }
func (s *satProbingAssigned) Reset() error {
	if e := s.values.shrink(0); e != nil {
		return e
	}
	return s.membership.shrink(0)
}
func (s *satProbingAssigned) Finalize() error {
	// Both source explicit finalize and reverse declaration destructor order.
	if e := s.values.finalize(); e != nil {
		return e
	}
	if e := s.membership.finalize(); e != nil {
		return e
	}
	s.bind(s.membership.runtime)
	s.values.raw.closed = false
	s.membership.closed = false
	return nil
}
func (s *satProbingAssigned) Close() error { return s.Finalize() }
