// Source: src/sat/sat_types.h:153–289; src/util/vector.h:102–120,187–201.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Nikolaj Bjorner.
package g3

import "github.com/glycerine/g3/golang/memory"

// Actual literal_set members. The enclosing charged lexical owner includes
// this cold metadata; both source buffers belong to the solver Runtime.
type satMUSLiteralSet struct {
	membership satSolverAssignmentColumn
	values     satSolverRuntimeUintColumn
}

func (s *satMUSLiteralSet) bind(r *memory.Runtime) { s.membership.bind(r); s.values.bind(r) }
func (s *satMUSLiteralSet) insert(l satLiteral) error {
	// Existing concrete source uint_set implementation, with identical buffers.
	return satMUSSetInsert(&s.membership, &s.values, l)
}
func satMUSSetInsert(m *satSolverAssignmentColumn, v *satSolverRuntimeUintColumn, l satLiteral) error {
	old, e := m.size()
	if e != nil {
		return e
	}
	n := l.index() + 1
	if n > old {
		for {
			c, e := m.capacity()
			if e != nil {
				return e
			}
			if c >= n {
				break
			}
			if e = m.expand(); e != nil {
				return e
			}
		}
		if e = satAssignmentWriteWord(m.runtime, m.block, 4, n); e != nil {
			return e
		}
		for i := old; i < n; i++ {
			if e = m.write(i, 0); e != nil {
				return e
			}
		}
	}
	b, e := m.read(l.index())
	if e != nil {
		return e
	}
	if b != 0 {
		return nil
	}
	if e = m.write(l.index(), 1); e != nil {
		return e
	}
	return v.push(l.index())
}
func (s *satMUSLiteralSet) size() (uint32, error) { return s.values.size() }
func (s *satMUSLiteralSet) contains(l satLiteral) (bool, error) {
	n, e := s.membership.size()
	if e != nil {
		return false, e
	}
	if l.index() >= n {
		return false, nil
	}
	b, e := s.membership.read(l.index())
	return b != 0, e
}
func (s *satMUSLiteralSet) reset() error {
	if e := s.values.shrink(0); e != nil {
		return e
	}
	return s.membership.shrink(0)
}
func (s *satMUSLiteralSet) close() error {
	if e := s.values.finalize(); e != nil {
		return e
	}
	return s.membership.finalize()
}
func (s *satMUSLiteralSet) intersect(other *satMUSLiteralSet) error {
	j := uint32(0)
	for i := uint32(0); ; i++ {
		n, e := s.values.size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		v, e := s.values.read(i)
		if e != nil {
			return e
		}
		yes, e := other.contains(satLiteral(v))
		if e != nil {
			return e
		}
		if yes {
			v, e = s.values.read(i)
			if e != nil {
				return e
			}
			if e = s.values.write(j, v); e != nil {
				return e
			}
			j++
		} else {
			v, e = s.values.read(i)
			if e != nil {
				return e
			}
			if e = s.membership.write(v, 0); e != nil {
				return e
			}
		}
	}
	return s.values.shrink(j)
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
		v, e := other.values.read(i)
		if e != nil {
			return e
		}
		if e = s.insert(satLiteral(v)); e != nil {
			return e
		}
	}
}
func (s *satMUSLiteralSet) assign(other *satMUSLiteralSet) error {
	if s == other {
		return nil
	}
	if e := s.membership.finalize(); e != nil {
		return e
	}
	s.membership.closed = false
	if e := satMUSCopyColumn(&s.membership, &other.membership, 1); e != nil {
		return e
	}
	if e := s.values.finalize(); e != nil {
		return e
	}
	s.values.raw.closed = false
	return satMUSCopyColumn(&s.values.raw, &other.values.raw, 4)
}

// Source vector assignment destroys the old member BEFORE copying the next;
// copy publishes capacity/size and m_data before ascending placement reads.
// A failed copy constructor has no source destructor. Pending allocation stays
// a named member root for separate owning cleanup, without repairing headers.
func satMUSCopyColumn(dst, src *satSolverAssignmentColumn, stride uint32) error {
	if src.block == (memory.Block{}) {
		dst.block = memory.Block{}
		return nil
	}
	n, e := src.size()
	if e != nil {
		return e
	}
	c, e := src.capacity()
	if e != nil {
		return e
	}
	h, e := src.runtime.Allocate(satMUSCopyRequestFor(c, stride, src.runtime.SizeTBits()))
	dst.pending = h
	if e != nil {
		return e
	}
	if e = satAssignmentWriteWord(src.runtime, h, 0, c); e != nil {
		return e
	}
	if e = satAssignmentWriteWord(src.runtime, h, 4, n); e != nil {
		return e
	}
	dst.block = h
	dst.pending = memory.Block{}
	for i := uint32(0); i < n; i++ {
		for k := uint32(0); k < stride; k++ {
			b, e := src.runtime.Read(src.block, 8+uint64(i)*uint64(stride)+uint64(k))
			if e != nil {
				return e
			}
			if e = dst.runtime.Write(h, 8+uint64(i)*uint64(stride)+uint64(k), b); e != nil {
				return e
			}
		}
	}
	return nil
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
