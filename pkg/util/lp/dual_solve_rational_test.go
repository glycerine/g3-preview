// Source: src/util/lp/lp_dual_core_solver.hpp:162-204,731-761.
package lp

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func lpRationalDualStandardFixture(t *testing.T, h *lpPairHarness, a [][]*big.Rat, b, c []*big.Rat, basis []uint32) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, uint32(len(b)), uint32(len(c)))
	f.settings.SimplexStrategy = StrategyLU
	f.settings.MaxTotalIterations = 200
	f.basis = append([]uint32(nil), basis...)
	can := make([]bool, len(c))
	for j := range c {
		can[j] = true
		f.types[j] = ColumnLowBound
		h.must(h.m.Copy(&f.costs.Data[j], lpScalarBoundValue(h, c[j].RatString())))
	}
	for i, row := range a {
		h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, b[i].RatString())))
		for j, v := range row {
			h.must(f.a.Set(uint32(i), uint32(j), lpScalarBoundValue(h, v.RatString())))
		}
	}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.RowsNZ = make([]uint32, len(b))
	m.ColumnsNZ = make([]uint32, len(c))
	h.must(m.RecalculateD())
	h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
	return f, m
}

// Enumerate basic feasible vertices with independent dense exact elimination.
// No project LU, pivot, ratio, price or solver operation is used by this oracle.
func lpIndependentRationalVertexMinimum(a [][]*big.Rat, b, c []*big.Rat) *big.Rat {
	rows, cols := len(b), len(c)
	var best *big.Rat
	selected := make([]int, 0, rows)
	var visit func(int)
	visit = func(next int) {
		if len(selected) < rows {
			for j := next; j <= cols-(rows-len(selected)); j++ {
				selected = append(selected, j)
				visit(j + 1)
				selected = selected[:len(selected)-1]
			}
			return
		}
		aug := make([][]*big.Rat, rows)
		for i := 0; i < rows; i++ {
			aug[i] = make([]*big.Rat, rows+1)
			for j := 0; j < rows; j++ {
				aug[i][j] = new(big.Rat).Set(a[i][selected[j]])
			}
			aug[i][rows] = new(big.Rat).Set(b[i])
		}
		for j := 0; j < rows; j++ {
			pivot := j
			for pivot < rows && aug[pivot][j].Sign() == 0 {
				pivot++
			}
			if pivot == rows {
				return
			}
			aug[j], aug[pivot] = aug[pivot], aug[j]
			scale := new(big.Rat).Set(aug[j][j])
			for k := j; k <= rows; k++ {
				aug[j][k].Quo(aug[j][k], scale)
			}
			for i := 0; i < rows; i++ {
				if i == j {
					continue
				}
				factor := new(big.Rat).Set(aug[i][j])
				for k := j; k <= rows; k++ {
					aug[i][k].Sub(aug[i][k], new(big.Rat).Mul(factor, aug[j][k]))
				}
			}
		}
		cost := new(big.Rat)
		for i := 0; i < rows; i++ {
			if aug[i][rows].Sign() < 0 {
				return
			}
			cost.Add(cost, new(big.Rat).Mul(c[selected[i]], aug[i][rows]))
		}
		if best == nil || cost.Cmp(best) < 0 {
			best = new(big.Rat).Set(cost)
		}
	}
	visit(0)
	return best
}
func TestLPRationalDualSolveIndependentVertices(t *testing.T) {
	rng := rand.New(rand.NewSource(745122))
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		rows := 1 + rng.Intn(3)
		structurals := 2 + rng.Intn(3)
		cols := structurals + rows
		a := make([][]*big.Rat, rows)
		b := make([]*big.Rat, rows)
		c := make([]*big.Rat, cols)
		basis := make([]uint32, rows)
		for j := 0; j < cols; j++ {
			c[j] = new(big.Rat)
			if j < structurals {
				c[j].SetFrac64(int64(1+rng.Intn(8)), int64(2+rng.Intn(7)))
			}
		}
		for i := 0; i < rows; i++ {
			a[i] = make([]*big.Rat, cols)
			b[i] = big.NewRat(-int64(1+rng.Intn(20)), int64(2+rng.Intn(7)))
			for j := 0; j < cols; j++ {
				a[i][j] = new(big.Rat)
				if j < structurals {
					a[i][j].SetFrac64(-int64(1+rng.Intn(8)), int64(2+rng.Intn(7)))
				}
			}
			a[i][structurals+i].SetInt64(1)
			basis[i] = uint32(structurals + i)
		}
		want := lpIndependentRationalVertexMinimum(a, b, c)
		if want == nil {
			t.Fatal("oracle has no feasible vertex", world)
		}
		f, m := lpRationalDualStandardFixture(t, h, a, b, c, basis)
		f.settings.RandomSeed(uint32(world + 1))
		steps, e := m.Solve()
		if sourceDebugEnabled {
			if e != ErrAssertion || steps != 1 || !reflect.DeepEqual(f.basis, basis) {
				t.Fatal(world, steps, e, f.basis, basis)
			}
		} else {
			h.must(e)
			if m.Status != StatusOptimal || steps == 0 {
				t.Fatal(world, m.Status, steps)
			}
			cost := lpScalarBoundValue(h, "0")
			h.must(m.GetCost(cost))
			lpSparseRationalCheck(t, h, cost, want)
			for j := range c {
				text, e := h.m.String(&f.x.Data[j])
				h.must(e)
				if lpRationalTestBig(text).Sign() < 0 {
					t.Fatal("negative final value", world, j, text)
				}
			}
			off, e := m.AMultXIsOff()
			h.must(e)
			if off {
				t.Fatal("final residual", world)
			}
			dual, e := m.ProblemIsDualFeasible()
			h.must(e)
			if !dual {
				t.Fatal("final dual feasibility", world)
			}
			correct, e := m.DIsCorrect()
			h.must(e)
			if !correct {
				t.Fatal("final reduced costs", world)
			}
			// Source Solve is do/while even if already optimal.
			again, e := m.Solve()
			h.must(e)
			if again != 1 || m.Status != StatusOptimal {
				t.Fatal("optimal reentry", world, again, m.Status)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalDualPricingFixture(t *testing.T, h *lpPairHarness, n uint32) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, n, n+1)
	f.settings.SimplexStrategy = StrategyLU
	for i := uint32(0); i < n; i++ {
		h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
		f.types[i] = ColumnLowBound
	}
	can := make([]bool, n+1)
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.RowsNZ = make([]uint32, n)
	m.ColumnsNZ = make([]uint32, n+1)
	h.must(m.InitBetas())
	return f, m
}
func TestLPRationalDualSolvePricingIndependentRecordBudget(t *testing.T) {
	rng := rand.New(rand.NewSource(7211))
	for world := 0; world < 500; world++ {
		h := newLPPairHarness(t)
		n := 1 + rng.Intn(9)
		f, m := lpRationalDualPricingFixture(t, h, uint32(n))
		scores, values := make([]*big.Rat, n), make([]*big.Rat, n)
		m.ForbiddenRows = map[uint32]bool{}
		for i := 0; i < n; i++ {
			values[i] = big.NewRat(-int64(rng.Intn(7)), int64(1+rng.Intn(5)))
			beta := big.NewRat(int64(1+rng.Intn(5)), int64(1+rng.Intn(7)))
			scores[i] = new(big.Rat).Quo(new(big.Rat).Mul(values[i], values[i]), beta)
			h.must(h.m.Copy(&f.x.Data[i], lpScalarBoundValue(h, values[i].RatString())))
			h.must(h.m.Copy(&m.Betas.Data[i], lpScalarBoundValue(h, beta.RatString())))
			if rng.Intn(3) == 0 {
				m.ForbiddenRows[uint32(i)] = true
			}
		}
		offset, budget := rng.Intn(n), rng.Intn(n+3)
		chosen := -1
		best := new(big.Rat)
		records := 0
		for visit := 0; visit < n; visit++ {
			i := (offset + visit) % n
			if !m.ForbiddenRows[uint32(i)] && scores[i].Cmp(best) > 0 {
				chosen = i
				best.Set(scores[i])
				records++
			}
			if budget == 0 || records == budget {
				break
			}
		}
		m.TotalIterations = f.settings.MaxTotalIterations + 1
		m.Status = StatusUnknown
		h.must(m.PricingLoop(uint32(budget), uint32(offset)))
		if int(m.R) != chosen {
			t.Fatal(world, n, budget, offset, m.R, chosen)
		}
		if chosen >= 0 {
			if m.P != int32(chosen) || len(m.ForbiddenRows) != 0 || m.Status != StatusIterationsExhausted {
				t.Fatal(world, m.P, m.ForbiddenRows, m.Status)
			}
			lpSparseRationalCheck(t, h, &m.Delta, values[chosen])
		} else if m.Status != StatusOptimal {
			t.Fatal(world, m.Status)
		}
		if m.loopValues != nil {
			t.Fatal("loop maximum escaped successful return")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalDualSolveOneRow(t *testing.T, h *lpPairHarness, entering int64) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	return lpRationalDualStandardFixture(t, h, [][]*big.Rat{{big.NewRat(1, 1), big.NewRat(entering, 1)}}, []*big.Rat{big.NewRat(-3, 1)}, []*big.Rat{new(big.Rat), big.NewRat(2, 1)}, []uint32{0})
}

func TestLPRationalDualSolveLimitsCancellationAndUnbounded(t *testing.T) {
	for _, phase := range []string{"zero-limit", "one-limit", "unbounded", "head-cancel", "head-error", "advance-cancel", "advance-error", "time"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			entering := int64(-1)
			if phase == "unbounded" {
				entering = 0
			}
			f, m := lpRationalDualSolveOneRow(t, h, entering)
			limit := &lpRationalSolveCancelSequence{}
			f.settings.SetResourceLimit(limit)
			m.TotalIterations = 91
			m.IterationsNoCostGrowth = 77
			f.settings.Stats.TotalIterations = 17
			wantStatus, wantCount, wantCalls, wantErr := StatusOptimal, uint32(2), 2, error(nil)
			switch phase {
			case "zero-limit":
				f.settings.MaxTotalIterations = 0
				wantStatus, wantCount, wantCalls = StatusIterationsExhausted, 1, 1
			case "one-limit":
				f.settings.MaxTotalIterations = 1
			case "unbounded":
				wantStatus, wantCount, wantCalls = StatusDualUnbounded, 2, 2
			case "head-cancel":
				limit.values = []bool{true}
				wantStatus, wantCount, wantCalls = StatusTimeExhausted, 1, 1
			case "head-error":
				limit.errorAt = 1
				wantStatus, wantCount, wantCalls, wantErr = StatusFeasible, 1, 1, lpErrRationalSolveCancel
			case "advance-cancel":
				limit.values = []bool{false, true}
				wantStatus, wantCount, wantCalls = StatusTimeExhausted, 2, 2
			case "advance-error":
				limit.errorAt = 2
				wantStatus, wantCount, wantCalls, wantErr = StatusFeasible, 2, 2, lpErrRationalSolveCancel
			case "time":
				f.settings.SetResourceLimit(nil)
				f.settings.TimeLimit = -1
				wantStatus, wantCount, wantCalls = StatusTimeExhausted, 1, 0
			}
			if sourceDebugEnabled && (phase == "one-limit" || phase == "advance-cancel" || phase == "advance-error") {
				wantStatus, wantCount, wantCalls, wantErr = StatusFeasible, 1, 1, ErrAssertion
			}
			count, e := m.Solve()
			if count != wantCount || e != wantErr || m.Status != wantStatus || limit.calls != wantCalls || f.settings.Stats.TotalIterations != 17+wantCount {
				t.Fatal(count, e, m.Status, limit.calls, "want", wantCount, wantErr, wantStatus, wantCalls)
			}
			if sourceDebugEnabled || (phase != "one-limit" && phase != "advance-cancel" && phase != "advance-error") {
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-3, 1))
			} else {
				lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(3, 1))
			}
			if e != nil {
				if _, again := m.Solve(); again != ErrExactVectorPending {
					t.Fatal("retry", again)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualSolveIterationRNGAndForbidden(t *testing.T) {
	for _, status := range []Status{StatusUnknown, StatusUnstable, StatusTentativeDualUnbounded} {
		for _, forbidden := range []bool{false, true} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPricingFixture(t, h, 5)
			f.settings.RandomSeed(791)
			rng := f.settings.random
			rng.Next()
			rng.Next()
			m.Status = status
			if forbidden {
				m.ForbiddenRows = map[uint32]bool{}
				for i := uint32(0); i < 5; i++ {
					m.ForbiddenRows[i] = true
					h.must(h.m.SetInt32(&f.x.Data[i], -1))
				}
				h.must(m.PricingLoop(5, 0))
				want := StatusOptimal
				if status == StatusUnstable {
					want = StatusUnstable
				}
				if m.R != -1 || m.Status != want || len(m.ForbiddenRows) != 5 {
					t.Fatal(m.R, m.Status, m.ForbiddenRows)
				}
				m.Status = status
			}
			h.must(m.OneIteration())
			if m.Status != StatusOptimal || m.R != -1 || f.settings.random != rng {
				t.Fatal(m.Status, m.R, f.settings.random, rng)
			}
			if forbidden {
				for i := range f.basis {
					lpSparseRationalCheck(t, h, &f.x.Data[i], big.NewRat(-1, 1))
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalDualSolveEmptyAndPricingGuards(t *testing.T) {
	for _, operation := range []string{"offset", "empty-price", "empty-iteration", "empty-solve", "score", "basis"} {
		t.Run(operation, func(t *testing.T) {
			h := newLPPairHarness(t)
			n := uint32(1)
			if operation == "empty-price" || operation == "empty-iteration" || operation == "empty-solve" {
				n = 0
			}
			f, m := lpRationalDualPricingFixture(t, h, n)
			m.P = 8
			h.must(h.m.SetInt32(&m.Delta, 9))
			oldTypes, oldBasis := f.types, f.basis
			var e error
			switch operation {
			case "offset":
				e = m.PricingLoop(1, 1)
			case "empty-price":
				e = m.PricingLoop(0, 0)
			case "empty-iteration":
				e = m.OneIteration()
			case "empty-solve":
				_, e = m.Solve()
			case "score":
				f.types = nil
				e = m.PricingLoop(1, 0)
			case "basis":
				h.must(h.m.SetInt32(&f.x.Data[0], -1))
				f.basis = nil
				e = m.PricingLoop(1, 0)
			}
			if e != ErrBounds {
				t.Fatal(e)
			}
			if m.P != 8 {
				t.Fatal("early publication", m.P)
			}
			lpSparseRationalCheck(t, h, &m.Delta, big.NewRat(9, 1))
			if _, e = m.Solve(); e != ErrExactVectorPending {
				t.Fatal("stopped retry", e)
			}
			f.types, f.basis = oldTypes, oldBasis
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualSolveReportingBytePrefixes(t *testing.T) {
	const want = " iterations = 0, cost = 0, nonzeros = 1\n"
	for cut := 0; cut <= len(want); cut++ {
		h := newLPPairHarness(t)
		f, m := lpRationalDualSolveOneRow(t, h, -1)
		f.settings.PrintStatistics = true
		f.settings.ReportFrequency = 1
		f.settings.TimeLimit = -1
		w := &lpResidualFailWriter{remaining: cut}
		f.settings.MessageOut = lpResidualWriterOnly{writer: w}
		count, e := m.Solve()
		if count != 1 || w.String() != want[:cut] {
			t.Fatal(cut, count, e, w.String())
		}
		if cut < len(want) {
			if e != lpErrResidualWrite || m.Status != StatusFeasible || m.RationalCoreSolverBase.statisticsCost == nil {
				t.Fatal(cut, e, m.Status)
			}
		} else if e != nil || m.Status != StatusTimeExhausted {
			t.Fatal(cut, e, m.Status)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	h := newLPPairHarness(t)
	f, m := lpRationalDualSolveOneRow(t, h, -1)
	var out bytes.Buffer
	f.settings.PrintStatistics = true
	f.settings.ReportFrequency = 1
	f.settings.MessageOut = &out
	f.settings.MaxTotalIterations = 0
	n, e := m.Solve()
	h.must(e)
	if n != 1 || out.String() != want || m.Status != StatusIterationsExhausted {
		t.Fatal(n, out.String(), m.Status)
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func lpRationalDualSolveOperation(m *RationalDualCoreSolver, op string) error {
	switch op {
	case "price":
		return m.PricingLoop(m.A.RowCount(), 0)
	case "iteration":
		return m.OneIteration()
	case "solve":
		_, e := m.Solve()
		return e
	}
	panic(op)
}
func lpRationalDualSolveLargeFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	f, m := lpRationalDualAdvanceLargeFixture(t, h)
	v, e := f.a.ColumnValue(f.a.Columns[1][0])
	h.must(e)
	h.must(h.m.SetInt32(v, -1))
	f.settings.MaxTotalIterations = 20
	return f, m
}
func TestLPRationalDualSolvePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"price", "iteration", "solve"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 4000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalDualSolveLargeFixture(t, h)
					h.budget.FailAfter(ordinal)
					e := lpRationalDualSolveOperation(m, op)
					h.budget.FailAfter(-1)
					if e == nil || sourceDebugEnabled && e == ErrAssertion {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if retry := lpRationalDualSolveOperation(m, op); retry != ErrExactVectorPending {
							t.Fatal("stopped retry", retry)
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
			if !success || direct && failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}
func TestLPRationalDualSolvePreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"price", "iteration", "solve"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualSolveLargeFixture(t, h)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, before, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				_ = lpRationalDualSolveOperation(m, op)
				if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			} else if e != nil && !lpExactPriorityPhysicalRefusal(e) {
				t.Fatal(e)
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

func TestLPRationalDualSolveCheckedPreconditionOrder(t *testing.T) {
	for _, phase := range []string{"d-false", "dual-false", "dual-metadata", "heading-false", "heading-metadata"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPricingFixture(t, h, 1)
			m.TotalIterations = 99
			m.IterationsNoCostGrowth = 77
			oldTypes, oldHeading := f.types, f.heading
			want := error(ErrAssertion)
			switch phase {
			case "d-false":
				f.nb = []uint32{1}
				h.must(h.m.SetInt32(&m.D.Data[1], 1))
			case "dual-false":
				f.types[1] = ColumnLowBound
				h.must(h.m.SetInt32(&f.costs.Data[1], -1))
				h.must(h.m.SetInt32(&m.D.Data[1], -1))
				f.nb = []uint32{1}
			case "dual-metadata":
				f.nb = []uint32{1}
				f.types = f.types[:1]
				want = ErrBounds
			case "heading-false":
				f.heading[0] = 1
			case "heading-metadata":
				f.heading = nil
			}
			n, e := m.Solve()
			if sourceDebugEnabled {
				if e != want || n != 99 || m.IterationsNoCostGrowth != 77 || f.settings.Stats.TotalIterations != 0 {
					t.Fatal(n, e, m.IterationsNoCostGrowth, f.settings.Stats.TotalIterations)
				}
			} else if e != nil || n != 1 || m.Status != StatusOptimal {
				t.Fatal(n, e, m.Status)
			}
			f.types, f.heading = oldTypes, oldHeading
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualSolvePricingActualRecovery(t *testing.T) {
	for _, singular := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		// Column 2 cannot enter for this negative leaving displacement. Its stale
		// negative reduced cost survives the exchange and forces actual rollback.
		h.must(h.m.SetInt32(&m.D.Data[2], -10))
		if singular {
			v, e := f.a.ColumnValue(f.a.Columns[0][0])
			h.must(e)
			h.must(h.m.SetInt32(v, 0))
		}
		m.ForbiddenRows = map[uint32]bool{9: true}
		e := m.PricingLoop(1, 0)
		if sourceDebugEnabled {
			if e != ErrAssertion || m.loopValues == nil {
				t.Fatal(e, m.loopValues)
			}
		} else {
			h.must(e)
			want := StatusUnstable
			if singular {
				want = StatusFloatingPointError
			}
			if m.Status != want || !m.ForbiddenRows[9] || m.ForbiddenRows[0] == singular || m.loopValues != nil || !reflect.DeepEqual(f.basis, []uint32{0}) {
				t.Fatal(m.Status, m.ForbiddenRows, m.loopValues, f.basis)
			}
			if !singular {
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-3, 1))
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualSolveTentativeFullScan(t *testing.T) {
	for seed := uint32(1); seed <= 40; seed++ {
		for _, tentative := range []bool{false, true} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPricingFixture(t, h, 5)
			f.settings.RandomSeed(seed)
			rng := f.settings.random
			budget, offset := rng.Next()%5+1, rng.Next()%5
			for visit := uint32(0); visit < 5; visit++ {
				i := (offset + visit) % 5
				h.must(h.m.SetInt32(&f.x.Data[i], -int32(visit+1)))
			}
			m.Status = StatusUnstable
			if tentative {
				m.Status = StatusTentativeDualUnbounded
				budget = 5
			}
			m.TotalIterations = f.settings.MaxTotalIterations + 1
			h.must(m.OneIteration())
			want := (offset + budget - 1) % 5
			if m.R != int32(want) || m.P != int32(want) || m.Status != StatusIterationsExhausted || f.settings.random != rng {
				t.Fatal(seed, tentative, m.R, m.P, m.Status)
			}
			lpSparseRationalCheck(t, h, &m.Delta, big.NewRat(-int64(budget), 1))
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}
func TestLPRationalDualSolveNoGrowthStrictLimit(t *testing.T) {
	for _, above := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualSolveOneRow(t, h, 0)
		f.settings.MaxIterationsWithoutImprovements = 1
		limit := &lpRationalSolveMutatingLimit{at: 1, mutate: func() {
			m.IterationsNoCostGrowth = 1
			if above {
				m.IterationsNoCostGrowth++
			}
		}}
		f.settings.SetResourceLimit(limit)
		n, e := m.Solve()
		h.must(e)
		count, status := uint32(2), StatusDualUnbounded
		if above {
			count, status = 1, StatusTentativeDualUnbounded
		}
		if n != count || m.Status != status || limit.calls != int(count) {
			t.Fatal(above, n, m.Status, limit.calls)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
