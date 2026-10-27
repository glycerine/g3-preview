// Source: src/util/lp/lp_primal_core_solver.h:140-154,174-318.
package lp

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	z3rand "github.com/glycerine/g3/pkg/util/random"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactRowPricingIndependentSelection(t *testing.T) {
	staleLonger := 0
	for world := 0; world < 480; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 4, 12)
		f.settings.SimplexStrategy = StrategyTableauRows
		grow := world%2 == 0
		f.types[0] = ColumnBoxed
		if grow {
			h.must(f.low.Data[0].Copy(h.m, h.pair("1", "0")))
			h.must(f.up.Data[0].Copy(h.m, h.pair("2", "0")))
		} else {
			h.must(f.low.Data[0].Copy(h.m, h.pair("-2", "0")))
			h.must(f.up.Data[0].Copy(h.m, h.pair("-1", "0")))
		}
		for i := uint32(0); i < 4; i++ {
			h.must(f.a.Set(i, i, &h.pair("1", "0").X))
			if i > 0 {
				f.types[i] = ColumnFree
				if world/3%2 == 0 && (int(i)+world)%2 == 0 {
					f.types[i] = ColumnLowBound
				}
			}
		}
		eligible := make([]bool, 12)
		counts := make([]uint32, 12)
		damages := make([]uint32, 12)
		coefs := make([]*big.Rat, 12)
		order := []uint32{11, 7, 5, 10, 4, 9, 6, 8}
		for _, j := range order {
			kind := ColumnType((world + int(j)) % 5)
			f.types[j] = kind
			low, up := int64(-1), int64(1)
			if world/2%2 == 0 {
				low = 0
			}
			if world/4%2 == 0 {
				up = 0
			}
			if kind == ColumnFixed {
				low, up = 0, 0
			}
			h.must(f.low.Data[j].Copy(h.m, h.pair(fmt.Sprint(low), "0")))
			h.must(f.up.Data[j].Copy(h.m, h.pair(fmt.Sprint(up), "0")))
			a := big.NewRat(int64(1+(world+int(j))%9), int64(3+j))
			if (world+int(j))%3 == 0 {
				a.Neg(a)
			}
			coefs[j] = a
			h.must(f.a.Set(0, j, &h.pair(a.RatString(), "0").X))
			count := 1 + (world+int(j))%4
			counts[j] = uint32(count)
			for i := 1; i < count; i++ {
				h.must(f.a.Set(uint32(i), j, &h.pair("2/7", "0").X))
			}
			for i := 0; i < count; i++ {
				if f.types[i] != ColumnFree {
					damages[j]++
				}
			}
			sign := a.Sign() > 0
			if !grow {
				sign = a.Sign() < 0
			}
			switch kind {
			case ColumnFree:
				eligible[j] = true
			case ColumnLowBound:
				eligible[j] = !sign || low < 0
			case ColumnUpperBound:
				eligible[j] = sign || up > 0
			case ColumnBoxed:
				if sign {
					eligible[j] = low < 0
				} else {
					eligible[j] = up > 0
				}
			}
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, 12)
		m.RowsNZ = []uint32{99, 98, 97, 96}
		for _, j := range order {
			m.ColumnsNZ[j] = 99 - counts[j]
			damage, e := m.NumberOfBasicVarsThatMightBecomeInf(j)
			h.must(e)
			if damage != damages[j] {
				t.Fatal("damage", world, j, damage, damages[j])
			}
		}
		m.InfRowIndexForTableau = 91
		m.BlandModeTableau = world/7%2 == 0
		seed := uint32(world + 1927)
		f.settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		want := int32(-1)
		bestDamage, bestLength := uint32(1000000), uint32(100000000)
		var choices []uint32
		for _, j := range order {
			if !eligible[j] {
				continue
			}
			if m.BlandModeTableau {
				if j < uint32(want) {
					want = int32(j)
				}
				continue
			}
			if damages[j] < bestDamage {
				bestDamage, bestLength = damages[j], counts[j]
				choices = []uint32{j}
			} else if damages[j] == bestDamage && counts[j] <= bestLength && rng.Next()%2 != 0 {
				choices = append(choices, j)
				bestLength = counts[j]
			}
		}
		if !m.BlandModeTableau && len(choices) > 0 {
			want = int32(choices[0])
			if len(choices) > 1 {
				want = int32(choices[rng.Next()%uint32(len(choices))])
			}
			if counts[want] > bestLength {
				staleLonger++
			}
		}
		out := h.pair("17/19", "0")
		got, e := m.FindBeneficialColumnInRowTableauRows(0, &out.X)
		h.must(e)
		if got != want || f.settings.RandomNext() != rng.Next() {
			t.Fatal("selection/RNG", world, got, want)
		}
		if want == -1 {
			lpSparseRationalCheck(t, h, &out.X, big.NewRat(17, 19))
			if m.InfRowIndexForTableau != 0 {
				t.Fatal("infeasible row")
			}
		} else {
			lpSparseRationalCheck(t, h, &out.X, coefs[want])
			if m.InfRowIndexForTableau != 91 {
				t.Fatal("unexpected infeasible row publication")
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	if staleLonger == 0 {
		t.Fatal("did not distinguish source append from replacing shorter-row choices")
	}
}

func TestLPExactRowMonoidIndependentLexicographicBounds(t *testing.T) {
	for world := 0; world < 300; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 1)
		kind := ColumnType(world % 5)
		f.types[0] = kind
		lowX, lowY := big.NewRat(-1, 1), big.NewRat(-1, 13)
		upX, upY := big.NewRat(1, 1), big.NewRat(1, 13)
		x, y := big.NewRat(int64(world/5%3-1), 1), big.NewRat(int64(world/15%3-1), 13)
		if kind == ColumnFixed {
			lowX.Set(x)
			lowY.Set(y)
			upX.Set(x)
			upY.Set(y)
		}
		h.must(f.low.Data[0].Copy(h.m, h.pair(lowX.RatString(), lowY.RatString())))
		h.must(f.up.Data[0].Copy(h.m, h.pair(upX.RatString(), upY.RatString())))
		h.must(f.x.Data[0].Copy(h.m, h.pair(x.RatString(), y.RatString())))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		coefficient := h.pair(fmt.Sprint(world/45%3-1), "0")
		// Borrow the test-owned coefficient for the duration of both pure predicates.
		rc := RationalRowCell{Column: 0, Value: coefficient.X}
		aboveLow := x.Cmp(lowX) > 0 || (x.Cmp(lowX) == 0 && y.Cmp(lowY) > 0)
		belowUp := x.Cmp(upX) < 0 || (x.Cmp(upX) == 0 && y.Cmp(upY) < 0)
		for _, decrease := range []bool{false, true} {
			sign := world/45%3-1 < 0
			if decrease {
				sign = world/45%3-1 > 0
			}
			want := false
			switch kind {
			case ColumnFree:
				want = true
			case ColumnLowBound:
				want = !sign || aboveLow
			case ColumnUpperBound:
				want = sign || belowUp
			case ColumnBoxed:
				if sign {
					want = aboveLow
				} else {
					want = belowUp
				}
			}
			var got bool
			if decrease {
				got, e = m.MonoidCanDecrease(&rc)
			} else {
				got, e = m.MonoidCanIncrease(&rc)
			}
			h.must(e)
			if got != want {
				t.Fatal("monoid", world, decrease, got, want)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactRowNeedsToGrowIndependentModels(t *testing.T) {
	for world := 0; world < 200; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 1)
		// Explicit kind selection avoids relying on enum ordering for infeasibility.
		kinds := []ColumnType{ColumnFixed, ColumnLowBound, ColumnBoxed, ColumnUpperBound}
		kind := kinds[world%4]
		f.types[0] = kind
		below := world/4%2 == 0
		if kind == ColumnLowBound {
			below = true
		}
		if kind == ColumnUpperBound {
			below = false
		}
		h.must(f.low.Data[0].Copy(h.m, h.pair("0", "0")))
		h.must(f.up.Data[0].Copy(h.m, h.pair("0", "0")))
		x, y := "0", "-1/13"
		if !below {
			y = "1/13"
		}
		if world/8%2 == 0 {
			x = "-1/7"
			y = "0"
			if !below {
				x = "1/7"
			}
		}
		h.must(f.x.Data[0].Copy(h.m, h.pair(x, y)))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		grow, e := m.NeedsToGrow(0)
		h.must(e)
		if grow != below {
			t.Fatal(world, grow, below)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactRowPricingBlandLiveOutputAlias(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 1, 6)
	f.types[0] = ColumnLowBound
	h.must(f.low.Data[0].Copy(h.m, h.pair("1", "0")))
	f.types[5] = ColumnFree
	f.types[4] = ColumnLowBound
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.a.Set(0, 5, &h.pair("2", "0").X))
	h.must(f.a.Set(0, 4, &h.pair("-3", "0").X))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	m.BlandModeTableau = true
	output, e := f.a.ColumnValue(f.a.Columns[4][0])
	h.must(e)
	got, e := m.FindBeneficialColumnInRowTableauRows(0, output)
	h.must(e)
	// Copying candidate5's coefficient changes the later candidate4 from negative
	// to positive. At its lower bound that later monomial can no longer decrease.
	if got != 5 {
		t.Fatal("snapshotted later coefficient", got)
	}
	lpSparseRationalCheck(t, h, output, big.NewRat(2, 1))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactRowPricingPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, profile := range []string{"bland", "damage", "bound", "grow"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", profile, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 1, 3)
					f.settings.SimplexStrategy = StrategyTableauRows
					f.types[0] = ColumnLowBound
					h.must(f.low.Data[0].Copy(h.m, h.pair("1", "0")))
					f.types[1], f.types[2] = ColumnFree, ColumnFree
					huge := strings.Repeat("1234567890", 100)
					h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
					h.must(f.a.Set(0, 2, &h.pair(huge+"1/11", "0").X))
					h.must(f.a.Set(0, 1, &h.pair(huge+"3/13", "0").X))
					m, e := NewPairPrimalCoreSolver(f.input())
					h.must(e)
					m.BlandModeTableau = profile == "bland"
					if profile == "bound" {
						f.types[1] = ColumnLowBound
						h.must(f.x.Data[1].Copy(h.m, h.pair(huge+"1/7", "0")))
						h.must(f.low.Data[1].Copy(h.m, h.pair(huge+"1/11", "0")))
					} else if profile == "grow" {
						h.must(f.x.Data[0].Copy(h.m, h.pair(huge+"1/11", "0")))
						h.must(f.low.Data[0].Copy(h.m, h.pair(huge+"1/7", "0")))
					}
					out := h.pair("17/19", "0")
					m.Settings.RandomSeed(91)
					h.budget.FailAfter(ordinal)
					var err error
					var got int32
					var pred bool
					switch profile {
					case "bound":
						pred, err = m.MonoidCanDecrease(&f.a.Rows[0][2])
					case "grow":
						pred, err = m.NeedsToGrow(0)
					default:
						got, err = m.FindBeneficialColumnInRowTableauRows(0, &out.X)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						if profile == "bound" || profile == "grow" {
							if !pred {
								t.Fatal("predicate")
							}
						} else {
							want := int32(1)
							if profile == "damage" {
								rng := z3rand.New(91)
								want = 2
								if rng.Next()%2 != 0 {
									choices := []int32{2, 1}
									want = choices[rng.Next()%2]
								}
							}
							if got != want {
								t.Fatal("column", got, want)
							}
							value := huge + "3/13"
							if got == 2 {
								value = huge + "1/11"
							}
							lpSparseRationalCheck(t, h, &out.X, mustLPRat(t, value))
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.PairCoreSolverBase.calls.pending() {
							t.Fatal("adopted child receipt")
						}
						if _, e = m.FindBeneficialColumnInRowTableauRows(0, &out.X); e != ErrExactVectorPending {
							t.Fatal("retry", e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if t.Failed() {
					return
				}
				if done {
					break
				}
			}
			if !success || (direct && failures == 0) {
				t.Fatal("campaign", profile, direct, failures)
			}
		}
	}
}

func TestLPExactRowPricingReachedPrefixes(t *testing.T) {
	for _, bland := range []bool{false, true} {
		for _, profile := range []string{"stopped", "negative-row", "basis-row", "basis-index", "row-extent", "types", "nil-output", "no-choice", "unknown-kind", "column-extent", "cell-row", "damage-types"} {
			t.Run(fmt.Sprintf("%t/%s", bland, profile), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 1, 3)
				f.types[0] = ColumnLowBound
				f.types[1], f.types[2] = ColumnFree, ColumnFree
				h.must(f.low.Data[0].Copy(h.m, h.pair("1", "0")))
				h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
				h.must(f.a.Set(0, 2, &h.pair("-3", "0").X))
				h.must(f.a.Set(0, 1, &h.pair("-2", "0").X))
				m, e := NewPairPrimalCoreSolver(f.input())
				h.must(e)
				m.BlandModeTableau = bland
				m.InfRowIndexForTableau = 91
				output := h.pair("17/19", "0")
				out := &output.X
				row := int32(0)
				want := error(nil)
				wantColumn := int32(-2)
				savedRows, savedColumns := f.a.Rows, f.a.Columns
				savedCell := f.a.Columns[2][0]
				switch profile {
				case "stopped":
					m.stopped = true
					if _, e = m.NumberOfBasicVarsThatMightBecomeInf(0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if _, e = m.FindBeneficialColumnInRowTableauRowsBlandMode(0, nil); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					want = ErrExactVectorPending
				case "negative-row":
					row = -1
					want = ErrBounds
				case "basis-row":
					row = 7
					want = ErrBounds
				case "basis-index":
					f.basis[0] = 7
					want = ErrBounds
				case "row-extent":
					f.a.Rows = nil
					want = ErrBounds
				case "types":
					f.types = f.types[:1]
					want = ErrBounds
				case "nil-output":
					out = nil
					want = ErrBounds
				case "no-choice":
					f.types[1], f.types[2] = ColumnFixed, ColumnFixed
					out = nil
					wantColumn = -1
				case "unknown-kind":
					f.types[2] = ColumnType(99)
					f.types[1] = ColumnFixed
					wantColumn = -1
					if sourceDebugEnabled {
						want = ErrAssertion
					}
				case "column-extent":
					f.a.Columns = f.a.Columns[:2]
					if !bland {
						want = ErrBounds
					} else {
						wantColumn = 1
					}
				case "cell-row":
					f.a.Columns[2][0].Row = 9
					if !bland {
						want = ErrBounds
					} else {
						wantColumn = 1
					}
				case "damage-types":
					// NeedsToGrow uses basic0; damage for candidate2 reaches a different,
					// invalid basic index through the real column cell.
					f.basis = append(f.basis, 9)
					f.a.Columns[2][0].Row = 1
					if !bland {
						want = ErrBounds
					} else {
						wantColumn = 1
					}
				}
				got, err := m.FindBeneficialColumnInRowTableauRows(row, out)
				if err != want || (err == nil && wantColumn != -2 && got != wantColumn) {
					t.Fatal(got, err, wantColumn, want)
				}
				if err != nil {
					if _, retry := m.FindBeneficialColumnInRowTableauRows(0, &output.X); retry != ErrExactVectorPending {
						t.Fatal("retry", retry)
					}
				}
				if err == nil && got == -1 {
					if m.InfRowIndexForTableau != 0 {
						t.Fatal("missing infeasible row")
					}
				} else if m.InfRowIndexForTableau != 91 {
					t.Fatal("premature row publication")
				}
				f.a.Rows = savedRows
				f.a.Columns = savedColumns
				f.a.Columns[2][0] = savedCell
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactRowPricingPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"grow", "monoid", "bland", "damage"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 2)
			f.types[0], f.types[1] = ColumnLowBound, ColumnLowBound
			huge := strings.Repeat("1234567890", 100)
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("1", "0").X))
			h.must(f.x.Data[0].Copy(h.m, h.pair(huge+"1/11", "0")))
			h.must(f.low.Data[0].Copy(h.m, h.pair(huge+"1/7", "0")))
			h.must(f.x.Data[1].Copy(h.m, h.pair(huge+"1/7", "0")))
			h.must(f.low.Data[1].Copy(h.m, h.pair(huge+"1/11", "0")))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			m.BlandModeTableau = op == "bland"
			out := h.pair("17/19", "0")
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
				switch op {
				case "grow":
					_, err = m.NeedsToGrow(0)
				case "monoid":
					_, err = m.MonoidCanDecrease(&f.a.Rows[0][1])
				default:
					_, err = m.FindBeneficialColumnInRowTableauRows(0, &out.X)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) {
					t.Fatal(op, err)
				}
				if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			h.must(m.Close())
			f.close()
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !matched {
			t.Fatal("no actual older comparison", op)
		}
	}
}

func TestLPExactRowPredicatesReachedShortCircuits(t *testing.T) {
	for _, profile := range []string{"stopped", "nil", "free-grow", "grow-low", "monoid-x", "monoid-low", "monoid-up", "monoid-types", "monoid-infeasible", "unknown", "free-x"} {
		for _, decrease := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", profile, decrease), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 0, 1)
				f.types[0] = ColumnBoxed
				h.must(f.low.Data[0].Copy(h.m, h.pair("-1", "0")))
				h.must(f.up.Data[0].Copy(h.m, h.pair("1", "0")))
				coeff := h.pair("1", "0")
				if !decrease {
					h.must(h.m.SetInt32(&coeff.X, -1))
				}
				cell := &RationalRowCell{Column: 0, Value: coeff.X}
				m, e := NewPairPrimalCoreSolver(f.input())
				h.must(e)
				savedX, savedLow, savedUp := f.x.Data, f.low.Data, f.up.Data
				want := error(nil)
				growCall := false
				value := false
				switch profile {
				case "stopped":
					m.stopped = true
					if _, e = m.NumberOfBasicVarsThatMightBecomeInf(0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if _, e = m.FindBeneficialColumnInRowTableauRowsBlandMode(0, nil); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					want = ErrExactVectorPending
				case "nil":
					cell = nil
					want = ErrBounds
				case "free-grow":
					f.types[0] = ColumnFree
					growCall = true
					if sourceDebugEnabled {
						want = ErrAssertion
					}
				case "grow-low":
					f.types[0] = ColumnLowBound
					h.must(f.x.Data[0].Copy(h.m, h.pair("-2", "0")))
					f.low.Data = nil
					growCall = true
					want = ErrBounds
				case "monoid-x":
					f.x.Data = nil
					want = ErrBounds
				case "monoid-low":
					f.low.Data = nil
					want = ErrBounds
				case "monoid-up":
					f.up.Data = nil
					h.must(h.m.Neg(&coeff.X))
					cell.Value = coeff.X
					want = ErrBounds
				case "monoid-infeasible":
					h.must(f.x.Data[0].Copy(h.m, h.pair("2", "0")))
					value = true
					if sourceDebugEnabled {
						want = ErrAssertion
					}
				case "monoid-types":
					f.types = nil
					want = ErrBounds
				case "unknown":
					f.types[0] = ColumnType(99)
					if sourceDebugEnabled {
						want = ErrAssertion
					}
				case "free-x":
					f.types[0] = ColumnFree
					f.x.Data = nil
					value = true
					if sourceDebugEnabled {
						want = ErrBounds
					}
				}
				var got bool
				var err error
				if growCall {
					got, err = m.NeedsToGrow(0)
				} else if decrease {
					got, err = m.MonoidCanDecrease(cell)
				} else {
					got, err = m.MonoidCanIncrease(cell)
				}
				if err != want || (err == nil && got != value) {
					t.Fatal(got, err, value, want)
				}
				f.x.Data, f.low.Data, f.up.Data = savedX, savedLow, savedUp
				if err != nil {
					if _, e = m.NeedsToGrow(0); e != ErrExactVectorPending {
						t.Fatal("retry", e)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}
