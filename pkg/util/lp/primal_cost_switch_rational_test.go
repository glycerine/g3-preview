package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPRationalPrimalCostSwitchIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyLU
		f.settings.UseBreakpointsInFeasibilitySearch = world%2 == 0
		if world%2 == 0 {
			f.basis = []uint32{2, 0}
		} else {
			f.basis = []uint32{1, 3}
		}
		coeff := make([][]*big.Rat, 2)
		for i := range coeff {
			coeff[i] = make([]*big.Rat, 4)
			for j := range coeff[i] {
				coeff[i][j] = big.NewRat(int64(3+i-j), 7)
			}
		}
		coeff[0][f.basis[0]] = big.NewRat(2, 1)
		coeff[0][f.basis[1]] = big.NewRat(1, 1)
		coeff[1][f.basis[0]] = big.NewRat(1, 1)
		coeff[1][f.basis[1]] = big.NewRat(3, 1)
		for i := range coeff {
			for j, a := range coeff[i] {
				if a.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(a.RatString(), "0").X))
				}
			}
		}
		original, current := make([]*big.Rat, 4), make([]*big.Rat, 4)
		infeas := make([]int64, 4)
		for j := range original {
			original[j] = big.NewRat(int64(world%19+j-9), 13)
			current[j] = big.NewRat(int64(31+j), 17)
			h.must(h.m.SetString(&f.costs.Data[j], original[j].RatString()))
			f.types[j] = ColumnType((world + j) % 5)
			upper := int64(2)
			if f.types[j] == ColumnFixed {
				upper = 0
			}
			x, eps := int64((world+j)%5-1), int64((world+j)%3-1)
			xr := new(big.Rat).Add(big.NewRat(x, 1), new(big.Rat).Mul(big.NewRat(eps, 1), new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))))
			h.must(h.m.SetString(&f.x.Data[j], xr.RatString()))
			h.must(f.up.copySlot(&f.up.Data[j], &h.pair(fmt.Sprint(upper), "0").X))
			above := xr.Cmp(big.NewRat(upper, 1)) > 0
			below := xr.Sign() < 0
			switch f.types[j] {
			case ColumnFixed, ColumnBoxed:
				if above {
					infeas[j] = 1
				} else if below {
					infeas[j] = -1
				}
			case ColumnLowBound:
				if below {
					infeas[j] = -1
				}
			case ColumnUpperBound:
				if above {
					infeas[j] = 1
				}
			}
			if !f.settings.UseBreakpointsInFeasibilitySearch {
				infeas[j] = -infeas[j]
			}
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		// Dense copies include entries absent from the wrapper's sparse metadata.
		f.costs.Index = []uint32{3}
		h.must(m.BackupAndNormalizeCosts())
		for j := range original {
			lpSparseRationalCheck(t, h, &m.CostsBackup.Data[j], original[j])
			lpSparseRationalCheck(t, h, &f.costs.Data[j], original[j])
			h.must(h.m.SetString(&f.costs.Data[j], current[j].RatString()))
			h.must(h.m.SetInt32(&m.D.Data[j], 97))
		}
		for i := range m.Y.Data {
			h.must(h.m.SetInt32(&m.Y.Data[i], 101))
		}
		feasible, using, only := world%2 == 0, (world/2)%2 == 0, (world/4)%2 == 0
		if !feasible {
			h.must(m.InfSet.Insert(0))
		}
		m.UsingInfeasCosts = using
		m.LookForFeasibleSolutionOnly = only
		skip := feasible && using && only
		wantUsing := using
		if !feasible && !using {
			wantUsing = true
			for j := range current {
				current[j] = new(big.Rat)
				if f.heading[j] >= 0 {
					current[j].SetInt64(infeas[j])
				}
			}
		} else if feasible && using && !only {
			wantUsing = false
			for j := range current {
				current[j] = new(big.Rat).Set(original[j])
			}
		}
		h.must(m.InitReducedCosts())
		if m.UsingInfeasCosts != wantUsing {
			t.Fatal(world, "using flag")
		}
		for j := range current {
			lpSparseRationalCheck(t, h, &f.costs.Data[j], current[j])
			lpSparseRationalCheck(t, h, &m.CostsBackup.Data[j], original[j])
		}
		if skip {
			for j := range m.D.Data {
				lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
			}
			for i := range m.Y.Data {
				lpSparseRationalCheck(t, h, &m.Y.Data[i], big.NewRat(101, 1))
			}
		} else {
			cb0, cb1 := current[f.basis[0]], current[f.basis[1]]
			y0 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), cb0), cb1), big.NewRat(5, 1))
			y1 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), cb1), cb0), big.NewRat(5, 1))
			lpSparseRationalCheck(t, h, &m.Y.Data[0], y0)
			lpSparseRationalCheck(t, h, &m.Y.Data[1], y1)
			for j := range current {
				want := new(big.Rat)
				if f.heading[j] < 0 {
					want.Set(current[j])
					want.Sub(want, new(big.Rat).Mul(y0, coeff[0][j]))
					want.Sub(want, new(big.Rat).Mul(y1, coeff[1][j]))
				}
				lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			}
		}
		if !feasible && !using {
			for j := range current {
				if m.InfSet.Contains(uint32(j)) != (current[j].Sign() != 0) {
					t.Fatal(world, j, "membership")
				}
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalCostBackupDenseReplacement(t *testing.T) {
	for _, phase := range []string{"replace", "empty", "only", "self"} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 3)
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		huge := strings.Repeat("9876543210", 20) + "/13"
		for j := range f.costs.Data {
			h.must(h.m.SetString(&f.costs.Data[j], huge))
		}
		h.must(m.CostsBackup.resizeData(5))
		for j := range m.CostsBackup.Data {
			h.must(h.m.SetString(&m.CostsBackup.Data[j], huge))
		}
		m.CostsBackup.Index = []uint32{0} // Algorithm operates on dense values only.
		want := 5
		oldCosts := m.Costs
		switch phase {
		case "replace":
			want = 3
		case "empty":
			h.must(f.costs.resizeData(0))
			want = 0
		case "only":
			m.LookForFeasibleSolutionOnly = true
		case "self":
			m.Costs = &m.CostsBackup.Data
		}
		h.must(m.BackupAndNormalizeCosts())
		if len(m.CostsBackup.Data) != want || len(m.CostsBackup.Index) != 1 {
			t.Fatal(phase, "dense assignment extent")
		}
		for j := range m.CostsBackup.Data {
			lpSparseRationalCheck(t, h, &m.CostsBackup.Data[j], mustLPRat(t, huge))
		}
		m.Costs = oldCosts
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalCostSwitchPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"backup", "restore", "enable"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 2, 4)
					f.settings.SimplexStrategy = StrategyLU
					huge := strings.Repeat("1234567890", 20)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &h.pair(fmt.Sprint(i+3), "0").X))
						h.must(f.a.Set(i, 2, &h.pair(huge+"7/13", "0").X))
						h.must(f.a.Set(i, 3, &h.pair(huge+"9/17", "0").X))
					}
					for j := range f.costs.Data {
						h.must(h.m.SetString(&f.costs.Data[j], huge+"3/23"))
					}
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					h.must(m.BackupAndNormalizeCosts())
					// Replacement must destroy all six old values before copying four.
					if op == "backup" {
						h.must(m.CostsBackup.resizeData(6))
						for j := range m.CostsBackup.Data {
							h.must(h.m.SetString(&m.CostsBackup.Data[j], huge+"1/19"))
						}
					}
					if op == "restore" {
						h.must(f.costs.resizeData(6))
						for j := range f.costs.Data {
							h.must(h.m.SetString(&f.costs.Data[j], huge+"1/19"))
						}
						m.UsingInfeasCosts = true
					}
					if op == "enable" {
						h.must(m.InfSet.Insert(0))
						for j := 0; j < 2; j++ {
							f.types[j] = ColumnLowBound
							h.must(f.low.copySlot(&f.low.Data[j], &h.pair(huge+"7/13", "0").X))
							h.must(f.x.copySlot(&f.x.Data[j], &h.pair(huge+"1/17", "0").X))
						}
					}
					for j := range m.D.Data {
						h.must(h.m.SetInt32(&m.D.Data[j], 97))
					}
					h.budget.FailAfter(ordinal)
					var err error
					if op == "backup" {
						err = m.BackupAndNormalizeCosts()
					} else {
						err = m.InitReducedCosts()
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && (m.RationalCoreSolverBase.calls.pending() || m.Factorization.stopped) {
							t.Fatal("adopted delegated call")
						}
						if op == "backup" && len(m.CostsBackup.Data) != 4 {
							t.Fatal("staged replacement")
						}
						if op == "restore" && m.UsingInfeasCosts {
							if len(f.costs.Data) != 4 {
								t.Fatal("failed restoration retained old storage")
							}
							for j := range m.D.Data {
								lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
							}
						}
						if direct && ordinal == 0 && (op == "backup" || op == "restore") {
							dst := m.CostsBackup.Data
							if op == "restore" {
								dst = f.costs.Data
							}
							lpSparseRationalCheck(t, h, &dst[3], big.NewRat(0, 1))
						}
						if e = m.BackupAndNormalizeCosts(); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e = m.InitReducedCosts(); e != ErrExactVectorPending {
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
			// Reused pooled cells can satisfy copying with no fresh physical request.
			if !success || (direct && failures == 0) {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalPrimalCostSwitchSourcePrefixes(t *testing.T) {
	for _, phase := range []string{"tableau", "only-no-lu", "restore-empty", "restore-short", "enable-bounds", "backup-only-no-destination", "self-restore"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			h.must(h.m.SetInt32(&f.costs.Data[0], 7))
			h.must(h.m.SetInt32(&f.costs.Data[1], 11))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			oldCosts := m.Costs
			for j := range m.D.Data {
				h.must(h.m.SetInt32(&m.D.Data[j], 97))
			}
			m.UsingInfeasCosts = true
			want := error(nil)
			switch phase {
			case "tableau":
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = ErrBounds
				}
			case "only-no-lu":
				m.LookForFeasibleSolutionOnly = true
				m.Settings.SimplexStrategy = StrategyLU
			case "backup-only-no-destination":
				m.LookForFeasibleSolutionOnly = true
			case "restore-empty":
				m.Settings.SimplexStrategy = StrategyLU
				want = ErrBounds
			case "restore-short":
				m.Settings.SimplexStrategy = StrategyLU
				h.must(m.CostsBackup.resizeData(1))
				h.must(h.m.SetInt32(&m.CostsBackup.Data[0], 19))
				want = ErrBounds
			case "self-restore":
				m.Settings.SimplexStrategy = StrategyLU
				m.Costs = &m.CostsBackup.Data
				want = ErrBounds
			case "enable-bounds":
				m.Settings.SimplexStrategy = StrategyLU
				m.UsingInfeasCosts = false
				h.must(m.InfSet.Insert(0))
				f.types[0] = ColumnLowBound
				h.must(f.low.resizeData(0))
				want = ErrBounds
			}
			var err error
			if phase == "backup-only-no-destination" {
				// The source feasibility-only branch returns before touching backup.
				backup := m.CostsBackup
				m.CostsBackup = nil
				err = m.BackupAndNormalizeCosts()
				m.CostsBackup = backup
			} else {
				err = m.InitReducedCosts()
			}
			if err != want {
				t.Fatal(err, want)
			}
			for j := range m.D.Data {
				lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
			}
			if phase == "enable-bounds" {
				if m.UsingInfeasCosts {
					t.Fatal("published using before complete initializer")
				}
				lpSparseRationalCheck(t, h, &f.costs.Data[1], big.NewRat(0, 1))
				lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(7, 1))
			}
			if phase == "restore-empty" || phase == "restore-short" || phase == "self-restore" || (phase == "tableau" && !sourceDebugEnabled) {
				if m.UsingInfeasCosts {
					t.Fatal("restore flag not published before failed solve")
				}
				if phase == "restore-short" {
					lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(19, 1))
					lpSparseRationalCheck(t, h, &m.Y.Data[0], big.NewRat(19, 1))
				}
			}
			if phase == "tableau" && sourceDebugEnabled {
				if !m.UsingInfeasCosts || len(f.costs.Data) != 2 {
					t.Fatal("mutation before source assertion")
				}
			}
			m.Costs = oldCosts
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalCostSwitchLeavesOlderComparisonOwned(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"backup", "restore"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 1)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
				m, err := NewRationalPrimalCoreSolver(f.input())
				h.must(err)
				m.UsingInfeasCosts = true
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.low.Data[0], &v.X))
				h.must(h.m.Copy(&f.x.Data[0], &v.Y))
				h.must(h.m.SetInt32(&f.costs.Data[0], 17))
				h.must(m.CostsBackup.resizeData(1))
				h.must(h.m.Copy(&m.CostsBackup.Data[0], &v.X))
				h.must(h.m.Copy(&f.costs.Data[0], &v.Y))
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
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
					if op == "backup" {
						err = m.BackupAndNormalizeCosts()
					} else {
						err = m.InitReducedCosts()
					}
					if m.calls.pending() || m.RationalCoreSolverBase.calls.pending() || m.work != nil {
						t.Fatal("adopted older comparison", err)
					}
					if err == nil {
						if op == "backup" {
							lpSparseRationalCheck(t, h, &m.CostsBackup.Data[0], mustLPRat(t, strings.Repeat("1234567890", 100)+"/13"))
						} else {
							expected := mustLPRat(t, strings.Repeat("9876543210", 100)+"/7")
							lpSparseRationalCheck(t, h, &f.costs.Data[0], expected)
							lpSparseRationalCheck(t, h, &m.Y.Data[0], new(big.Rat).Quo(expected, big.NewRat(3, 1)))
						}
					}

				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(m.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
				f.close()
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("fixture consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}
