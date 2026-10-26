// Source: src/util/lp/lar_solver.h:1278-1303.
package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func lpLARGetModelFixture(t *testing.T, h *lpPairHarness, pairs [][2]*big.Rat) *LARSolver {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	for j, pair := range pairs {
		_, err = s.AddVar(uint32(17 + 13*j))
		h.must(err)
		h.must(s.Core.RX.Data[j].Copy(h.m, h.pair(pair[0].RatString(), pair[1].RatString())))
	}
	return s
}

func lpLARGetModelCheck(t *testing.T, h *lpPairHarness, model *LARRationalModel, pairs [][2]*big.Rat, delta *big.Rat) {
	t.Helper()
	for j, p := range pairs {
		v := model.Values[uint32(j)]
		if v == nil {
			t.Fatal("missing model column", j)
		}
		want := new(big.Rat).Add(p[0], new(big.Rat).Mul(delta, p[1]))
		lpSparseRationalCheck(t, h, v, want)
		for k := 0; k < j; k++ {
			pk := pairs[k]
			other := new(big.Rat).Add(pk[0], new(big.Rat).Mul(delta, pk[1]))
			if (p[0].Cmp(pk[0]) == 0 && p[1].Cmp(pk[1]) == 0) != (want.Cmp(other) == 0) {
				t.Fatal("independent pair/single distinctness", j, k)
			}
		}
	}
}

func TestLARGetModelIndependentCollisionChains(t *testing.T) {
	for world := 0; world < 160; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			shift, scale := big.NewRat(int64(world-79), 7), big.NewRat(int64(1+world%11), 13)
			pairs := [][2]*big.Rat{{new(big.Rat).Set(shift), new(big.Rat).Set(scale)}}
			collisions := world % 8
			for k := 1; k <= collisions; k++ {
				value := new(big.Rat).Add(shift, new(big.Rat).Quo(scale, new(big.Rat).SetInt64(1<<k)))
				pairs = append(pairs, [2]*big.Rat{value, new(big.Rat)})
			}
			// Repeated pairs are allowed. Distinct rational values with similar
			// hash keys must remain separate even when their numerators match.
			pairs = append(pairs, pairs[0])
			for _, denominator := range []int64{3, 5} {
				pairs = append(pairs, [2]*big.Rat{new(big.Rat).Add(shift, new(big.Rat).Quo(scale, big.NewRat(denominator, 1))), new(big.Rat)})
			}
			s := lpLARGetModelFixture(t, h, pairs)
			out := NewLARRationalModel(h.m)
			old := h.pair("97/19", "0")
			h.must(out.set(99, &old.X))
			h.must(out.set(0, &old.X))
			limit := &lpTestResourceLimit{err: lpErrExactSolveCancel}
			s.Settings.SetResourceLimit(limit)
			h.must(s.GetModel(out))
			delta := big.NewRat(1, int64(1<<(collisions+1)))
			lpLARGetModelCheck(t, h, out, pairs, delta)
			lpSparseRationalCheck(t, h, out.Values[99], big.NewRat(97, 19))
			if limit.calls != 0 || len(out.Values) != len(pairs)+1 || s.rationalWork != nil || s.modelPairs != nil || s.modelSingles != nil {
				t.Fatal("source cancellation/ownership/output extent")
			}
			// Source X and output have independent owning values.
			h.must(h.m.SetInt32(&s.Core.RX.Data[0].X, 71))
			lpSparseRationalCheck(t, h, out.Values[0], new(big.Rat).Add(shift, new(big.Rat).Mul(delta, scale)))
			h.must(out.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARGetModelStrictBoundsAndRepeatedOutput(t *testing.T) {
	h := newLPPairHarness(t)
	pairs := [][2]*big.Rat{{new(big.Rat), big.NewRat(1, 1)}, {big.NewRat(1, 8), new(big.Rat)}, {big.NewRat(1, 3), new(big.Rat)}, {big.NewRat(1, 5), new(big.Rat)}}
	s := lpLARGetModelFixture(t, h, pairs)
	s.Bounds.Types.values[0] = ColumnBoxed
	h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", "0")))
	h.must(s.Bounds.Upper.values[0].Copy(h.m, h.pair("1/8", "0")))
	out := NewLARRationalModel(h.m)
	for repeat := 0; repeat < 3; repeat++ {
		h.must(s.GetModel(out))
		lpLARGetModelCheck(t, h, out, pairs, big.NewRat(1, 16))
	}
	h.must(out.Close())
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARGetModelEmptyAndReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"empty", "stopped", "nil-output", "foreign-output", "stopped-output", "delta-error", "status"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			var pairs [][2]*big.Rat
			if phase != "empty" {
				pairs = [][2]*big.Rat{{new(big.Rat), big.NewRat(1, 1)}}
			}
			s := lpLARGetModelFixture(t, h, pairs)
			out := NewLARRationalModel(h.m)
			argument := out
			want := error(nil)
			switch phase {
			case "stopped":
				s.stopped = true
				want = ErrExactVectorPending
			case "nil-output":
				argument = nil
				want = ErrExactVectorOwner
			case "foreign-output":
				argument = NewLARRationalModel(nil)
				want = ErrExactVectorOwner
			case "stopped-output":
				out.stopped = true
				want = ErrExactVectorPending
			case "delta-error":
				s.Bounds.Types.values[0] = ColumnLowBound
				s.Bounds.Types.values = nil
				want = ErrBounds
			case "status":
				s.Bounds.Status.Value = StatusInfeasible
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			if err := s.GetModel(argument); err != want {
				t.Fatal("entry/source stage", phase, err, want)
			}
			if want != nil {
				if len(out.Values) != 0 || s.GetModel(out) != ErrExactVectorPending {
					t.Fatal("failure prefix/stopped retry")
				}
				if phase == "status" && (s.rationalWork == nil || s.Core.strictDelta != nil || s.modelPairs != nil) {
					t.Fatal("source status assertion follows delta selection")
				}
			}
			h.must(out.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARGetModelPhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, collision := range []bool{false, true} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 2000 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%v/%v/%d", direct, collision, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					x := mustLPRat(t, strings.Repeat("1234567890", 20)+"/7")
					y := mustLPRat(t, strings.Repeat("9876543210", 20)+"/13")
					pairs := [][2]*big.Rat{{x, y}, {new(big.Rat).Add(x, new(big.Rat).Mul(big.NewRat(1, 2), y)), new(big.Rat)}}
					if !collision {
						pairs[1][0].Add(pairs[1][0], big.NewRat(1, 1))
					}
					pairs = append(pairs, pairs[0])
					s := lpLARGetModelFixture(t, h, pairs)
					out := NewLARRationalModel(h.m)
					h.budget.FailAfter(ordinal)
					err := s.GetModel(out)
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
						delta := big.NewRat(1, 2)
						if collision {
							delta.SetFrac64(1, 4)
						}
						lpLARGetModelCheck(t, h, out, pairs, delta)
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) || s.GetModel(out) != ErrExactVectorPending {
							t.Fatal("actual numerical allocation failure", err)
						}
						if out.stopped && out.set(99, &s.Core.RX.Data[0].X) != ErrExactVectorPending {
							t.Fatal("stopped output accepted another assignment")
						}
					}
					// A stopped assignment may borrow the solver's current single.
					h.must(out.Close())
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			if !done || direct && failures == 0 {
				t.Fatal("physical campaign did not reach failure and success", direct, collision, failures)
			}
		}
	}
}

func TestLARGetModelRetryAndLateErrorPublication(t *testing.T) {
	h := newLPPairHarness(t)
	pairs := [][2]*big.Rat{{new(big.Rat), big.NewRat(1, 1)}, {big.NewRat(1, 2), new(big.Rat)}, {big.NewRat(7, 1), new(big.Rat)}}
	s := lpLARGetModelFixture(t, h, pairs)
	out := NewLARRationalModel(h.m)
	old := h.pair("29", "0")
	h.must(out.set(2, &old.X))
	h.must(out.set(99, &old.X))
	saved := s.Core.RX.Data[2].X
	expired := h.pair(strings.Repeat("1234567890", 20)+"/7", "0")
	s.Core.RX.Data[2].X = expired.X
	h.must(h.m.Del(&expired.X))
	if err := s.GetModel(out); err != arena.ErrInvalidSlot {
		t.Fatal("late source-value inspection error", err)
	}
	// The first attempt published 1/2 at column0, then collided at column1.
	// Retry overwrites that prefix with 1/4 and publishes column1 before the
	// malformed third pair is read. Existing later and unrelated values stay.
	lpSparseRationalCheck(t, h, out.Values[0], big.NewRat(1, 4))
	lpSparseRationalCheck(t, h, out.Values[1], big.NewRat(1, 2))
	lpSparseRationalCheck(t, h, out.Values[2], big.NewRat(29, 1))
	lpSparseRationalCheck(t, h, out.Values[99], big.NewRat(29, 1))
	lpSparseRationalCheck(t, h, &s.rationalWork.Data[0], big.NewRat(1, 4))
	if s.GetModel(out) != ErrExactVectorPending || len(s.modelPairs.Data) != 2 || len(s.modelSingles.Data) != 2 {
		t.Fatal("reached set prefix and sticky solver")
	}
	s.Core.RX.Data[2].X = saved
	h.must(out.Close())
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARGetModelPreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpLARGetModelFixture(t, h, [][2]*big.Rat{{big.NewRat(1, 13), new(big.Rat)}})
			s.Bounds.Types.values[0] = ColumnLowBound
			h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", "123456789012345678901/7")))
			out := NewLARRationalModel(h.m)
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
				if err = s.GetModel(out); err == nil || s.calls.comparison != 0 || s.Core.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(out.Close())
			h.must(s.Close())
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

func TestLARGetModelDeepCollisionPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	// Forty successive collisions take the denominator beyond inline integer
	// storage and reach real allocation refusal during epsilon halving.
	for _, direct := range []bool{false, true} {
		done, failures := false, 0
		for ordinal := int64(0); ordinal < 6000 && !done; ordinal++ {
			t.Run(fmt.Sprintf("%v/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				pairs := [][2]*big.Rat{{new(big.Rat), big.NewRat(1, 1)}}
				for k := uint(1); k <= 40; k++ {
					pairs = append(pairs, [2]*big.Rat{new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), k)), new(big.Rat)})
				}
				s := lpLARGetModelFixture(t, h, pairs)
				out := NewLARRationalModel(h.m)
				h.budget.FailAfter(ordinal)
				err := s.GetModel(out)
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
					lpLARGetModelCheck(t, h, out, pairs, new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 41)))
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) || s.GetModel(out) != ErrExactVectorPending {
						t.Fatal("deep collision numerical failure", err)
					}
				}
				h.must(out.Close())
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !done || direct && failures == 0 {
			t.Fatal("deep collision campaign", direct, failures)
		}
	}
}
