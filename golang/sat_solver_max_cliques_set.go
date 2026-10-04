// Source: src/util/uint_set.h:24-225; src/util/vector.h:191-216.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

// Actual uint_set bit words, distinct from sat::literal_set's char membership.
type satCliqueUintSet struct{ words satSolverRuntimeUintColumn }

func (s *satCliqueUintSet) contains(v uint32) (bool, error) {
	n, e := s.words.size()
	if e != nil {
		return false, e
	}
	i := v >> 5
	if i >= n {
		return false, nil
	}
	w, e := s.words.read(i)
	return w&(uint32(1)<<(v&31)) != 0, e
}
func (s *satCliqueUintSet) insert(v uint32) error {
	idx := v >> 5
	old, e := s.words.size()
	if e != nil {
		return e
	}
	if idx >= old {
		n := idx + 1
		for {
			cap, e := s.words.capacity()
			if e != nil {
				return e
			}
			if cap >= n {
				break
			}
			if e = s.words.raw.expandStride(4); e != nil {
				return e
			}
		}
		if e = s.words.raw.runtime.WriteUint32(s.words.raw.block, 4, n); e != nil {
			return e
		}
		for i := old; i < n; i++ {
			if e = s.words.write(i, 0); e != nil {
				return e
			}
		}
	}
	w, e := s.words.read(idx)
	if e != nil {
		return e
	}
	return s.words.write(idx, w|(uint32(1)<<(v&31)))
}
func (s *satCliqueUintSet) remove(v uint32) error {
	n, e := s.words.size()
	if e != nil {
		return e
	}
	i := v >> 5
	if i >= n {
		return nil
	}
	w, e := s.words.read(i)
	if e != nil {
		return e
	}
	return s.words.write(i, w & ^(uint32(1)<<(v&31)))
}
func (s *satCliqueUintSet) reset() error { return s.words.shrink(0) }
func (s *satCliqueUintSet) empty() (bool, error) {
	n, e := s.words.size()
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		w, e := s.words.read(i)
		if e != nil {
			return false, e
		}
		if w != 0 {
			return false, nil
		}
	}
	return true, nil
}
func (s *satCliqueUintSet) first() (uint32, error) {
	n, e := s.words.size()
	if e != nil {
		return 0, e
	}
	last := n * 32
	index := uint32(0)
	for index != last {
		w, e := s.words.read(index >> 5)
		if e != nil {
			return 0, e
		}
		if w == 0 {
			index += 32
			continue
		}
		for {
			yes, e := s.contains(index)
			if e != nil {
				return 0, e
			}
			if yes {
				return index, nil
			}
			index++
		}
	}
	return index, nil
}
func (s *satCliqueUintSet) assign(other *satCliqueUintSet) error {
	if s == other {
		return nil
	}
	r := s.words.raw.runtime
	if e := s.words.finalize(); e != nil {
		return e
	}
	s.words = satSolverRuntimeUintColumn{}
	s.words.bind(r)
	return satMUSCopyColumn(&s.words.raw, &other.words.raw, 4)
}
