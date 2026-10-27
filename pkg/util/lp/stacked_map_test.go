package lp

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func TestLPStackedIntFloatMapSource(t *testing.T) {
	var s StackedInt32FloatMap
	s.Set(0, 3)
	s.Set(1, 4)
	s.Push()
	s.Set(1, 5)
	s.Set(2, 2)
	s.Pop(1)
	lpStaticMust(t, s.Erase(2))
	s.Set(2, 3)
	lpStaticMust(t, s.Erase(1))
	s.Push()
	s.Set(3, 100)
	s.Set(4, 200)
	lpStaticMust(t, s.Erase(1))
	s.Push()
	s.Set(5, 300)
	s.Set(6, 400)
	s.Set(5, 301)
	lpStaticMust(t, s.Erase(5))
	s.Set(3, 122)
	s.Pop(2)
	s.Pop(1)
	if !reflect.DeepEqual(s.values, map[int32]float64{0: 3, 2: 3}) || len(s.stack) != 0 {
		t.Fatal(s.values, s.stack)
	}
}

func lpStackedMapSnapshot(m map[int32]float64) map[int32]float64 {
	result := make(map[int32]float64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
func TestLPStackedIntFloatMapSnapshotOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for seed := int64(0); seed < 40; seed++ {
		r := rand.New(rand.NewSource(seed))
		var s StackedInt32FloatMap
		want := map[int32]float64{}
		var snapshots []map[int32]float64
		for step := 0; step < 2500; step++ {
			key := int32(r.Intn(31) - 15)
			switch r.Intn(6) {
			case 0, 1, 2:
				value := float64(r.Intn(21) - 10)
				s.Set(key, value)
				want[key] = value
			case 3:
				err := s.Erase(key)
				if err == nil {
					delete(want, key)
				} else if !sourceDebugEnabled || err != ErrAssertion {
					t.Fatal(seed, step, err)
				}
			case 4:
				s.Push()
				snapshots = append(snapshots, lpStackedMapSnapshot(want))
			case 5:
				k := uint32(r.Intn(5))
				s.Pop(k)
				for k > 0 && len(snapshots) > 0 {
					last := len(snapshots) - 1
					want = snapshots[last]
					snapshots = snapshots[:last]
					k--
				}
			}
			if len(s.values) != len(want) {
				t.Fatal(seed, step, s.values, want)
			}
			for k, v := range want {
				if got, ok := s.values[k]; !ok || got != v {
					t.Fatal(seed, step, k, got, v)
				}
			}
			if len(s.stack) != len(snapshots) {
				t.Fatal(seed, step, "depth")
			}
		}
		s.Pop(math.MaxUint32)
		if len(s.stack) != 0 {
			t.Fatal("remaining snapshots")
		}
		// Popped frames release references in retained slice capacity.
		for _, d := range s.stack[:cap(s.stack)] {
			if d.newKeys != nil || d.original != nil {
				t.Fatal("retained popped maps")
			}
		}
	}
}
func TestLPStackedIntFloatMapSourceCancellationAndIEEE(t *testing.T) {
	var s StackedInt32FloatMap
	s.Set(1, 1)
	s.Push()
	s.Set(1, 2)
	s.Set(1, 1)
	if len(s.stack[0].original) != 0 {
		t.Fatal("original-value cancellation")
	}
	s.Set(1, 3)
	lpStaticMust(t, s.Erase(1))
	s.Set(1, 4)
	err := s.Erase(1)
	if sourceDebugEnabled {
		if err != ErrAssertion || s.values[1] != 4 {
			t.Fatal(err, s.values)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	s.Pop(1)
	if s.values[1] != 1 {
		t.Fatal(s.values)
	}
	s.Set(2, math.Copysign(0, -1))
	s.Push()
	s.Set(2, 0)
	if !math.Signbit(s.values[2]) {
		t.Fatal("equal zero replaces inside scope")
	}
	s.Pop(1)
	s.Set(2, 0)
	if math.Signbit(s.values[2]) {
		t.Fatal("outside scope assignment lost sign")
	}
	nan := math.Float64frombits(0x7ff8000000000013)
	s.Set(3, nan)
	s.Push()
	s.Set(3, 7)
	s.Set(3, nan)
	if len(s.stack[0].original) != 1 {
		t.Fatal("NaN cancellation")
	}
	s.Pop(1)
	if math.Float64bits(s.values[3]) != math.Float64bits(nan) {
		t.Fatal("NaN payload")
	}
	s.Set(math.MinInt32, math.Inf(-1))
	s.Set(math.MaxInt32, math.Inf(1))
	s.Push()
	s.Set(math.MinInt32, 1)
	s.Set(math.MaxInt32, 2)
	s.Pop(math.MaxUint32)
	if !math.IsInf(s.values[math.MinInt32], -1) || !math.IsInf(s.values[math.MaxInt32], 1) {
		t.Fatal(s.values)
	}
}
