package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPPairMakeFeasibleIndependentLexicographicOracle(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 0, 1)
	m, err := NewPairCoreSolverBase(f.input())
	h.must(err)
	delta := h.pair("0", "0")
	rng := rand.New(rand.NewPCG(731819, 271))
	lex := func(a, b [2]*big.Rat) int {
		if c := a[0].Cmp(b[0]); c != 0 {
			return c
		}
		return a[1].Cmp(b[1])
	}
	for world := 0; world < 1600; world++ {
		kind := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}[world%5]
		lo := [2]*big.Rat{big.NewRat(int64(rng.IntN(17)-8), int64(rng.IntN(11)+1)), big.NewRat(int64(rng.IntN(9)-4), int64(rng.IntN(7)+1))}
		up := [2]*big.Rat{new(big.Rat).Add(lo[0], big.NewRat(int64(rng.IntN(5)), 3)), big.NewRat(int64(rng.IntN(9)-4), int64(rng.IntN(7)+1))}
		if lex(lo, up) > 0 {
			lo, up = up, lo
		}
		if kind == ColumnFixed {
			up = [2]*big.Rat{new(big.Rat).Set(lo[0]), new(big.Rat).Set(lo[1])}
		}
		x := [2]*big.Rat{new(big.Rat).Set(lo[0]), big.NewRat(int64(rng.IntN(13)-6), int64(rng.IntN(5)+1))}
		if world%3 == 0 {
			x[0].Add(x[0], big.NewRat(int64(rng.IntN(7)-3), 5))
		}
		if world%11 == 0 {
			x = [2]*big.Rat{new(big.Rat).Set(up[0]), new(big.Rat).Set(up[1])}
		}
		beforeDelta := [2]*big.Rat{big.NewRat(int64(world+3), 19), big.NewRat(int64(-world-7), 23)}
		for k, slot := range []*RationalPair{&f.low.Data[0], &f.up.Data[0], &f.x.Data[0], delta} {
			want := [][2]*big.Rat{lo, up, x, beforeDelta}[k]
			h.must(h.m.SetString(&slot.X, want[0].RatString()))
			h.must(h.m.SetString(&slot.Y, want[1].RatString()))
		}
		f.types[0] = kind
		feasible := true
		if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnLowBound) && lex(x, lo) < 0 {
			feasible = false
		}
		if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnUpperBound) && lex(x, up) > 0 {
			feasible = false
		}
		if world%2 == 0 {
			h.must(m.InfSet.Insert(0))
		} else {
			m.InfSet.Erase(0)
		}
		h.must(m.UpdateColumnInInfSet(0))
		if m.InfSet.Contains(0) == feasible {
			t.Fatal("membership oracle", world, kind)
		}
		want := x
		changed := false
		if kind == ColumnFixed && lex(x, lo) != 0 {
			want = lo
			changed = true
		} else if (kind == ColumnLowBound || kind == ColumnBoxed) && lex(x, lo) < 0 {
			want = lo
			changed = true
		} else if (kind == ColumnUpperBound || kind == ColumnBoxed) && lex(x, up) > 0 {
			want = up
			changed = true
		}
		got, e := m.MakeColumnFeasible(0, delta)
		h.must(e)
		if got != changed {
			t.Fatal("clamp oracle", world, kind, got, changed)
		}
		h.check(&f.x.Data[0], want[0], want[1])
		if changed {
			h.check(delta, new(big.Rat).Sub(want[0], x[0]), new(big.Rat).Sub(want[1], x[1]))
		} else {
			h.check(delta, beforeDelta[0], beforeDelta[1])
		}
		// Clamping alone does not alter membership; the separate source operation does.
		if m.InfSet.Contains(0) == feasible {
			t.Fatal("clamp changed membership")
		}
		h.must(m.UpdateColumnInInfSet(0))
		if m.InfSet.Contains(0) {
			t.Fatal("clamped column remained infeasible")
		}
		if m.pivotWork != nil || m.feasibilityWork != nil {
			t.Fatal("completed expression retained")
		}
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairMakeFeasibleAliasesAndSourceOrder(t *testing.T) {
	for _, op := range []string{"delta-x", "delta-low", "basic", "fixed-mismatch", "fixed-short-upper", "inverted-box", "unknown"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 0, 1)
			m, err := NewPairCoreSolverBase(f.input())
			h.must(err)
			h.must(f.x.Data[0].Copy(h.m, h.pair("1", "2")))
			h.must(f.low.Data[0].Copy(h.m, h.pair("5", "7")))
			h.must(f.up.Data[0].Copy(h.m, h.pair("9", "11")))
			delta := h.pair("17", "19")
			f.types[0] = ColumnLowBound
			expectedErr := error(nil)
			changed := true
			switch op {
			case "delta-x":
				delta = &f.x.Data[0]
			case "delta-low":
				delta = &f.low.Data[0]
			case "basic":
				f.heading[0] = 0
				if sourceDebugEnabled {
					expectedErr = ErrAssertion
					changed = false
				}
			case "fixed-mismatch":
				f.types[0] = ColumnFixed
				if sourceDebugEnabled {
					expectedErr = ErrAssertion
					changed = false
				}
			case "fixed-short-upper":
				f.types[0] = ColumnFixed
				f.up.Data = nil
				if sourceDebugEnabled {
					expectedErr = ErrBounds
					changed = false
				}
			case "inverted-box":
				f.types[0] = ColumnBoxed
				h.must(f.up.Data[0].Copy(h.m, h.pair("-3", "-4")))
			case "unknown":
				f.types[0] = ColumnType(99)
				changed = false
				if sourceDebugEnabled {
					expectedErr = ErrAssertion
				}
			}
			got, e := m.MakeColumnFeasible(0, delta)
			if got != changed || e != expectedErr {
				t.Fatal(got, e, changed, expectedErr)
			}
			if e == nil && changed {
				if op == "delta-low" {
					h.check(&f.x.Data[0], big.NewRat(4, 1), big.NewRat(5, 1))
					h.check(delta, big.NewRat(4, 1), big.NewRat(5, 1))
				} else {
					h.check(&f.x.Data[0], big.NewRat(5, 1), big.NewRat(7, 1))
				}
				if op == "delta-x" {
					h.check(delta, big.NewRat(5, 1), big.NewRat(7, 1))
				}
			}
			if e != nil || op == "unknown" {
				h.check(delta, big.NewRat(17, 1), big.NewRat(19, 1))
				h.check(&f.x.Data[0], big.NewRat(1, 1), big.NewRat(2, 1))
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPPairMakeFeasibleMetadataAndStoppedGuards(t *testing.T) {
	for _, op := range []string{"index", "nil-delta", "low", "fixed-low", "upper", "boxed-upper", "heading", "membership-index", "membership-domain"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 0, 1)
			m, err := NewPairCoreSolverBase(f.input())
			h.must(err)
			f.types[0] = ColumnLowBound
			delta := h.pair("7", "9")
			want := ErrBounds
			switch op {
			case "index":
				_, err = m.MakeColumnFeasible(9, delta)
			case "nil-delta":
				_, err = m.MakeColumnFeasible(0, nil)
			case "low":
				f.low.Data = nil
				_, err = m.MakeColumnFeasible(0, delta)
			case "fixed-low":
				f.types[0] = ColumnFixed
				f.low.Data = nil
				_, err = m.MakeColumnFeasible(0, delta)
			case "upper":
				f.types[0] = ColumnUpperBound
				f.up.Data = nil
				_, err = m.MakeColumnFeasible(0, delta)
			case "boxed-upper":
				f.types[0] = ColumnBoxed
				f.up.Data = nil
				_, err = m.MakeColumnFeasible(0, delta)
			case "heading":
				f.heading = nil
				_, err = m.MakeColumnFeasible(0, delta)
				if !sourceDebugEnabled {
					want = nil
				}
			case "membership-index":
				err = m.UpdateColumnInInfSet(99)
			case "membership-domain":
				h.must(h.m.SetInt64(&f.low.Data[0].X, 2))
				h.must(m.InfSet.Resize(0))
				err = m.UpdateColumnInInfSet(0)
			}
			if err != want {
				t.Fatal(err, want)
			}
			if err != nil {
				if _, e := m.MakeColumnFeasible(0, delta); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if m.UpdateColumnInInfSet(0) != ErrExactVectorPending {
					t.Fatal("stopped membership")
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPPairMakeFeasiblePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"low", "upper", "fixed"} {
			failures, expressions := 0, 0
			success := false
			for ordinal := int64(0); ordinal < 240; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 0, 1)
					m, err := NewPairCoreSolverBase(f.input())
					h.must(err)
					a := strings.Repeat("1234567890", 100) + "/7"
					b := strings.Repeat("9876543210", 100) + "/13"
					bound := h.pair(a, b)
					x := h.pair("-"+b, "-"+a)
					f.types[0] = ColumnLowBound
					if op == "upper" {
						f.types[0] = ColumnUpperBound
						bound, x = x, bound
					}
					if op == "fixed" {
						f.types[0] = ColumnFixed
					}
					h.must(f.low.Data[0].Copy(h.m, bound))
					h.must(f.up.Data[0].Copy(h.m, bound))
					h.must(f.x.Data[0].Copy(h.m, x))
					delta := h.pair("0", "0")
					h.budget.FailAfter(ordinal)
					changed, err := m.MakeColumnFeasible(0, delta)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if !changed {
							t.Fatal("missing clamp")
						}
						bx, by := mustLPRat(t, a), mustLPRat(t, b)
						xx, xy := new(big.Rat).Neg(by), new(big.Rat).Neg(bx)
						if op == "upper" {
							bx, xx = xx, bx
							by, xy = xy, by
						}
						h.check(&f.x.Data[0], bx, by)
						h.check(delta, new(big.Rat).Sub(bx, xx), new(big.Rat).Sub(by, xy))
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.feasibilityWork != nil {
							expressions++
						}
						if _, e := m.MakeColumnFeasible(0, delta); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if m.UpdateColumnInInfSet(0) != ErrExactVectorPending {
							t.Fatal("stopped mutation")
						}
					}
					// Delta remains a separately owned caller root after the core closes.
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || expressions == 0 || !success {
				t.Fatal("incomplete physical campaign", direct, op, failures, expressions, success)
			}
		}
	}
}

func TestLPPairMakeFeasibleLeavesOlderComparisonOwned(t *testing.T) {
	for _, zero := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("zero=%t/%d", zero, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 0, 1)
				m, err := NewPairCoreSolverBase(f.input())
				h.must(err)
				v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				f.types[0] = ColumnLowBound
				h.must(f.low.Data[0].Copy(h.m, v))
				if !zero {
					h.must(f.x.Data[0].Copy(h.m, h.pair("3/17", "5/19")))
				}
				delta := h.pair("0", "0")
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
					_, err = m.MakeColumnFeasible(0, delta)
					if (!errors.Is(err, numeral.ErrRationalRawArithmeticPending) && !errors.Is(err, memory.ErrAllocatorPending) && !(sourceDebugEnabled && zero && errors.Is(err, arena.ErrInvalidSlot))) || m.calls.comparison != 0 {
						t.Fatal("adopted older comparison", err)
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(m.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed older comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual older comparison")
		}
	}

}

func TestLPPairMakeFeasibleDeltaBeforeFinalCopyRefusal(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		failures, finalCopies := 0, 0
		success := false
		for ordinal := int64(0); ordinal < 160; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("direct=%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f := lpNewPairCoreFixture(t, h, 0, 1)
				m, err := NewPairCoreSolverBase(f.input())
				h.must(err)
				a := strings.Repeat("1234567890", 100) + "/7"
				b := strings.Repeat("9876543210", 100) + "/13"
				bound := h.pair(a, b)
				h.must(f.low.Data[0].Copy(h.m, bound))
				f.types[0] = ColumnLowBound
				delta := h.pair("0", "0")
				h.budget.FailAfter(ordinal)
				changed, err := m.MakeColumnFeasible(0, delta)
				h.budget.FailAfter(-1)
				if err == nil {
					success = true
					done = true
					if !changed {
						t.Fatal("missing clamp")
					}
					h.check(&f.x.Data[0], mustLPRat(t, a), mustLPRat(t, b))
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if m.feasibilityWork == nil && m.pivotWork == nil {
						finalCopies++
						h.check(delta, mustLPRat(t, a), mustLPRat(t, b))
						if !m.stopped {
							t.Fatal("lost final assignment stop")
						}
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
		if failures == 0 || !success {
			t.Fatal("incomplete cold destination campaign", direct, failures, success)
		}
		// Pooled copies can reuse the just-released expression payloads. Direct
		// allocation exposes the final copy refusal after delta has been assigned.
		if direct && finalCopies == 0 {
			t.Fatal("final-copy failure prefix was not reached")
		}
	}
}

func TestLPPairMakeFeasibleComparisonBeforeDeltaPublication(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, kind := range []ColumnType{ColumnUpperBound, ColumnBoxed} {
			failures, comparisons := 0, 0
			success := false
			for ordinal := int64(0); ordinal < 160; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/type=%d/%d", direct, kind, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 0, 1)
					m, err := NewPairCoreSolverBase(f.input())
					h.must(err)
					a := strings.Repeat("9876543210", 100) + "/7"
					b := strings.Repeat("1234567890", 100) + "/13"
					if mustLPRat(t, a).Cmp(mustLPRat(t, b)) <= 0 {
						t.Fatal("comparison fixture must exceed upper bound")
					}
					h.must(f.x.Data[0].Copy(h.m, h.pair(a, b)))
					h.must(f.up.Data[0].Copy(h.m, h.pair(b, a)))
					f.types[0] = kind
					delta := h.pair("17", "19")
					h.budget.FailAfter(ordinal)
					changed, err := m.MakeColumnFeasible(0, delta)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if !changed {
							t.Fatal("missing clamp")
						}
						h.check(&f.x.Data[0], mustLPRat(t, b), mustLPRat(t, a))
						h.check(delta, new(big.Rat).Sub(mustLPRat(t, b), mustLPRat(t, a)), new(big.Rat).Sub(mustLPRat(t, a), mustLPRat(t, b)))
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.comparison != 0 {
							comparisons++
							if m.pivotWork != nil || m.feasibilityWork != nil {
								t.Fatal("comparison failure constructed delta expression")
							}
							h.check(delta, big.NewRat(17, 1), big.NewRat(19, 1))
							h.check(&f.x.Data[0], mustLPRat(t, a), mustLPRat(t, b))
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
			if failures == 0 || comparisons == 0 || !success {
				t.Fatal("incomplete comparison campaign", direct, kind, failures, comparisons, success)
			}
		}
	}
}
