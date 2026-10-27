// Source: src/util/lp/lp_dual_core_solver.h/.hpp; static_matrix.h:170-174.
package lp

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalDualFlipIndependentSystems(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewSource(661738))
	for world := 0; world < 180; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 3, 6)
		f.settings.SimplexStrategy = StrategyLU
		f.basis = []uint32{2, 0, 4}
		basis := make([][]*big.Rat, 3)
		for i := 0; i < 3; i++ {
			basis[i] = make([]*big.Rat, 3)
			for k := 0; k < 3; k++ {
				v := new(big.Rat)
				if i == k {
					v.SetFrac64(int64(i+2), int64(i+3))
				}
				if k < i {
					v.SetFrac(new(big.Int).Lsh(big.NewInt(int64(world+k+1)), uint(81+world%29)), big.NewInt(23))
				}
				basis[i][k] = v
				h.must(f.a.Set(uint32(i), f.basis[k], lpScalarBoundValue(h, v.RatString())))
			}
		}
		// Each nonbasic column is independently formed as B*direction. The
		// expected basis displacement is therefore known before running the LU.
		directions := make([][]*big.Rat, 3)
		spans := make([]*big.Rat, 3)
		start := make([]*big.Rat, 6)
		wave := []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat)}
		displacement := []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat)}
		for j := 0; j < 6; j++ {
			start[j] = big.NewRat(int64(world+j+1), 29)
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, start[j].RatString())))
		}
		can := []bool{false, true, false, true, false, true}
		for k, j := range []uint32{1, 3, 5} {
			directions[k] = make([]*big.Rat, 3)
			for i := 0; i < 3; i++ {
				directions[k][i] = big.NewRat(int64(rng.Intn(17)-8), int64(2+rng.Intn(11)))
			}
			low := big.NewRat(int64(k-3), 7)
			span := big.NewRat(int64(1+rng.Intn(19)), 13)
			up := new(big.Rat).Add(low, span)
			spans[k] = new(big.Rat).Set(span)
			start[j] = new(big.Rat).Set(low)
			if (world+k)%2 == 0 {
				start[j].Set(up)
				spans[k].Neg(spans[k])
			}
			f.types[j] = ColumnBoxed
			h.must(h.m.Copy(&f.low.Data[j], lpScalarBoundValue(h, low.RatString())))
			h.must(h.m.Copy(&f.up.Data[j], lpScalarBoundValue(h, up.RatString())))
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, start[j].RatString())))
			for i := 0; i < 3; i++ {
				c := new(big.Rat)
				for z := 0; z < 3; z++ {
					c.Add(c, new(big.Rat).Mul(basis[i][z], directions[k][z]))
				}
				h.must(f.a.Set(uint32(i), j, lpScalarBoundValue(h, c.RatString())))
				wave[i].Add(wave[i], new(big.Rat).Mul(c, spans[k]))
				displacement[i].Add(displacement[i], new(big.Rat).Mul(directions[k][i], spans[k]))
			}
		}
		// Encode the initial residual independently and verify the flip/update
		// sequence leaves the complete equation A*x=b intact.
		for i := 0; i < 3; i++ {
			rhs := new(big.Rat)
			for k, j := range f.basis {
				rhs.Add(rhs, new(big.Rat).Mul(basis[i][k], start[j]))
			}
			for k, j := range []uint32{1, 3, 5} {
				c := new(big.Rat)
				for z := 0; z < 3; z++ {
					c.Add(c, new(big.Rat).Mul(basis[i][z], directions[k][z]))
				}
				rhs.Add(rhs, new(big.Rat).Mul(c, start[j]))
			}
			h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, rhs.RatString())))
		}
		m, e := NewRationalDualCoreSolver(f.input(), &can)
		h.must(e)
		m.FlippedBoxed = map[uint32]bool{5: true, 1: true, 3: true}
		theta := big.NewRat(int64(world-90), 17)
		h.must(h.m.Copy(&m.ThetaD, lpScalarBoundValue(h, theta.RatString())))
		m.P = 2
		costs := map[uint32]*big.Rat{}
		for _, j := range f.nb {
			d := big.NewRat(int64(j+uint32(world)), 19)
			p := big.NewRat(int64(int(j)-4), 11)
			h.must(h.m.Copy(&m.D.Data[j], lpScalarBoundValue(h, d.RatString())))
			h.must(h.m.Copy(&m.PivotRow.Data[j], lpScalarBoundValue(h, p.RatString())))
			costs[j] = new(big.Rat).Sub(d, new(big.Rat).Mul(theta, p))
		}
		h.must(m.ProcessFlipped())
		for i := 0; i < 3; i++ {
			lpSparseRationalCheck(t, h, &m.AWave.Data[i], wave[i])
		}
		h.must(m.UpdateDAndXB())
		for _, j := range f.nb {
			lpSparseRationalCheck(t, h, &m.D.Data[j], costs[j])
			lpSparseRationalCheck(t, h, &f.x.Data[j], start[j])
		}
		lpSparseRationalCheck(t, h, &m.D.Data[2], new(big.Rat).Neg(theta))
		for i, j := range f.basis {
			lpSparseRationalCheck(t, h, &m.AWave.Data[i], displacement[i])
			lpSparseRationalCheck(t, h, &f.x.Data[j], new(big.Rat).Sub(start[j], displacement[i]))
		}
		h.must(m.ApplyFlips())
		for k, j := range []uint32{1, 3, 5} {
			lpSparseRationalCheck(t, h, &f.x.Data[j], new(big.Rat).Add(start[j], spans[k]))
		}
		bad, e := m.AMultXIsOff()
		h.must(e)
		if bad {
			t.Fatal(world, bad)
		}
		beta := new(big.Rat)
		for i := 0; i < 3; i++ {
			v := big.NewRat(int64(world+i-70), 31)
			if i == 2 {
				v.SetFrac(new(big.Int).Lsh(big.NewInt(int64(world+1)), uint(90+world%20)), big.NewInt(37))
			}
			h.must(h.m.Copy(&m.PivotRowB1.Data[i], lpScalarBoundValue(h, v.RatString())))
			beta.Add(beta, new(big.Rat).Mul(v, v))
		}
		m.R = int32(world % 3)
		h.must(m.CalculateBetaRPrecisely())
		lpSparseRationalCheck(t, h, &m.Betas.Data[m.R], beta)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalDualFlipLargeFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, 2, 4)
	f.settings.SimplexStrategy = StrategyLU
	huge := strings.Repeat("1234567890", 10)
	for i := uint32(0); i < 2; i++ {
		h.must(f.a.Set(i, i, lpScalarBoundValue(h, fmt.Sprint(i+2))))
	}
	for j := uint32(2); j < 4; j++ {
		f.types[j] = ColumnBoxed
		h.must(h.m.SetString(&f.low.Data[j], "-"+huge+"1/13"))
		h.must(h.m.SetString(&f.up.Data[j], huge+"3/17"))
		h.must(h.m.Copy(&f.x.Data[j], &f.low.Data[j]))
		if j == 3 {
			h.must(h.m.Copy(&f.x.Data[j], &f.up.Data[j]))
		}
		for i := uint32(0); i < 2; i++ {
			h.must(f.a.Set(i, j, lpScalarBoundValue(h, huge+fmt.Sprint(i+j)+"/19")))
		}
	}
	for i := 0; i < 2; i++ {
		h.must(h.m.SetString(&f.x.Data[i], huge+"7/23"))
	}
	can := []bool{false, false, true, true}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	m.P = 0
	m.R = 1
	m.FlippedBoxed = map[uint32]bool{3: true, 2: true}
	h.must(h.m.SetString(&m.ThetaD, huge+"9/29"))
	for j := 0; j < 4; j++ {
		h.must(h.m.SetString(&m.D.Data[j], huge+"1/31"))
		h.must(h.m.SetString(&m.PivotRow.Data[j], huge+"3/37"))
	}
	for i := 0; i < 2; i++ {
		h.must(h.m.SetString(&m.AWave.Data[i], huge+"1/41"))
		h.must(h.m.SetString(&m.PivotRowB1.Data[i], huge+"3/43"))
	}
	return f, m
}
func lpRationalDualFlipOperation(m *RationalDualCoreSolver, op string, out *numeral.Rational) error {
	switch op {
	case "signed":
		return m.SignedSpanOfBoxed(3, out)
	case "wave":
		return m.UpdateAWave(&m.ThetaD, 2)
	case "process":
		return m.ProcessFlipped()
	case "xb":
		return m.UpdateXBAfterBoundFlips()
	case "update":
		return m.UpdateDAndXB()
	case "apply":
		return m.ApplyFlips()
	case "beta":
		return m.CalculateBetaRPrecisely()
	}
	panic(op)
}
func TestLPRationalDualFlipPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"signed", "wave", "process", "xb", "update", "apply", "beta"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 3000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalDualFlipLargeFixture(t, h)
					if op == "apply" {
						h.must(h.m.SetInt32(&f.low.Data[2], 0))
						h.must(h.m.SetInt32(&f.x.Data[2], 0))
					}
					out := lpScalarBoundValue(h, "97")
					h.budget.FailAfter(ordinal)
					e := lpRationalDualFlipOperation(m, op, out)
					h.budget.FailAfter(-1)
					if e == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if retry := lpRationalDualFlipOperation(m, op, out); retry != ErrExactVectorPending {
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
func TestLPRationalDualFlipSignedAliases(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, upper := range []bool{false, true} {
		for _, alias := range []string{"out", "low", "up", "x"} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualFlipLargeFixture(t, h)
			if !upper {
				h.must(h.m.Copy(&f.x.Data[3], &f.low.Data[3]))
			}
			lo, _ := new(big.Rat).SetString("-" + strings.Repeat("1234567890", 10) + "1/13")
			up, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 10) + "3/17")
			want := new(big.Rat).Sub(up, lo)
			if upper {
				want.Neg(want)
			}
			out := lpScalarBoundValue(h, "97")
			switch alias {
			case "low":
				out = &f.low.Data[3]
			case "up":
				out = &f.up.Data[3]
			case "x":
				out = &f.x.Data[3]
			}
			h.must(m.SignedSpanOfBoxed(3, out))
			lpSparseRationalCheck(t, h, out, want)
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}
func TestLPRationalDualFlipColumnLiveDeltaAndOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualFlipLargeFixture(t, h)
		// Stored order is set by insertion, not by row identity. An early update
		// changes the live scalar used by the later row's product.
		a := NewRationalStaticMatrix(h.m, 2, 1)
		rows := []uint32{0, 1}
		if reverse {
			rows = []uint32{1, 0}
		}
		for _, i := range rows {
			h.must(a.Set(i, 0, lpScalarBoundValue(h, fmt.Sprint(i+2))))
		}
		h.must(h.m.SetInt32(&m.AWave.Data[0], 5))
		h.must(h.m.SetInt32(&m.AWave.Data[1], 7))
		h.must(a.AddColumnToVector(&m.AWave.Data[0], 0, m.AWave.Data))
		want1 := int64(52)
		if reverse {
			want1 = 22
		}
		lpSparseRationalCheck(t, h, &m.AWave.Data[0], big.NewRat(15, 1))
		lpSparseRationalCheck(t, h, &m.AWave.Data[1], big.NewRat(want1, 1))
		h.must(a.Close())
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualFlipEmptyAndNoFlips(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpRationalDualBreakpointFixture(t, h, 1)
	m.FlippedBoxed = nil
	h.must(m.ProcessFlipped())
	lpSparseRationalCheck(t, h, &m.AWave.Data[0], new(big.Rat))
	h.must(h.m.SetInt32(&m.ThetaD, 7))
	h.must(h.m.SetInt32(&m.D.Data[1], 9))
	h.must(h.m.SetInt32(&m.PivotRow.Data[1], 2))
	// With no flips, the update must not reach the factorization.
	lu := m.Factorization
	m.Factorization = nil
	h.must(m.UpdateDAndXB())
	h.must(m.ApplyFlips())
	m.Factorization = lu
	lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(-5, 1))
	lpSparseRationalCheck(t, h, &m.D.Data[0], big.NewRat(-7, 1))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
func TestLPRationalDualFlipPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"signed", "wave", "process", "xb", "update", "apply", "beta"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualFlipLargeFixture(t, h)
			out := lpScalarBoundValue(h, "97")
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
				_ = lpRationalDualFlipOperation(m, op, out)
				if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 || m.A.calls.comparison != 0 {
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

func TestLPRationalDualFlipMetadataAndPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, phase := range []string{"signed-x", "signed-up", "signed-local", "process-local", "wave-nil", "wave-column", "xb-factor", "xb-basis", "cost-prefix", "cost-p", "beta-pivot", "beta-r", "apply-upper", "apply-interior", "matrix-offset", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalDualFlipLargeFixture(t, h)
			out := lpScalarBoundValue(h, "97")
			xs, ups, piv, ds, basis := f.x.Data, f.up.Data, m.PivotRowB1.Data, m.D.Data, f.basis
			wave, lu := m.AWave, m.Factorization
			col := append([]ColumnCell(nil), f.a.Columns[2]...)
			var e error
			want := error(ErrBounds)
			switch phase {
			case "signed-x":
				f.x.Data = f.x.Data[:3]
				e = m.SignedSpanOfBoxed(3, out)
			case "signed-up":
				f.up.Data = f.up.Data[:3]
				e = m.SignedSpanOfBoxed(3, out)
			case "signed-local":
				m.signedSpan = NewIndexedRationalVector(h.m, 1)
				e = m.SignedSpanOfBoxed(3, out)
				want = ErrExactVectorPending
			case "process-local":
				m.processSpan = NewIndexedRationalVector(h.m, 1)
				e = m.ProcessFlipped()
				want = ErrExactVectorPending
			case "wave-nil":
				m.AWave = nil
				e = m.UpdateAWave(&m.ThetaD, 2)
			case "wave-column":
				e = m.UpdateAWave(&m.ThetaD, 99)
			case "xb-factor":
				m.Factorization = nil
				e = m.UpdateXBAfterBoundFlips()
			case "xb-basis":
				f.basis = f.basis[:1]
				e = m.UpdateXBAfterBoundFlips()
			case "cost-prefix":
				f.nb = []uint32{2, 3}
				m.D.Data = m.D.Data[:3]
				theta, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 10) + "9/29")
				d, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 10) + "1/31")
				p, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 10) + "3/37")
				e = m.UpdateDAndXB()
				lpSparseRationalCheck(t, h, &m.D.Data[2], new(big.Rat).Sub(d, new(big.Rat).Mul(theta, p)))
			case "cost-p":
				m.P = -1
				e = m.UpdateDAndXB()
			case "beta-pivot":
				m.PivotRowB1.Data = m.PivotRowB1.Data[:1]
				e = m.CalculateBetaRPrecisely()
			case "beta-r":
				m.R = -1
				e = m.CalculateBetaRPrecisely()
			case "apply-upper":
				h.must(h.m.Copy(&f.x.Data[3], &f.low.Data[3]))
				f.up.Data = f.up.Data[:3]
				e = m.ApplyFlips()
				lpSparseRationalCheck(t, h, &f.x.Data[2], lpRationalTestBig(strings.Repeat("1234567890", 10)+"3/17"))
			case "apply-interior":
				h.must(h.m.SetInt32(&f.x.Data[2], 0))
				e = m.ApplyFlips()
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = nil
					lpSparseRationalCheck(t, h, &f.x.Data[2], lpRationalTestBig("-"+strings.Repeat("1234567890", 10)+"1/13"))
				}
			case "matrix-offset":
				f.a.Columns[2][1].Offset = 999
				e = m.UpdateAWave(&m.ThetaD, 2)
			case "stopped":
				m.stopped = true
				e = m.ApplyFlips()
				want = ErrExactVectorPending
			}
			if e != want {
				t.Fatal(phase, e, want)
			}
			f.x.Data, f.up.Data, m.PivotRowB1.Data, m.D.Data, f.basis = xs, ups, piv, ds, basis
			m.AWave, m.Factorization = wave, lu
			f.a.Columns[2] = col
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
func lpRationalTestBig(s string) *big.Rat {
	v, ok := new(big.Rat).SetString(s)
	if !ok {
		panic(s)
	}
	return v
}

func TestLPRationalDualFlipSourcePublicationOrder(t *testing.T) {
	for _, updateCosts := range []bool{false, true} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualFlipLargeFixture(t, h)
		low := f.low.Data
		f.low.Data = f.low.Data[:3]
		var e error
		if updateCosts {
			e = m.UpdateDAndXB()
		} else {
			e = m.ProcessFlipped()
		}
		if e != ErrBounds {
			t.Fatal(e)
		}
		lo := lpRationalTestBig("-" + strings.Repeat("1234567890", 10) + "1/13")
		up := lpRationalTestBig(strings.Repeat("1234567890", 10) + "3/17")
		span := new(big.Rat).Sub(up, lo)
		for i := 0; i < 2; i++ {
			coefficient := lpRationalTestBig(strings.Repeat("1234567890", 10) + fmt.Sprint(i+2) + "/19")
			lpSparseRationalCheck(t, h, &m.AWave.Data[i], new(big.Rat).Mul(span, coefficient))
		}
		if updateCosts {
			theta := lpRationalTestBig(strings.Repeat("1234567890", 10) + "9/29")
			lpSparseRationalCheck(t, h, &m.D.Data[0], new(big.Rat).Neg(theta))
		}
		f.low.Data = low
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	// Bounds are live borrowed references. The first ascending flip reads x[2]
	// before the second flip changes it. Reversed map iteration would assign11.
	h := newLPPairHarness(t)
	f, m := lpRationalDualBreakpointFixture(t, h, 3)
	for i, v := range []int32{0, 5, 7, 11} {
		h.must(h.m.SetInt32(&f.x.Data[i], v))
		h.must(h.m.SetInt32(&f.low.Data[i], v))
	}
	up := f.up.Data
	f.up.Data = f.x.Data[1:]
	m.FlippedBoxed = map[uint32]bool{2: true, 1: true}
	h.must(m.ApplyFlips())
	lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(7, 1))
	lpSparseRationalCheck(t, h, &f.x.Data[2], big.NewRat(11, 1))
	f.up.Data = up
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
