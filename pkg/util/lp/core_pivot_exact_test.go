package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLPPairPivotIndependentBasisRows(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(287431, 437))
	value := h.pair("0", "0")
	for world := 0; world < 100; world++ {
		n := 1 + rng.IntN(5)
		f := lpNewPairCoreFixture(t, h, uint32(n), uint32(n+3))
		f.settings.SimplexStrategy = StrategyLU
		perm := rng.Perm(n + 3)
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		b := make([][]*big.Rat, n)
		for i := 0; i < n; i++ {
			b[i] = make([]*big.Rat, n)
			for j := 0; j < n; j++ {
				b[i][j] = new(big.Rat)
				if j <= i {
					b[i][j].SetFrac64(int64(rng.IntN(11)-5), int64(rng.IntN(7)+1))
				}
				if i == j {
					b[i][j].SetFrac64(int64(i+2), int64(rng.IntN(7)+1))
				}
				h.must(h.m.SetString(&value.X, b[i][j].RatString()))
				h.must(f.a.Set(uint32(i), f.basis[j], &value.X))
			}
		}
		// Choose D independently, then construct nonbasic columns as B*D.
		// Each pivot row must recover the corresponding row of D exactly.
		d := make([][]*big.Rat, n)
		for i := range d {
			d[i] = make([]*big.Rat, 3)
			for j := range d[i] {
				d[i][j] = big.NewRat(int64(rng.IntN(9)-4), int64(rng.IntN(11)+1))
			}
		}
		for i := 0; i < n; i++ {
			for j := 0; j < 3; j++ {
				a := new(big.Rat)
				for k := 0; k < n; k++ {
					a.Add(a, new(big.Rat).Mul(b[i][k], d[k][j]))
				}
				h.must(h.m.SetString(&value.X, a.RatString()))
				h.must(f.a.Set(uint32(i), uint32(perm[n+j]), &value.X))
			}
		}
		m, err := NewPairCoreSolverBase(f.input())
		h.must(err)
		m.RowsNZ = make([]uint32, n)
		for row := 0; row < n; row++ {
			h.must(m.CalculatePivotRowOfBInverse(uint32(row)))
			y := make([]*big.Rat, n)
			for j := n - 1; j >= 0; j-- {
				y[j] = new(big.Rat)
				if j == row {
					y[j].SetInt64(1)
				}
				for i := j + 1; i < n; i++ {
					y[j].Sub(y[j], new(big.Rat).Mul(y[i], b[i][j]))
				}
				y[j].Quo(y[j], b[j][j])
				lpSparseRationalCheck(t, h, &m.PivotRowB1.Data[j], y[j])
			}
			h.must(m.CalculatePivotRowWhenInverseReady(uint32(row)))
			nz := uint32(0)
			for j := 0; j < n+3; j++ {
				want := new(big.Rat)
				for k := 0; k < 3; k++ {
					if perm[n+k] == j {
						want = d[row][k]
					}
				}
				lpSparseRationalCheck(t, h, &m.PivotRow.Data[j], want)
				if want.Sign() != 0 {
					nz++
				}
			}
			if m.RowsNZ[row] != nz || m.PivotRow.Size() != nz || m.pivotWork != nil {
				t.Fatal("exact cancellation/membership", world, row, m.RowsNZ, m.PivotRow.Index)
			}
		}
		h.must(m.Close())
		f.close()
	}
	lpExactVectorNamedCensus(t, h, nil)
}

// Build the real solver's prepared LU state without terms. Source prefix_r
// prepares these fields before row scheduling; it is not part of calculation.
func lpPreparedLARPivot(t *testing.T, h *lpPairHarness, a0, a1, basis string, externalZero ...bool) *LARSolver {
	t.Helper()
	pair := h.pair
	if len(a0) > 256 || len(a1) > 256 || len(basis) > 256 {
		pair = h.pairExactLarge
	}
	s, err := NewLARSolver(h.m)
	h.must(err)
	s.Settings.SimplexStrategy = StrategyLU
	external := []uint32{31, 67, 97}
	if len(externalZero) != 0 && externalZero[0] {
		external[0] = 0
	}
	for _, ext := range external {
		_, err = s.AddVar(ext)
		h.must(err)
	}
	h.must(s.Core.RA.AddRow())
	v := pair(a0, a1)
	h.must(s.Core.RA.Set(0, 0, &v.X))
	h.must(s.Core.RA.Set(0, 1, &v.Y))
	h.must(s.Core.RA.Set(0, 2, &pair(basis, "0").X))
	s.Core.RBasis = []uint32{2}
	m := s.Core.RPrimal
	h.must(m.InitBasisHeadingAndNonBasicColumns())
	m.RowsNZ = make([]uint32, 1)
	h.must(InitPairFactorization(&m.Factorization, s.Core.RA, s.Core.RBasis, s.Settings))
	lo, hi := h.pair("-3", "2"), h.pair("7", "11")
	for j := uint32(0); j < 2; j++ {
		low, high := &lo.X, &hi.X
		if j == 1 {
			low, high = &lo.Y, &hi.Y
		}
		_, err = s.AddVarBound(j, LConstraintGE, low)
		h.must(err)
		kind := LConstraintLE
		if j == 1 {
			kind = LConstraintLT
		}
		_, err = s.AddVarBound(j, kind, high)
		h.must(err)
	}
	return s
}

func TestLARPivotActualRowScheduling(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpPreparedLARPivot(t, h, "-2", "-3", "1")
	p := s.NewBoundPropagator()
	h.must(s.CalculateImpliedBoundsForRow(0, p))
	if len(p.FoundBounds) != 2 || len(s.Core.RPrimal.PivotRow.Index) != 2 || s.Core.RPrimal.RowsNZ[0] != 2 {
		t.Fatal("actual calculated row dispatch")
	}
	for _, b := range p.FoundBounds {
		want := int64(47)
		if b.IsLowBound {
			want = 0
		}
		if b.Column != 2 || b.Strict == b.IsLowBound || b.RowOrTermIndex != 0 {
			t.Fatal("calculated row evidence", b)
		}
		lpSparseRationalCheck(t, h, &b.Bound, big.NewRat(want, 1))
	}
	h.must(s.CalculateImpliedBoundsForRow(0, p))
	if len(p.FoundBounds) != 2 {
		t.Fatal("repeat calculated row")
	}
	h.must(p.Close())
	h.must(s.Close())
	// Tableau scheduling consumes the actual matrix and bypasses LU/substitution.
	s, term := lpDispatchModel(t, h, false, "2", "3")
	p = s.NewBoundPropagator()
	h.must(s.CalculateImpliedBoundsForRow(0, p))
	if len(p.FoundBounds) != 2 || p.FoundBounds[0].Column != term {
		t.Fatal("tableau branch")
	}
	h.must(s.Close())
	h.must(p.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARPivotTermSubstitutionSourceOffset(t *testing.T) {
	h := newLPPairHarness(t)
	for _, skip := range []bool{false, true} {
		// A real external variable zero collides with the literal term offset.
		s := lpPreparedLARPivot(t, h, "-2", "-3", "1", skip)
		v := h.pair("4", "5")
		input := NewSparseRationalVector(h.m)
		h.must(input.Append(2, &v.X))
		h.must(input.Append(0, &v.Y))
		term, err := s.AddTerm(input.Data, &h.pair("7", "0").X)
		h.must(err)
		used, err := s.TermIsUsedAsRow(term)
		h.must(err)
		if used {
			t.Fatal("unregistered term is a row")
		}
		p := s.NewBoundPropagator()
		err = s.CalculateImpliedBoundsForRow(0, p)
		if sourceDebugEnabled {
			if !errors.Is(err, ErrAssertion) || len(p.FoundBounds) != 0 || !s.Terms[0].Contains(2) {
				t.Fatal("source offset assertion must precede substitution/analysis", err)
			}
		} else {
			h.must(err)
			if s.Terms[0].Contains(2) != skip {
				t.Fatal("source literal offset lookup", skip)
			}
			if !skip {
				for j, want := range map[uint32]int64{0: 13, 1: 12} {
					coefficient, ok := s.Terms[0].Coefficients().Get2(j)
					if !ok {
						t.Fatal("missing substituted coefficient")
					}
					lpSparseRationalCheck(t, h, coefficient, big.NewRat(want, 1))
				}
			}
		}
		if !s.OriginalTerms[0].Contains(2) {
			t.Fatal("substitution mutated original term")
		}
		lpSparseRationalCheck(t, h, &s.Terms[0].Value, big.NewRat(7, 1))
		h.must(s.Close())
		h.must(p.Close())
		h.must(input.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARPivotPhysicalFailurePrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	large := "1" + strings.Repeat("0", 999) + "7"
	basis := "3" + strings.Repeat("0", 999) + "1"
	for _, direct := range []bool{false, true} {
		operations := []string{"inverse", "accumulate", "schedule"}
		if !sourceDebugEnabled {
			operations = append(operations, "substitute")
		}
		for _, operation := range operations {
			failures, published := 0, 0
			success := false
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%s/%d", direct, operation, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpPreparedLARPivot(t, h, "-"+large+"/19", large+"/23", basis)
					m := s.Core.RPrimal.PairCoreSolverBase
					if operation != "schedule" {
						h.must(s.Core.CalculatePivotRow(0))
					}
					if operation == "substitute" {
						input := NewSparseRationalVector(h.m)
						v := h.pairExactLarge(large+"/29", large+"/31")
						h.must(input.Append(2, &v.X))
						h.must(input.Append(0, &v.Y))
						_, err := s.AddTerm(input.Data, &h.pair("7", "0").X)
						h.must(err)
						h.must(input.Close())
					}
					p := s.NewBoundPropagator()
					h.budget.FailAfter(ordinal)
					var err error
					if operation == "inverse" {
						err = m.CalculatePivotRowOfBInverse(0)
					} else if operation == "accumulate" {
						err = m.CalculatePivotRowWhenInverseReady(0)
					} else if operation == "substitute" {
						err = s.SubstituteBasisVarInTermsForRow(0)
					} else {
						err = s.CalculateImpliedBoundsForRow(0, p)
					}
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if len(p.FoundBounds) > 0 {
							published++
						}
						raw, div, cmp := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
						var again error
						if operation == "schedule" {
							again = s.CalculateImpliedBoundsForRow(0, p)
						} else if operation == "substitute" {
							again = s.SubstituteBasisVarInTermsForRow(0)
						} else {
							again = m.CalculatePivotRowOfBInverse(0)
						}
						if !errors.Is(again, ErrExactVectorPending) || raw != h.m.RawInvocationState() || div != h.m.DivisionInvocationState() || cmp != h.m.ComparisonInvocationState() {
							t.Fatal("stopped pivot owner changed numeric calls", again)
						}
						if s.calls.pending() || operation == "inverse" && m.calls.pending() {
							t.Fatal("outer owner adopted child receipts")
						}
					} else {
						done = true
					}
					h.must(s.Close())
					h.must(p.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					success = true
					break
				}
			}
			if !success || failures == 0 || operation == "schedule" && published == 0 {
				t.Fatal(direct, operation, success, failures, published)
			}
		}
	}
}

func TestLARPivotSourceBranchesAndMetadataGuards(t *testing.T) {
	h := newLPPairHarness(t)
	s, _ := lpDispatchModel(t, h, false, "2", "3")
	v := s.Core.RPrimal.PivotRow
	h.must(v.Resize(3))
	h.must(v.SetValue(&h.pair("17", "0").X, 0))
	err := s.Core.CalculatePivotRow(0)
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) || len(v.Index) != 1 {
			t.Fatal("outer tableau assertion must precede scratch clearing", err)
		}
		lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(17, 1))
	} else {
		h.must(err)
		if len(v.Index) != 2 || s.Core.RPrimal.PivotRowB1.Size() != 0 {
			t.Fatal("native direct tableau row path")
		}
		for j, want := range []int64{-2, -3, 0} {
			lpSparseRationalCheck(t, h, &v.Data[j], big.NewRat(want, 1))
		}
	}
	h.must(s.Close())

	for _, mode := range []string{"zero", "bad-index", "bad-matrix-row", "bad-heading", "bad-count"} {
		s = lpPreparedLARPivot(t, h, "-2", "-3", "1")
		h.must(s.Core.CalculatePivotRow(0))
		m := s.Core.RPrimal.PairCoreSolverBase
		switch mode {
		case "zero":
			h.must(m.PivotRowB1.Clear())
			h.must(m.PivotRowB1.SetValue(&h.pair("0", "0").X, 0))
		case "bad-index":
			m.PivotRowB1.Index = []uint32{3}
		case "bad-matrix-row":
			h.must(m.PivotRowB1.Resize(2))
			h.must(m.PivotRowB1.SetValue(&h.pair("1", "0").X, 1))
		case "bad-heading":
			*m.Heading = nil
		case "bad-count":
			m.RowsNZ = nil
		}
		err = m.CalculatePivotRowWhenInverseReady(0)
		if mode == "zero" {
			h.must(err)
			if m.RowsNZ[0] != 0 || m.PivotRow.Size() != 0 {
				t.Fatal("indexed zero inverse entry must be skipped")
			}
		} else {
			if !errors.Is(err, ErrBounds) || m.pivotWork == nil {
				t.Fatal(mode, "metadata failure prefix", err)
			}
			if mode == "bad-count" && m.PivotRow.Size() != 2 {
				t.Fatal("RowsNZ failure must follow coefficient publication")
			}
			if e := m.CalculatePivotRowWhenInverseReady(0); !errors.Is(e, ErrExactVectorPending) {
				t.Fatal("stopped accumulation", e)
			}
		}
		h.must(s.Close())
	}

	for _, mode := range []string{"missing-lu", "invalid-row", "tableau"} {
		s = lpPreparedLARPivot(t, h, "-2", "-3", "1")
		h.must(s.Core.CalculatePivotRow(0))
		m := s.Core.RPrimal.PairCoreSolverBase
		row := uint32(0)
		if mode == "missing-lu" {
			h.must(m.Factorization.Close())
			m.Factorization = nil
		} else if mode == "invalid-row" {
			row = 99
		} else {
			s.Settings.SimplexStrategy = StrategyTableauRows
		}
		err = m.CalculatePivotRowOfBInverse(row)
		want := ErrBounds
		if mode == "tableau" {
			if sourceDebugEnabled {
				want = ErrAssertion
			} else {
				want = nil
			}
		}
		if !errors.Is(err, want) {
			t.Fatal(mode, err, want)
		}
		if err != nil {
			if e := s.Core.CalculatePivotRow(0); !errors.Is(e, ErrExactVectorPending) {
				t.Fatal("failed precise core accepted", e)
			}
		}
		h.must(s.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARPivotAdditionalEntryAndClearGuards(t *testing.T) {
	h := newLPPairHarness(t)
	for _, mode := range []string{"inverse-clear", "pivot-clear", "outer-inverse-clear", "outer-pivot-clear", "substitute-range"} {
		s := lpPreparedLARPivot(t, h, "-2", "-3", "1")
		h.must(s.Core.CalculatePivotRow(0))
		m := s.Core.RPrimal.PairCoreSolverBase
		var err, again error
		switch mode {
		case "inverse-clear", "outer-inverse-clear":
			m.PivotRowB1.Index = []uint32{99}
		case "pivot-clear", "outer-pivot-clear":
			m.PivotRow.Index = []uint32{99}
		}
		// Explicit public-vector metadata errors exercise bounds guards,
		// not allocation refusals or ordinary solver states.
		switch mode {
		case "inverse-clear":
			err = m.CalculatePivotRowOfBInverse(0)
			again = m.CalculatePivotRowOfBInverse(0)
		case "pivot-clear":
			err = m.CalculatePivotRowWhenInverseReady(0)
			again = m.CalculatePivotRowWhenInverseReady(0)
		case "substitute-range":
			err = s.SubstituteBasisVarInTermsForRow(99)
			again = s.SubstituteBasisVarInTermsForRow(0)
		default:
			err = s.Core.CalculatePivotRow(0)
			again = s.Core.CalculatePivotRow(0)
		}
		if !errors.Is(err, ErrBounds) || !errors.Is(again, ErrExactVectorPending) {
			t.Fatal(mode, err, again)
		}
		h.must(s.Close())
	}
	s, _ := lpDispatchModel(t, h, false, "2", "3")
	err := s.Core.CalculatePivotRow(99)
	want := ErrBounds
	if sourceDebugEnabled {
		want = ErrAssertion
	}
	if !errors.Is(err, want) {
		t.Fatal("tableau row guard", err)
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARPivotNativeTableauPhysicalCopies(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	if sourceDebugEnabled {
		// The preceding source assertion rejects this branch in debug.
		return
	}
	large := "1" + strings.Repeat("0", 999) + "7"
	for _, direct := range []bool{false, true} {
		failures, success := 0, false
		for ordinal := int64(0); ordinal < 100; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%v/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				s, _ := lpDispatchModel(t, h, false, large, "-"+large)
				h.budget.FailAfter(ordinal)
				err := s.Core.CalculatePivotRow(0)
				h.budget.FailAfter(-1)
				if err != nil {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if e := s.Core.CalculatePivotRow(0); !errors.Is(e, ErrExactVectorPending) {
						t.Fatal("stopped native tableau copy", e)
					}
				} else {
					done = true
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				success = true
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("native tableau campaign", direct, success, failures)
		}
	}
}
