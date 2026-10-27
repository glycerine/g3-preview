package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLARInfeasibilityReducedCostIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		c := lpNewCoreScopeFixture(t, h)
		if world%2 == 0 {
			c.settings.SimplexStrategy = StrategyTableauCosts
		}
		c.settings.UseBreakpointsInFeasibilitySearch = world%3 == 0
		cost := big.NewRat(int64(world%17-8), int64(1+world%7))
		a := big.NewRat(int64(world%13-6), int64(1+world%5))
		b := big.NewRat(int64(world%11-5), int64(1+world%3))
		h.must(h.m.SetString(&c.Costs.Data[0], cost.RatString()))
		for j, v := range []*big.Rat{a, b} {
			h.must(h.m.SetString(&c.RA.Rows[0][j+1].Value, v.RatString()))
			d := new(big.Rat).Neg(new(big.Rat).Mul(cost, v))
			h.must(h.m.SetString(&c.RPrimal.D.Data[j+1], d.RatString()))
		}
		// A stale prior explanation must be destroyed, including its numeral.
		h.must(c.Infeasibility.Append(99, &h.pair(strings.Repeat("123456789", 20), "0").X))
		c.InfeasibleSumSign = 77
		h.must(c.FillNotImprovableZeroSum())
		sign := int32(1)
		if world%3 == 0 {
			sign = -1
		}
		if c.InfeasibleSumSign != sign {
			t.Fatal("sign", world, c.InfeasibleSumSign)
		}
		var indices []uint32
		var values []*big.Rat
		for j, v := range []*big.Rat{cost, new(big.Rat).Mul(cost, a), new(big.Rat).Mul(cost, b)} {
			if v.Sign() != 0 {
				indices = append(indices, uint32(j))
				values = append(values, v)
			}
		}
		lpCheckConstraintVector(t, h, c.Infeasibility, indices, values)
		// These coefficients are cost times the equation x0+a*x1+b*x2=0.
		// Check two independently chosen points on that plane.
		for _, scale := range []int64{-7, 19} {
			x1, x2 := big.NewRat(scale, 3), big.NewRat(scale+2, 5)
			x0 := new(big.Rat).Neg(new(big.Rat).Add(new(big.Rat).Mul(a, x1), new(big.Rat).Mul(b, x2)))
			sum := new(big.Rat)
			xs := []*big.Rat{x0, x1, x2}
			for k, j := range indices {
				sum.Add(sum, new(big.Rat).Mul(values[k], xs[j]))
			}
			if sum.Sign() != 0 {
				t.Fatal("independent zero sum", world, sum)
			}
		}
		h.must(c.FillNotImprovableZeroSum())
		lpCheckConstraintVector(t, h, c.Infeasibility, indices, values)
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfeasibilityStoredRowIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		c := lpNewCoreScopeFixture(t, h)
		c.settings.SimplexStrategy = StrategyTableauRows
		h.must(c.RightSides.resizeData(1))
		kinds := []ColumnType{ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}
		kind := kinds[world%4]
		c.Bounds.Types.values[0] = kind
		sign := int32(1)
		initial := big.NewRat(-int64(1+world%19), int64(1+world%7))
		if kind == ColumnUpperBound || (kind != ColumnLowBound && world/4%2 == 0) {
			sign = -1
			initial.Neg(initial)
		}
		a := big.NewRat(int64(1+world%13), int64(1+world%5))
		b := big.NewRat(int64(world%11-5), int64(1+world%3))
		epsilon := big.NewRat(int64(world%7-3), 17)
		h.must(h.m.SetString(&c.RA.Rows[0][1].Value, a.RatString()))
		h.must(h.m.SetString(&c.RA.Rows[0][2].Value, b.RatString()))
		// The stored row is x0+a*x1+b*x2=0. A free nonbasic value
		// balances the deliberately infeasible basic variable exactly.
		h.must(c.RX.Data[0].Copy(h.m, h.pair(initial.RatString(), epsilon.RatString())))
		h.must(c.RX.Data[1].Copy(h.m, h.pair(new(big.Rat).Quo(new(big.Rat).Neg(initial), a).RatString(), new(big.Rat).Quo(new(big.Rat).Neg(epsilon), a).RatString())))
		indices := []uint32{0, 1, 2}
		values := []*big.Rat{big.NewRat(1, 1), a, b}
		if world%2 == 0 {
			c.RA.Rows[0][0], c.RA.Rows[0][2] = c.RA.Rows[0][2], c.RA.Rows[0][0]
			for k, cell := range c.RA.Rows[0] {
				c.RA.Columns[cell.Column][cell.Offset].Offset = uint32(k)
			}
			indices[0], indices[2] = indices[2], indices[0]
			values[0], values[2] = values[2], values[0]
		}
		c.RPrimal.InfRowIndexForTableau = 0
		h.must(c.Infeasibility.Append(99, &h.pair("13", "0").X))
		h.must(c.FillNotImprovableZeroSum())
		if c.InfeasibleSumSign != sign {
			t.Fatal("row sign", world, c.InfeasibleSumSign, sign)
		}
		lpCheckConstraintVector(t, h, c.Infeasibility, indices, values)
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfeasibilityPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, row := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%t/%d", row, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					c := lpNewCoreScopeFixture(t, h)
					huge := strings.Repeat("1234567890", 20)
					if row {
						c.settings.SimplexStrategy = StrategyTableauRows
						h.must(c.RightSides.resizeData(1))
						c.Bounds.Types.values[0] = ColumnLowBound
						h.must(c.RX.Data[0].Copy(h.m, h.pair("-2", "0")))
						h.must(c.RX.Data[1].Copy(h.m, h.pair("1", "0")))
						// x2=0 keeps this large row coefficient irrelevant to AX.
						h.must(h.m.SetString(&c.RA.Rows[0][2].Value, huge+"7/13"))
					} else {
						h.must(h.m.SetString(&c.Costs.Data[0], huge+"3/17"))
						h.must(h.m.SetString(&c.RPrimal.D.Data[1], huge+"7/13"))
						h.must(h.m.SetInt32(&c.RPrimal.D.Data[2], -2147483648))
					}
					h.budget.FailAfter(ordinal)
					err := c.FillNotImprovableZeroSum()
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if c.FillNotImprovableZeroSum() != ErrExactVectorPending {
							t.Fatal("stopped retry")
						}
					}
					h.must(c.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || (direct && failures == 0) {
				t.Fatal("campaign", row, direct, success, failures)
			}
		}
	}
}

func TestLARInfeasibilityActualSolveExplanation(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for world := 0; world < 60; world++ {
			h := newLPPairHarness(t)
			c := lpNewCoreScopeFixture(t, h)
			c.settings.SimplexStrategy = strategy
			h.must(c.RightSides.resizeData(1))
			a := big.NewRat(int64(1+world%13), int64(1+world%5))
			gap := big.NewRat(int64(1+world%17), int64(1+world%7))
			negative := new(big.Rat).Neg(gap)
			h.must(h.m.SetString(&c.RA.Rows[0][1].Value, a.RatString()))
			h.must(h.m.SetInt32(&c.RA.Rows[0][2].Value, -1))
			c.Bounds.Types.values[0], c.Bounds.Types.values[1], c.Bounds.Types.values[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
			h.must(c.RX.Data[0].Copy(h.m, h.pair(negative.RatString(), "0")))
			for _, p := range []*RationalPair{&c.RX.Data[2], &c.Bounds.Low.values[2], &c.Bounds.Upper.values[2]} {
				h.must(p.Copy(h.m, h.pair(negative.RatString(), "0")))
			}
			h.must(c.PrefixR())
			h.must(c.RPrimal.InfSet.Resize(c.RA.ColumnCount()))
			h.must(c.RPrimal.InitInfSet())
			if strategy == StrategyLU {
				h.must(c.RPrimal.InitLU())
			} else if strategy == StrategyTableauCosts {
				h.must(c.RPrimal.InitInfeasibilityCosts())
				h.must(c.RPrimal.InitReducedCostsTableau())
			}
			c.RPrimal.Names = lpFeasibleNames{}
			c.settings.SetResourceLimit(&lpTestResourceLimit{})
			_, err := c.RPrimal.Solve()
			h.must(err)
			if c.RPrimal.Status != StatusInfeasible {
				t.Fatal("actual solve", strategy, world, c.RPrimal.Status)
			}
			h.must(c.FillNotImprovableZeroSum())
			// For every strategy the explanation is a positive multiple of
			// x0+a*x1-x2=0. Its least value under the bounds is gap>0.
			lpCheckConstraintVector(t, h, c.Infeasibility, []uint32{0, 1, 2}, []*big.Rat{big.NewRat(1, 1), a, big.NewRat(-1, 1)})
			if c.InfeasibleSumSign != 1 || gap.Sign() <= 0 {
				t.Fatal("contradiction sign")
			}
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARInfSignIndependentBoundAndEpsilonModels(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 1)
		f.settings.SimplexStrategy = StrategyTableauRows
		kinds := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}
		kind := kinds[world%len(kinds)]
		f.types[0] = kind
		value := h.pair("0", "0")
		want := int32(1)
		if kind == ColumnUpperBound || ((kind == ColumnFixed || kind == ColumnBoxed) && world/5%2 == 0) {
			want = -1
		}
		if kind == ColumnFree {
			want = 0
		}
		delta := big.NewRat(int64(-want), 7)
		if world/10%2 == 0 {
			h.must(h.m.SetString(&value.X, delta.RatString()))
		} else {
			h.must(h.m.SetString(&value.Y, delta.RatString()))
		}
		h.must(f.x.Data[0].Copy(h.m, value))
		m, err := NewPairPrimalCoreSolver(f.input())
		h.must(err)
		sign, err := m.InfSignOfColumn(0)
		if kind == ColumnFree && sourceDebugEnabled {
			if err != ErrAssertion {
				t.Fatal("free debug", sign, err)
			}
		} else if err != nil || sign != want {
			t.Fatal(world, kind, sign, err, want)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfeasibilityReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"stopped", "primal-stopped", "short-costs", "short-heading", "short-reduced", "row-short-basis", "row-missing", "row-feasible", "row-residual"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			c := lpNewCoreScopeFixture(t, h)
			h.must(c.RightSides.resizeData(1))
			h.must(h.m.SetInt32(&c.Costs.Data[0], 3))
			h.must(c.Infeasibility.Append(99, &h.pair("17", "0").X))
			c.InfeasibleSumSign = 77
			costs, heading, basis, rows := c.Costs.Data, c.RHeading, c.RBasis, c.RA.Rows
			reduced := c.RPrimal.D.Data
			wantSign, wantSize := int32(1), uint32(0)
			if strings.HasPrefix(phase, "row-") {
				c.settings.SimplexStrategy = StrategyTableauRows
				c.Bounds.Types.values[0] = ColumnLowBound
				h.must(c.RX.Data[0].Copy(h.m, h.pair("-2", "0")))
				h.must(c.RX.Data[1].Copy(h.m, h.pair("1", "0")))
			}
			switch phase {
			case "stopped":
				c.stopped = true
				wantSign, wantSize = 77, 1
			case "primal-stopped":
				c.RPrimal.stopped = true
				wantSign, wantSize = 77, 1
			case "short-costs":
				c.Costs.Data = nil
			case "short-heading":
				c.RHeading = nil
				wantSize = 1
			case "short-reduced":
				c.RPrimal.D.Data = nil
				wantSize = 1
			case "row-short-basis":
				c.RBasis = nil
				wantSign, wantSize = 77, 1
			case "row-missing":
				c.RA.Rows = nil
			case "row-feasible":
				h.must(c.RX.Data[0].Copy(h.m, h.pair("0", "0")))
				h.must(c.RX.Data[1].Copy(h.m, h.pair("0", "0")))
				if sourceDebugEnabled {
					wantSign, wantSize = 77, 1
				} else {
					wantSize = 3
				}
			case "row-residual":
				h.must(c.RX.Data[0].Copy(h.m, h.pair("-1", "0")))
				if sourceDebugEnabled {
					wantSign, wantSize = 77, 1
				} else {
					wantSize = 3
				}
			}
			var err error
			lpCaptureFeasibleStdout(t, func() { err = c.FillNotImprovableZeroSum() })
			succeeds := (phase == "row-feasible" || phase == "row-residual") && !sourceDebugEnabled
			if (err == nil) != succeeds || c.InfeasibleSumSign != wantSign || c.Infeasibility.Size() != wantSize {
				t.Fatal("prefix", err, c.InfeasibleSumSign, c.Infeasibility.Size(), wantSign, wantSize)
			}
			if !succeeds && c.FillNotImprovableZeroSumFromInfRow() != ErrExactVectorPending {
				t.Fatal("direct stopped retry")
			}
			// Restore borrowed slice headers before their owning vectors are closed.
			c.Costs.Data, c.RHeading, c.RBasis, c.RA.Rows, c.RPrimal.D.Data = costs, heading, basis, rows, reduced
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARInfeasibilityPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			c := lpNewCoreScopeFixture(t, h)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&c.Costs.Data[0], &v.X))
			h.must(h.m.Copy(&c.RPrimal.D.Data[1], &v.Y))
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
				_ = c.FillNotImprovableZeroSum()
				if c.Infeasibility.calls.comparison != 0 || (c.evidenceNegated != nil && c.evidenceNegated.calls.comparison != 0) {
					t.Fatal("adopted caller comparison")
				}
			} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			h.must(c.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
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

func TestLARInfeasibilityBasisOrderAndNonbasicScan(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		settings, err := NewSettings()
		h.must(err)
		settings.SimplexStrategy = StrategyLU
		c, err := NewLARCoreSolver(h.m, settings, nil)
		h.must(err)
		for j := 0; j < 5; j++ {
			lpCoreScopeAddColumn(t, h, c, j%2 == 0)
		}
		basis := []uint32{0, 2, 4}
		rotate := world % 3
		c.RBasis = append(append([]uint32{}, basis[rotate:]...), basis[:rotate]...)
		for i, j := range c.RBasis {
			c.RHeading[j] = int32(i)
		}
		var indices []uint32
		var values []*big.Rat
		expected := make(map[uint32]*big.Rat)
		for j := uint32(0); j < 5; j++ {
			v := big.NewRat(int64((world+int(j))%7-3), int64(j+1))
			if j%2 == 0 {
				h.must(h.m.SetString(&c.Costs.Data[j], v.RatString()))
				expected[j] = v
			} else {
				h.must(h.m.SetString(&c.RPrimal.D.Data[j], v.RatString()))
				expected[j] = new(big.Rat).Neg(v)
			}
		}
		for _, j := range append(append([]uint32{}, c.RBasis...), 1, 3) {
			if expected[j].Sign() != 0 {
				indices = append(indices, j)
				values = append(values, expected[j])
			}
		}
		h.must(c.FillNotImprovableZeroSum())
		lpCheckConstraintVector(t, h, c.Infeasibility, indices, values)
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfeasibilityStopsAtPendingExplanation(t *testing.T) {
	for _, row := range []bool{false, true} {
		h := lpNewDirectSparseHarness(t)
		c := lpNewCoreScopeFixture(t, h)
		if row {
			c.settings.SimplexStrategy = StrategyTableauRows
			h.must(c.RightSides.resizeData(1))
			c.Bounds.Types.values[0] = ColumnLowBound
			h.must(c.RX.Data[0].Copy(h.m, h.pair("-2", "0")))
			h.must(c.RX.Data[1].Copy(h.m, h.pair("1", "0")))
		}
		h.must(c.Infeasibility.Append(99, &h.pair("17", "0").X))
		bigValue := h.pair(strings.Repeat("1234567890", 40), "0")
		h.budget.FailAfter(0)
		err := c.Infeasibility.appendTermCoefficient(100, &bigValue.X)
		h.budget.FailAfter(-1)
		if !lpExactPriorityPhysicalRefusal(err) || c.Infeasibility.appendValues == nil {
			t.Fatal("no actual stopped append", err)
		}
		if err = c.FillNotImprovableZeroSum(); err != ErrExactVectorPending || c.InfeasibleSumSign != 1 {
			t.Fatal("pending explanation prefix", err, c.InfeasibleSumSign)
		}
		lpCheckConstraintVector(t, h, c.Infeasibility, []uint32{99}, []*big.Rat{big.NewRat(17, 1)})
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfSignPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 0, 1)
			f.settings.SimplexStrategy = StrategyTableauRows
			f.types[0] = ColumnBoxed
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&f.x.Data[0].X, &v.X))
			h.must(h.m.Copy(&f.up.Data[0].X, &v.Y))
			m, err := NewPairPrimalCoreSolver(f.input())
			h.must(err)
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
				if _, err = m.InfSignOfColumn(0); err == nil || m.PairCoreSolverBase.calls.comparison != 0 {
					t.Fatal("caller comparison ownership", err)
				}
				if _, err = m.InfSignOfColumn(0); err != ErrExactVectorPending {
					t.Fatal("stopped sign retry", err)
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
		})
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}

func TestLARInfeasibilityOwnsCopiedCoefficients(t *testing.T) {
	for _, row := range []bool{false, true} {
		h := newLPPairHarness(t)
		c := lpNewCoreScopeFixture(t, h)
		huge := strings.Repeat("1234567890", 30) + "7/13"
		original := mustLPRat(t, huge)
		if row {
			c.settings.SimplexStrategy = StrategyTableauRows
			h.must(c.RightSides.resizeData(1))
			c.Bounds.Types.values[0] = ColumnLowBound
			h.must(c.RX.Data[0].Copy(h.m, h.pair("-2", "0")))
			h.must(c.RX.Data[1].Copy(h.m, h.pair("1", "0")))
			h.must(h.m.SetString(&c.RA.Rows[0][2].Value, huge))
		} else {
			h.must(h.m.SetString(&c.Costs.Data[0], huge))
		}
		h.must(c.FillNotImprovableZeroSum())
		index := 0
		if row {
			index = 2
		}
		if row {
			h.must(h.m.SetInt32(&c.RA.Rows[0][2].Value, 17))
		} else {
			h.must(h.m.SetInt32(&c.Costs.Data[0], 17))
		}
		lpSparseRationalCheck(t, h, &c.Infeasibility.Data[index].Value, original)
		h.must(h.m.SetInt32(&c.Infeasibility.Data[index].Value, 23))
		if row {
			lpSparseRationalCheck(t, h, &c.RA.Rows[0][2].Value, big.NewRat(17, 1))
		} else {
			lpSparseRationalCheck(t, h, &c.Costs.Data[0], big.NewRat(17, 1))
		}
		h.must(c.FillNotImprovableZeroSum())
		lpSparseRationalCheck(t, h, &c.Infeasibility.Data[index].Value, big.NewRat(17, 1))
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}
