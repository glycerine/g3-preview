package lp

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactTableauValuesIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 3, 6)
		f.settings.SimplexStrategy = StrategyTableauCosts
		using := world%2 == 0
		if !using && world/2%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		f.settings.UseBreakpointsInFeasibilitySearch = world/3%2 == 0
		a := make([][]*big.Rat, 3)
		x, y := make([]*big.Rat, 6), make([]*big.Rat, 6)
		oldCost := make([]*big.Rat, 6)
		for j := 0; j < 6; j++ {
			x[j] = big.NewRat(int64((world+j*7)%17-8), 3)
			y[j] = big.NewRat(int64((world+j)%5-2), 7)
			oldCost[j] = big.NewRat(int64(j+1), 11)
			h.must(f.x.Data[j].Copy(h.m, h.pair(x[j].RatString(), y[j].RatString())))
			h.must(h.m.SetString(&f.costs.Data[j], oldCost[j].RatString()))
			if j < 3 {
				f.types[j] = ColumnType((world/2 + j) % 5)
			}
			h.must(f.low.Data[j].Copy(h.m, h.pair("-1", "0")))
			h.must(f.up.Data[j].Copy(h.m, h.pair("1", "0")))
			if f.types[j] == ColumnFixed {
				h.must(f.low.Data[j].Copy(h.m, h.pair("0", "0")))
				h.must(f.up.Data[j].Copy(h.m, h.pair("0", "0")))
			}
		}
		for i := range a {
			a[i] = make([]*big.Rat, 6)
			bx, by := new(big.Rat), new(big.Rat)
			for j := range a[i] {
				v := new(big.Rat)
				if j == i {
					v.SetInt64(1)
				} else if j >= 3 {
					v.SetFrac64(int64((world+i*3+j*5)%13-6), int64(i+j+2))
				}
				a[i][j] = v
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
				}
				bx.Add(bx, new(big.Rat).Mul(v, x[j]))
				by.Add(by, new(big.Rat).Mul(v, y[j]))
			}
			h.must(f.b.Data[i].Copy(h.m, h.pair(bx.RatString(), by.RatString())))
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitInfSet())
		if using {
			h.must(m.InitInfeasibilityCosts())
			for j := range oldCost {
				v := int64(0)
				if j < 3 {
					lo, up := big.NewRat(-1, 1), big.NewRat(1, 1)
					if f.types[j] == ColumnFixed {
						lo.SetInt64(0)
						up.SetInt64(0)
					}
					v = lpAdvanceEqualOracleCost(x[j], y[j], f.types[j], lo, up)
					if f.settings.UseBreakpointsInFeasibilitySearch {
						v = -v
					}
				}
				oldCost[j] = big.NewRat(v, 1)
			}
		}
		// Independent initial reduced costs, avoiding the production initializer.
		for j := 0; j < 6; j++ {
			d := new(big.Rat)
			if j >= 3 {
				d.Set(oldCost[j])
				for i := range a {
					d.Sub(d, new(big.Rat).Mul(oldCost[i], a[i][j]))
				}
			}
			h.must(h.m.SetString(&m.D.Data[j], d.RatString()))
		}
		en := 3 + world%3
		dx, dy := big.NewRat(int64(world%9-4), 5), big.NewRat(int64(world%3-1), 11)
		delta := h.pair(dx.RatString(), dy.RatString())
		x[en].Add(x[en], dx)
		y[en].Add(y[en], dy)
		for i := range a {
			x[i].Sub(x[i], new(big.Rat).Mul(dx, a[i][en]))
			y[i].Sub(y[i], new(big.Rat).Mul(dy, a[i][en]))
		}
		newCost := make([]*big.Rat, 6)
		for j := range newCost {
			newCost[j] = new(big.Rat).Set(oldCost[j])
			if using && j < 3 {
				lo, up := big.NewRat(-1, 1), big.NewRat(1, 1)
				if f.types[j] == ColumnFixed {
					lo.SetInt64(0)
					up.SetInt64(0)
				}
				v := lpAdvanceEqualOracleCost(x[j], y[j], f.types[j], lo, up)
				if f.settings.UseBreakpointsInFeasibilitySearch {
					v = -v
				}
				newCost[j].SetInt64(v)
			}
		}
		m.Status = StatusOptimal
		m.IterationsNoCostGrowth = 73
		m.D.Index = []uint32{5, 3}
		h.must(m.UpdateXTableau(uint32(en), delta))
		for j := range x {
			h.check(&f.x.Data[j], x[j], y[j])
			lpSparseRationalCheck(t, h, &f.costs.Data[j], newCost[j])
			if j < 3 {
				lo, up := big.NewRat(-1, 1), big.NewRat(1, 1)
				if f.types[j] == ColumnFixed {
					lo.SetInt64(0)
					up.SetInt64(0)
				}
				if m.InfSet.Contains(uint32(j)) != (lpAdvanceEqualOracleCost(x[j], y[j], f.types[j], lo, up) != 0) {
					t.Fatal("membership", world, j)
				}
			}
			d := new(big.Rat)
			if j >= 3 {
				c := oldCost
				if using {
					c = newCost
				}
				d.Set(c[j])
				for i := range a {
					d.Sub(d, new(big.Rat).Mul(c[i], a[i][j]))
				}
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], d)
		}
		if m.Status != StatusOptimal || m.IterationsNoCostGrowth != 73 || !reflect.DeepEqual(m.D.Index, []uint32{5, 3}) || m.Factorization != nil {
			t.Fatal("state")
		}
		off, e := m.AMultXIsOff()
		h.must(e)
		if off {
			t.Fatal("residual")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactTableauBasisIndependentElimination(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyTableauCosts
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		p := big.NewRat(int64(1+world%7), int64(2+world%5))
		q := big.NewRat(int64(world%9-4), 3)
		if world%3 == 0 {
			p.Neg(p)
		}
		a := [][]*big.Rat{{big.NewRat(1, 1), new(big.Rat), new(big.Rat).Set(p), big.NewRat(2, 7)}, {new(big.Rat), big.NewRat(1, 1), new(big.Rat).Set(q), big.NewRat(-3, 11)}}
		x := []*big.Rat{big.NewRat(3, 2), big.NewRat(-2, 3), big.NewRat(1, 7), big.NewRat(-1, 5)}
		if world%3 == 0 {
			for i := range a {
				x[i].Neg(new(big.Rat).Add(new(big.Rat).Mul(a[i][2], x[2]), new(big.Rat).Mul(a[i][3], x[3])))
			}
		}

		rhs := []*big.Rat{new(big.Rat), new(big.Rat)}
		cost := []*big.Rat{big.NewRat(3, 1), big.NewRat(-2, 1), big.NewRat(5, 1), big.NewRat(7, 1)}
		for j := range x {
			h.must(f.x.Data[j].Copy(h.m, h.pair(x[j].RatString(), "0")))
			h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
		}
		for i := range a {
			for j, v := range a[i] {
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
				}
				rhs[i].Add(rhs[i], new(big.Rat).Mul(v, x[j]))
			}
			h.must(f.b.Data[i].Copy(h.m, h.pair(rhs[i].RatString(), "0")))
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitReducedCostsTableau())
		m.StartTracingBasisChanges()
		delta := big.NewRat(int64(world%11-5), 13)
		x[2].Add(x[2], delta)
		x[0].Sub(x[0], new(big.Rat).Mul(p, delta))
		x[1].Sub(x[1], new(big.Rat).Mul(q, delta))
		for j := range a[0] {
			a[0][j].Quo(a[0][j], p)
		}
		rhs[0].Quo(rhs[0], p)
		for j := range a[1] {
			a[1][j].Sub(a[1][j], new(big.Rat).Mul(q, a[0][j]))
		}
		// Source pivots matrix rows but does not eliminate the nonpivot RHS.
		// Keep its stored value distinct from the mathematically transformed RHS.
		transformedRHS := new(big.Rat).Sub(rhs[1], new(big.Rat).Mul(q, rhs[0]))
		oldD := make([]*big.Rat, 4)
		for j := range oldD {
			oldD[j] = new(big.Rat)
			if j >= 2 {
				oldD[j].Sub(cost[j], new(big.Rat).Add(new(big.Rat).Mul(cost[0], []*big.Rat{p, big.NewRat(2, 7)}[j-2]), new(big.Rat).Mul(cost[1], []*big.Rat{q, big.NewRat(-3, 11)}[j-2])))
			}
		}
		ok, e := m.UpdateBasisAndXTableau(2, 0, h.pair(delta.RatString(), "0"))
		h.must(e)
		if !ok {
			t.Fatal("outcome")
		}
		if !reflect.DeepEqual(f.basis, []uint32{2, 1}) || !reflect.DeepEqual(f.nb, []uint32{3, 0}) || !reflect.DeepEqual(m.Trace, []uint32{2, 0}) {
			t.Fatal("basis", world, f.basis, f.nb, m.Trace)
		}
		for i := range a {
			h.check(&f.b.Data[i], rhs[i], new(big.Rat))
			for j := range a[i] {
				got := new(big.Rat)
				for _, cell := range f.a.Rows[i] {
					if int(cell.Column) == j {
						v, e := h.m.String(&cell.Value)
						h.must(e)
						got.SetString(v)
					}
				}
				if got.Cmp(a[i][j]) != 0 {
					t.Fatal("matrix", world, i, j, got, a[i][j])
				}
			}
		}
		for j := range x {
			h.check(&f.x.Data[j], x[j], new(big.Rat))
			want := new(big.Rat).Set(oldD[j])
			if f.settings.SimplexStrategy == StrategyTableauCosts {
				want.SetInt64(0)
				if f.heading[j] < 0 {
					want.Set(cost[j])
					want.Sub(want, new(big.Rat).Mul(cost[2], a[0][j]))
					want.Sub(want, new(big.Rat).Mul(cost[1], a[1][j]))
				}
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], want)
		}
		actualDot := h.pair("0", "0")
		h.must(f.a.DotPairRow(1, f.x.Data, actualDot))
		h.check(actualDot, transformedRHS, new(big.Rat))
		if world%3 == 0 {
			off, e := m.AMultXIsOff()
			h.must(e)
			if off {
				t.Fatal("homogeneous AX")
			}
		} else if q.Sign() != 0 && rhs[0].Sign() != 0 && transformedRHS.Cmp(rhs[1]) == 0 {
			t.Fatal("missing independent nonpivot RHS discrepancy")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactTableauValuesPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"values", "infeasible", "basis"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1500; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 1, 2)
					f.settings.SimplexStrategy = StrategyTableauCosts
					h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
					huge := strings.Repeat("1234567890", 12)
					h.must(f.a.Set(0, 1, &h.pair(huge+"1/13", "0").X))
					m, e := NewPairPrimalCoreSolver(f.input())
					h.must(e)
					if op == "infeasible" {
						f.types[0] = ColumnLowBound
						h.must(f.low.Data[0].Copy(h.m, h.pair(huge+"1/19", "0")))
						h.must(m.InitInfeasibilityCosts())
						h.must(m.InitReducedCostsTableau())
					}
					delta := h.pair(huge+"1/17", huge+"3/23")
					h.budget.FailAfter(ordinal)
					var err error
					if op == "basis" {
						_, err = m.UpdateBasisAndXTableau(1, 0, delta)
					} else {
						err = m.UpdateXTableau(1, delta)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.PairCoreSolverBase.calls.pending() {
							t.Fatal("adopted child")
						}
						if e = m.UpdateXTableau(1, delta); e != ErrExactVectorPending {
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
				t.Fatal("campaign", op, direct, failures)
			}
		}
	}
}

func TestLPExactTableauValuesLiveDeltaAliases(t *testing.T) {
	for alias := 0; alias < 3; alias++ {
		for _, reverse := range []bool{false, true} {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 2, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(1, 1, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 2, &h.pair("2", "0").X))
			h.must(f.a.Set(1, 2, &h.pair("3", "0").X))
			if reverse {
				f.a.Columns[2][0], f.a.Columns[2][1] = f.a.Columns[2][1], f.a.Columns[2][0]
			}
			x := []*big.Rat{big.NewRat(5, 1), big.NewRat(7, 1), big.NewRat(1, 1)}
			y := []*big.Rat{big.NewRat(1, 3), big.NewRat(-1, 7), big.NewRat(1, 11)}
			for j := range x {
				h.must(f.x.Data[j].Copy(h.m, h.pair(x[j].RatString(), y[j].RatString())))
			}
			// Model references, not snapshots: every full expression reads the current
			// aliased argument. Use its resulting RHS to isolate the final AX assertion.
			x[2].Add(x[2], x[alias])
			y[2].Add(y[2], y[alias])
			order := []int{0, 1}
			if reverse {
				order = []int{1, 0}
			}
			for _, j := range order {
				c := big.NewRat(int64(j+2), 1)
				x[j].Sub(x[j], new(big.Rat).Mul(x[alias], c))
				y[j].Sub(y[j], new(big.Rat).Mul(y[alias], c))
			}
			for i := 0; i < 2; i++ {
				c := big.NewRat(int64(i+2), 1)
				bx := new(big.Rat).Add(x[i], new(big.Rat).Mul(x[2], c))
				by := new(big.Rat).Add(y[i], new(big.Rat).Mul(y[2], c))
				h.must(f.b.Data[i].Copy(h.m, h.pair(bx.RatString(), by.RatString())))
			}
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.UpdateXTableau(2, &f.x.Data[alias]))
			for j := range x {
				h.check(&f.x.Data[j], x[j], y[j])
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactTableauValuesReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"nil", "entering", "column", "row", "basic", "offset", "second-cell", "type-after-value", "entering-feasibility", "entering-cost", "row-strategy", "leaving-extent", "negative-leaving", "missing-pivot", "zero-pivot", "entering-feasibility-error", "entering-cost-extent", "strategy", "change-basis-error", "post-residual"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			h.must(f.x.Data[0].Copy(h.m, h.pair("4", "0")))
			h.must(f.x.Data[1].Copy(h.m, h.pair("1", "0")))
			h.must(f.b.Data[0].Copy(h.m, h.pair("6", "0")))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			savedColumns := f.a.Columns
			delta := h.pair("1", "0")
			en := uint32(1)
			lv := int32(0)
			basisCall := false
			want := error(ErrBounds)
			wantEnter, wantBasic := big.NewRat(2, 1), big.NewRat(4, 1)
			switch phase {
			case "nil":
				delta = nil
				wantEnter.SetInt64(1)
			case "entering":
				en = 99
				wantEnter.SetInt64(1)
			case "column":
				f.a.Columns = f.a.Columns[:1]
			case "row":
				f.a.Columns[1][0].Row = 99
			case "basic":
				f.basis[0] = 99
			case "offset":
				f.a.Columns[1][0].Offset = 99
			case "second-cell":
				bad := f.a.Columns[1][0]
				bad.Row = 99
				f.a.Columns[1] = append(f.a.Columns[1], bad)
				wantBasic.SetInt64(2)
			case "type-after-value":
				f.types = nil
				wantBasic.SetInt64(2)
			case "entering-feasibility":
				m.UsingInfeasCosts = true
				f.types[1] = ColumnUpperBound
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
					wantBasic.SetInt64(2)
				}
			case "entering-cost":
				m.UsingInfeasCosts = true
				h.must(h.m.SetInt32(&f.costs.Data[1], 7))
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
					wantBasic.SetInt64(2)
				}
			case "row-strategy":
				m.UsingInfeasCosts = true
				f.settings.SimplexStrategy = StrategyTableauRows
				wantBasic.SetInt64(2)
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
				}
			case "leaving-extent":
				basisCall = true
				lv = 99
				wantBasic.SetInt64(2)
			case "negative-leaving":
				basisCall = true
				lv = -1
				wantBasic.SetInt64(2)
			case "entering-feasibility-error":
				m.UsingInfeasCosts = true
				f.types = nil
				if !sourceDebugEnabled {
					wantBasic.SetInt64(2)
				}
			case "entering-cost-extent":
				m.UsingInfeasCosts = true
				h.must(f.costs.Resize(1))
				if !sourceDebugEnabled {
					want = nil
					wantBasic.SetInt64(2)
				}
			case "strategy":
				basisCall = true
				f.settings.SimplexStrategy = StrategyLU
				if sourceDebugEnabled {
					want = ErrAssertion
					wantEnter.SetInt64(1)
				} else {
					want = nil
					wantBasic.SetInt64(2)
				}
			case "change-basis-error":
				basisCall = true
				f.nb = nil
				wantBasic.SetInt64(2)
			case "post-residual":
				h.must(f.b.Data[0].Copy(h.m, h.pair("7", "0")))
				wantBasic.SetInt64(2)
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
				}
			case "missing-pivot":
				basisCall = true
				want = nil
				f.a.Columns[1] = nil
				f.a.Rows[0] = f.a.Rows[0][:1]
				h.must(f.b.Data[0].Copy(h.m, h.pair("4", "0")))
			case "zero-pivot":
				basisCall = true
				want = nil
				coeff, e := f.a.ColumnValue(f.a.Columns[1][0])
				h.must(e)
				h.must(h.m.SetInt32(coeff, 0))
				h.must(f.b.Data[0].Copy(h.m, h.pair("4", "0")))
			}
			var err error
			if basisCall {
				ok, e := m.UpdateBasisAndXTableau(int32(en), lv, delta)
				err = e
				if want == nil && !ok {
					t.Fatal("source bool")
				}
			} else {
				err = m.UpdateXTableau(en, delta)
			}
			if err != want {
				t.Fatal(phase, err, want)
			}
			h.check(&f.x.Data[1], wantEnter, new(big.Rat))
			h.check(&f.x.Data[0], wantBasic, new(big.Rat))
			if want != nil {
				if e = m.UpdateXTableau(1, h.pair("1", "0")); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			if phase == "missing-pivot" || phase == "zero-pivot" {
				if f.basis[0] != 1 || f.heading[1] != 0 {
					t.Fatal("ignored pivot bool did not change basis")
				}
			}
			if phase == "change-basis-error" && f.basis[0] != 1 {
				t.Fatal("basis publication before NB extent")
			}
			f.a.Columns = savedColumns
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactTableauValuesPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"values", "infeasible", "basis"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2/7", "0").X))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&f.costs.Data[0], &v.X))
			h.must(m.CostsBackup.Resize(3))
			h.must(h.m.Copy(&m.CostsBackup.Data[0], &v.Y))
			if op == "infeasible" {
				m.UsingInfeasCosts = true
			}
			delta := h.pair("1/7", "0")

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
				// Real numeric allocation may encounter the older comparison
				// allocator receipt; the solver must neither adopt nor release it.
				if op == "basis" {
					_, err = m.UpdateBasisAndXTableau(1, 0, delta)
				} else {
					err = m.UpdateXTableau(1, delta)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) && !errors.Is(err, umemory.ErrAllocatorPending) {
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

func TestLPExactTableauBasisNonzeroRHSSourceLimitation(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 2, 3)
	f.settings.SimplexStrategy = StrategyTableauCosts
	for _, c := range []struct {
		i, j uint32
		v    string
	}{{0, 0, "1"}, {1, 1, "1"}, {0, 2, "2"}, {1, 2, "3"}} {
		h.must(f.a.Set(c.i, c.j, &h.pair(c.v, "0").X))
	}
	for j, v := range []string{"4", "7", "1"} {
		h.must(f.x.Data[j].Copy(h.m, h.pair(v, "0")))
	}
	for i, v := range []string{"6", "10"} {
		h.must(f.b.Data[i].Copy(h.m, h.pair(v, "0")))
	}
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	ok, e := m.UpdateBasisAndXTableau(2, 0, h.pair("1", "0"))
	h.must(e)
	if !ok {
		t.Fatal("source return")
	}
	for j, v := range []int64{2, 4, 2} {
		h.check(&f.x.Data[j], big.NewRat(v, 1), new(big.Rat))
	}
	h.check(&f.b.Data[0], big.NewRat(3, 1), new(big.Rat))
	h.check(&f.b.Data[1], big.NewRat(10, 1), new(big.Rat))
	// Correct row elimination gives (-3/2)*x0+x1 = 1, whereas source
	// retains the old RHS 10. The returned success does not prove AX=B.
	out := h.pair("0", "0")
	h.must(f.a.DotPairRow(1, f.x.Data, out))
	h.check(out, big.NewRat(1, 1), new(big.Rat))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
