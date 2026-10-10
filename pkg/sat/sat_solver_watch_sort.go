package sat

// satWatchSortWork is optional caller-owned diagnostic work, never retained.
type satWatchSortWork struct{ Comparisons, Moves uint64 }

// Two stable partitions implement the three equivalence classes of watched_lt.
// Recursive rotations use only stack scalar records: O(N) comparisons and
// O(N log N) assignments, without libstdc++ scratch allocation equivalence.
func (s *satSolverWatchStore) sortByKindWithWork(r satSolverWatchListRef, work *satWatchSortWork) error {
	x, e := s.record(r)
	if e != nil {
		return e
	}
	if work != nil {
		*work = satWatchSortWork{}
	}
	if uint32(len(x.values())) == 0 {
		return nil
	}
	ternary, e := newSATTernaryWatch(2, 4)
	if e != nil {
		return e
	}
	binaryEnd, e := s.stableWatchPartition(r, 0, uint32(len(x.values())), ternary, work)
	if e != nil {
		return e
	}
	clause, e := newSATClauseWatch(0, satClauseOffset{})
	if e != nil {
		return e
	}
	_, e = s.stableWatchPartition(r, binaryEnd, uint32(len(x.values())), clause, work)
	return e
}

func (s *satSolverWatchStore) stableWatchPartition(r satSolverWatchListRef, first, last uint32, pivot satWatched, work *satWatchSortWork) (uint32, error) {
	if first == last {
		return first, nil
	}
	if last-first == 1 {
		w, e := s.Read(r, first)
		if e != nil {
			return 0, e
		}
		if work != nil {
			work.Comparisons++
		}
		if satWatchedLess(w, pivot) {
			return last, nil
		}
		return first, nil
	}
	middle := first + (last-first)/2
	a, e := s.stableWatchPartition(r, first, middle, pivot, work)
	if e != nil {
		return 0, e
	}
	b, e := s.stableWatchPartition(r, middle, last, pivot, work)
	if e != nil {
		return 0, e
	}
	if a != middle && middle != b {
		if e = s.reverseWatchRange(r, a, middle, work); e != nil {
			return 0, e
		}
		if e = s.reverseWatchRange(r, middle, b, work); e != nil {
			return 0, e
		}
		if e = s.reverseWatchRange(r, a, b, work); e != nil {
			return 0, e
		}
	}
	return a + (b - middle), nil
}

func (s *satSolverWatchStore) reverseWatchRange(r satSolverWatchListRef, first, last uint32, work *satWatchSortWork) error {
	for last-first > 1 {
		last--
		a, e := s.Read(r, first)
		if e != nil {
			return e
		}
		b, e := s.Read(r, last)
		if e != nil {
			return e
		}
		if e = s.Write(r, first, b); e != nil {
			return e
		}
		if work != nil {
			work.Moves++
		}
		if e = s.Write(r, last, a); e != nil {
			return e
		}
		if work != nil {
			work.Moves++
		}
		first++
	}
	return nil
}
