package lp

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestLPRationalFeasibilityIndependentMembership(t *testing.T) {
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 0, 5)
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.Names = lpFeasibleNames{}
		all, count := true, uint32(0)
		for j, kind := range []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed} {
			f.types[j] = kind
			lo, up := big.NewRat(-2, 1), big.NewRat(2, 1)
			if kind == ColumnFixed {
				lo.SetInt64(1)
				up.SetInt64(1)
			}
			x := big.NewRat(int64((world+j)%7-3), 1)
			if world%4 == 0 {
				x.Add(lo, new(big.Rat).SetFrac(big.NewInt(-1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil)))
			}
			for k, v := range []*big.Rat{lo, up, x} {
				p := []*IndexedRationalVector{f.low, f.up, f.x}[k]
				h.must(h.m.SetString(&p.Data[j], v.RatString()))
			}
			feasible := true
			if kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnLowBound {
				feasible = feasible && x.Cmp(lo) >= 0
			}
			if kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnUpperBound {
				feasible = feasible && x.Cmp(up) <= 0
			}
			h.must(m.UpdateColumnInInfSet(uint32(j)))
			if m.InfSet.Contains(uint32(j)) == feasible {
				t.Fatal(world, j, "membership")
			}
			all = all && feasible
			if !feasible {
				count++
			}
		}
		ok, e := m.CalcCurrentXIsFeasibleIncludeNonBasis()
		h.must(e)
		if ok != all || m.InfSet.Size() != count || m.CurrentXIsFeasible() != all {
			t.Fatal(world, ok, all, count, m.InfSet.Size())
		}
		ok, e = m.InfSetIsCorrect()
		h.must(e)
		if !ok {
			t.Fatal("independent membership rejected")
		}
		// Repeated updates erase old membership and are idempotent.
		for j := range f.x.Data {
			h.must(f.x.copySlot(&f.x.Data[j], &f.low.Data[j]))
			h.must(m.UpdateColumnInInfSet(uint32(j)))
			h.must(m.UpdateColumnInInfSet(uint32(j)))
		}
		ok, e = m.CalcCurrentXIsFeasibleIncludeNonBasis()
		h.must(e)
		if !ok || m.InfSet.Size() != 0 {
			t.Fatal("repaired membership")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalFeasibilityDiagnosticsAndGuards(t *testing.T) {
	for _, phase := range []string{"missing-member", "extra-member", "closed-stdout", "bad-update", "bad-types", "descending-skip", "closed-entry", "nil-settings-cost", "nil-settings-reduced"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 0, 2)
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			m.Names = lpFeasibleNames{}
			var err error
			var ok bool
			switch phase {
			case "missing-member", "closed-stdout":
				f.types[1] = ColumnLowBound
				h.must(h.m.SetInt32(&f.x.Data[1], -1))
				if phase == "closed-stdout" {
					closed, e := os.CreateTemp(t.TempDir(), "g3-feasibility-closed-")
					h.must(e)
					h.must(closed.Close())
					old := os.Stdout
					os.Stdout = closed
					ok, err = m.InfSetIsCorrect()
					os.Stdout = old
					h.must(os.Remove(closed.Name()))
					if err == nil || ok {
						t.Fatal("closed diagnostic accepted")
					}
				} else {
					out := lpCaptureFeasibleStdout(t, func() { ok, err = m.InfSetIsCorrect() })
					h.must(err)
					if ok || out != "column_index = 1, name = v type = low_bound\n0\nbasis heading = -1\nx = -1\nbelongs_to_set = 0\ninf\n" {
						t.Fatal(out, ok)
					}
				}
			case "extra-member":
				h.must(m.InfSet.Insert(1))
				out := lpCaptureFeasibleStdout(t, func() { ok, err = m.InfSetIsCorrect() })
				h.must(err)
				if ok || !strings.HasSuffix(out, "belongs_to_set = 1\nfeas\n") {
					t.Fatal(out, ok)
				}
			case "bad-update":
				err = m.UpdateColumnInInfSet(99)
				if err != ErrBounds {
					t.Fatal(err)
				}
			case "bad-types":
				f.types = nil
				_, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
				if err != ErrBounds {
					t.Fatal(err)
				}
			case "descending-skip":
				f.types[1] = ColumnLowBound
				h.must(h.m.SetInt32(&f.x.Data[1], -1))
				f.types[0] = ColumnBoxed
				owner := f.up.Data
				f.up.Data = nil
				ok, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
				f.up.Data = owner
				h.must(err)
				if ok {
					t.Fatal("lost descending early false")
				}
			case "closed-entry":
				h.must(m.Close())
				_, err = m.InfSetIsCorrect()
				if err != ErrExactVectorPending {
					t.Fatal(err)
				}
			case "nil-settings-cost":
				m.Settings = nil
				f.types = nil
				_, err = m.InfeasibilityCostIsCorrectForColumn(99)
				if err != ErrBounds {
					t.Fatal("settings before metadata", err)
				}
			case "nil-settings-reduced":
				m.Settings = nil
				_, err = m.ReducedCostsAreCorrectTableau()
				if err != ErrBounds {
					t.Fatal(err)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if e = m.UpdateColumnInInfSet(0); e != ErrExactVectorPending {
				t.Fatal("closed membership", e)
			}
			if _, e = m.CalcCurrentXIsFeasibleIncludeNonBasis(); e != ErrExactVectorPending {
				t.Fatal("closed all-column check", e)
			}
		})
	}
}

func TestLPRationalFeasibilityPhysicalComparisons(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, phase := range []string{"update", "all", "validate"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 100 && !done; ordinal++ {
				h := newLPPairHarness(t)
				if direct {
					h = lpNewDirectSparseHarness(t)
				}
				f := lpNewRationalCoreFixture(t, h, 0, 1)
				m, e := NewRationalCoreSolverBase(f.input())
				h.must(e)
				m.Names = lpFeasibleNames{}
				f.types[0] = ColumnLowBound
				large := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.must(f.low.copySlot(&f.low.Data[0], &large.X))
				h.must(f.x.copySlot(&f.x.Data[0], &large.Y))
				if phase == "validate" {
					h.must(m.InfSet.Insert(0))
				}
				h.budget.FailAfter(ordinal)
				var err error
				switch phase {
				case "update":
					err = m.UpdateColumnInInfSet(0)
				case "all":
					_, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
				case "validate":
					_, err = m.InfSetIsCorrect()
				}
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(fmt.Sprint(direct, phase, ordinal), err)
					}
					if _, e = m.InfSetIsCorrect(); e != ErrExactVectorPending {
						t.Fatal("stopped feasibility", e)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
			if !done || direct && failures == 0 {
				t.Fatal("missing physical comparison", direct, phase, failures)
			}
		}
	}
}
