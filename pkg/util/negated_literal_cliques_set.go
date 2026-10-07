// Source: src/util/uint_set.h:24-225; src/util/vector.h:191-216.
package util

// Actual uint_set bit words, distinct from sat::literal_set's char membership.
type cliqueUintSet struct{ words RuntimeUintColumn }

func (s *cliqueUintSet) contains(v uint32) (bool, error) {
	n, e := s.words.Size()
	if e != nil {
		return false, e
	}
	i := v >> 5
	if i >= n {
		return false, nil
	}
	w, e := s.words.Read(i)
	return w&(uint32(1)<<(v&31)) != 0, e
}
func (s *cliqueUintSet) insert(v uint32) error {
	idx := v >> 5
	old, e := s.words.Size()
	if e != nil {
		return e
	}
	if idx >= old {
		n := idx + 1
		for {
			cap, e := s.words.Capacity()
			if e != nil {
				return e
			}
			if cap >= n {
				break
			}
			if e = s.words.Raw.ExpandStride(4); e != nil {
				return e
			}
		}
		if e = s.words.Raw.Runtime.WriteUint32(s.words.Raw.Block, 4, n); e != nil {
			return e
		}
		for i := old; i < n; i++ {
			if e = s.words.Write(i, 0); e != nil {
				return e
			}
		}
	}
	w, e := s.words.Read(idx)
	if e != nil {
		return e
	}
	return s.words.Write(idx, w|(uint32(1)<<(v&31)))
}
func (s *cliqueUintSet) remove(v uint32) error {
	n, e := s.words.Size()
	if e != nil {
		return e
	}
	i := v >> 5
	if i >= n {
		return nil
	}
	w, e := s.words.Read(i)
	if e != nil {
		return e
	}
	return s.words.Write(i, w & ^(uint32(1)<<(v&31)))
}
func (s *cliqueUintSet) reset() error { return s.words.Shrink(0) }
func (s *cliqueUintSet) empty() (bool, error) {
	n, e := s.words.Size()
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		w, e := s.words.Read(i)
		if e != nil {
			return false, e
		}
		if w != 0 {
			return false, nil
		}
	}
	return true, nil
}
func (s *cliqueUintSet) first() (uint32, error) {
	n, e := s.words.Size()
	if e != nil {
		return 0, e
	}
	last := n * 32
	index := uint32(0)
	for index != last {
		w, e := s.words.Read(index >> 5)
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
func (s *cliqueUintSet) assign(other *cliqueUintSet) error {
	if s == other {
		return nil
	}
	r := s.words.Raw.Runtime
	if e := s.words.Finalize(); e != nil {
		return e
	}
	s.words = RuntimeUintColumn{}
	s.words.Bind(r)
	return copyCliqueColumn(&s.words.Raw, &other.words.Raw, 4)
}
