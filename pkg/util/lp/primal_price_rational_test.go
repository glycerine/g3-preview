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

func lpRationalPricingFixture(t *testing.T, h *lpPairHarness, rows, cols uint32) (*lpRationalCoreFixture, *RationalPrimalCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, rows, cols)
	f.settings.SimplexStrategy = StrategyLU
	for i := uint32(0); i < rows; i++ {
		h.must(f.a.Set(i, i, &h.pair("1", "0").X))
	}
	m, e := NewRationalPrimalCoreSolver(f.input())
	h.must(e)
	return f, m
}

func TestLPRationalPrimalPricingIndependentEligibility(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpRationalPricingFixture(t, h, 0, 1)
	lo, up := big.NewRat(-3, 7), big.NewRat(5, 13)
	tiny := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))
	h.must(h.m.SetString(&f.low.Data[0], lo.RatString()))
	h.must(h.m.SetString(&f.up.Data[0], up.RatString()))
	eps := big.NewRat(1, 10000000)
	negEps := new(big.Rat).Neg(eps)
	values := []*big.Rat{new(big.Rat).Sub(lo, tiny), lo, new(big.Rat).Add(lo, tiny), new(big.Rat), new(big.Rat).Sub(up, tiny), up, new(big.Rat).Add(up, tiny)}
	for _, kind := range []ColumnType{ColumnFree, ColumnFixed, ColumnLowBound, ColumnUpperBound, ColumnBoxed} {
		f.types[0] = kind
		for _, ds := range []string{"-19/23", "-1/10000000", "-1/1" + strings.Repeat("0", 100), "0", "1/1" + strings.Repeat("0", 100), "1/10000000", "29/31"} {
			d := mustLPRat(t, ds)
			h.must(h.m.SetString(&m.D.Data[0], ds))
			for _, x := range values {
				h.must(h.m.SetString(&f.x.Data[0], x.RatString()))
				want := false
				switch kind {
				case ColumnFree:
					want = d.Sign() != 0
				case ColumnLowBound:
					want = d.Sign() > 0 || (d.Sign() < 0 && x.Cmp(lo) > 0)
				case ColumnUpperBound:
					want = d.Sign() < 0 || (d.Sign() > 0 && x.Cmp(up) < 0)
				case ColumnBoxed:
					want = (d.Sign() > 0 && x.Cmp(up) < 0) || (d.Sign() < 0 && x.Cmp(lo) > 0)
				}
				got, e := m.ColumnIsBeneficialForEnteringBasis(0)
				h.must(e)
				if got != want {
					t.Fatal(kind, ds, x, got, want)
				}
			}
			for _, upper := range []bool{false, true} {
				atUpper := upper
				if kind == ColumnLowBound {
					atUpper = false
				}
				if kind == ColumnUpperBound {
					atUpper = true
				}
				x := lo
				if atUpper {
					x = up
				}
				h.must(h.m.SetString(&f.x.Data[0], x.RatString()))
				m.UsingInfeasCosts = true
				m.Settings.UseBreakpointsInFeasibilitySearch = true
				want := false
				switch kind {
				case ColumnFree:
					want = d.Cmp(eps) > 0 || d.Cmp(negEps) < 0
				case ColumnLowBound:
					want = d.Cmp(negEps) < 0
				case ColumnUpperBound:
					want = d.Cmp(eps) > 0
				case ColumnBoxed:
					if atUpper {
						want = d.Cmp(eps) > 0
					} else {
						want = d.Cmp(negEps) < 0
					}
				}
				got, e := m.ColumnIsBeneficialForEnteringBasis(0)
				h.must(e)
				if got != want {
					t.Fatal("breakpoint", kind, ds, atUpper, got, want)
				}
				m.UsingInfeasCosts = false
				m.Settings.UseBreakpointsInFeasibilitySearch = false
			}
		}
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalPrimalPricingTerminalStatus(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for _, feasible := range []bool{false, true} {
			for _, using := range []bool{false, true} {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 0, 1)
				m.Settings.SimplexStrategy = strategy
				m.UsingInfeasCosts = using
				if !feasible {
					h.must(m.InfSet.Insert(0))
				}
				m.Status = StatusTentativeUnbounded
				m.TotalIterations = 17
				m.IterationsNoCostGrowth = 31
				want := StatusInfeasible
				if feasible {
					want = StatusOptimal
				}
				err := m.DecideOnStatusWhenCannotFindEntering()
				if sourceDebugEnabled && strategy != StrategyTableauRows && feasible == using {
					if err != ErrAssertion || m.Status != StatusTentativeUnbounded {
						t.Fatal(err, m.Status)
					}
					if e := m.DecideOnStatusWhenCannotFindEntering(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
				} else {
					h.must(err)
					if m.Status != want {
						t.Fatal(m.Status, want)
					}
				}
				if m.TotalIterations != 17 || m.IterationsNoCostGrowth != 31 {
					t.Fatal("counter changed")
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPRationalPrimalPricingActualInitializationPipeline(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		rows := uint32(1 + world%4)
		f := lpNewRationalCoreFixture(t, h, rows, rows+2)
		f.settings.SimplexStrategy = StrategyLU
		wanted := []*big.Rat{new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil)), big.NewRat(-7, 13)}
		cost := []*big.Rat{new(big.Rat).Set(wanted[0]), new(big.Rat).Set(wanted[1])}
		for i := uint32(0); i < rows; i++ {
			diag := new(big.Rat).SetFrac(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(81+world%120)), big.NewInt(int64(i+3))), big.NewInt(7))
			cb := big.NewRat(int64(3+i+uint32(world%11)), 17)
			x := big.NewRat(int64(i+1), 19)
			h.must(h.m.SetString(&f.costs.Data[i], cb.RatString()))
			h.must(h.m.SetString(&f.x.Data[i], x.RatString()))
			h.must(h.m.SetString(&f.b.Data[i], new(big.Rat).Mul(diag, x).RatString()))
			h.must(f.a.Set(i, i, &h.pair(diag.RatString(), "0").X))
			f.types[i] = ColumnLowBound
			for k := 0; k < 2; k++ {
				if k == 0 && i > 0 {
					continue
				}
				a := big.NewRat(int64(i+3), int64(k+2))
				h.must(f.a.Set(i, rows+uint32(k), &h.pair(a.RatString(), "0").X))
				cost[k].Add(cost[k], new(big.Rat).Quo(new(big.Rat).Mul(cb, a), diag))
			}
		}
		for k, c := range cost {
			h.must(h.m.SetString(&f.costs.Data[rows+uint32(k)], c.RatString()))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitRun())
		if m.UsingInfeasCosts || !m.CurrentXIsFeasible() || m.Factorization.GetStatus() != LUOK {
			t.Fatal("actual preparation")
		}
		for k, d := range wanted {
			lpSparseRationalCheck(t, h, &m.D.Data[rows+uint32(k)], d)
		}
		if m.ColumnsNZ[rows] != 1 || m.ColumnsNZ[rows+1] != rows {
			t.Fatal("actual sparse counts", m.ColumnsNZ)
		}
		seed := uint32(world + 31)
		m.Settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		got, e := m.ChooseEnteringColumn(2)
		h.must(e)
		want := int32(rows)
		if rows == 1 {
			want = int32((*m.NBasis)[0])
			if rng.Next()%2 == 0 {
				want = int32((*m.NBasis)[1])
			}
		}
		if got != want || m.Settings.RandomNext() != rng.Next() {
			t.Fatal(world, got, want, "pipeline RNG")
		}
		wantSign := int32(1)
		if got == int32(rows+1) {
			wantSign = -1
		}
		if m.SignOfEnteringDelta != wantSign {
			t.Fatal("pipeline direction", m.SignOfEnteringDelta, wantSign)
		}
		ok, e := m.FindXBySolving()
		h.must(e)
		if !ok {
			t.Fatal("initialization changed equations")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalPricingSettingsAndEarlyReturns(t *testing.T) {
	for _, phase := range []string{"run-early", "run-settings", "price-settings", "price-bypass", "sort-settings", "choose-zero", "choose-tie", "choose-bypass", "count-zero", "count-one", "count-large", "count-tentative", "status-settings"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 0, 2)
			m.Settings = nil
			m.TotalIterations = 77
			m.UsingInfeasCosts = true
			h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 9))
			h.must(h.m.SetInt32(&m.D.Data[0], 1))
			h.must(h.m.SetInt32(&m.D.Data[1], 1))
			m.NonBasisList = []uint32{0, 1}
			m.ColumnsNZ = []uint32{0, 0}
			m.BasisSortCounter = 3
			var err error
			want := error(nil)
			switch phase {
			case "run-early":
				m.LookForFeasibleSolutionOnly = true
				err = m.InitRun()
				if !m.UsingInfeasCosts || m.TotalIterations != 0 {
					t.Fatal("early run prefix")
				}
				lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, big.NewRat(9, 1))
			case "run-settings":
				err = m.InitRun()
				want = ErrBounds
				if m.UsingInfeasCosts || m.TotalIterations != 0 {
					t.Fatal("late settings prefix")
				}
				lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, big.NewRat(9, 1))
			case "price-settings":
				_, err = m.ColumnIsBeneficialForEnteringBasis(99)
				want = ErrBounds
			case "price-bypass":
				m.UsingInfeasCosts = false
				var ok bool
				ok, err = m.ColumnIsBeneficialForEnteringBasis(0)
				if !ok {
					t.Fatal("unreached settings read")
				}
			case "sort-settings":
				err = m.SortNonBasis()
				want = ErrBounds
			case "choose-zero":
				_, err = m.ChooseEnteringColumn(0)
				if m.BasisSortCounter != 3 {
					t.Fatal("zero budget counter")
				}
			case "choose-tie":
				m.UsingInfeasCosts = false
				_, err = m.ChooseEnteringColumn(2)
				want = ErrBounds
				if m.BasisSortCounter != 2 || !reflect.DeepEqual(m.NonBasisList, []uint32{0, 1}) {
					t.Fatal("tie failure publication")
				}
			case "choose-bypass":
				m.UsingInfeasCosts = false
				var got int32
				got, err = m.ChooseEnteringColumn(1)
				if got != 0 {
					t.Fatal(got)
				}
			case "count-zero":
				f.nb = nil
				var got uint32
				got, err = m.NumberOfNonBasicColumnsToTryForEnter()
				if got != 0 {
					t.Fatal(got)
				}
			case "count-one":
				f.nb = []uint32{0}
				_, err = m.NumberOfNonBasicColumnsToTryForEnter()
				want = ErrBounds
			case "count-large":
				f.nb = make([]uint32, 301)
				_, err = m.NumberOfNonBasicColumnsToTryForEnter()
				want = ErrBounds
			case "count-tentative":
				f.nb = make([]uint32, 301)
				m.Status = StatusTentativeUnbounded
				var got uint32
				got, err = m.NumberOfNonBasicColumnsToTryForEnter()
				if got != 301 {
					t.Fatal(got)
				}
			case "status-settings":
				err = m.DecideOnStatusWhenCannotFindEntering()
				if sourceDebugEnabled {
					want = ErrBounds
				} else if m.Status != StatusOptimal {
					t.Fatal(m.Status)
				}
			}
			if err != want {
				t.Fatal(err, want)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if e := m.InitRun(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.InitColumnRowNonZeroes(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.SortNonBasis(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := m.ChooseEnteringColumn(0); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := m.ColumnIsBeneficialForEnteringOnBreakpoints(0); e != ErrExactVectorPending {
				t.Fatal(e)
			}
		})
	}
}

func TestLPRationalPrimalPricingIndependentScanRotationAndRNG(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(8771, 481))
	for world := 0; world < 160; world++ {
		f, m := lpRationalPricingFixture(t, h, 7, 16)
		h.must(m.D.resizeData(16))
		m.ColumnsNZ = make([]uint32, 16)
		m.BasisSortCounter = 3
		order := rng.Perm(9)
		m.NonBasisList = nil
		for _, j := range order {
			m.NonBasisList = append(m.NonBasisList, uint32(j+7))
		}
		signs := make([]int, 16)
		for j := 7; j < 16; j++ {
			m.ColumnsNZ[j] = uint32(rng.IntN(8))
			signs[j] = rng.IntN(3) - 1
			h.must(h.m.SetInt32(&m.D.Data[j], int32(signs[j])))
		}
		seed := uint32(world + 19)
		f.settings.RandomSeed(seed)
		oracle := z3rand.New(seed)
		budget := uint32(world % 5)
		remaining := budget
		best, position := uint32(8), -1
		before := append([]uint32(nil), m.NonBasisList...)
		for i, j := range before {
			if remaining == 0 {
				break
			}
			if signs[j] == 0 {
				continue
			}
			if m.ColumnsNZ[j] < best {
				best, position = m.ColumnsNZ[j], i
				remaining--
			} else if m.ColumnsNZ[j] == best && oracle.Next()%2 == 0 {
				position = i
			}
		}
		got, e := m.ChooseEnteringColumn(budget)
		h.must(e)
		want := int32(-1)
		expected := append([]uint32(nil), before...)
		if position >= 0 {
			want = int32(before[position])
			expected = append(append(expected[:position:position], expected[position+1:]...), uint32(want))
		}
		if got != want || !reflect.DeepEqual(m.NonBasisList, expected) {
			t.Fatal(world, got, want, m.NonBasisList, expected)
		}
		if position >= 0 && m.SignOfEnteringDelta != int32(signs[want]) {
			t.Fatal("entering direction")
		}
		wantCounter := uint32(2)
		if budget == 0 {
			wantCounter = 3
		}
		if m.BasisSortCounter != wantCounter {
			t.Fatal("zero budget changed counter")
		}
		if f.settings.RandomNext() != oracle.Next() {
			t.Fatal("random consumption", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalPricingSortSourcesAndCounter(t *testing.T) {
	for _, tableau := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 4, 8)
		h.must(m.D.resizeData(8))
		m.ColumnsNZ = make([]uint32, 8)
		for j := uint32(4); j < 8; j++ {
			m.ColumnsNZ[j] = 8 - j
			h.must(h.m.SetInt32(&m.D.Data[j], 1))
			for row := uint32(0); row < j-3; row++ {
				h.must(m.A.Set(row, j, &h.pair("1", "0").X))
			}
		}
		if tableau {
			f.settings.SimplexStrategy = StrategyTableauCosts
		}
		h.must(m.SortNonBasis())
		want := []uint32{7, 6, 5, 4}
		if tableau {
			want = []uint32{4, 5, 6, 7}
		}
		if !reflect.DeepEqual(*m.NBasis, want) || !reflect.DeepEqual(m.NonBasisList, want) {
			t.Fatal(tableau, *m.NBasis, want)
		}
		for i, j := range want {
			if (*m.Heading)[j] != -int32(i)-1 {
				t.Fatal("heading", i, j)
			}
		}
		got, e := m.ChooseEnteringColumn(1)
		h.must(e)
		if got != int32(want[0]) || m.BasisSortCounter != 20 {
			t.Fatal("sort counter reset", got, m.BasisSortCounter)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalPricingPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"bound", "epsilon-positive", "epsilon-negative", "breakpoint-low", "breakpoint-boxed", "choose"} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalPricingFixture(t, h, 0, 1)
					huge := strings.Repeat("1234567890", 100)
					a := huge + "1/7"
					b := huge + "3/11"
					h.must(h.m.SetString(&m.D.Data[0], a))
					if op == "bound" || op == "choose" {
						f.types[0] = ColumnUpperBound
						h.must(h.m.Copy(&f.up.Data[0], &h.pair(b, "0").X))
						h.must(h.m.Copy(&f.x.Data[0], &h.pair(a, "0").X))
						m.BasisSortCounter = 1
						m.ColumnsNZ = []uint32{0}
						m.NonBasisList = []uint32{0}
					} else {
						h.must(h.m.SetString(&m.EpsilonOfReducedCost, b))
						m.UsingInfeasCosts = true
						f.settings.UseBreakpointsInFeasibilitySearch = true
						if op == "breakpoint-low" || op == "breakpoint-boxed" {
							kind := ColumnLowBound
							if op == "breakpoint-boxed" {
								kind = ColumnBoxed
							}
							f.types[0] = kind
							h.must(h.m.Copy(&f.low.Data[0], &h.pair("0", "0").X))
							h.must(h.m.Copy(&f.up.Data[0], &h.pair("1", "0").X))
						}
						if op == "epsilon-negative" || op == "breakpoint-low" || op == "breakpoint-boxed" {
							h.must(h.m.Neg(&m.D.Data[0]))
						}
					}
					h.budget.FailAfter(ordinal)
					var err error
					if op == "choose" {
						_, err = m.ChooseEnteringColumn(1)
					} else {
						_, err = m.ColumnIsBeneficialForEnteringBasis(0)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.RationalCoreSolverBase.calls.pending() && m.calls.pending() {
							t.Fatal("primal adopted base comparison")
						}
						if _, e := m.ColumnIsBeneficialForEnteringBasis(0); e != ErrExactVectorPending {
							t.Fatal("lost stopped owner")
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
				t.Fatal("incomplete campaign", direct, op, success, failures)
			}
		}
	}
}

func TestLPRationalPrimalPricingSourceGuardsAndPrefixes(t *testing.T) {
	for _, op := range []string{"kind", "D", "X", "bound", "off-low", "off-upper", "boxed-interior", "sort-key", "sort-heading", "scan-count", "zero-sort", "normal-D", "low-extent", "breakpoint-X", "breakpoint-low", "breakpoint-upper", "breakpoint-kind", "tableau-key", "sort-in-scan"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			cols := uint32(1)
			if op == "zero-sort" {
				cols = 2
			}
			f, m := lpRationalPricingFixture(t, h, 0, cols)
			h.must(h.m.Copy(&f.low.Data[0], &h.pair("0", "0").X))
			h.must(h.m.Copy(&f.up.Data[0], &h.pair("2", "0").X))
			h.must(h.m.SetInt32(&m.D.Data[0], 1))
			m.ColumnsNZ = []uint32{0}
			m.NonBasisList = []uint32{0}
			m.BasisSortCounter = 3
			var err error
			want := ErrBounds
			switch op {
			case "normal-D":
				_, err = m.ColumnIsBeneficialForEnteringBasis(99)
			case "low-extent":
				f.types[0] = ColumnLowBound
				h.must(h.m.SetInt32(&m.D.Data[0], -1))
				saved := f.low.Data
				f.low.Data = nil
				_, err = m.ColumnIsBeneficialForEnteringBasis(0)
				f.low.Data = saved
			case "breakpoint-X":
				f.types[0] = ColumnBoxed
				saved := f.x.Data
				f.x.Data = nil
				_, err = m.ColumnIsBeneficialForEnteringOnBreakpoints(0)
				f.x.Data = saved
			case "breakpoint-low":
				f.types[0] = ColumnBoxed
				saved := f.low.Data
				f.low.Data = nil
				_, err = m.ColumnIsBeneficialForEnteringOnBreakpoints(0)
				f.low.Data = saved
			case "breakpoint-upper":
				f.types[0] = ColumnBoxed
				h.must(h.m.Copy(&f.x.Data[0], &h.pair("1", "0").X))
				saved := f.up.Data
				f.up.Data = nil
				_, err = m.ColumnIsBeneficialForEnteringOnBreakpoints(0)
				f.up.Data = saved
				want = nil
				if sourceDebugEnabled {
					want = ErrBounds
				}
			case "breakpoint-kind":
				f.types[0] = ColumnType(99)
				_, err = m.ColumnIsBeneficialForEnteringOnBreakpoints(0)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "tableau-key":
				f.settings.SimplexStrategy = StrategyTableauRows
				*m.NBasis = []uint32{99}
				err = m.SortNonBasis()
			case "sort-in-scan":
				*m.NBasis = []uint32{99}
				m.BasisSortCounter = 0
				_, err = m.ChooseEnteringColumn(1)
			case "kind":
				f.types[0] = ColumnType(99)
				_, err = m.ColumnIsBeneficialForEnteringBasis(0)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "D":
				_, err = m.ColumnIsBeneficialForEnteringOnBreakpoints(99)
			case "X":
				f.types[0] = ColumnUpperBound
				saved := f.x.Data
				f.x.Data = nil
				_, err = m.ColumnIsBeneficialForEnteringBasis(0)
				f.x.Data = saved
			case "bound":
				f.types[0] = ColumnUpperBound
				saved := f.up.Data
				f.up.Data = nil
				_, err = m.ColumnIsBeneficialForEnteringBasis(0)
				f.up.Data = saved
			case "off-low", "off-upper", "boxed-interior":
				kind := ColumnLowBound
				if op == "off-upper" {
					kind = ColumnUpperBound
				}
				if op == "boxed-interior" {
					kind = ColumnBoxed
				}
				f.types[0] = kind
				h.must(h.m.Copy(&f.x.Data[0], &h.pair("1", "0").X))
				_, err = m.ColumnIsBeneficialForEnteringOnBreakpoints(0)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "sort-key":
				m.ColumnsNZ = nil
				f.settings.SimplexStrategy = StrategyLU
				err = m.SortNonBasis()
			case "sort-heading":
				f.heading = nil
				err = m.SortNonBasis()
				if len(m.NonBasisList) != 0 {
					t.Fatal("list cleared before heading publication")
				}
			case "scan-count":
				m.ColumnsNZ = nil
				_, err = m.ChooseEnteringColumn(1)
				if m.BasisSortCounter != 2 {
					t.Fatal("counter prefix")
				}
			case "zero-sort":
				m.ColumnsNZ = []uint32{0, 2}
				*m.NBasis = []uint32{1, 0}
				f.settings.SimplexStrategy = StrategyLU
				err = m.SortNonBasis()
				want = nil
				if len(m.NonBasisList) != 2 || m.NonBasisList[0] == m.NonBasisList[1] {
					t.Fatal("zero-comparator permutation")
				}
				for i, j := range m.NonBasisList {
					if (*m.Heading)[j] != -int32(i)-1 {
						t.Fatal("zero-comparator heading")
					}
				}
			}
			if err != want {
				t.Fatal(op, err, want)
			}
			if err != nil {
				for _, call := range []func() error{m.SortNonBasis, func() error { _, e := m.ColumnIsBeneficialForEnteringBasis(0); return e }, func() error { _, e := m.ColumnIsBeneficialForEnteringOnBreakpoints(0); return e }, func() error { _, e := m.ChooseEnteringColumn(1); return e }} {
					if e := call(); e != ErrExactVectorPending {
						t.Fatal("lost stopped owner", e)
					}
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalPricingOlderComparisonOwner(t *testing.T) {
	for _, op := range []string{"bound", "epsilon", "choose", "breakpoint"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 0, 1)
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnUpperBound
				h.must(h.m.Copy(&f.up.Data[0], &v.X))
				h.must(h.m.Copy(&f.x.Data[0], &v.Y))
				h.must(h.m.SetInt32(&m.D.Data[0], 1))
				m.BasisSortCounter = 1
				m.ColumnsNZ = []uint32{0}
				m.NonBasisList = []uint32{0}
				if op == "epsilon" {
					f.types[0] = ColumnFree
					m.UsingInfeasCosts = true
					f.settings.UseBreakpointsInFeasibilitySearch = true
					h.must(h.m.SetInt32(&m.D.Data[0], -1))
				}
				if op == "breakpoint" {
					f.types[0] = ColumnLowBound
					h.must(h.m.Copy(&f.low.Data[0], &h.pair("0", "0").X))
					h.must(h.m.Copy(&f.x.Data[0], &h.pair("0", "0").X))
					m.UsingInfeasCosts = true
					f.settings.UseBreakpointsInFeasibilitySearch = true
					h.must(h.m.SetInt32(&m.D.Data[0], -1))
				}
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
					if op == "choose" {
						_, err = m.ChooseEnteringColumn(1)
					} else {
						_, err = m.ColumnIsBeneficialForEnteringBasis(0)
					}
					if err == nil || m.calls.pending() || m.RationalCoreSolverBase.calls.pending() {
						t.Fatal("adopted older comparison", err)
					}
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
			t.Fatal("no actual caller comparison", op)
		}
	}
}

func TestLPRationalPrimalPricingBreakpointDirection(t *testing.T) {
	for _, sign := range []int32{-1, 1} {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 0, 1)
		h.must(h.m.SetInt32(&m.D.Data[0], sign))
		m.NonBasisList = []uint32{0}
		m.ColumnsNZ = []uint32{0}
		m.BasisSortCounter = 1
		m.UsingInfeasCosts = true
		f.settings.UseBreakpointsInFeasibilitySearch = true
		j, e := m.ChooseEnteringColumn(1)
		h.must(e)
		if j != 0 || m.SignOfEnteringDelta != -sign {
			t.Fatal(j, m.SignOfEnteringDelta, sign)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalPricingCountIndependentRNG(t *testing.T) {
	for _, count := range []int{0, 1, 2, 300, 301, 401} {
		for _, percent := range []uint32{0, 1, 30, 100, 1000, ^uint32(0)} {
			for _, tentative := range []bool{false, true} {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 0, 0)
				f.nb = make([]uint32, count)
				m.Settings.PercentOfEnteringToCheck = percent
				m.Status = StatusUnknown
				if tentative {
					m.Status = StatusTentativeUnbounded
				}
				seed := uint32(count) + percent + 97
				m.Settings.RandomSeed(seed)
				rng := z3rand.New(seed)
				want := uint32(count)
				if !tentative {
					if want > 300 {
						want = uint32((uint64(want)*uint64(percent))&0xffffffff) / 100
					}
					if want != 0 {
						want = rng.Next() % want
						if want < 1 {
							want = 1
						}
					}
				}
				got, e := m.NumberOfNonBasicColumnsToTryForEnter()
				h.must(e)
				if got != want || m.Settings.RandomNext() != rng.Next() {
					t.Fatal(count, percent, tentative, got, want)
				}
				m.stopped = true
				if e := m.DecideOnStatusWhenCannotFindEntering(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if _, e := m.NumberOfNonBasicColumnsToTryForEnter(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}
