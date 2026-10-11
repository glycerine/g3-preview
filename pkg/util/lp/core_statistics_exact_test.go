package lp

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPPairStatisticsIndependentCostOracle(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		cols := 1 + world%5
		f := lpNewPairCoreFixture(t, h, 1, uint32(cols))
		m, e := NewPairCoreSolverBase(f.input())
		h.must(e)
		sx, sy := new(big.Rat), new(big.Rat)
		for j := 0; j < cols; j++ {
			c := big.NewRat(int64((world*3+j*7)%31-15), 13)
			x, y := big.NewRat(int64((world+j)%17-8), 7), big.NewRat(int64((world*5+j)%23-11), 11)
			h.must(h.m.SetString(&f.costs.Data[j], c.RatString()))
			h.must(f.x.Data[j].Copy(h.m, h.pair(x.RatString(), y.RatString())))
			sx.Add(sx, new(big.Rat).Mul(c, x))
			sy.Add(sy, new(big.Rat).Mul(c, y))
		}
		out := h.pair("97", "101")
		if world%3 == 0 {
			out = &f.x.Data[cols-1]
		}
		h.must(m.GetCost(out))
		h.check(out, sx, sy)
		if m.valueWork != nil || m.valueScalars != nil {
			t.Fatal("cost scratch retained")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairStatisticsPreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"cost", "print", "combined", "feasible"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(f.x.Data[0].Copy(h.m, v))
			h.must(h.m.SetInt32(&f.costs.Data[0], 3))
			f.types[2] = ColumnLowBound
			h.must(f.x.Data[2].Copy(h.m, v))
			h.must(f.low.Data[2].Copy(h.m, h.pair("1", "0")))
			m.Settings.PrintStatistics = true
			m.Settings.ReportFrequency = 1
			out := h.pair("0", "0")
			var buf bytes.Buffer
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
				case "cost":
					err = m.GetCost(out)
				case "print":
					err = m.PrintStatistics(nil, v, &buf)
				case "combined":
					_, err = m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(nil, &buf)
				case "feasible":
					_, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
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

func TestLPPairStatisticsGatesCountersAndCancellation(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		f, m := lpAdvanceDifferentFixture(t, h)
		sx, sy := new(big.Rat), new(big.Rat)
		for j := 0; j < 3; j++ {
			c := big.NewRat(int64(world%7+j+1), 13)
			x, y := big.NewRat(int64(world%9+j), 7), big.NewRat(int64(world%5-j), 11)
			h.must(h.m.SetString(&f.costs.Data[j], c.RatString()))
			h.must(f.x.Data[j].Copy(h.m, h.pair(x.RatString(), y.RatString())))
			sx.Add(sx, new(big.Rat).Mul(c, x))
			sy.Add(sy, new(big.Rat).Mul(c, y))
		}
		old := uint32(world % 4)
		if world%7 == 0 {
			old = ^uint32(0)
		}
		m.TotalIterations = old
		m.Settings.Stats.TotalIterations = ^uint32(0)
		m.Settings.ReportFrequency = int32(world % 4)
		if world%13 == 0 {
			m.Settings.ReportFrequency = -1
		}
		m.Settings.PrintStatistics = world/4%2 == 0
		limit := &lpTestResourceLimit{cancel: world%3 == 0}
		m.Settings.SetResourceLimit(limit)
		prefix := (*string)(nil)
		text := "feas"
		if world%2 == 0 {
			prefix = &text
		}
		nz := 1
		if world/8%2 == 0 {
			h.must(m.Factorization.Close())
			m.Factorization = nil
			nz = 3
		}
		var buf bytes.Buffer
		over, e := m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(prefix, &buf)
		h.must(e)
		if over != limit.cancel || limit.calls != 1 || m.TotalIterations != old+1 || m.Settings.Stats.TotalIterations != 0 {
			t.Fatal("counter/cancel prefix", world)
		}
		if over && m.Status != StatusTimeExhausted {
			t.Fatal("cancel status")
		}
		want := ""
		if m.Settings.ReportFrequency != 0 && m.Settings.PrintStatistics && old%uint32(m.Settings.ReportFrequency) == 0 {
			number, _ := new(big.Rat).Add(sx, new(big.Rat).Quo(sy, big.NewRat(1000, 1))).Float64()
			if prefix != nil {
				want = *prefix + " "
			}
			want += fmt.Sprintf("iterations = %d, cost = %s, nonzeros = %d\n", old, strconv.FormatFloat(number, 'g', 6, 64), nz)
		}
		if buf.String() != want || m.statisticsArgument != nil || m.statisticsCost != nil {
			t.Fatal("statistics", world, buf.String(), want)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairStatisticsAllWriterPrefixes(t *testing.T) {
	line := "inf iterations = 5, cost = 5, nonzeros = 1\n"
	for stop := 0; stop < len(line); stop++ {
		h := newLPPairHarness(t)
		f, m := lpAdvanceDifferentFixture(t, h)
		h.must(f.x.Data[0].Copy(h.m, h.pair("2", "3000")))
		h.must(h.m.SetInt32(&f.costs.Data[0], 1))
		m.TotalIterations = 5
		m.Settings.ReportFrequency = 1
		m.Settings.PrintStatistics = true
		limit := &lpTestResourceLimit{cancel: true}
		m.Settings.SetResourceLimit(limit)
		w := &lpResidualFailWriter{remaining: stop}
		prefix := "inf"
		_, err := m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(&prefix, lpResidualWriterOnly{writer: w})
		if err != lpErrResidualWrite || w.String() != line[:stop] || limit.calls != 0 || m.TotalIterations != 6 || m.Settings.Stats.TotalIterations != 1 || m.Status == StatusTimeExhausted {
			t.Fatal(stop, err, w.String())
		}
		if m.statisticsArgument == nil || m.statisticsCost == nil {
			t.Fatal("lost source locals")
		}
		if _, e := m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(&prefix, w); e != ErrExactVectorPending {
			t.Fatal("retry", e)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

type lpStatisticsMutatingWriter struct {
	bytes.Buffer
	mutate  func()
	changed bool
}

func (w *lpStatisticsMutatingWriter) Write(p []byte) (int, error) {
	if !w.changed {
		w.changed = true
		w.mutate()
	}
	return w.Buffer.Write(p)
}

func TestLPPairStatisticsByValueAndEvaluationOrder(t *testing.T) {
	for _, phase := range []string{"argument", "factorization"} {
		h := newLPPairHarness(t)
		f, m := lpAdvanceDifferentFixture(t, h)
		cost := h.pair("1", "2000")
		prefix := "p"
		w := &lpStatisticsMutatingWriter{}
		w.mutate = func() {
			if phase == "argument" {
				h.must(cost.Copy(h.m, h.pair("99", "0")))
			} else {
				h.must(m.Factorization.Close())
				m.Factorization = nil
			}
		}
		h.must(m.PrintStatistics(&prefix, cost, w))
		nz := 1
		if phase == "factorization" {
			nz = 3
		}
		want := fmt.Sprintf("p iterations = 4294967295, cost = 3, nonzeros = %d\n", nz)
		if w.String() != want {
			t.Fatal(phase, w.String(), want)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairStatisticsExactProjectionBeforeFloat(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpAdvanceDifferentFixture(t, h)
	x := mustLPRat(t, strings.Repeat("1234567890", 20)+"1/7")
	y := new(big.Rat).Sub(big.NewRat(1000, 1), new(big.Rat).Mul(x, big.NewRat(1000, 1)))
	cost := h.pair(x.RatString(), y.RatString())
	var buf bytes.Buffer
	empty := ""
	h.must(m.PrintStatistics(&empty, cost, &buf))
	if buf.String() != " iterations = 4294967295, cost = 1, nonzeros = 1\n" {
		t.Fatal("converted operands before exact cancellation", buf.String())
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairStatisticsLiveLocalReentryGuard(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpAdvanceDifferentFixture(t, h)
	m.Settings.PrintStatistics = true
	m.Settings.ReportFrequency = 1
	cost := h.pair("3", "0")
	var nested bytes.Buffer
	w := &lpStatisticsMutatingWriter{}
	w.mutate = func() {
		before := m.TotalIterations
		if e := m.PrintStatistics(nil, cost, &nested); e != ErrExactVectorPending {
			t.Fatal("live argument", e)
		}
		if _, e := m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(nil, &nested); e != ErrExactVectorPending {
			t.Fatal("live cost", e)
		}
		if m.TotalIterations != before || nested.Len() != 0 {
			t.Fatal("nested publication")
		}
	}
	over, e := m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(nil, w)
	h.must(e)
	if over || m.TotalIterations != 1 || m.statisticsCost != nil || m.statisticsArgument != nil {
		t.Fatal("outer completion")
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairStatisticsPhysicalOwnership(t *testing.T) {
	for _, op := range []string{"cost", "print", "combined", "feasible"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1800; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpAdvanceDifferentFixture(t, h)
					huge := strings.Repeat("1234567890", 12)
					for j := range f.costs.Data {
						h.must(h.m.SetString(&f.costs.Data[j], huge+fmt.Sprintf("%d/13", j+1)))
						h.must(f.x.Data[j].Copy(h.m, h.pair(huge+fmt.Sprintf("%d/17", j+2), huge+fmt.Sprintf("%d/19", j+3))))
					}
					out := h.pair(huge+"1/23", huge+"1/29")
					if op == "feasible" {
						f.types[2] = ColumnLowBound
						h.must(f.low.Data[2].Copy(h.m, h.pair(huge+"7/31", "0")))
					}
					m.Settings.PrintStatistics = true
					m.Settings.ReportFrequency = 1
					var buf bytes.Buffer
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "feasible":
						_, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
					case "cost":
						err = m.GetCost(out)
					case "print":
						err = m.PrintStatistics(nil, out, &buf)
					case "combined":
						_, err = m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(nil, &buf)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if _, e := m.TimeIsOver(); e != ErrExactVectorPending {
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

func TestLPPairAllColumnsFeasibilityIndependentOracle(t *testing.T) {
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 5)
		m, e := NewPairCoreSolverBase(f.input())
		h.must(e)
		want := true
		for j := 0; j < 5; j++ {
			kind := ColumnType((world + j) % 5)
			f.types[j] = kind
			x, y := big.NewRat(int64((world+j)%7-3), 2), big.NewRat(int64((world*3+j)%5-2), 7)
			if world%8 == 0 {
				x.SetInt64(0)
				y.SetInt64(0)
			}
			low, up := big.NewRat(-1, 1), big.NewRat(1, 1)
			if kind == ColumnFixed {
				low.SetInt64(0)
				up.SetInt64(0)
			}
			h.must(f.x.Data[j].Copy(h.m, h.pair(x.RatString(), y.RatString())))
			h.must(f.low.Data[j].Copy(h.m, h.pair(low.RatString(), "0")))
			h.must(f.up.Data[j].Copy(h.m, h.pair(up.RatString(), "0")))
			below := x.Cmp(low) < 0 || (x.Cmp(low) == 0 && y.Sign() < 0)
			above := x.Cmp(up) > 0 || (x.Cmp(up) == 0 && y.Sign() > 0)
			if (kind == ColumnLowBound && below) || (kind == ColumnUpperBound && above) || ((kind == ColumnBoxed || kind == ColumnFixed) && (below || above)) {
				want = false
			}
		}
		got, e := m.CalcCurrentXIsFeasibleIncludeNonBasis()
		h.must(e)
		if got != want {
			t.Fatal(world, got, want)
		}
		if m.InfSet.Size() != 0 {
			t.Fatal("validator mutated stored feasibility")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairStatisticsReachedGuardsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"nil-cost-out", "short-x", "long-x", "empty-dot", "nil-print-cost", "nil-writer", "cancel-error", "stopped-factor", "erased-gate", "feasible-range", "feasible-first-stop", "feasible-unknown"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
			h.must(h.m.SetInt32(&f.costs.Data[0], 2))
			h.must(f.x.Data[0].Copy(h.m, h.pair("3", "5")))
			out := h.pair("97", "101")
			var buf bytes.Buffer
			want := error(nil)
			var err error
			switch phase {
			case "nil-cost-out":
				want = ErrBounds
				err = m.GetCost(nil)
			case "short-x":
				h.must(f.x.resizeData(1))
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				err = m.GetCost(out)
				h.check(out, big.NewRat(97, 1), big.NewRat(101, 1))
			case "long-x":
				h.must(f.x.resizeData(4))
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				err = m.GetCost(out)
				if want == nil {
					h.check(out, big.NewRat(6, 1), big.NewRat(10, 1))
				}
			case "empty-dot":
				h.must(f.x.resizeData(0))
				h.must(f.costs.resizeData(0))
				err = m.GetCost(out)
				h.check(out, new(big.Rat), new(big.Rat))
			case "nil-print-cost":
				want = ErrBounds
				err = m.PrintStatistics(nil, nil, &buf)
			case "nil-writer":
				want = ErrBounds
				err = m.PrintStatistics(nil, out, nil)
				if m.statisticsArgument == nil {
					t.Fatal("argument copy prefix")
				}
			case "cancel-error":
				limit := &lpTestResourceLimit{cancel: true, err: lpErrResidualWrite}
				m.Settings.SetResourceLimit(limit)
				want = lpErrResidualWrite
				_, err = m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(nil, nil)
				if limit.calls != 1 || m.TotalIterations != 1 || m.Status != StatusUnknown {
					t.Fatal("cancellation error prefix")
				}
			case "stopped-factor":
				if e := m.Factorization.SolveBy(nil); e == nil {
					t.Fatal("expected stopped child")
				}
				want = ErrLUStopped
				err = m.PrintStatistics(nil, out, &buf)
				if buf.String() != "iterations = 4294967295, cost = 97.101, nonzeros = " {
					t.Fatal("LU access prefix", buf.String())
				}
			case "erased-gate":
				h.must(f.x.resizeData(0))
				limit := &lpTestResourceLimit{}
				m.Settings.SetResourceLimit(limit)
				m.Settings.PrintStatistics = false
				m.Settings.ReportFrequency = 1
				_, err = m.PrintStatisticsWithIterationsAndNonzeroesAndCostAndCheckTimeOver(nil, nil)
				if limit.calls != 1 || m.TotalIterations != 1 || m.statisticsCost != nil {
					t.Fatal("disabled report gate")
				}
			case "feasible-range":
				f.types = nil
				want = ErrBounds
				_, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
			case "feasible-first-stop":
				f.types[0] = ColumnType(99)
				f.types[2] = ColumnLowBound
				h.must(f.x.Data[2].Copy(h.m, h.pair("-1", "0")))
				var ok bool
				ok, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
				if ok {
					t.Fatal("descending early stop")
				}
			case "feasible-unknown":
				f.types[2] = ColumnType(99)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				var ok bool
				ok, err = m.CalcCurrentXIsFeasibleIncludeNonBasis()
				if ok {
					t.Fatal("unknown type")
				}
			}
			if err != want {
				t.Fatal(phase, err, want)
			}
			// All public entries must reject the same stopped owner, without counters
			// or borrowed input traversal, after any actual interrupted operation.
			if want != nil {
				if _, e := m.IncTotalIterations(); e != ErrExactVectorPending {
					t.Fatal("counter retry", e)
				}
				if e := m.GetCost(out); e != ErrExactVectorPending {
					t.Fatal("cost retry", e)
				}
				if e := m.PrintStatistics(nil, out, &buf); e != ErrExactVectorPending {
					t.Fatal("print retry", e)
				}
				if _, e := m.CalcCurrentXIsFeasibleIncludeNonBasis(); e != ErrExactVectorPending {
					t.Fatal("feasible retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
