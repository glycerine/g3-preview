package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

// Independent scalar boundary geometry, with exact Harris epsilon zero.
func lpScalarLeavingLimit(kind ColumnType, x, lo, up, mult *big.Rat, feasible bool) (*big.Rat, bool) {
	positive := mult.Sign() > 0
	var target *big.Rat
	active, zero := false, false
	if feasible {
		switch kind {
		case ColumnUpperBound:
			if positive {
				target = up
				active = true
			}
		case ColumnLowBound:
			if !positive {
				target = lo
				active = true
			}
		case ColumnFixed, ColumnBoxed:
			active = true
			if positive {
				target = up
			} else {
				target = lo
			}
		}
	} else {
		switch kind {
		case ColumnUpperBound:
			target = up
			if positive {
				active = x.Cmp(up) < 0
				zero = x.Cmp(up) == 0
			} else {
				active = x.Cmp(up) > 0
			}
		case ColumnLowBound:
			target = lo
			if positive {
				active = x.Cmp(lo) < 0
			} else {
				active = x.Cmp(lo) > 0
				zero = x.Cmp(lo) == 0
			}
		case ColumnFixed, ColumnBoxed:
			if positive {
				if x.Cmp(lo) < 0 {
					target = lo
					active = true
				} else {
					target = up
					active = x.Cmp(up) < 0
					zero = x.Cmp(up) == 0
				}
			} else {
				if x.Cmp(up) > 0 {
					target = up
					active = true
				} else {
					target = lo
					active = x.Cmp(lo) > 0
					zero = x.Cmp(lo) == 0
				}
			}
		}
	}
	if zero {
		return new(big.Rat), true
	}
	if !active {
		return nil, false
	}
	q := new(big.Rat).Quo(new(big.Rat).Sub(target, x), mult)
	if q.Sign() < 0 {
		q.SetInt64(0)
	}
	return q, true
}

func TestLPRationalPrimalLeavingIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewPCG(971, 391))
	counters := map[string]int{}
	for _, feasible := range []bool{true, false} {
		for world := 0; world < 480; world++ {
			h := newLPPairHarness(t)
			n := 1 + world%6
			f, m := lpRationalPricingFixture(t, h, uint32(n), uint32(n+2))
			entering := uint32(n)
			m.RowsNZ = make([]uint32, n)
			direction := int32(1)
			if world%2 == 1 {
				direction = -1
			}
			m.SignOfEnteringDelta = direction
			type limit struct {
				v  *big.Rat
				nz uint32
			}
			limits := map[uint32]limit{}
			order := rng.Perm(n)
			for _, i := range order {
				ed := big.NewRat(int64(1+rng.IntN(7)), int64(1+rng.IntN(5)))
				if rng.IntN(2) == 0 {
					ed.Neg(ed)
				}
				h.must(f.a.Set(uint32(i), entering, lpScalarBoundValue(h, ed.RatString())))
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, ed.RatString()), uint32(i)))
				kind := ColumnType((world + i) % 5)
				if feasible && world%11 == 0 {
					kind = ColumnFree
				}
				if !feasible && i == 0 {
					kind = ColumnBoxed
				}
				f.types[i] = kind
				x, lo, up := new(big.Rat), new(big.Rat), new(big.Rat)
				if feasible {
					distance := big.NewRat(int64((world+i)%4), 3)
					if kind == ColumnFixed {
						distance.SetInt64(0)
					}
					up.Mul(distance, new(big.Rat).Abs(ed))
					lo.Neg(up)
				} else {
					x.SetInt64(int64((world+i)%6 - 2))
					lo.SetInt64(-1)
					up.SetInt64(2)
					if i == 0 {
						x.SetInt64(3)
					}
					if kind == ColumnFixed {
						up.Set(lo)
					}
					if x.Sign() != 0 {
						h.must(f.a.Set(uint32(i), uint32(n+1), lpScalarBoundValue(h, new(big.Rat).Neg(x).RatString())))
					}
					if (kind == ColumnLowBound && x.Cmp(lo) < 0) || (kind == ColumnUpperBound && x.Cmp(up) > 0) || ((kind == ColumnFixed || kind == ColumnBoxed) && (x.Cmp(lo) < 0 || x.Cmp(up) > 0)) {
						h.must(m.InfSet.Insert(uint32(i)))
					}
				}
				for k, v := range []*big.Rat{x, lo, up} {
					h.must(h.m.SetString(&[]*IndexedRationalVector{f.x, f.low, f.up}[k].Data[i], v.RatString()))
				}
				m.RowsNZ[i] = uint32((world+i)%3 + 1)
				mult := new(big.Rat).Mul(new(big.Rat).Neg(ed), big.NewRat(int64(direction), 1))
				v, active := lpScalarLeavingLimit(kind, x, lo, up, mult, feasible)
				if active {
					limits[uint32(i)] = limit{v, m.RowsNZ[i]}
				}
			}
			if !feasible {
				h.must(h.m.SetInt32(&f.x.Data[n+1], 1))
				if m.CurrentXIsFeasible() {
					t.Fatal("infeasible fixture")
				}
			}
			kind := ColumnType(world / 3 % 5)
			f.types[entering] = kind
			lo, up := big.NewRat(-int64(world%5+1), 3), big.NewRat(int64(world%5+1), 3)
			h.must(h.m.SetString(&f.low.Data[entering], lo.RatString()))
			h.must(h.m.SetString(&f.up.Data[entering], up.RatString()))
			seed := uint32(world + 17)
			f.settings.RandomSeed(seed)
			random := z3rand.New(seed)
			start := int(random.Next() % uint32(n))
			var best *limit
			for _, v := range limits {
				v := v
				if best == nil || v.v.Cmp(best.v) < 0 || (v.v.Cmp(best.v) == 0 && v.nz < best.nz) {
					best = &v
				}
			}
			var candidates []uint32
			if best != nil {
				for k := 0; k < n; k++ {
					j := uint32(order[(start+k)%n])
					if v, ok := limits[j]; ok && v.v.Cmp(best.v) == 0 && v.nz == best.nz {
						candidates = append(candidates, j)
					}
				}
			}
			out := lpScalarBoundValue(h, "97")
			alias := world%13 == 0
			before := big.NewRat(97, 1)
			if alias {
				out = &f.x.Data[entering]
				before.SetInt64(0)
			}
			wantValue := new(big.Rat).Set(before)
			if best != nil {
				wantValue.Set(best.v)
			}
			enteringX := new(big.Rat)
			if alias {
				enteringX.Set(wantValue)
			}
			step := new(big.Rat)
			if direction > 0 {
				step.Sub(up, enteringX)
			} else {
				step.Sub(enteringX, lo)
			}
			flip := kind == ColumnBoxed
			if best != nil {
				flip = (flip || kind == ColumnUpperBound && direction > 0 || kind == ColumnLowBound && direction < 0) && step.Cmp(best.v) <= 0
			}
			if best != nil && flip && step.Sign() < 0 {
				// Accepted bounded negative flips violate the source assertion; exercised separately.
				out = lpScalarBoundValue(h, "97")
				alias = false
				enteringX.SetInt64(0)
				if direction > 0 {
					step.Set(up)
				} else {
					step.Neg(lo)
				}
				flip = step.Cmp(best.v) <= 0
			}
			want := -1
			if flip {
				want = int(entering)
				wantValue.Set(step)
				counters["flip"]++
			} else if best != nil {
				want = int(candidates[random.Next()%uint32(len(candidates))])
				counters["basic"]++
				if len(candidates) > 1 {
					counters["ties"]++
				}
			} else {
				counters["unlimited"]++
			}
			got, e := m.FindLeavingAndTPrecise(entering, out)
			h.must(e)
			if got != want || !reflect.DeepEqual(m.LeavingCandidates, candidates) {
				t.Fatal(feasible, world, got, want, m.LeavingCandidates, candidates)
			}
			lpSparseRationalCheck(t, h, out, wantValue)
			if f.settings.RandomNext() != random.Next() {
				t.Fatal("RNG", feasible, world)
			}
			if m.leavingMultiplier != nil || m.leavingRatio != nil {
				t.Fatal("scratch retained")
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
	for _, k := range []string{"flip", "basic", "ties", "unlimited"} {
		if counters[k] == 0 {
			t.Fatal(k, counters)
		}
	}
}

func TestLPRationalPrimalLeavingPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"ratio", "tie", "flip", "inline-minimum"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1600; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalPricingFixture(t, h, 2, 3)
					m.RowsNZ = []uint32{2, 2}
					m.SignOfEnteringDelta = 1
					seed := uint32(1)
					for z3rand.New(seed).Next()%2 != 1 {
						seed++
					}
					f.settings.RandomSeed(seed)
					huge := strings.Repeat("1234567890", 12)
					for i := uint32(0); i < 2; i++ {
						h.must(m.Ed.SetValue(lpScalarBoundValue(h, huge+"1/7"), i))
						f.types[i] = ColumnLowBound
						h.must(h.m.SetString(&f.low.Data[i], "-"+huge+fmt.Sprint(i+3)+"/11"))
						if op == "inline-minimum" {
							h.must(h.m.SetInt32(&m.Ed.Data[i], -2147483648))
							f.types[i] = ColumnUpperBound
							h.must(h.m.SetString(&f.up.Data[i], huge+fmt.Sprint(i+3)+"/11"))
						}
					}
					if op == "tie" {
						h.must(h.m.Copy(&f.low.Data[1], &f.low.Data[0]))
					}
					if op == "flip" {
						f.types[2] = ColumnBoxed
						h.must(h.m.SetString(&f.up.Data[2], "1/3"))
					}
					out := lpScalarBoundValue(h, "97")
					h.budget.FailAfter(ordinal)
					_, err := m.FindLeavingAndTPrecise(2, out)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted delegated receipt")
						}
						if _, e := m.FindLeavingAndTPrecise(2, out); e != ErrExactVectorPending {
							t.Fatal(e)
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
			if !success || failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalPrimalLeavingGuardsAndBreakpointDispatch(t *testing.T) {
	for _, op := range []string{"nil", "empty", "zero", "ed-index", "basis-index", "row-nz", "unknown", "breakpoints"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 2)
			h.must(m.Ed.SetValue(lpScalarBoundValue(h, "1"), 0))
			m.RowsNZ = []uint32{2}
			m.SignOfEnteringDelta = 1
			f.types[0] = ColumnLowBound
			h.must(h.m.SetInt32(&f.low.Data[0], -1))
			out := lpScalarBoundValue(h, "97")
			want := ErrBounds
			m.LeavingCandidates = []uint32{77}
			seed := uint32(37)
			f.settings.RandomSeed(seed)
			random := z3rand.New(seed)
			switch op {
			case "nil":
				out = nil
			case "empty":
				h.must(m.Ed.Clear())
				random.Next()
			case "zero":
				h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "ed-index":
				m.Ed.Index[0] = 99
			case "basis-index":
				f.basis = nil
			case "row-nz":
				m.RowsNZ = nil
			case "unknown":
				f.types[0] = ColumnType(99)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "breakpoints":
				h.must(m.InfSet.Insert(0))
				f.settings.UseBreakpointsInFeasibilitySearch = true
				h.must(m.BreakpointQueue.Resize(4))
				h.must(h.m.SetInt32(&f.x.Data[0], 4))
				h.must(h.m.SetInt32(&f.low.Data[0], 0))
				h.must(h.m.SetInt32(&m.D.Data[1], -1))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			got, err := m.FindLeavingAndTPrecise(1, out)
			if err != want {
				t.Fatal(got, err, want)
			}
			if op == "empty" || op == "nil" || op == "breakpoints" {
				if !reflect.DeepEqual(m.LeavingCandidates, []uint32{77}) {
					t.Fatal("early candidate prefix")
				}
				if f.settings.RandomNext() != random.Next() {
					t.Fatal("early RNG prefix")
				}
			}
			if op == "row-nz" {
				if !reflect.DeepEqual(m.LeavingCandidates, []uint32{0}) {
					t.Fatal("candidate append prefix")
				}
				lpSparseRationalCheck(t, h, out, big.NewRat(1, 1))
			}
			if op == "breakpoints" && !sourceDebugEnabled {
				if got != 0 {
					t.Fatal(got)
				}
				lpSparseRationalCheck(t, h, out, big.NewRat(4, 1))
			}
			if err != nil {
				if _, e := m.FindLeavingAndTPrecise(1, lpScalarBoundValue(h, "0")); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalLeavingLiveAliases(t *testing.T) {
	for _, alias := range []string{"x", "low", "upper", "ed", "entering-negative-flip"} {
		t.Run(alias, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 2, 3)
			m.RowsNZ = []uint32{2, 2}
			m.SignOfEnteringDelta = 1
			for i := uint32(0); i < 2; i++ {
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, "1"), i))
				f.types[i] = ColumnLowBound
			}
			h.must(h.m.SetInt32(&f.low.Data[0], -1))
			h.must(h.m.SetInt32(&f.low.Data[1], -3))
			seed := uint32(1)
			for z3rand.New(seed).Next()%2 != 1 {
				seed++
			}
			f.settings.RandomSeed(seed)
			out := &f.x.Data[0]
			wantJ, wantV := 1, int64(3)
			switch alias {
			case "low":
				out = &f.low.Data[0]
				wantJ, wantV = 0, 0
			case "upper":
				out = &f.up.Data[0]
				wantJ, wantV = 0, 1
			case "ed":
				out = &m.Ed.Data[0]
				wantJ, wantV = 0, 0 // Row1 publishes3, then row0 uses Ed=3, yielding1/3 below.
			case "entering-negative-flip":
				h.must(m.Ed.Clear())
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, "1"), 1))
				out = &f.x.Data[2]
				f.types[2] = ColumnBoxed
				h.must(h.m.SetInt32(&f.up.Data[2], 1))
				wantJ, wantV = 2, -2
			}
			got, err := m.FindLeavingAndTPrecise(2, out)
			if alias == "entering-negative-flip" && sourceDebugEnabled {
				if err != ErrAssertion || got != -1 || m.leavingRatio == nil {
					t.Fatal(got, err)
				}
				lpSparseRationalCheck(t, h, out, big.NewRat(3, 1))
				lpSparseRationalCheck(t, h, &m.leavingRatio.Data[0], big.NewRat(-2, 1))
			} else {
				h.must(err)
				if got != wantJ {
					t.Fatal(got, wantJ)
				}
				v := big.NewRat(wantV, 1)
				if alias == "ed" {
					v.SetFrac64(1, 3)
				}
				lpSparseRationalCheck(t, h, out, v)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalLeavingSecondRowMetadataAndSparseIndex(t *testing.T) {
	for _, op := range []string{"missing-nz", "only-row0", "bad-entering"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 2, 3)
			m.RowsNZ = []uint32{2}
			m.SignOfEnteringDelta = 1
			h.must(m.Ed.SetValue(lpScalarBoundValue(h, "1"), 0))
			if op != "only-row0" {
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, "1"), 1))
			}
			for j := 0; j < 2; j++ {
				f.types[j] = ColumnLowBound
				h.must(h.m.SetInt32(&f.low.Data[j], -1))
			}
			seed := uint32(1)
			for z3rand.New(seed).Next()%2 != 0 {
				seed++
			}
			f.settings.RandomSeed(seed)
			entering := uint32(2)
			want := ErrBounds
			if op == "only-row0" {
				want = nil
			}
			if op == "bad-entering" {
				entering = 99
				f.types[0] = ColumnFree
				f.types[1] = ColumnFree
			}
			out := lpScalarBoundValue(h, "97")
			got, err := m.FindLeavingAndTPrecise(entering, out)
			if err != want {
				t.Fatal(got, err, want)
			}
			if op == "missing-nz" {
				if !reflect.DeepEqual(m.LeavingCandidates, []uint32{0}) || m.leavingRatio == nil {
					t.Fatal("lost row prefix")
				}
				lpSparseRationalCheck(t, h, out, big.NewRat(1, 1))
			}
			if op == "only-row0" {
				if got != 0 {
					t.Fatal(got)
				}
				lpSparseRationalCheck(t, h, out, big.NewRat(1, 1))
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalLeavingPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 2, 3)
			m.RowsNZ = []uint32{2, 2}
			m.SignOfEnteringDelta = 1
			for i := uint32(0); i < 2; i++ {
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, "1"), i))
				f.types[i] = ColumnLowBound
				h.must(h.m.SetString(&f.low.Data[i], "-1/7"))
			}
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, err := h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_, err = m.FindLeavingAndTPrecise(2, lpScalarBoundValue(h, "97"))
				if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted caller comparison", err)
				}
			}
			h.must(m.Close())
			f.close()
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("destroyed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual caller comparison")
	}
}
