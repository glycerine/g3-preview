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
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalTableauIndependentEliminationAndCosts(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(83219, 527))
	value := h.pair("0", "0")
	for world := 0; world < 160; world++ {
		rows := 2 + rng.IntN(4)
		cols := rows + 3
		pivot := rng.IntN(rows)
		f := lpNewRationalCoreFixture(t, h, uint32(rows), uint32(cols))
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauCosts
		}
		a := make([][]*big.Rat, rows)
		rhs := make([]*big.Rat, rows)
		for i := 0; i < rows; i++ {
			a[i] = make([]*big.Rat, cols)
			for j := range a[i] {
				a[i][j] = new(big.Rat)
			}
			sign := int64(1)
			if rng.IntN(2) == 0 {
				sign = -1
			}
			a[i][0].SetFrac64(sign*int64(1+rng.IntN(11)), int64(1+rng.IntN(13)))
			a[i][i+1].SetInt64(1)
			for j := rows + 1; j < cols; j++ {
				a[i][j].SetFrac64(int64(rng.IntN(15)-7), int64(rng.IntN(11)+1))
			}
			for j, x := range a[i] {
				h.must(h.m.SetString(&value.X, x.RatString()))
				h.must(f.a.Set(uint32(i), uint32(j), &value.X))
			}
			rhs[i] = big.NewRat(int64(i+7), 3)
			h.must(h.m.SetString(&f.b.Data[i], rhs[i].RatString()))
		}
		m, err := NewRationalCoreSolverBase(f.input())
		h.must(err)
		d := make([]*big.Rat, cols)
		for j := range d {
			d[j] = big.NewRat(int64(j-2), int64(j+3))
			h.must(h.m.SetString(&m.D.Data[j], d[j].RatString()))
		}
		m.PivotedRows, err = NewIntSet(uint32(rows))
		h.must(err)
		basis := append([]uint32(nil), f.basis...)
		heading := append([]int32(nil), f.heading...)
		p := new(big.Rat).Set(a[pivot][0])
		for j := range a[pivot] {
			a[pivot][j].Quo(a[pivot][j], p)
		}
		rhs[pivot].Quo(rhs[pivot], p)
		for i := range a {
			if i != pivot {
				scale := new(big.Rat).Set(a[i][0])
				for j := range a[i] {
					a[i][j].Sub(a[i][j], new(big.Rat).Mul(scale, a[pivot][j]))
				}
			}
		}
		if f.settings.SimplexStrategy == StrategyTableauCosts {
			scale := new(big.Rat).Set(d[0])
			for j := 1; j < cols; j++ {
				d[j].Sub(d[j], new(big.Rat).Mul(scale, a[pivot][j]))
			}
			d[0].SetInt64(0)
		}
		ok, err := m.PivotColumnTableau(0, uint32(pivot))
		h.must(err)
		if !ok {
			t.Fatal("nonempty elimination failed", world)
		}
		want := make(map[IndexPair]*big.Rat)
		for i := range a {
			for j, x := range a[i] {
				if x.Sign() != 0 {
					want[IndexPair{uint32(i), uint32(j)}] = x
				}
			}
			lpSparseRationalCheck(t, h, &f.b.Data[i], rhs[i])
			if m.PivotedRows.Contains(uint32(i)) != (i != pivot) {
				t.Fatal("pivoted rows", world, i)
			}
		}
		lpExactStaticCheck(t, h, f.a, want)
		for j := range d {
			lpSparseRationalCheck(t, h, &m.D.Data[j], d[j])
		}
		if !reflect.DeepEqual(basis, f.basis) || !reflect.DeepEqual(heading, f.heading) {
			t.Fatal("pivot altered basis")
		}
		h.must(m.Close())
		f.close()
	}
}

func TestLPRationalTableauEmptyTargetAndNoPivot(t *testing.T) {
	for _, op := range []string{"empty", "absent", "zero", "missing-link", "cost-short", "cost-zero"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 3)
			v := h.pair("2", "6")
			h.must(f.a.Set(0, 0, &v.X))
			h.must(f.a.Set(0, 2, &v.Y))
			h.must(f.a.Set(1, 0, &v.X))
			h.must(f.a.Set(1, 2, &v.Y))
			f.settings.SimplexStrategy = StrategyTableauCosts
			m, err := NewRationalCoreSolverBase(f.input())
			h.must(err)
			defer f.close()
			defer func() { h.must(m.Close()) }()
			m.PivotedRows, err = NewIntSet(2)
			h.must(err)
			h.must(h.m.SetInt64(&m.D.Data[0], 7))
			var ok bool
			switch op {
			case "empty":
				ok, err = m.PivotColumnTableau(0, 0)
				if len(f.a.Rows[1]) != 0 || m.PivotedRows.Size() != 0 {
					t.Fatal("empty target published as pivoted")
				}
				lpSparseRationalCheck(t, h, &m.D.Data[0], big.NewRat(7, 1))
			case "absent":
				ok, err = m.DivideRowByPivot(0, 1)
			case "zero":
				h.must(h.m.SetInt64(&f.a.Rows[0][0].Value, 0))
				ok, err = m.DivideRowByPivot(0, 0)
			case "missing-link":
				f.a.Columns[0] = f.a.Columns[0][1:]
				ok, err = m.PivotColumnTableau(0, 0)
				lpSparseRationalCheck(t, h, &f.a.Rows[0][0].Value, big.NewRat(1, 1))
			case "cost-short":
				err = m.PivotToReducedCostsTableau(99, 99)
			case "cost-zero":
				h.must(h.m.SetInt64(&m.D.Data[0], 0))
				err = m.PivotToReducedCostsTableau(99, 0)
			}
			h.must(err)
			if ok {
				t.Fatal("source false branch")
			}
		})
	}
}

func TestLPRationalTableauMetadataAndStoppedGuards(t *testing.T) {
	for _, op := range []string{"row", "rhs", "column", "head-offset", "head-row", "head-row-offset", "cost-row", "cost-column", "pivoted-domain"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 3)
			v := h.pair("2", "3")
			h.must(f.a.Set(0, 0, &v.X))
			h.must(f.a.Set(0, 1, &v.Y))
			h.must(f.a.Set(1, 0, &v.Y))
			h.must(f.a.Set(1, 2, &v.X))
			m, err := NewRationalCoreSolverBase(f.input())
			h.must(err)
			defer f.close()
			defer func() { h.must(m.Close()) }()
			switch op {
			case "row":
				_, err = m.DivideRowByPivot(9, 0)
			case "rhs":
				owned := f.b.Data
				f.b.Data = nil
				_, err = m.DivideRowByPivot(0, 0)
				f.b.Data = owned
			case "column":
				f.a.Rows[0][0].Column = 9
				_, err = m.PivotColumnTableau(9, 0)
			case "head-offset":
				f.a.Columns[0][1].Offset = 9
				_, err = m.PivotColumnTableau(0, 1)
			case "head-row":
				f.a.Columns[0][0].Row = 9
				_, err = m.PivotColumnTableau(0, 1)
			case "head-row-offset":
				f.a.Columns[0][0].Offset = 9
				_, err = m.PivotColumnTableau(0, 1)
			case "cost-row":
				h.must(h.m.SetInt64(&m.D.Data[0], 1))
				err = m.PivotToReducedCostsTableau(9, 0)
			case "cost-column":
				h.must(h.m.SetInt64(&m.D.Data[0], 1))
				f.a.Rows[0][1].Column = 9
				err = m.PivotToReducedCostsTableau(0, 0)
			case "pivoted-domain":
				m.PivotedRows, err = NewIntSet(0)
				h.must(err)
				_, err = m.PivotColumnTableau(0, 0)
			}
			if err != ErrBounds {
				t.Fatal(err)
			}
			if _, e := m.DivideRowByPivot(0, 0); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := m.PivotColumnTableau(0, 0); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.PivotToReducedCostsTableau(0, 0); e != ErrExactVectorPending {
				t.Fatal(e)
			}
		})
	}
}

func TestLPRationalTableauPhysicalFailureOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"divide", "eliminate", "costs", "pivot-costs"} {
			failures := 0
			coreFailures, matrixFailures, costFailures := 0, 0, 0
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
					f := lpNewRationalCoreFixture(t, h, 2, 4)
					v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
					h.must(f.a.Set(0, 0, &v.X))
					h.must(f.a.Set(0, 1, &v.Y))
					h.must(f.a.Set(1, 0, &v.Y))
					h.must(f.a.Set(1, 2, &v.X))
					h.must(h.m.Copy(&f.b.Data[0], &v.X))
					m, err := NewRationalCoreSolverBase(f.input())
					h.must(err)
					if op == "pivot-costs" {
						f.settings.SimplexStrategy = StrategyTableauCosts
					}
					if op == "costs" || op == "pivot-costs" {
						h.must(h.m.Copy(&m.D.Data[0], &v.X))
					}
					h.budget.FailAfter(ordinal)
					switch op {
					case "divide":
						_, err = m.DivideRowByPivot(0, 0)
					case "eliminate", "pivot-costs":
						_, err = m.PivotColumnTableau(0, 0)
					case "costs":
						err = m.PivotToReducedCostsTableau(0, 0)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						x := mustLPRat(t, strings.Repeat("1234567890", 100)+"/7")
						y := mustLPRat(t, strings.Repeat("9876543210", 100)+"/13")
						if op != "costs" {
							lpSparseRationalCheck(t, h, &f.b.Data[0], big.NewRat(1, 1))
							lpSparseRationalCheck(t, h, &f.a.Rows[0][0].Value, big.NewRat(1, 1))
						}
						if op == "costs" || op == "pivot-costs" {
							want := new(big.Rat).Neg(y)
							if op == "costs" {
								want.Mul(want, x)
							}
							lpSparseRationalCheck(t, h, &m.D.Data[1], want)
							lpSparseRationalCheck(t, h, &m.D.Data[0], big.NewRat(0, 1))
						}
					} else {
						failures++
						if m.calls.pending() {
							coreFailures++
						}
						if f.a.work != nil {
							matrixFailures++
							lpSparseRationalCheck(t, h, &f.b.Data[0], big.NewRat(1, 1))
							lpSparseRationalCheck(t, h, &f.a.Rows[0][0].Value, big.NewRat(1, 1))
						}
						if m.pivotWork != nil {
							costFailures++
							lpSparseRationalCheck(t, h, &m.D.Data[0], mustLPRat(t, strings.Repeat("1234567890", 100)+"/7"))
						}
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if !m.stopped {
							t.Fatal("lost parent stop")
						}
						if f.a.work != nil && m.calls.pending() {
							t.Fatal("core captured matrix-owned failure")
						}
						if _, e := m.PivotColumnTableau(0, 0); e != ErrExactVectorPending {
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
			if (op == "eliminate" || op == "pivot-costs") && matrixFailures == 0 {
				t.Fatal("no actual matrix refusal reached")
			}
			if coreFailures == 0 {
				t.Fatal("no core-owned stopped arithmetic reached")
			}
			if (op == "costs" || op == "pivot-costs") && costFailures == 0 {
				t.Fatal("no actual cost refusal reached")
			}
			if failures == 0 || !success {
				t.Fatal("incomplete refusal campaign", direct, op, failures, success)
			}
		}
	}
}

func TestLPRationalTableauStoppedMatrixBeforeNormalization(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 1, 2)
	small := h.pair("2", "3")
	large := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	h.must(f.a.Set(0, 0, &small.X))
	h.must(h.m.Copy(&f.b.Data[0], &small.X))
	m, err := NewRationalCoreSolverBase(f.input())
	h.must(err)
	h.budget.FailAfter(0)
	err = f.a.Set(0, 1, &large.X)
	h.budget.FailAfter(-1)
	if !lpExactPriorityPhysicalRefusal(err) {
		t.Fatal(err)
	}
	if _, err = m.DivideRowByPivot(0, 0); err != ErrExactVectorPending {
		t.Fatal(err)
	}
	lpSparseRationalCheck(t, h, &f.b.Data[0], big.NewRat(2, 1))
	if m.calls.pending() {
		t.Fatal("parent adopted matrix-owned allocation")
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

// The caller comparison predates the tableau owner. Core and matrix capture
// must leave its receipt intact, including when their own arithmetic stops.
func TestLPRationalTableauPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"divide", "cost", "pivot"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "2")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "3")))
			h.must(f.a.Set(1, 0, lpScalarBoundValue(h, "5")))
			h.must(f.a.Set(1, 2, lpScalarBoundValue(h, "7")))
			h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "11")))
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			h.must(h.m.SetInt32(&m.D.Data[0], 13))
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
				case "divide":
					_, err = m.DivideRowByPivot(0, 0)
				case "cost":
					err = m.PivotToReducedCostsTableau(0, 0)
				case "pivot":
					_, err = m.PivotColumnTableau(0, 0)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) {
					t.Fatal(err)
				}
				if m.calls.comparison != 0 || f.a.calls.comparison != 0 {
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
			t.Fatal("no actual caller comparison", op)
		}
	}
}
