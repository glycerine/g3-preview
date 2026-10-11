package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPRationalTableauLeavingIndependentModels(t *testing.T) {
	rng := rand.New(rand.NewPCG(971, 391))
	counters := map[string]int{}
	for _, feasible := range []bool{true, false} {
		for world := 0; world < 480; world++ {
			h := newLPPairHarness(t)
			n := 1 + world%6
			f, m := lpRationalTableauSelectionFixture(t, h, uint32(n), uint32(n+4))
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
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, new(big.Rat).Neg(ed).RatString()), uint32(i)))
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
				for extra := 0; extra < (world+i)%3; extra++ {
					h.must(f.a.Set(uint32(i), uint32(n+2+extra), lpScalarBoundValue(h, "1")))
				}
				actualNZ := uint32(len(f.a.Rows[i]))
				m.RowsNZ[i] = 99 - actualNZ
				mult := new(big.Rat).Mul(new(big.Rat).Neg(ed), big.NewRat(int64(direction), 1))
				v, active := lpScalarLeavingLimit(kind, x, lo, up, mult, feasible)
				if active {
					limits[uint32(i)] = limit{v, actualNZ}
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
			start := 0
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
				want = int(candidates[0])
				if len(candidates) > 1 {
					want = int(candidates[random.Next()%uint32(len(candidates))])
				}
				counters["basic"]++
				if len(candidates) > 1 {
					counters["ties"]++
				}
			} else {
				counters["unlimited"]++
			}
			got, e := m.FindLeavingAndTTableau(entering, out)
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

func TestLPRationalTableauLeavingPhysicalOwnership(t *testing.T) {
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
					f, m := lpRationalTableauSelectionFixture(t, h, 2, 3)
					m.RowsNZ = []uint32{2, 2}
					m.SignOfEnteringDelta = 1
					seed := uint32(1)
					for z3rand.New(seed).Next()%2 != 1 {
						seed++
					}
					f.settings.RandomSeed(seed)
					huge := strings.Repeat("1234567890", 12)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, 2, lpScalarBoundValue(h, huge+"1/7")))
						f.types[i] = ColumnLowBound
						h.must(h.m.SetString(&f.low.Data[i], "-"+huge+fmt.Sprint(i+3)+"/11"))
						if op == "inline-minimum" {
							coefficient, e := f.a.ColumnValue(f.a.Columns[2][i])
							h.must(e)
							h.must(h.m.SetInt32(coefficient, -2147483648))
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
					_, err := m.FindLeavingAndTTableau(2, out)
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
						if _, e := m.FindLeavingAndTTableau(2, out); e != ErrExactVectorPending {
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

func TestLPRationalTableauLeavingLiveAliases(t *testing.T) {
	for _, alias := range []string{"x", "low", "upper", "coefficient", "entering-negative-flip"} {
		t.Run(alias, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalTableauSelectionFixture(t, h, 2, 3)
			m.RowsNZ = []uint32{2, 2}
			m.SignOfEnteringDelta = 1
			for i := uint32(0); i < 2; i++ {
				if alias != "entering-negative-flip" || i != 0 {
					h.must(f.a.Set(i, 2, lpScalarBoundValue(h, "1")))
				}
				f.types[i] = ColumnLowBound
			}
			h.must(h.m.SetInt32(&f.low.Data[0], -1))
			h.must(h.m.SetInt32(&f.low.Data[1], -3))
			seed := uint32(1)
			for z3rand.New(seed).Next()%2 != 1 {
				seed++
			}
			f.settings.RandomSeed(seed)
			if len(f.a.Columns[2]) == 2 {
				f.a.Columns[2][0], f.a.Columns[2][1] = f.a.Columns[2][1], f.a.Columns[2][0]
			}
			for k, c := range f.a.Columns[2] {
				f.a.Rows[c.Row][c.Offset].Offset = uint32(k)
			}
			out := &f.x.Data[0]
			wantJ, wantV := 1, int64(3)
			var e error
			switch alias {
			case "low":
				out = &f.low.Data[0]
				wantJ, wantV = 0, 0
			case "upper":
				out = &f.up.Data[0]
				wantJ, wantV = 0, 1
			case "coefficient":
				out, e = f.a.ColumnValue(f.a.Columns[2][1])
				h.must(e)
				wantJ, wantV = 0, 0 // Row1 publishes3, then row0 uses coefficient=3, yielding1/3 below.
			case "entering-negative-flip":
				out = &f.x.Data[2]
				f.types[2] = ColumnBoxed
				h.must(h.m.SetInt32(&f.up.Data[2], 1))
				wantJ, wantV = 2, -2
			}
			got, err := m.FindLeavingAndTTableau(2, out)
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
				if alias == "coefficient" {
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

func TestLPRationalTableauLeavingPreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalTableauSelectionFixture(t, h, 2, 3)
			m.RowsNZ = []uint32{2, 2}
			m.SignOfEnteringDelta = 1
			for i := uint32(0); i < 2; i++ {
				h.must(f.a.Set(i, 2, lpScalarBoundValue(h, "1")))
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
				_, err = m.FindLeavingAndTTableau(2, lpScalarBoundValue(h, "97"))
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) && !errors.Is(err, umemory.ErrAllocatorPending) {
					t.Fatal("unexpected older-owner stop", err)
				}
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
func TestLPRationalTableauLeavingReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"nil", "entering", "empty", "empty-boxed", "zero-first", "zero-second", "basis-first", "basis-second", "offset-first", "offset-second", "type-second", "unknown-first", "jump-unlimited-bound", "jump-bounded-bound"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			for i := uint32(0); i < 2; i++ {
				h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
				h.must(f.a.Set(i, 2, lpScalarBoundValue(h, "1")))
				f.types[i] = ColumnLowBound
				h.must(h.m.Copy(&f.low.Data[i], lpScalarBoundValue(h, fmt.Sprint(-1-2*int(i)))))
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.SignOfEnteringDelta = 1
			m.LeavingCandidates = []uint32{77}
			savedColumns := f.a.Columns
			out := lpScalarBoundValue(h, "97")
			en := uint32(2)
			want := error(ErrBounds)
			wantJ := -1
			first := false
			unchanged := true
			switch phase {
			case "nil":
				out = nil
			case "entering":
				en = 99
			case "empty":
				f.a.Columns[2] = nil
				want = nil
			case "empty-boxed":
				f.a.Columns[2] = nil
				f.types[2] = ColumnBoxed
				h.must(h.m.Copy(&f.up.Data[2], lpScalarBoundValue(h, "2")))
				want = nil
				wantJ = 2
				unchanged = false
			case "zero-first":
				coeff, e := f.a.ColumnValue(f.a.Columns[2][0])
				h.must(e)
				h.must(h.m.SetInt32(coeff, 0))
				f.types[0] = ColumnFree
				f.types[1] = ColumnFree
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
				}
			case "zero-second":
				coeff, e := f.a.ColumnValue(f.a.Columns[2][1])
				h.must(e)
				h.must(h.m.SetInt32(coeff, 0))
				f.types[1] = ColumnFree
				first = true
				unchanged = false
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
					wantJ = 0
				}
			case "basis-first":
				f.basis = nil
			case "basis-second":
				f.basis = f.basis[:1]
				first = true
				unchanged = false
			case "offset-first":
				f.a.Columns[2][0].Offset = 99
			case "offset-second":
				f.a.Columns[2][1].Offset = 99
				first = true
				unchanged = false
			case "type-second":
				f.types = f.types[:1]
				first = true
				unchanged = false
			case "jump-unlimited-bound":
				f.a.Columns[2] = nil
				f.types[2] = ColumnBoxed
				h.must(f.up.Resize(2))
			case "jump-bounded-bound":
				f.types[2] = ColumnBoxed
				h.must(f.up.Resize(2))
				first = true
				unchanged = false
			case "unknown-first":
				f.types[0] = ColumnType(99)
				f.types[1] = ColumnFree
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
				}
			}
			seed := uint32(29)
			f.settings.RandomSeed(seed)
			oracle := z3rand.New(seed)
			got, err := m.FindLeavingAndTTableau(en, out)
			if err != want || got != wantJ {
				t.Fatal(phase, got, err, wantJ, want)
			}
			candidates := []uint32{}
			if phase == "nil" {
				candidates = []uint32{77}
			} else if first {
				candidates = []uint32{0}
			}
			if !reflect.DeepEqual(m.LeavingCandidates, candidates) {
				t.Fatal("candidate prefix", phase, m.LeavingCandidates, candidates)
			}
			if out != nil {
				if unchanged {
					lpSparseRationalCheck(t, h, out, big.NewRat(97, 1))
				} else if phase == "empty-boxed" {
					lpSparseRationalCheck(t, h, out, big.NewRat(2, 1))
				} else {
					lpSparseRationalCheck(t, h, out, big.NewRat(1, 1))
				}
			}
			if f.settings.RandomNext() != oracle.Next() {
				t.Fatal("unexpected RNG")
			}
			if first && want != nil && m.leavingRatio == nil {
				t.Fatal("missing source ratio local")
			}
			if want != nil {
				if _, e = m.FindLeavingAndTTableau(2, lpScalarBoundValue(h, "0")); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			f.a.Columns = savedColumns
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func lpRationalTableauSelectionFixture(t *testing.T, h *lpPairHarness, rows, cols uint32) (*lpRationalCoreFixture, *RationalPrimalCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, rows, cols)
	f.settings.SimplexStrategy = StrategyTableauCosts
	for i := uint32(0); i < rows; i++ {
		h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
	}
	m, e := NewRationalPrimalCoreSolver(f.input())
	h.must(e)
	return f, m
}
