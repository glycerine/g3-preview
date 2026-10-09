// Source: src/sat/sat_types.h:153–289; src/util/vector.h:102–120,187–201.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

// Actual literal_set members. The enclosing charged lexical owner includes
// this cold metadata; both source buffers belong to the solver Runtime.
type satMUSLiteralSet struct {
	membership RuntimeVectorColumn
	values     RuntimeUintColumn
}

func (s *satMUSLiteralSet) bind(r *util.Runtime) { s.membership.Bind(r); s.values.Bind(r) }
func (s *satMUSLiteralSet) insert(l Literal) error {
	// Existing concrete source uint_set implementation, with identical buffers.
	return satMUSSetInsert(&s.membership, &s.values, l)
}
func satMUSSetInsert(m *RuntimeVectorColumn, v *RuntimeUintColumn, l Literal) error {
	old, e := m.Size()
	if e != nil {
		return e
	}
	n := l.index() + 1
	if n > old {
		for {
			c, e := m.Capacity()
			if e != nil {
				return e
			}
			if c >= n {
				break
			}
			if e = m.Expand(); e != nil {
				return e
			}
		}
		if e = WriteRuntimeWord(m.Runtime, m.Block, 4, n); e != nil {
			return e
		}
		for i := old; i < n; i++ {
			if e = m.Write(i, 0); e != nil {
				return e
			}
		}
	}
	b, e := m.Read(l.index())
	if e != nil {
		return e
	}
	if b != 0 {
		return nil
	}
	if e = m.Write(l.index(), 1); e != nil {
		return e
	}
	return v.Push(l.index())
}
func (s *satMUSLiteralSet) size() (uint32, error) { return s.values.Size() }
func (s *satMUSLiteralSet) contains(l Literal) (bool, error) {
	n, e := s.membership.Size()
	if e != nil {
		return false, e
	}
	if l.index() >= n {
		return false, nil
	}
	b, e := s.membership.Read(l.index())
	return b != 0, e
}
func (s *satMUSLiteralSet) reset() error {
	if e := s.values.Shrink(0); e != nil {
		return e
	}
	return s.membership.Shrink(0)
}
func (s *satMUSLiteralSet) close() error {
	if e := s.values.Finalize(); e != nil {
		return e
	}
	return s.membership.Finalize()
}
func (s *satMUSLiteralSet) intersect(other *satMUSLiteralSet) error {
	j := uint32(0)
	for i := uint32(0); ; i++ {
		n, e := s.values.Size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		v, e := s.values.Read(i)
		if e != nil {
			return e
		}
		yes, e := other.contains(Literal(v))
		if e != nil {
			return e
		}
		if yes {
			v, e = s.values.Read(i)
			if e != nil {
				return e
			}
			if e = s.values.Write(j, v); e != nil {
				return e
			}
			j++
		} else {
			v, e = s.values.Read(i)
			if e != nil {
				return e
			}
			if e = s.membership.Write(v, 0); e != nil {
				return e
			}
		}
	}
	return s.values.Shrink(j)
}
func (s *satMUSLiteralSet) union(other *satMUSLiteralSet) error {
	for i := uint32(0); ; i++ {
		n, e := other.size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		v, e := other.values.Read(i)
		if e != nil {
			return e
		}
		if e = s.insert(Literal(v)); e != nil {
			return e
		}
	}
}
func (s *satMUSLiteralSet) assign(other *satMUSLiteralSet) error {
	if s == other {
		return nil
	}
	if e := s.membership.Finalize(); e != nil {
		return e
	}
	s.membership.Closed = false
	if e := satMUSCopyColumn(&s.membership, &other.membership, 1); e != nil {
		return e
	}
	if e := s.values.Finalize(); e != nil {
		return e
	}
	s.values.Raw.Closed = false
	return satMUSCopyColumn(&s.values.Raw, &other.values.Raw, 4)
}

// Source vector assignment destroys the old member BEFORE copying the next;
// copy publishes capacity/size and m_data before ascending placement reads.
// A failed copy constructor has no source destructor. Pending allocation stays
// a named member root for separate owning cleanup, without repairing headers.
func satMUSCopyColumn(dst, src *RuntimeVectorColumn, stride uint32) error {
	return copyRuntimeVectorColumn(dst, src, stride)
}

func satMUSCopyRequest(capacity, stride uint32) uint64 { return uint64(capacity)*uint64(stride) + 8 }

// sizeof(T) arithmetic uses the selected actual source size_t width before
// memory::allocate receives the request. Runtime expects an already-sized value.
func satMUSCopyRequestFor(capacity, stride uint32, bits uint8) uint64 {
	n := satMUSCopyRequest(capacity, stride)
	if bits == 32 {
		return uint64(uint32(n))
	}
	return n
}
