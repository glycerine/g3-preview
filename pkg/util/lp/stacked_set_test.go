package lp

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func lpStackedSetSnapshot(m map[int32]struct{}) map[int32]struct{} {
	result := make(map[int32]struct{}, len(m))
	for k := range m {
		result[k] = struct{}{}
	}
	return result
}
func TestLPStackedIntSetSource(t *testing.T) {
	// Original consumer is Z3DEBUG-only; running the same sequence natively
	// additionally verifies behavior when its final source assertion is erased.
	var s StackedInt32Set
	s.Insert(1)
	s.Insert(2)
	s.Insert(3)
	snapshot := lpStackedSetSnapshot(s.Values())
	s.Push()
	s.Insert(4)
	lpStaticMust(t, s.Pop(1))
	if !reflect.DeepEqual(s.Values(), snapshot) {
		t.Fatal(s.Values())
	}
	s.Push()
	s.Push()
	s.Insert(4)
	s.Insert(5)
	s.Push()
	s.Insert(4)
	lpStaticMust(t, s.Pop(3))
	if !reflect.DeepEqual(s.Values(), snapshot) {
		t.Fatal(s.Values())
	}
}
func TestLPStackedIntSetSnapshotOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for seed := int64(0); seed < 40; seed++ {
		r := rand.New(rand.NewSource(seed))
		var s StackedInt32Set
		want := map[int32]struct{}{}
		var snapshots []map[int32]struct{}
		for step := 0; step < 2500; step++ {
			switch r.Intn(4) {
			case 0, 1:
				key := int32(r.Intn(41) - 20)
				s.Insert(key)
				want[key] = struct{}{}
			case 2:
				s.Push()
				snapshots = append(snapshots, lpStackedSetSnapshot(want))
			case 3:
				k := uint32(r.Intn(5))
				lpStaticMust(t, s.Pop(k))
				for k > 0 && len(snapshots) > 0 {
					last := len(snapshots) - 1
					want = snapshots[last]
					snapshots = snapshots[:last]
					k--
				}
			}
			if !reflect.DeepEqual(s.Values(), want) && !(len(s.Values()) == 0 && len(want) == 0) {
				t.Fatal(seed, step, s.Values(), want)
			}
			if len(s.stack) != len(snapshots) {
				t.Fatal("depth")
			}
		}
		lpStaticMust(t, s.Pop(math.MaxUint32))
		for _, d := range s.stack[:cap(s.stack)] {
			if d.inserted != nil || d.erased != nil || d.debugCopy != nil {
				t.Fatal("retained popped maps")
			}
		}
	}
}
func TestLPStackedIntSetCheckedFailurePrefix(t *testing.T) {
	for _, sameSize := range []bool{false, true} {
		var s StackedInt32Set
		s.Insert(math.MinInt32)
		s.Insert(math.MaxInt32)
		s.Push()
		s.Insert(7)
		// Source operator() is const. Deliberately corrupt its borrowed Go view
		// only to exercise the checked rollback assertion before frame removal.
		delete(s.values, math.MinInt32)
		if sameSize {
			s.values[9] = struct{}{}
		}
		err := s.Pop(1)
		if sourceDebugEnabled {
			if err != ErrAssertion || len(s.stack) != 1 {
				t.Fatal(sameSize, err)
			}
			s.values[math.MinInt32] = struct{}{}
			delete(s.values, 9)
			lpStaticMust(t, s.Pop(1))
		} else if err != nil || len(s.stack) != 0 {
			t.Fatal(sameSize, err)
		}
		if _, ok := s.values[7]; ok {
			t.Fatal("rollback insertion prefix missing")
		}
	}
}
