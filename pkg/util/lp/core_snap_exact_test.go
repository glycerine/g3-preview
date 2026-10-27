// Source: src/util/lp/lp_core_solver_base.h:346-385; mpq/impq.
package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExactSnapColumnIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	kinds := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed, ColumnType(99)}
	for world := 0; world < 600; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 1)
		kind := kinds[world%len(kinds)]
		f.types[0] = kind
		lo, ly := big.NewRat(int64(-1-world%11), 7), big.NewRat(1, 13)
		up, uy := big.NewRat(int64(2+world%17), 5), big.NewRat(-1, 19)
		if kind == ColumnFixed {
			up.Set(lo)
			uy.Set(ly)
		}
		x, y := new(big.Rat), new(big.Rat)
		switch world / len(kinds) % 4 {
		case 0:
			x.Set(lo)
			y.Set(ly)
		case 1:
			x.Set(up)
			y.Set(uy)
		case 2:
			x.Quo(new(big.Rat).Add(lo, up), big.NewRat(2, 1))
		case 3:
			x.Add(up, big.NewRat(1, 1))
			y.Set(uy)
		}
		h.must(f.low.Data[0].Copy(h.m, h.pair(lo.RatString(), ly.RatString())))
		h.must(f.up.Data[0].Copy(h.m, h.pair(up.RatString(), uy.RatString())))
		h.must(f.x.Data[0].Copy(h.m, h.pair(x.RatString(), y.RatString())))
		m, err := NewPairCoreSolverBase(f.input())
		h.must(err)
		seed := uint32(world*7919 + 17)
		f.settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		equal := func(a, b, c, d *big.Rat) bool { return a.Cmp(b) == 0 && c.Cmp(d) == 0 }
		atLow, atUpper := equal(x, lo, y, ly), equal(x, up, y, uy)
		changed := false
		wantX, wantY := new(big.Rat).Set(x), new(big.Rat).Set(y)
		switch kind {
		case ColumnFixed:
			if !atLow && !atUpper {
				changed = true
				wantX.Set(lo)
				wantY.Set(ly)
			}
		case ColumnLowBound:
			if !atLow {
				changed = true
				wantX.Set(lo)
				wantY.Set(ly)
			}
		case ColumnUpperBound:
			if !atUpper {
				changed = true
				wantX.Set(up)
				wantY.Set(uy)
			}
		case ColumnBoxed:
			if !atLow && !atUpper {
				changed = true
				if rng.Next()%2 == 1 {
					wantX.Set(lo)
					wantY.Set(ly)
				} else {
					wantX.Set(up)
					wantY.Set(uy)
				}
			}
		}
		got, err := m.SnapColumnToBound(0)
		h.must(err)
		if got != changed || f.settings.RandomNext() != rng.Next() {
			t.Fatal("bound selection/RNG prefix", world, got, changed)
		}
		h.check(&f.x.Data[0], wantX, wantY)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactSnapNonBasicVisitsEveryColumn(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 0, 4)
	for j := range f.types {
		f.types[j] = ColumnLowBound
		h.must(f.low.Data[j].Copy(h.m, h.pair(fmt.Sprint(j+1), "1/13")))
	}
	m, err := NewPairCoreSolverBase(f.input())
	h.must(err)
	changed, err := m.SnapNonBasicXToBound()
	h.must(err)
	if !changed {
		t.Fatal("changed")
	}
	for j := range f.types {
		h.check(&f.x.Data[j], big.NewRat(int64(j+1), 1), big.NewRat(1, 13))
	}
	changed, err = m.SnapNonBasicXToBound()
	h.must(err)
	if changed {
		t.Fatal("second pass")
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSnapFixedPreservesEitherEndpoint(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 0, 1)
	f.types[0] = ColumnFixed
	h.must(f.low.Data[0].Copy(h.m, h.pair("2", "0")))
	h.must(f.up.Data[0].Copy(h.m, h.pair("3", "0")))
	h.must(f.x.Data[0].Copy(h.m, h.pair("3", "0")))
	m, err := NewPairCoreSolverBase(f.input())
	h.must(err)
	changed, err := m.SnapColumnToBound(0)
	h.must(err)
	if changed {
		t.Fatal("literal source fixed branch preserves upper even if bounds differ")
	}
	h.check(&f.x.Data[0], big.NewRat(3, 1), new(big.Rat))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSnapReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"stopped", "short-types", "short-x", "short-low", "short-upper", "boxed-short-low", "boxed-short-upper", "free-short-x", "unknown-short-x", "late-nonbasic"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 0, 2)
			f.types[0], f.types[1] = ColumnLowBound, ColumnLowBound
			h.must(f.low.Data[0].Copy(h.m, h.pair("2", "0")))
			h.must(f.low.Data[1].Copy(h.m, h.pair("3", "0")))
			m, err := NewPairCoreSolverBase(f.input())
			h.must(err)
			types, x, low, up := f.types, f.x.Data, f.low.Data, f.up.Data
			want := error(ErrBounds)
			switch phase {
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
			case "short-types":
				f.types = nil
			case "short-x":
				f.x.Data = nil
			case "short-low":
				f.low.Data = nil
			case "short-upper":
				f.types[0] = ColumnUpperBound
				f.up.Data = nil
			case "boxed-short-low":
				f.types[0] = ColumnBoxed
				f.low.Data = nil
			case "boxed-short-upper":
				f.types[0] = ColumnBoxed
				f.up.Data = nil
			case "free-short-x":
				f.types[0] = ColumnFree
				f.x.Data = nil
				want = nil
			case "unknown-short-x":
				f.types[0] = ColumnType(99)
				f.x.Data = nil
				want = nil
			case "late-nonbasic":
				*m.NBasis = []uint32{0, 2}
			}
			var changed bool
			if phase == "late-nonbasic" {
				changed, err = m.SnapNonBasicXToBound()
			} else {
				changed, err = m.SnapColumnToBound(0)
			}
			if err != want || changed != (phase == "late-nonbasic") {
				t.Fatal("reached prefix", changed, err, want)
			}
			f.types = types
			f.x.Data = x
			f.low.Data = low
			f.up.Data = up
			if phase == "late-nonbasic" {
				h.check(&f.x.Data[0], big.NewRat(2, 1), new(big.Rat))
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactSnapPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 1000; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f := lpNewPairCoreFixture(t, h, 0, 2)
				f.types[0], f.types[1] = ColumnLowBound, ColumnBoxed
				huge := strings.Repeat("1234567890", 20) + "7/13"
				h.must(f.low.Data[0].Copy(h.m, h.pair(huge, "1/17")))
				h.must(f.low.Data[1].Copy(h.m, h.pair("-"+huge, "1/19")))
				h.must(f.up.Data[1].Copy(h.m, h.pair(huge, "-1/23")))
				m, err := NewPairCoreSolverBase(f.input())
				h.must(err)
				h.budget.FailAfter(ordinal)
				changed, err := m.SnapNonBasicXToBound()
				h.budget.FailAfter(-1)
				if err == nil {
					success, done = true, true
					if !changed {
						t.Fatal("actual snap")
					}
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if _, e := m.SnapNonBasicXToBound(); e != ErrExactVectorPending {
						t.Fatal("stopped retry", e)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				break
			}
		}
		if !success || direct && failures == 0 {
			t.Fatal("physical campaign", direct, success, failures)
		}
	}
}

func TestLPExactSnapPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 0, 1)
			f.types[0] = ColumnLowBound
			huge := strings.Repeat("1234567890", 20)
			h.must(f.low.Data[0].Copy(h.m, h.pair(huge+"7/13", "0")))
			h.must(f.x.Data[0].Copy(h.m, h.pair(huge+"9/17", "0")))
			m, err := NewPairCoreSolverBase(f.input())
			h.must(err)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_, err = m.SnapNonBasicXToBound()
				if err == nil || m.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(m.Close())
			f.close()
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed older comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}
