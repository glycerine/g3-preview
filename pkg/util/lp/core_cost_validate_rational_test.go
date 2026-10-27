package lp

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
)

func TestLPRationalCostValidatorsIndependentOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 1, 1)
	m, e := NewRationalCoreSolverBase(f.input())
	h.must(e)
	rng := rand.New(rand.NewPCG(9138, 1782))
	tiny := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))
	for world := 0; world < 1600; world++ {
		kind := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}[world%5]
		lo := big.NewRat(int64(rng.IntN(7)-3), 3)
		up := big.NewRat(int64(rng.IntN(7)-3), 5)
		x := new(big.Rat).Set(lo)
		if world%3 == 0 {
			x.Set(up)
		}
		if world%4 == 0 {
			x.Add(x, tiny)
		}
		if world%7 == 0 {
			x.Sub(x, tiny)
		}
		for k, v := range []*big.Rat{lo, up, x} {
			p := []*IndexedRationalVector{f.low, f.up, f.x}[k]
			h.must(h.m.SetString(&p.Data[0], v.RatString()))
		}
		f.types[0] = kind
		f.settings.UseBreakpointsInFeasibilitySearch = world%2 == 0
		want := int32(0)
		if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnUpperBound) && x.Cmp(up) > 0 {
			want = 1
		} else if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnLowBound) && x.Cmp(lo) < 0 {
			want = -1
		}
		if !f.settings.UseBreakpointsInFeasibilitySearch {
			want = -want
		}
		h.must(h.m.SetInt32(&f.costs.Data[0], want))
		ok, e := m.InfeasibilityCostIsCorrectForColumn(0)
		h.must(e)
		if !ok {
			t.Fatal(world, want)
		}
		h.must(h.m.SetInt32(&f.costs.Data[0], want+3))
		ok, e = m.InfeasibilityCostIsCorrectForColumn(0)
		h.must(e)
		if ok {
			t.Fatal("incorrect cost", world)
		}
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalReducedCostsIndependentMatrixOracle(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(71839, 173))
	v := h.pair("0", "0")
	for world := 0; world < 160; world++ {
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyTableauCosts
		coeff := [2][2]*big.Rat{}
		cost := [4]*big.Rat{}
		for j := 0; j < 4; j++ {
			cost[j] = big.NewRat(int64(rng.IntN(17)-8), int64(rng.IntN(11)+1))
			h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
		}
		for row := 0; row < 2; row++ {
			h.must(h.m.SetInt32(&v.X, 1))
			h.must(f.a.Set(uint32(row), uint32(row), &v.X))
			for k := 0; k < 2; k++ {
				coeff[row][k] = big.NewRat(int64(rng.IntN(11)-5), int64(rng.IntN(7)+1))
				h.must(h.m.SetString(&v.X, coeff[row][k].RatString()))
				h.must(f.a.Set(uint32(row), uint32(k+2), &v.X))
			}
		}
		m, err := NewRationalCoreSolverBase(f.input())
		h.must(err)
		m.Names = lpFeasibleNames{}
		want := [2]*big.Rat{}
		for k := 0; k < 2; k++ {
			want[k] = new(big.Rat).Set(cost[k+2])
			for row := 0; row < 2; row++ {
				want[k].Sub(want[k], new(big.Rat).Mul(cost[row], coeff[row][k]))
			}
			h.must(h.m.SetString(&m.D.Data[k+2], want[k].RatString()))
		}
		ok, err := m.ReducedCostsAreCorrectTableau()
		h.must(err)
		if !ok {
			t.Fatal("matrix oracle", world)
		}
		if world%16 == 0 {
			h.must(h.m.SetString(&m.D.Data[3], new(big.Rat).Add(want[1], big.NewRat(1, 17)).RatString()))
			out := lpCaptureFeasibleStdout(t, func() { ok, err = m.ReducedCostsAreCorrectTableau() })
			h.must(err)
			if ok || !strings.HasPrefix(out, "case b\ncolumn_index = 3") {
				t.Fatal(out)
			}
			h.must(h.m.SetString(&m.D.Data[3], want[1].RatString()))
			h.must(h.m.SetInt32(&m.D.Data[0], 1))
			out = lpCaptureFeasibleStdout(t, func() { ok, err = m.ReducedCostsAreCorrectTableau() })
			h.must(err)
			if ok || !strings.HasPrefix(out, "case a\ncolumn_index = 0") {
				t.Fatal(out)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCostValidatorSourceDiagnosticsAndGuards(t *testing.T) {
	for _, which := range []string{"disabled", "cost", "basic-d", "nonbasic-cost", "unknown", "rows", "bad-matrix", "types", "costs", "x", "low", "upper", "heading", "d", "basis", "column-cost", "column-link"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyTableauCosts
			v := h.pair("1", "0")
			h.must(f.a.Set(0, 0, &v.X))
			h.must(f.a.Set(0, 1, &v.X))
			m, err := NewRationalCoreSolverBase(f.input())
			h.must(err)
			m.Names = lpFeasibleNames{}
			m.UsingInfeasCosts = true
			savedX, savedLow, savedUp, savedCost := f.x.Data, f.low.Data, f.up.Data, f.costs.Data
			ok := false
			var want error
			expected := true
			stdout := lpCaptureFeasibleStdout(t, func() {
				switch which {
				case "disabled":
					m.UsingInfeasCosts = false
					f.costs.Data = nil
					ok, err = m.InfeasibilityCostsAreCorrect()
				case "cost":
					f.types[0] = ColumnLowBound
					h.must(h.m.SetInt32(&f.x.Data[0], -1))
					expected = false
					ok, err = m.ReducedCostsAreCorrectTableau()
				case "basic-d":
					h.must(h.m.SetInt32(&m.D.Data[0], 2))
					expected = false
					ok, err = m.InfeasibilityCostsAreCorrect()
				case "nonbasic-cost":
					h.must(h.m.SetInt32(&f.costs.Data[1], 2))
					ok, err = m.CostsOnNBasisAreZeros()
					if sourceDebugEnabled {
						want = ErrAssertion
						expected = false
					}
				case "unknown":
					f.types[0] = ColumnType(99)
					ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					if sourceDebugEnabled {
						want = ErrAssertion
					}
				case "rows":
					f.settings.SimplexStrategy = StrategyTableauRows
					f.heading = nil
					ok, err = m.ReducedCostsAreCorrectTableau()
				case "bad-matrix":
					f.a.Rows[0][0].Offset = 99
					ok, err = m.ReducedCostsAreCorrectTableau()
					if sourceDebugEnabled {
						want = ErrAssertion
						expected = false
					}
				case "types":
					f.types = nil
					ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					want = ErrBounds
					expected = false
				case "costs":
					f.costs.Data = nil
					ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					want = ErrBounds
					expected = false
				case "x":
					f.types[0] = ColumnLowBound
					f.x.Data = nil
					ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					want = ErrBounds
					expected = false
				case "low":
					f.types[0] = ColumnLowBound
					f.low.Data = nil
					ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					want = ErrBounds
					expected = false
				case "upper":
					f.types[0] = ColumnBoxed
					f.up.Data = nil
					ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					want = ErrBounds
					expected = false
				case "heading":
					m.UsingInfeasCosts = false
					f.heading = nil
					ok, err = m.ReducedCostsAreCorrectTableau()
					want = ErrBounds
					expected = false
				case "d":
					m.UsingInfeasCosts = false
					h.must(m.D.Resize(0))
					ok, err = m.ReducedCostsAreCorrectTableau()
					want = ErrBounds
					expected = false
				case "basis":
					m.UsingInfeasCosts = false
					f.basis = nil
					ok, err = m.ReducedCostsAreCorrectTableau()
					want = ErrBounds
					expected = false
				case "column-cost":
					m.UsingInfeasCosts = false
					f.costs.Data = f.costs.Data[:1]
					ok, err = m.ReducedCostsAreCorrectTableau()
					want = ErrBounds
					expected = false
				case "column-link":
					m.UsingInfeasCosts = false
					f.a.Columns[1][0].Offset = 99
					ok, err = m.ReducedCostsAreCorrectTableau()
					want = ErrBounds
					expected = false
				}
			})
			if !errors.Is(err, want) || ok != expected {
				t.Fatal(ok, err, expected, want, stdout)
			}
			if which == "cost" && !strings.Contains(stdout, "infeasibility_costs_are_correct() does not hold\n") {
				t.Fatal(stdout)
			}
			if which == "basic-d" && !strings.HasPrefix(stdout, "m_d is not zero\n") {
				t.Fatal(stdout)
			}
			f.x.Data, f.low.Data, f.up.Data, f.costs.Data = savedX, savedLow, savedUp, savedCost
			if err != nil {
				if _, e := m.ReducedCostsAreCorrectTableau(); e != ErrExactVectorPending {
					t.Fatal("stopped", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrintColumnInfoExactSplitAndRefusals(t *testing.T) {
	for _, kind := range []ColumnType{ColumnFree, ColumnFixed, ColumnBoxed, ColumnLowBound, ColumnUpperBound} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 1)
			m, err := NewRationalCoreSolverBase(f.input())
			h.must(err)
			m.Names = lpFeasibleNames{}
			f.types[0] = kind
			h.must(h.m.Copy(&f.low.Data[0], &h.pair("1/3", "0").X))
			h.must(h.m.Copy(&f.up.Data[0], &h.pair("5/3", "0").X))
			h.must(h.m.Copy(&f.x.Data[0], &h.pair("2/3", "0").X))
			var out bytes.Buffer
			stdout := lpCaptureFeasibleStdout(t, func() { h.must(m.PrintColumnInfo(0, &out)) })
			typ, e := ColumnTypeString(kind)
			h.must(e)
			want := "column_index = 0, name = v type = " + typ + "\n"
			switch kind {
			case ColumnFixed, ColumnBoxed:
				want += "(1/3, 5/3)\n"
			case ColumnLowBound:
				want += "1/3\n"
			case ColumnUpperBound:
				want += "5/3\n"
			}
			if out.String() != want || stdout != "basis heading = 0\nx = 2/3\n" {
				t.Fatal(out.String(), stdout, want)
			}
			failure := errors.New("writer refused")
			if e := m.PrintColumnInfo(0, lpStaticRefusingWriter{failure}); !errors.Is(e, failure) {
				t.Fatal(e)
			}
			if m.PrintColumnInfo(0, nil) != ErrBounds {
				t.Fatal("nil writer")
			}
			old := os.Stdout
			closed, e := os.CreateTemp(t.TempDir(), "g3-closed-stdout-")
			h.must(e)
			h.must(closed.Close())
			os.Stdout = closed
			e = m.PrintColumnInfo(0, &out)
			os.Stdout = old
			h.must(os.Remove(closed.Name()))
			if e == nil {
				t.Fatal("closed stdout accepted")
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalCostValidatorsPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"low", "upper", "reduced"} {
			success := false
			failures := 0
			owned := 0
			for ordinal := int64(0); ordinal < 240; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 2)
					f.settings.SimplexStrategy = StrategyTableauCosts
					a := strings.Repeat("9876543210", 100) + "/7"
					b := strings.Repeat("1234567890", 100) + "/13"
					v := h.pair(a, b)
					h.must(f.a.Set(0, 1, &v.X))
					m, err := NewRationalCoreSolverBase(f.input())
					h.must(err)
					m.Names = lpFeasibleNames{}
					if op == "low" {
						f.types[0] = ColumnLowBound
						h.must(h.m.Copy(&f.low.Data[0], &v.X))
						h.must(h.m.Copy(&f.x.Data[0], &v.Y))
						h.must(h.m.SetInt32(&f.costs.Data[0], 1))
					} else if op == "upper" {
						f.types[0] = ColumnUpperBound
						h.must(h.m.Copy(&f.up.Data[0], &v.Y))
						h.must(h.m.Copy(&f.x.Data[0], &v.X))
						h.must(h.m.SetInt32(&f.costs.Data[0], -1))
					} else {
						h.must(h.m.Copy(&f.costs.Data[0], &v.Y))
						h.must(h.m.Copy(&f.costs.Data[1], &v.X))
						h.must(h.m.SetString(&m.D.Data[1], new(big.Rat).Sub(mustLPRat(t, a), new(big.Rat).Mul(mustLPRat(t, a), mustLPRat(t, b))).RatString()))
					}
					h.budget.FailAfter(ordinal)
					var ok bool
					if op == "reduced" {
						ok, err = m.ReducedCostsAreCorrectTableau()
					} else {
						ok, err = m.InfeasibilityCostIsCorrectForColumn(0)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if !ok {
							t.Fatal("oracle rejected")
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() || m.pivotWork != nil {
							owned++
						}
						if _, e := m.ReducedCostsAreCorrectTableau(); e != ErrExactVectorPending {
							t.Fatal("stopped", e)
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
			if !success || failures == 0 || owned == 0 {
				t.Fatal("incomplete campaign", direct, op, success, failures, owned)
			}
		}
	}
}

func TestLPRationalPrintColumnInfoMetadataAndBorrowedValueGuards(t *testing.T) {
	for _, which := range []string{"unknown", "fixed-low", "boxed-upper", "low", "upper", "heading", "x", "low-value", "upper-value", "single-value", "x-value", "bound-writer"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			foreign := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 1)
			m, err := NewRationalCoreSolverBase(f.input())
			h.must(err)
			m.Names = lpFeasibleNames{}
			savedX, savedLow, savedUp := f.x.Data, f.low.Data, f.up.Data
			low, up, x := f.low.Data[0], f.up.Data[0], f.x.Data[0]
			bad := foreign.pair("1234567890123456789012345678901234567890", "0")
			failure := errors.New("bound write refused")
			var out bytes.Buffer
			var writer interface{ Write([]byte) (int, error) } = &out
			expected := ErrBounds
			switch which {
			case "unknown":
				f.types[0] = ColumnType(99)
				expected = nil
				if sourceDebugEnabled {
					expected = ErrAssertion
				}
			case "fixed-low":
				f.types[0] = ColumnFixed
				f.low.Data = nil
			case "boxed-upper":
				f.types[0] = ColumnBoxed
				f.up.Data = nil
			case "low":
				f.types[0] = ColumnLowBound
				f.low.Data = nil
			case "upper":
				f.types[0] = ColumnUpperBound
				f.up.Data = nil
			case "heading":
				f.heading = nil
			case "x":
				f.x.Data = nil
			case "low-value":
				f.types[0] = ColumnBoxed
				f.low.Data[0] = bad.X
				expected = nil
			case "upper-value":
				f.types[0] = ColumnBoxed
				f.up.Data[0] = bad.X
				expected = nil
			case "single-value":
				f.types[0] = ColumnLowBound
				f.low.Data[0] = bad.X
				expected = nil
			case "x-value":
				f.x.Data[0] = bad.X
				expected = nil
			case "bound-writer":
				f.types[0] = ColumnBoxed
				writer = &lpValidationWriter{allowed: 1, err: failure}
				expected = failure
			}
			stdout := lpCaptureFeasibleStdout(t, func() { err = m.PrintColumnInfo(0, writer) })
			if strings.HasSuffix(which, "value") {
				if err == nil {
					t.Fatal("missing foreign reference consumed", which)
				}
			} else if !errors.Is(err, expected) {
				t.Fatal(err, expected, stdout)
			}
			if which == "unknown" && !sourceDebugEnabled && !strings.Contains(out.String(), "type = unknown\n") {
				t.Fatal(out.String())
			}
			f.x.Data, f.low.Data, f.up.Data = savedX, savedLow, savedUp
			f.x.Data[0], f.low.Data[0], f.up.Data[0] = x, low, up
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalCostValidatorClosedDiagnosticAndDebugGuards(t *testing.T) {
	for _, which := range []string{"cost-output", "basic-d-output", "case-a-output", "case-b-output", "basis-heading", "basis-duplicate", "basis-inverse", "nonbasic-extent", "infeas-d-extent", "infeas-types", "basic-cost-index"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyTableauCosts
			v := h.pair("1", "0")
			h.must(f.a.Set(0, 1, &v.X))
			m, err := NewRationalCoreSolverBase(f.input())
			h.must(err)
			m.Names = lpFeasibleNames{}
			savedCost := f.costs.Data
			var ok bool
			var want error
			switch which {
			case "cost-output":
				m.UsingInfeasCosts = true
				f.types[0] = ColumnLowBound
				h.must(h.m.SetInt32(&f.x.Data[0], -1))
			case "basic-d-output":
				m.UsingInfeasCosts = true
				h.must(h.m.SetInt32(&m.D.Data[0], 1))
			case "case-a-output":
				h.must(h.m.SetInt32(&m.D.Data[0], 1))
			case "case-b-output":
				h.must(h.m.SetInt32(&m.D.Data[1], 1))
			case "basis-heading":
				f.heading = nil
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "basis-duplicate":
				f.nb = append(f.nb, f.nb[0])
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "basis-inverse":
				f.nb[0] = 0
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "nonbasic-extent":
				f.costs.Data = f.costs.Data[:1]
				want = nil
				if sourceDebugEnabled {
					want = ErrBounds
				}
			case "infeas-d-extent":
				m.UsingInfeasCosts = true
				h.must(m.D.Resize(0))
				want = ErrBounds
			case "infeas-types":
				m.UsingInfeasCosts = true
				f.types = nil
				want = ErrBounds
			case "basic-cost-index":
				m.UsingInfeasCosts = false
				f.basis[0] = 99
				want = ErrBounds
			}
			if strings.HasSuffix(which, "output") {
				old := os.Stdout
				closed, e := os.CreateTemp(t.TempDir(), "g3-closed-diagnostic-")
				h.must(e)
				h.must(closed.Close())
				os.Stdout = closed
				ok, err = m.ReducedCostsAreCorrectTableau()
				os.Stdout = old
				h.must(os.Remove(closed.Name()))
				if err == nil || ok {
					t.Fatal("closed output accepted")
				}
			} else {
				if strings.HasPrefix(which, "basis-") || which == "nonbasic-extent" {
					ok, err = m.CostsOnNBasisAreZeros()
				} else if strings.HasPrefix(which, "infeas-") {
					ok, err = m.InfeasibilityCostsAreCorrect()
				} else {
					ok, err = m.ReducedCostsAreCorrectTableau()
				}
				if !errors.Is(err, want) || (want == nil && !ok) {
					t.Fatal(ok, err, want)
				}
			}
			if err != nil {
				if _, e := m.InfeasibilityCostsAreCorrect(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if _, e := m.InfeasibilityCostIsCorrectForColumn(0); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if _, e := m.CostsOnNBasisAreZeros(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			f.costs.Data = savedCost
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
