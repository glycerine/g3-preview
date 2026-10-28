package lp

// Source: src/util/lp/lp_dual_core_solver.hpp:213-235,339-350,433-533.

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPRationalDualAdvanceIndependentExchanges(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	b := [][]int64{{2, 1, 0}, {0, 4, 2}, {0, 0, 8}}
	inverse := [][]*big.Rat{{big.NewRat(1, 2), big.NewRat(-1, 8), big.NewRat(1, 32)}, {new(big.Rat), big.NewRat(1, 4), big.NewRat(-1, 16)}, {new(big.Rat), new(big.Rat), big.NewRat(1, 8)}}
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 3, 4)
		f.settings.SimplexStrategy = StrategyLU
		r := world % 3
		u := []*big.Rat{big.NewRat(int64(world%5-2), 4), big.NewRat(1, 2), big.NewRat(-1, 4)}
		u[r] = big.NewRat(-int64(1+world%3), 2)
		x := []*big.Rat{big.NewRat(5, 1), big.NewRat(7, 1), big.NewRat(9, 1)}
		x[r] = big.NewRat(-int64(1+world%11), 4)
		if world%7 == 0 {
			x[r].SetFrac(new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(int64(world+1)), uint(81+world%30))), big.NewInt(17))
		}
		for i := 0; i < 3; i++ {
			col, rhs := new(big.Rat), new(big.Rat)
			for j := 0; j < 3; j++ {
				h.must(f.a.Set(uint32(i), uint32(j), lpScalarBoundValue(h, fmt.Sprint(b[i][j]))))
				col.Add(col, new(big.Rat).Mul(big.NewRat(b[i][j], 1), u[j]))
				rhs.Add(rhs, new(big.Rat).Mul(big.NewRat(b[i][j], 1), x[j]))
			}
			h.must(f.a.Set(uint32(i), 3, lpScalarBoundValue(h, col.RatString())))
			h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, rhs.RatString())))
		}
		for i := range f.types {
			f.types[i] = ColumnLowBound
		}
		h.must(h.m.SetInt32(&f.costs.Data[3], 1))
		can := []bool{true, true, true, true}
		m, e := NewRationalDualCoreSolver(f.input(), &can)
		h.must(e)
		m.RowsNZ = make([]uint32, 3)
		m.ColumnsNZ = make([]uint32, 4)
		h.must(m.RecalculateD())
		h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
		h.must(m.InitBetasPrecisely())
		m.P = int32(r)
		m.R = int32(r)
		h.must(m.GetDelta(&m.Delta))
		old := m.Factorization
		forced := world%2 != 0
		if forced {
			old.RefactorCounter = 200
		}
		ok, e := m.AdvanceOnKnownP()
		ready := []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat).Set(inverse[r][2])}
		ready[1].Sub(inverse[r][1], new(big.Rat).Mul(big.NewRat(1, 2), ready[2]))
		ready[0].Sub(inverse[r][0], new(big.Rat).Mul(big.NewRat(1, 2), ready[1]))
		for i := 0; i < 3; i++ {
			lpSparseRationalCheck(t, h, &m.PivotRowB1.Data[i], ready[i])
		}
		if sourceDebugEnabled {
			if e != ErrAssertion || ok || !reflect.DeepEqual(f.basis, []uint32{0, 1, 2}) {
				t.Fatal(world, ok, e, f.basis)
			}
			for i := range x {
				lpSparseRationalCheck(t, h, &f.x.Data[i], x[i])
			}
			lpSparseRationalCheck(t, h, &m.ThetaP, new(big.Rat))
		} else {
			h.must(e)
			if !ok || m.Q != 3 || f.basis[r] != 3 || f.heading[r] >= 0 || f.heading[3] != int32(r) {
				t.Fatal(world, ok, m.Q, f.basis, f.heading)
			}
			step := new(big.Rat).Quo(x[r], u[r])
			lpSparseRationalCheck(t, h, &m.ThetaP, step)
			norms := make([]*big.Rat, 3)
			for i := 0; i < 3; i++ {
				norms[i] = new(big.Rat)
				for _, v := range inverse[i] {
					norms[i].Add(norms[i], new(big.Rat).Mul(v, v))
				}
			}
			for i := 0; i < 3; i++ {
				want := new(big.Rat).Sub(x[i], new(big.Rat).Mul(step, u[i]))
				if i == r {
					want.SetInt64(0)
				}
				lpSparseRationalCheck(t, h, &f.x.Data[i], want)
				beta := new(big.Rat).Quo(norms[r], new(big.Rat).Mul(u[r], u[r]))
				if i != r {
					ratio := new(big.Rat).Quo(u[i], u[r])
					beta.Add(norms[i], new(big.Rat).Mul(new(big.Rat).Mul(ratio, ratio), norms[r]))
					beta.Sub(beta, new(big.Rat).Mul(big.NewRat(2, 1), new(big.Rat).Mul(ratio, ready[i])))
					if beta.Sign() < 0 {
						beta.SetInt64(0)
					}
				}
				lpSparseRationalCheck(t, h, &m.Betas.Data[i], beta)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[3], step)
			if forced {
				if old == m.Factorization || !old.closed {
					t.Fatal("full refactor lifecycle")
				}
			} else if old != m.Factorization || old.RefactorCounter != 1 {
				t.Fatal("replacement lifecycle")
			}
			off, e := m.AMultXIsOff()
			h.must(e)
			if off {
				t.Fatal("exchange changed residual")
			}
			for _, j := range f.basis {
				h.must(h.m.SetInt32(&f.x.Data[j], 97))
			}
			found, e := m.FindXBySolving()
			h.must(e)
			if !found {
				t.Fatal("new basis did not reconstruct")
			}
			correct, e := m.DIsCorrect()
			h.must(e)
			if !correct {
				t.Fatal("new reduced costs")
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalDualAdvanceFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = StrategyLU
	for j, v := range []string{"2", "-2", "1"} {
		h.must(f.a.Set(0, uint32(j), lpScalarBoundValue(h, v)))
	}
	h.must(h.m.SetInt32(&f.b.Data[0], -6))
	h.must(h.m.SetInt32(&f.costs.Data[1], 1))
	h.must(h.m.SetInt32(&f.costs.Data[2], 1))
	for i := range f.types {
		f.types[i] = ColumnLowBound
	}
	can := []bool{true, true, true}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.RowsNZ = make([]uint32, 1)
	m.ColumnsNZ = make([]uint32, 3)
	m.P, m.R = 0, 0
	h.must(m.RecalculateD())
	h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
	h.must(m.GetDelta(&m.Delta))
	return f, m
}
func lpRationalDualPrepareAdvance(t *testing.T, h *lpPairHarness, m *RationalDualCoreSolver) {
	t.Helper()
	h.must(m.CalculatePivotRowOfBInverse(uint32(m.R)))
	h.must(m.CalculatePivotRowWhenInverseReady(uint32(m.R)))
	found, e := m.RatioTest()
	h.must(e)
	if !found {
		t.Fatal("fixture has no candidate")
	}
	h.must(m.CalculateBetaRPrecisely())
	h.must(m.SolveBd(uint32(m.Q)))
	h.must(m.DSEFTran())
}
func TestLPRationalDualAdvanceActualRecovery(t *testing.T) {
	for _, forced := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		lpRationalDualPrepareAdvance(t, h, m)
		h.must(h.m.SetInt32(&m.D.Data[2], -10))
		if forced {
			m.Factorization.RefactorCounter = 200
		}
		ok, e := m.BasisChangeAndUpdate()
		if sourceDebugEnabled {
			if ok || e != ErrAssertion {
				t.Fatal(ok, e)
			}
		} else {
			h.must(e)
			if ok || m.IterationsNoCostGrowth != 1 || !reflect.DeepEqual(f.basis, []uint32{0}) {
				t.Fatal(ok, m.IterationsNoCostGrowth, f.basis)
			}
			for j, text := range []string{"-3", "0", "0"} {
				lpSparseRationalCheck(t, h, &f.x.Data[j], lpRationalTestBig(text))
			}
			for j, text := range []string{"0", "1", "1"} {
				lpSparseRationalCheck(t, h, &m.D.Data[j], lpRationalTestBig(text))
			}
			lpSparseRationalCheck(t, h, &m.Betas.Data[0], big.NewRat(1, 4))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualAdvanceRevertAndRecoverPositions(t *testing.T) {
	for _, position := range []NonBasicColumnValuePosition{PositionAtLowBound, PositionAtFixed, PositionAtUpperBound, PositionFreeOfBounds, PositionNotAtBound, NonBasicColumnValuePosition(99)} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		m.Q = 1
		m.EnteringBoundaryPosition = position
		h.must(h.m.SetInt32(&f.low.Data[1], -2))
		h.must(h.m.SetInt32(&f.up.Data[1], 5))
		h.must(h.m.SetInt32(&f.x.Data[1], 97))
		e := m.RecoverLeaving()
		want := int64(97)
		switch position {
		case PositionAtLowBound, PositionAtFixed:
			want = -2
		case PositionAtUpperBound:
			want = 5
		case PositionFreeOfBounds:
			want = 0
		}
		assertion := position == PositionFreeOfBounds || position == PositionNotAtBound || position == NonBasicColumnValuePosition(99)
		if assertion && sourceDebugEnabled {
			if e != ErrAssertion {
				t.Fatal(position, e)
			}
		} else {
			h.must(e)
		}
		lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(want, 1))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, singular := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		m.P, m.Q = 0, 1
		m.EnteringBoundaryPosition = PositionAtLowBound
		h.must(m.ChangeBasis(1, 0))
		h.must(h.m.SetInt32(&f.x.Data[1], 3))
		if singular {
			v, e := f.a.ColumnValue(f.a.Columns[0][0])
			h.must(e)
			h.must(h.m.SetInt32(v, 0))
		}
		var out bytes.Buffer
		f.settings.DebugOut = &out
		h.must(m.RevertToPreviousBasis())
		if !reflect.DeepEqual(f.basis, []uint32{0}) || out.String() != "revert to previous basis on ( 0, 1)\n" {
			t.Fatal(f.basis, out.String())
		}
		if singular {
			if m.Status != StatusFloatingPointError {
				t.Fatal(m.Status)
			}
		} else {
			lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-3, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[1], new(big.Rat))
			lpSparseRationalCheck(t, h, &m.Betas.Data[0], big.NewRat(1, 4))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualAdvanceSnapIndependentProfiles(t *testing.T) {
	for _, typ := range []ColumnType{ColumnFixed, ColumnLowBound, ColumnBoxed, ColumnUpperBound, ColumnFree, ColumnType(99)} {
		for _, text := range []string{"-3", "-2", "-1/1000000000000000000000000000000", "1", "3/2", "3", "5", "6"} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualAdvanceFixture(t, h)
			f.types[1] = typ
			h.must(h.m.SetInt32(&f.low.Data[1], -2))
			h.must(h.m.SetInt32(&f.up.Data[1], 5))
			h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, text)))
			x := lpRationalTestBig(text)
			want := new(big.Rat).Set(x)
			snapped := false
			switch typ {
			case ColumnFixed, ColumnLowBound:
				if x.Cmp(big.NewRat(-2, 1)) != 0 {
					want.SetInt64(-2)
					snapped = true
				}
			case ColumnUpperBound:
				if x.Cmp(big.NewRat(5, 1)) != 0 {
					want.SetInt64(5)
					snapped = true
				}
			case ColumnBoxed:
				lower := new(big.Rat).Abs(new(big.Rat).Sub(big.NewRat(-2, 1), x))
				upper := new(big.Rat).Abs(new(big.Rat).Sub(big.NewRat(5, 1), x))
				target := big.NewRat(5, 1)
				if lower.Cmp(upper) < 0 {
					target.SetInt64(-2)
				}
				if x.Cmp(target) != 0 {
					want.SetInt64(-2)
					snapped = true
				}
			}
			got, e := m.SnapRunawayNonBasicColumn(1)
			h.must(e)
			if got != snapped {
				t.Fatal(typ, text, got, snapped)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[1], want)
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}
func TestLPRationalDualAdvanceDoneAndNoCandidate(t *testing.T) {
	for _, done := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		if done {
			m.Status = StatusOptimal
			m.R = -1
		} else {
			f.types[1] = ColumnFixed
			f.types[2] = ColumnFixed
		}
		ok, e := m.AdvanceOnKnownP()
		h.must(e)
		want := StatusTentativeDualUnbounded
		if done {
			want = StatusOptimal
		}
		if !ok || m.Status != want {
			t.Fatal(done, ok, m.Status)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualAdvanceCostDiagnosticThresholds(t *testing.T) {
	for _, text := range []string{"0", "1/100000000000000000000000000000000000000000000000000", "999/1000000", "1/1000", "1001/1000000", "123456789012345678901234567890/17"} {
		for _, negative := range []bool{false, true} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualAdvanceFixture(t, h)
			delta := lpRationalTestBig(text)
			if negative {
				delta.Neg(delta)
			}
			value := new(big.Rat).Add(big.NewRat(1, 1), delta)
			h.must(h.m.Copy(&m.D.Data[2], lpScalarBoundValue(h, value.RatString())))
			number := lpLARSourceDouble(new(big.Rat).Abs(delta))
			want := number < 0.001
			var out bytes.Buffer
			f.settings.DebugOut = &out
			m.TotalIterations = 9
			correct, e := m.DIsCorrect()
			h.must(e)
			if correct != want {
				t.Fatal(text, negative, correct, want)
			}
			expected := ""
			if !want {
				expected = "total_iterations = 9\nd[2] = " + value.RatString() + " but should be 1\n"
			}
			if out.String() != expected {
				t.Fatal(text, out.String(), expected)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

type lpRationalDualDiagnosticWriter struct {
	bytes.Buffer
	m        *RationalDualCoreSolver
	mutate   bool
	flushes  int
	flushErr error
}

func (w *lpRationalDualDiagnosticWriter) Write(p []byte) (int, error) {
	if w.mutate && strings.HasPrefix(string(p), "total_iterations") {
		if e := w.m.manager.SetInt32(&w.m.D.Data[2], -19); e != nil {
			return 0, e
		}
	}
	return w.Buffer.Write(p)
}
func (w *lpRationalDualDiagnosticWriter) Flush() error { w.flushes++; return w.flushErr }
func TestLPRationalDualAdvanceDiagnosticWriterPrefixes(t *testing.T) {
	want := "total_iterations = 9\nd[2] = 2 but should be 1\n"
	for limit := 0; limit <= len(want)+1; limit++ {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		m.TotalIterations = 9
		h.must(h.m.SetInt32(&m.D.Data[2], 2))
		w := &lpResidualFailWriter{remaining: limit}
		f.settings.DebugOut = lpResidualWriterOnly{writer: w}
		correct, e := m.DIsCorrect()
		if limit < len(want) {
			if e != lpErrResidualWrite {
				t.Fatal(limit, e)
			}
		} else {
			h.must(e)
			if correct {
				t.Fatal("missed mismatch")
			}
		}
		n := limit
		if n > len(want) {
			n = len(want)
		}
		if w.String() != want[:n] {
			t.Fatal(limit, w.String(), want[:n])
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, failFlush := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		m.TotalIterations = 9
		h.must(h.m.SetInt32(&m.D.Data[2], 2))
		w := &lpRationalDualDiagnosticWriter{m: m, mutate: true}
		if failFlush {
			w.flushErr = lpErrResidualWrite
		}
		f.settings.DebugOut = w
		correct, e := m.DIsCorrect()
		if failFlush {
			if e != lpErrResidualWrite || w.flushes != 1 || w.String() != "total_iterations = 9\n" {
				t.Fatal(e, w.flushes, w.String())
			}
		} else {
			h.must(e)
			if correct || w.flushes != 2 || w.String() != "total_iterations = 9\nd[2] = -19 but should be 1\n" {
				t.Fatal(correct, w.flushes, w.String())
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	// Revert's output precedes basis publication; any writer refusal leaves the
	// current exchanged basis intact.
	message := "revert to previous basis on ( 0, 1)\n"
	for limit := 0; limit < len(message); limit++ {
		h := newLPPairHarness(t)
		f, m := lpRationalDualAdvanceFixture(t, h)
		h.must(m.ChangeBasis(1, 0))
		m.Q = 1
		w := &lpResidualFailWriter{remaining: limit}
		f.settings.DebugOut = lpResidualWriterOnly{writer: w}
		e := m.RevertToPreviousBasis()
		if e != lpErrResidualWrite || f.basis[0] != 1 {
			t.Fatal(limit, e, f.basis)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalDualAdvanceLargeFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = StrategyLU
	huge := strings.Repeat("1234567890", 10)
	a := lpScalarBoundValue(h, huge+"1/17")
	negative := lpScalarBoundValue(h, "-"+huge+"1/17")
	h.must(f.a.Set(0, 0, a))
	h.must(f.a.Set(0, 1, negative))
	h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "1")))
	factor := lpScalarBoundValue(h, "-"+huge+"3/19")
	h.must(h.m.Mul(a, factor, &f.b.Data[0]))
	for i := range f.types {
		f.types[i] = ColumnLowBound
	}
	h.must(h.m.SetInt32(&f.costs.Data[1], 1))
	h.must(h.m.SetInt32(&f.costs.Data[2], 1))
	can := []bool{true, true, true}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.RowsNZ = make([]uint32, 1)
	m.ColumnsNZ = make([]uint32, 3)
	m.P, m.R = 0, 0
	h.must(m.RecalculateD())
	h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
	h.must(m.GetDelta(&m.Delta))
	return f, m
}
func lpRationalDualAdvanceOperation(m *RationalDualCoreSolver, op string) error {
	switch op {
	case "advance":
		_, e := m.AdvanceOnKnownP()
		return e
	case "basis":
		_, e := m.BasisChangeAndUpdate()
		return e
	case "revert":
		return m.RevertToPreviousBasis()
	case "recover":
		return m.RecoverLeaving()
	case "snap":
		_, e := m.SnapRunawayNonBasicColumn(1)
		return e
	case "cost":
		_, e := m.DIsCorrect()
		return e
	}
	panic(op)
}
func lpRationalDualAdvanceOperationFixture(t *testing.T, h *lpPairHarness, op string) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f, m := lpRationalDualAdvanceLargeFixture(t, h)
	switch op {
	case "basis":
		// Keep a live consistent pivot with a large reciprocal, so checked
		// builds can physically refuse before the source beta assertion.
		v, e := f.a.ColumnValue(f.a.Columns[1][0])
		h.must(e)
		h.must(h.m.SetInt32(v, -1))
		lpRationalDualPrepareAdvance(t, h, m)
	case "revert":
		m.Q = 1
		m.EnteringBoundaryPosition = PositionAtLowBound
		h.must(m.ChangeBasis(1, 0))
	case "recover":
		m.Q = 1
		m.EnteringBoundaryPosition = PositionAtUpperBound
		h.must(h.m.SetString(&f.up.Data[1], strings.Repeat("1234567890", 10)+"7/23"))
	case "snap":
		f.types[1] = ColumnBoxed
		h.must(h.m.SetString(&f.low.Data[1], "-"+strings.Repeat("1234567890", 10)+"7/23"))
		h.must(h.m.SetString(&f.up.Data[1], strings.Repeat("1234567890", 10)+"9/29"))
		h.must(h.m.SetInt32(&f.x.Data[1], 1))
	case "cost":
		h.must(h.m.SetString(&m.D.Data[2], strings.Repeat("1234567890", 10)+"7/31"))
	}
	return f, m
}
func TestLPRationalDualAdvancePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"advance", "basis", "revert", "recover", "snap", "cost"} {
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
					f, m := lpRationalDualAdvanceOperationFixture(t, h, op)
					h.budget.FailAfter(ordinal)
					e := lpRationalDualAdvanceOperation(m, op)
					h.budget.FailAfter(-1)
					if e == nil || sourceDebugEnabled && (op == "advance" || op == "basis") && e == ErrAssertion {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if retry := lpRationalDualAdvanceOperation(m, op); retry != ErrExactVectorPending {
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
func TestLPRationalDualAdvancePreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"advance", "basis", "revert", "recover", "snap", "cost"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualAdvanceOperationFixture(t, h, op)
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
				_ = lpRationalDualAdvanceOperation(m, op)
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

func TestLPRationalDualAdvanceMetadataAndStoppedPrefixes(t *testing.T) {
	for _, phase := range []string{"snap-type", "snap-low", "snap-upper", "recover-x", "recover-low", "revert-heading", "cost-lu", "cost-local", "cost-costs", "cost-d", "cost-matrix", "advance-r", "advance-rows", "advance-columns", "basis-p", "basis-q", "basis-r", "basis-d", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalDualAdvanceFixture(t, h)
			xs, low, up, costs, types, heading, ds := f.x.Data, f.low.Data, f.up.Data, f.costs.Data, f.types, f.heading, m.D.Data
			lu := m.Factorization
			col := append([]ColumnCell(nil), f.a.Columns[2]...)
			var e error
			want := error(ErrBounds)
			switch phase {
			case "snap-type":
				f.types = f.types[:1]
				_, e = m.SnapRunawayNonBasicColumn(1)
			case "snap-low":
				f.low.Data = f.low.Data[:1]
				_, e = m.SnapRunawayNonBasicColumn(1)
			case "snap-upper":
				f.types[1] = ColumnBoxed
				f.up.Data = f.up.Data[:1]
				_, e = m.SnapRunawayNonBasicColumn(1)
			case "recover-x":
				m.Q = 1
				m.EnteringBoundaryPosition = PositionAtLowBound
				f.x.Data = f.x.Data[:1]
				e = m.RecoverLeaving()
			case "recover-low":
				m.Q = 1
				m.EnteringBoundaryPosition = PositionAtLowBound
				f.low.Data = f.low.Data[:1]
				e = m.RecoverLeaving()
			case "revert-heading":
				m.Q = 1
				h.must(m.ChangeBasis(1, 0))
				f.heading = f.heading[:1]
				e = m.RevertToPreviousBasis()
			case "cost-lu":
				m.Factorization = nil
				_, e = m.DIsCorrect()
			case "cost-local":
				m.diagnosticValues = NewIndexedRationalVector(h.m, 1)
				_, e = m.DIsCorrect()
				want = ErrExactVectorPending
			case "cost-costs":
				f.costs.Data = f.costs.Data[:1]
				_, e = m.DIsCorrect()
			case "cost-d":
				m.D.Data = m.D.Data[:1]
				_, e = m.DIsCorrect()
				lpSparseRationalCheck(t, h, &m.diagnosticValues.Data[1], big.NewRat(1, 1))
			case "cost-matrix":
				f.a.Columns[2][0].Offset = 999
				_, e = m.DIsCorrect()
			case "advance-r":
				m.R = -1
				_, e = m.AdvanceOnKnownP()
			case "advance-rows":
				m.RowsNZ = nil
				_, e = m.AdvanceOnKnownP()
			case "advance-columns":
				m.ColumnsNZ = nil
				_, e = m.AdvanceOnKnownP()
			case "basis-p", "basis-q", "basis-r", "basis-d":
				lpRationalDualPrepareAdvance(t, h, m)
				switch phase {
				case "basis-p":
					m.P = -1
				case "basis-q":
					m.Q = -1
				case "basis-r":
					m.R = -1
				case "basis-d":
					m.D.Data = m.D.Data[:1]
				}
				_, e = m.BasisChangeAndUpdate()
				if sourceDebugEnabled && phase != "basis-q" {
					want = ErrAssertion
				}
			case "stopped":
				m.stopped = true
				_, e = m.DIsCorrect()
				want = ErrExactVectorPending
			}
			if e != want {
				t.Fatal(phase, e, want)
			}
			f.x.Data, f.low.Data, f.up.Data, f.costs.Data, f.types, f.heading, m.D.Data = xs, low, up, costs, types, heading, ds
			m.Factorization = lu
			f.a.Columns[2] = col
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualAdvanceUpperLeavingSnapAndReconstruct(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 1, 2)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "2")))
	h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "2")))
	h.must(h.m.SetInt32(&f.b.Data[0], 6))
	f.types[0] = ColumnUpperBound
	f.types[1] = ColumnLowBound
	can := []bool{true, true}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.RowsNZ = make([]uint32, 1)
	m.ColumnsNZ = make([]uint32, 2)
	m.P, m.R = 0, 0
	h.must(m.RecalculateD())
	h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
	h.must(m.GetDelta(&m.Delta))
	// Source upper-only GetDelta is the squared edge value, so the initial
	// step overshoots. Native snaps the leaving value and reconstructs basics.
	lpSparseRationalCheck(t, h, &m.Delta, big.NewRat(9, 1))
	ok, e := m.AdvanceOnKnownP()
	if sourceDebugEnabled {
		if ok || e != ErrAssertion {
			t.Fatal(ok, e)
		}
		lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(3, 1))
	} else {
		h.must(e)
		if !ok || f.basis[0] != 1 {
			t.Fatal(ok, f.basis)
		}
		lpSparseRationalCheck(t, h, &m.ThetaP, big.NewRat(9, 1))
		lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
		lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(3, 1))
		off, e := m.AMultXIsOff()
		h.must(e)
		if off {
			t.Fatal("snap did not reconstruct")
		}
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
func TestLPRationalDualAdvanceActualSingularFallback(t *testing.T) {
	for _, onlyFeasible := range []bool{false, true} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 3)
		f.settings.SimplexStrategy = StrategyLU
		h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
		h.must(f.a.Set(1, 1, lpScalarBoundValue(h, "1")))
		h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "-1")))
		h.must(f.a.Set(1, 2, lpScalarBoundValue(h, "-1")))
		h.must(h.m.SetInt32(&f.b.Data[0], -1))
		h.must(h.m.SetInt32(&f.b.Data[1], 2))
		h.must(h.m.SetInt32(&f.costs.Data[2], 1))
		for i := range f.types {
			f.types[i] = ColumnLowBound
		}
		can := []bool{true, true, true}
		m, e := NewRationalDualCoreSolver(f.input(), &can)
		h.must(e)
		m.RowsNZ = make([]uint32, 2)
		m.ColumnsNZ = make([]uint32, 3)
		m.P, m.R = 0, 0
		h.must(m.RecalculateD())
		h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
		h.must(m.GetDelta(&m.Delta))
		lpRationalDualPrepareAdvance(t, h, m)
		for i := 0; i < 2; i++ {
			h.must(h.m.Copy(&m.CopyXB.Data[i], &f.x.Data[i]))
		}
		// A changed unused basis column leaves the pending entering direction
		// live but makes the proposed basis singular. The old basis stays valid;
		// its saved point satisfies the changed matrix and RHS.
		h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
		h.must(h.m.SetInt32(&f.b.Data[0], 1))
		m.Factorization.RefactorCounter = 200
		m.LookForFeasibleSolutionOnly = onlyFeasible
		ok, e := m.BasisChangeAndUpdate()
		if sourceDebugEnabled {
			if ok || e != ErrAssertion {
				t.Fatal(ok, e)
			}
		} else {
			h.must(e)
			if ok || m.Status != StatusUnstable || m.IterationsNoCostGrowth != 1 || !reflect.DeepEqual(f.basis, []uint32{0, 1}) {
				t.Fatal(ok, m.Status, m.IterationsNoCostGrowth, f.basis)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-1, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(2, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[2], new(big.Rat))
			lpSparseRationalCheck(t, h, &m.Betas.Data[0], big.NewRat(2, 1))
			lpSparseRationalCheck(t, h, &m.Betas.Data[1], big.NewRat(1, 1))
			off, e := m.AMultXIsOff()
			h.must(e)
			if off {
				t.Fatal("fallback did not restore point")
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
