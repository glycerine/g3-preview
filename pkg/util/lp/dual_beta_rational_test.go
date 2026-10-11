// Source: src/util/lp/lp_dual_core_solver.hpp:207-210,357-375,411-431.
package lp

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
)

func lpRationalDualBetaFixture(t *testing.T, h *lpPairHarness, n uint32, large bool) (*lpRationalCoreFixture, *RationalDualCoreSolver, [][]*big.Rat) {
	return lpRationalDualBetaTriangularFixture(t, h, n, large, false)
}

func lpRationalDualBetaTriangularFixture(t *testing.T, h *lpPairHarness, n uint32, large, upper bool) (*lpRationalCoreFixture, *RationalDualCoreSolver, [][]*big.Rat) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, n, n+1)
	f.settings.SimplexStrategy = StrategyLU
	b := make([][]*big.Rat, n)
	for i := uint32(0); i < n; i++ {
		b[i] = make([]*big.Rat, n)
		for j := uint32(0); j < n; j++ {
			v := new(big.Rat)
			if i == j {
				v.SetFrac64(int64(i+2), int64(i+3))
			}
			if (!upper && j < i) || (upper && i < j) {
				v.SetFrac64(int64(i+j+1), 17)
				if large {
					v.SetFrac(new(big.Int).Lsh(big.NewInt(int64(i+j+1)), 100), big.NewInt(19))
				}
			}
			b[i][j] = v
			h.must(f.a.Set(i, j, lpScalarBoundValue(h, v.RatString())))
		}
	}
	can := make([]bool, n+1)
	can[n] = true
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.Q = int32(n)
	m.R = int32(n) - 1
	for i := uint32(0); i < n; i++ {
		h.must(h.m.SetInt32(&m.Betas.Data[i], int32(i+1)))
		h.must(m.Ed.SetValue(lpScalarBoundValue(h, fmt.Sprint(i+2)), i))
		h.must(m.PivotRowB1.SetValue(lpScalarBoundValue(h, fmt.Sprint(i+3)), i))
	}
	h.must(h.m.Copy(&m.PivotRow.Data[n], lpScalarBoundValue(h, "2/3")))
	return f, m, b
}
func lpIndependentInverseRow(b [][]*big.Rat, r int) []*big.Rat {
	n := len(b)
	v := make([]*big.Rat, n)
	for j := n - 1; j >= 0; j-- {
		z := new(big.Rat)
		if j == r {
			z.SetInt64(1)
		}
		for i := j + 1; i < n; i++ {
			z.Sub(z, new(big.Rat).Mul(v[i], b[i][j]))
		}
		v[j] = z.Quo(z, b[j][j])
	}
	return v
}
func TestLPRationalDualBetaIndependentReinitialization(t *testing.T) {
	for world := 0; world < 100; world++ {
		h := newLPPairHarness(t)
		n := uint32(1 + world%5)
		f, m, b := lpRationalDualBetaFixture(t, h, n, world%2 == 0)
		expected := make([]*big.Rat, n)
		for r := 0; r < int(n); r++ {
			v := lpIndependentInverseRow(b, r)
			norm := new(big.Rat)
			for _, x := range v {
				norm.Add(norm, new(big.Rat).Mul(x, x))
			}
			expected[r] = norm
			h.must(m.InitBetaPrecisely(uint32(r)))
			lpSparseRationalCheck(t, h, &m.Betas.Data[r], norm)
		}
		for i := range m.Betas.Data {
			h.must(h.m.SetInt32(&m.Betas.Data[i], -97))
		}
		h.must(m.InitBetasPrecisely())
		for i := range expected {
			lpSparseRationalCheck(t, h, &m.Betas.Data[i], expected[i])
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualBetaIndependentRecurrence(t *testing.T) {
	rng := rand.New(rand.NewSource(757132))
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		n := uint32(1 + world%5)
		f, m, _ := lpRationalDualBetaFixture(t, h, n, false)
		m.R = int32(world % int(n))
		pivot := big.NewRat(int64(1+rng.Intn(19)), int64(1+rng.Intn(17)))
		if world%2 == 0 {
			pivot.Neg(pivot)
		}
		if world%7 == 0 {
			pivot.SetFrac(new(big.Int).Lsh(big.NewInt(int64(world+1)), uint(90+world%17)), big.NewInt(29))
		}
		h.must(h.m.Copy(&m.PivotRow.Data[n], lpScalarBoundValue(h, pivot.RatString())))
		old, ed, row := make([]*big.Rat, n), make([]*big.Rat, n), make([]*big.Rat, n)
		for i := uint32(0); i < n; i++ {
			old[i] = big.NewRat(int64(rng.Intn(19)-9), 37)
			ed[i] = big.NewRat(int64(rng.Intn(31)-15), 41)
			row[i] = big.NewRat(int64(rng.Intn(31)-15), 43)
			h.must(h.m.Copy(&m.Betas.Data[i], lpScalarBoundValue(h, old[i].RatString())))
			h.must(h.m.Copy(&m.Ed.Data[i], lpScalarBoundValue(h, ed[i].RatString())))
			h.must(h.m.Copy(&m.PivotRowB1.Data[i], lpScalarBoundValue(h, row[i].RatString())))
		}
		inv := new(big.Rat).Inv(pivot)
		selected := new(big.Rat).Mul(old[m.R], new(big.Rat).Mul(inv, inv))
		if selected.Sign() < 0 {
			selected.SetInt64(0)
		}
		k := new(big.Rat).Mul(big.NewRat(-2, 1), inv)
		e := m.UpdateBetas()
		if sourceDebugEnabled {
			if e != ErrAssertion {
				t.Fatal(world, e)
			}
			for i := range old {
				lpSparseRationalCheck(t, h, &m.Betas.Data[i], old[i])
			}
			// The copied reciprocal precedes the source double-constructor assertion.
			if m.priceWork == nil {
				t.Fatal("missing reciprocal owner")
			}
			lpSparseRationalCheck(t, h, &m.priceWork.Data[2], inv)
		} else {
			h.must(e)
			for i := range old {
				want := new(big.Rat).Set(selected)
				if int32(i) != m.R {
					term := new(big.Rat).Mul(ed[i], new(big.Rat).Add(new(big.Rat).Mul(ed[i], selected), new(big.Rat).Mul(k, row[i])))
					want.Add(old[i], term)
					if want.Sign() < 0 {
						want.SetInt64(0)
					}
				}
				lpSparseRationalCheck(t, h, &m.Betas.Data[i], want)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualBetaPreparedDiagonalStage(t *testing.T) {
	for world := 0; world < 100; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 3, 4)
		f.settings.SimplexStrategy = StrategyLU
		f.basis = []uint32{2, 0, 1}
		diag := []*big.Rat{big.NewRat(2, 3), big.NewRat(-5, 7), big.NewRat(11, 13)}
		for i, j := range f.basis {
			h.must(f.a.Set(uint32(i), j, lpScalarBoundValue(h, diag[i].RatString())))
		}
		can := []bool{false, false, false, true}
		m, e := NewRationalDualCoreSolver(f.input(), &can)
		h.must(e)
		input := make([]*big.Rat, 3)
		for i := 0; i < 3; i++ {
			input[i] = big.NewRat(int64(world+i-40), 17)
			if i == 2 {
				input[i].SetFrac(new(big.Int).Lsh(big.NewInt(int64(world+1)), uint(90+world%20)), big.NewInt(23))
			}
			h.must(m.PivotRowB1.SetValue(lpScalarBoundValue(h, input[i].RatString()), uint32(i)))
		}
		// Diagonal scaling is retained in LU tail operations. The source DSE
		// invokes only the prepared U/R stage. Check the unit U assumption and
		// apply the actual permutation mathematically without calling its helper.
		expected := make([]*big.Rat, 3)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				v, err := m.Factorization.U.Get(uint32(i), uint32(j))
				h.must(err)
				want := new(big.Rat)
				if i == j {
					want.SetInt64(1)
				}
				lpSparseRationalCheck(t, h, v, want)
			}
			expected[m.Factorization.R.Values()[i]] = input[i]
		}
		h.must(m.DSEFTran())
		for i := 0; i < 3; i++ {
			lpSparseRationalCheck(t, h, &m.PivotRowB1.Data[i], expected[i])
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func lpRationalDualBetaOperation(m *RationalDualCoreSolver, op string) error {
	switch op {
	case "init":
		return m.InitBetaPrecisely(1)
	case "all":
		return m.InitBetasPrecisely()
	case "dse":
		return m.DSEFTran()
	case "update":
		return m.UpdateBetas()
	}
	panic(op)
}
func TestLPRationalDualBetaPhysicalOwnership(t *testing.T) {
	for _, op := range []string{"init", "all", "dse", "update"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			completedDescendingPrefix := false
			for ordinal := int64(0); ordinal < 3000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m, basis := lpRationalDualBetaTriangularFixture(t, h, 3, true, op == "all")
					huge := strings.Repeat("1234567890", 10)
					h.must(h.m.SetString(&m.PivotRow.Data[3], huge+"1/13"))
					for i := 0; i < 3; i++ {
						h.must(h.m.SetString(&m.Betas.Data[i], huge+"3/17"))
						h.must(h.m.SetString(&m.PivotRowB1.Data[i], huge+"5/19"))
						h.must(h.m.SetString(&m.Ed.Data[i], huge+"7/23"))
					}
					h.budget.FailAfter(ordinal)
					e := lpRationalDualBetaOperation(m, op)
					h.budget.FailAfter(-1)
					if e == nil || sourceDebugEnabled && op == "update" && e == ErrAssertion {
						success, done = true, true
					} else {
						failures++
						if op == "all" {
							norm := new(big.Rat)
							for _, v := range lpIndependentInverseRow(basis, 2) {
								norm.Add(norm, new(big.Rat).Mul(v, v))
							}
							last, le := h.m.String(&m.Betas.Data[2])
							if le == nil {
								lv, lok := new(big.Rat).SetString(last)
								if lok && lv.Cmp(norm) == 0 {
									completedDescendingPrefix = true
								}
							}
						}
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if retry := lpRationalDualBetaOperation(m, op); retry != ErrExactVectorPending {
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
			if op == "all" && !completedDescendingPrefix {
				t.Fatal("missing completed descending beta prefix", direct)
			}
		}
	}
}
func TestLPRationalDualBetaPreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"init", "all", "dse", "update"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m, _ := lpRationalDualBetaFixture(t, h, 3, true)
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
				_ = lpRationalDualBetaOperation(m, op)
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

func TestLPRationalDualBetaMetadataAndPrefixes(t *testing.T) {
	for _, phase := range []string{"init-index", "init-local", "init-lu", "init-destination", "all-destination", "dse-nil", "dse-closed", "dse-vector", "update-q", "update-r", "update-ed", "update-row", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m, b := lpRationalDualBetaFixture(t, h, 3, false)
			lu, betas, ed, row := m.Factorization, m.Betas.Data, m.Ed.Data, m.PivotRowB1.Data
			var e error
			want := error(ErrBounds)
			switch phase {
			case "init-index":
				e = m.InitBetaPrecisely(3)
			case "init-local":
				m.betaVector = NewIndexedRationalVector(h.m, 1)
				e = m.InitBetaPrecisely(1)
				want = ErrExactVectorPending
			case "init-lu":
				m.Factorization = nil
				e = m.InitBetaPrecisely(1)
			case "init-destination":
				m.Betas.Data = m.Betas.Data[:1]
				e = m.InitBetaPrecisely(1)
			case "all-destination":
				// The first descending destination is invalid. The completed
				// child inverse-row solve must survive this late extent failure.
				m.Betas.Data = m.Betas.Data[:2]
				e = m.InitBetasPrecisely()
				if m.betaVector == nil {
					t.Fatal("missing reached inverse-row local")
				}
				v := lpIndependentInverseRow(b, 2)
				for i := range v {
					lpSparseRationalCheck(t, h, &m.betaVector.Data[i], v[i])
				}
			case "dse-nil":
				m.Factorization = nil
				e = m.DSEFTran()
			case "dse-closed":
				h.must(m.Factorization.Close())
				e = m.DSEFTran()
				want = ErrLUClosed
			case "dse-vector":
				m.PivotRowB1.Data = m.PivotRowB1.Data[:1]
				e = m.DSEFTran()
			case "update-q":
				m.Q = -1
				e = m.UpdateBetas()
			case "update-r":
				m.R = -1
				e = m.UpdateBetas()
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "update-ed":
				m.Ed.Data = m.Ed.Data[:1]
				e = m.UpdateBetas()
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "update-row":
				m.PivotRowB1.Data = m.PivotRowB1.Data[:1]
				e = m.UpdateBetas()
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "stopped":
				m.stopped = true
				e = m.InitBetasPrecisely()
				want = ErrExactVectorPending
			}
			if e != want {
				t.Fatal(phase, e, want)
			}
			if phase == "update-ed" || phase == "update-row" {
				wantBeta := big.NewRat(27, 4)
				if sourceDebugEnabled {
					wantBeta.SetInt64(3)
				}
				lpSparseRationalCheck(t, h, &m.Betas.Data[2], wantBeta)
			}
			m.Factorization = lu
			m.Betas.Data, m.Ed.Data, m.PivotRowB1.Data = betas, ed, row
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualBetaPreparedDenseStage(t *testing.T) {
	for world := 0; world < 100; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 3, 4)
		f.settings.SimplexStrategy = StrategyLU
		for i, row := range [][]int64{{2, 1, 3}, {1, 4, -2}, {3, -1, 5}} {
			for j, v := range row {
				h.must(f.a.Set(uint32(i), uint32(j), lpScalarBoundValue(h, fmt.Sprint(v))))
			}
		}
		can := []bool{false, false, false, true}
		m, e := NewRationalDualCoreSolver(f.input(), &can)
		h.must(e)
		target := []*big.Rat{big.NewRat(int64(world-50), 17), big.NewRat(int64(world%3), 19), new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(world+1)), uint(90+world%20)), big.NewInt(23))}
		// Independently form the prepared RHS U*R*target using the actual
		// factorization's mathematical coefficients. DSE must recover target;
		// applying the full B solve here would incorrectly replay the LU tails.
		hasOffDiagonal := false
		for i := 0; i < 3; i++ {
			rhs := new(big.Rat)
			for j := 0; j < 3; j++ {
				v, e := m.Factorization.U.Get(uint32(i), uint32(j))
				h.must(e)
				text, e := h.m.String(v)
				h.must(e)
				coefficient := lpRationalTestBig(text)
				if i != j && coefficient.Sign() != 0 {
					hasOffDiagonal = true
				}
				rhs.Add(rhs, new(big.Rat).Mul(coefficient, target[m.Factorization.R.Values()[j]]))
			}
			h.must(m.PivotRowB1.SetValue(lpScalarBoundValue(h, rhs.RatString()), uint32(i)))
		}
		if !hasOffDiagonal {
			t.Fatal("fixture must reach prepared triangular arithmetic")
		}
		h.must(m.DSEFTran())
		for i := range target {
			lpSparseRationalCheck(t, h, &m.PivotRowB1.Data[i], target[i])
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualBetaReinitializationHasNoFloor(t *testing.T) {
	for _, text := range []string{"123456789012345678901234567890123456789/13", "-123456789012345678901234567890123456789/17", "13/123456789012345678901234567890123456789", "-17/123456789012345678901234567890123456789"} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 2)
		f.settings.SimplexStrategy = StrategyLU
		h.must(f.a.Set(0, 0, lpScalarBoundValue(h, text)))
		can := []bool{false, true}
		m, e := NewRationalDualCoreSolver(f.input(), &can)
		h.must(e)
		h.must(h.m.SetInt32(&m.Betas.Data[0], 97))
		h.must(m.InitBetasPrecisely())
		reciprocal := new(big.Rat).Inv(lpRationalTestBig(text))
		lpSparseRationalCheck(t, h, &m.Betas.Data[0], new(big.Rat).Mul(reciprocal, reciprocal))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
