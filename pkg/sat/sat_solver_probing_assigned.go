// Source: src/sat/sat_types.h; src/sat/sat_probing.cpp.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

// The probing-used sat_types.h uint_set/literal_set operations. Both actual
// source member buffers share the solver Runtime; enclosing header owns cold
// metadata once. This is not a replacement of the general set facade.
type satProbingAssigned struct {
	membership RuntimeVectorColumn
	values     RuntimeUintColumn
}

func (s *satProbingAssigned) bind(r *util.Runtime) { s.membership.Bind(r); s.values.Bind(r) }
func (s *satProbingAssigned) reserve(n uint32) error {
	old, e := s.membership.Size()
	if e != nil {
		return e
	}
	if n <= old {
		return nil
	}
	for {
		capacity, e := s.membership.Capacity()
		if e != nil {
			return e
		}
		if capacity >= n {
			break
		}
		if e = s.membership.Expand(); e != nil {
			return e
		}
	}
	// resize(n,false) publishes size before ascending element construction.
	if e = WriteRuntimeWord(s.membership.Runtime, s.membership.Block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = s.membership.Write(i, 0); e != nil {
			return e
		}
	}
	return nil
}
func (s *satProbingAssigned) Insert(l Literal) error {
	v := l.index()
	if e := s.reserve(v + 1); e != nil {
		return e
	}
	member, e := s.membership.Read(v)
	if e != nil {
		return e
	}
	if member != 0 {
		return nil
	}
	if e = s.membership.Write(v, 1); e != nil {
		return e
	} // source mark BEFORE list push.
	return s.values.Push(v)
}
func (s *satProbingAssigned) contains(l Literal) (bool, error) {
	n, e := s.membership.Size()
	if e != nil {
		return false, e
	}
	if l.index() >= n {
		return false, nil
	}
	v, e := s.membership.Read(l.index())
	return v != 0, e
}
func (s *satProbingAssigned) size() (uint32, error) { return s.values.Size() }
func (s *satProbingAssigned) Reset() error {
	if e := s.values.Shrink(0); e != nil {
		return e
	}
	return s.membership.Shrink(0)
}
func (s *satProbingAssigned) Finalize() error {
	// Both source explicit finalize and reverse declaration destructor order.
	if e := s.values.Finalize(); e != nil {
		return e
	}
	if e := s.membership.Finalize(); e != nil {
		return e
	}
	s.bind(s.membership.Runtime)
	s.values.Raw.Closed = false
	s.membership.Closed = false
	return nil
}
func (s *satProbingAssigned) Close() error { return s.Finalize() }
