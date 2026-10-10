package lp

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpExactLUStatic(t *testing.T, h *lpPairHarness, b [][]*big.Rat, basis []uint32) *RationalStaticMatrix {
	t.Helper()
	a := NewRationalStaticMatrix(h.m, uint32(len(b)), uint32(2*len(b)+1))
	t.Cleanup(func() { h.must(a.Close()) })
	v := numeral.NewRational()
	defer func() { h.must(h.m.Del(&v)) }()
	for i := range b {
		for j, column := range basis {
			h.must(h.m.SetString(&v, b[i][j].RatString()))
			h.must(a.Set(uint32(i), column, &v))
		}
	}
	return a
}

func lpExactLUVector(h *lpPairHarness, values []*big.Rat) *IndexedRationalVector {
	v := NewIndexedRationalVector(h.m, uint32(len(values)))
	for i, x := range values {
		h.must(h.m.SetString(&v.Data[i], x.RatString()))
		if x.Sign() != 0 {
			v.Index = append(v.Index, uint32(i))
		}
	}
	return v
}

func lpExactLUCheckVector(t *testing.T, h *lpPairHarness, values []numeral.Rational, want []*big.Rat) {
	t.Helper()
	for i, x := range want {
		lpSparseRationalCheck(t, h, &values[i], x)
	}
}

func lpExactLUStaticCells(a *RationalStaticMatrix) []*numeral.Rational {
	var out []*numeral.Rational
	for i := range a.Rows {
		for j := range a.Rows[i] {
			out = append(out, &a.Rows[i][j].Value)
		}
	}
	for i := range a.WorkVector.Data {
		out = append(out, &a.WorkVector.Data[i])
	}
	return out
}

func lpExactLUCheckSource(t *testing.T, h *lpPairHarness, a *RationalStaticMatrix, b [][]*big.Rat, basis []uint32) {
	t.Helper()
	for i, row := range b {
		count := 0
		for j, value := range row {
			if value.Sign() == 0 {
				continue
			}
			count++
			found := false
			for k := range a.Rows[i] {
				cell := &a.Rows[i][k]
				if cell.Column == basis[j] {
					lpSparseRationalCheck(t, h, &cell.Value, value)
					found = true
				}
			}
			if !found {
				t.Fatal("borrowed source coefficient removed")
			}
		}
		if len(a.Rows[i]) != count {
			t.Fatal("borrowed source row changed")
		}
	}
}

func TestLPExactLUIndependentFactorizationAndSolves(t *testing.T) {
	rng := rand.New(rand.NewPCG(192311, 711319))
	denseCount, rowSwaps, columnSwaps := 0, 0, 0
	for world := 0; world < 56; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			n := 1 + world%7
			b := lpExactIdentity(n)
			order, columns := rng.Perm(n), rng.Perm(2*n+1)
			basis := make([]uint32, n)
			x, y := make([]*big.Rat, n), make([]*big.Rat, n)
			for i := range b {
				basis[i] = uint32(columns[i])
				x[i] = new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(rng.IntN(9)-4)), 73), big.NewInt(7))
				y[i] = big.NewRat(int64(rng.IntN(13)-6), 13)
				for j := range b[i] {
					v := int64(rng.IntN(5) - 2)
					if j == order[i] {
						v = int64(15*n + i)
					}
					b[i][j].SetFrac64(v, 7)
				}
			}
			a := lpExactLUStatic(t, h, b, basis)
			for _, density := range []float64{math.Inf(1), 0} {
				s := lpFloatLUSettings(density)
				r, err := NewRationalLU(a, basis, s)
				h.must(err)
				defer func() { h.must(r.Close()) }()
				p, err := NewPairLU(a, basis, s)
				h.must(err)
				defer func() { h.must(p.Close()) }()
				if r.GetStatus() != LUOK || p.GetStatus() != LUOK || r.Failure || p.Failure || s.Stats.Factorizations != 2 {
					t.Fatal("factorization status/statistics")
				}
				wantDense := density == 0 && n >= 5
				if (r.Dense != nil) != wantDense || (p.Dense != nil) != wantDense {
					t.Fatal("dense switch")
				}
				if wantDense {
					denseCount++
				}
				for i := range b {
					if r.Q.values[i] != uint32(i) {
						rowSwaps++
					}
					if r.R.values[i] != uint32(i) {
						columnSwaps++
					}
					for j := range b[i] {
						u, e := r.U.Get(uint32(i), uint32(j))
						h.must(e)
						if j < i {
							lpSparseRationalCheck(t, h, u, big.NewRat(0, 1))
						} else if i == j {
							lpSparseRationalCheck(t, h, u, big.NewRat(1, 1))
						}
					}
				}
				// Verify the source factorization identity with independently formed
				// scalar matrix products, rather than reimplementing its pivots.
				for j := range b {
					column := make([]*big.Rat, n)
					for i := range b {
						column[i] = b[i][j]
					}
					v := lpExactLUVector(h, column)
					for _, tail := range r.Tail {
						h.must(tail.ApplyLeft(v.Data))
					}
					for i := range b {
						want := new(big.Rat)
						for k := range b {
							for l := range b {
								if r.Q.values[i] != uint32(k) || r.R.values[l] != uint32(j) {
									continue
								}
								u, e := r.U.Get(uint32(k), uint32(l))
								h.must(e)
								text, e := h.m.String(u)
								h.must(e)
								value, ok := new(big.Rat).SetString(text)
								if !ok {
									t.Fatal(text)
								}
								want.Add(want, value)
							}
						}
						lpSparseRationalCheck(t, h, &v.Data[i], want)
					}
					h.must(v.Close())
				}
				for _, right := range []bool{false, true} {
					rhs := lpExactMatrixVector(b, x, right)
					v := lpExactLUVector(h, rhs)
					if right {
						h.must(r.SolveYB(v.Data))
					} else {
						h.must(r.SolveBy(v.Data))
					}
					lpExactLUCheckVector(t, h, v.Data, x)
					h.must(v.Close())
					if right {
						for _, pair := range []bool{false, true} {
							v = lpExactLUVector(h, rhs)
							if pair {
								h.must(p.SolveYBIndexed(v))
							} else {
								h.must(r.SolveYBIndexed(v))
							}
							lpExactLUCheckVector(t, h, v.Data, x)
							h.must(v.Close())
						}
						v = lpExactLUVector(h, rhs)
						h.must(p.SolveYB(v.Data))
						lpExactLUCheckVector(t, h, v.Data, x)
						h.must(v.Close())
					} else {
						v := NewIndexedPairVector(h.m, uint32(n))
						rhsY := lpExactMatrixVector(b, y, false)
						for i := range v.Data {
							h.must(h.m.SetString(&v.Data[i].X, rhs[i].RatString()))
							h.must(h.m.SetString(&v.Data[i].Y, rhsY[i].RatString()))
						}
						h.must(p.SolveBy(v.Data))
						for i := range v.Data {
							h.check(&v.Data[i], x[i], y[i])
						}
						h.must(v.Close())
					}
				}
				h.must(p.Close())
				h.must(r.Close())
				lpExactLUCheckSource(t, h, a, b, basis)
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			}
		})
	}
	if denseCount != 24 || rowSwaps == 0 || columnSwaps == 0 {
		t.Fatalf("dense=%d row swaps=%d column swaps=%d", denseCount, rowSwaps, columnSwaps)
	}
}

func TestLPExactLUConstructorGuardsAndDegeneration(t *testing.T) {
	for _, pair := range []bool{false, true} {
		for _, kind := range []string{"empty", "short", "outside", "duplicate", "tableau", "singular", "tiny", "dependent-sparse", "dependent-dense"} {
			t.Run(fmt.Sprint(pair, "/", kind), func(t *testing.T) {
				h := newLPPairHarness(t)
				n := 2
				if kind == "empty" {
					n = 0
				}
				if kind == "dependent-dense" {
					n = 6
				}
				b := lpExactIdentity(n)
				basis := make([]uint32, n)
				for i := range basis {
					basis[i] = uint32(i)
				}
				if kind == "singular" {
					b[1][1].SetInt64(0)
				}
				if kind == "tiny" {
					b[0][0].SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 100))
				}
				if kind == "dependent-sparse" {
					b[0][0].SetInt64(2)
					b[0][1].SetInt64(2)
					b[1][0].SetInt64(1)
				}
				if kind == "dependent-dense" {
					for i := range b {
						for j := range b[i] {
							b[i][j].SetInt64(1)
						}
						b[i][i].SetInt64(20)
					}
					for j := range b[5] {
						b[5][j].Set(b[4][j])
					}
				}
				a := lpExactLUStatic(t, h, b, basis)
				s := lpFloatLUSettings(0)
				switch kind {
				case "short":
					basis = basis[:1]
				case "outside":
					basis[1] = a.ColumnCount()
				case "duplicate":
					basis[1] = basis[0]
				case "tableau":
					s.SimplexStrategy = StrategyTableauRows
				}
				var state *exactLUState
				var err error
				var r *RationalLU
				var p *PairLU
				if pair {
					p, err = NewPairLU(a, basis, s)
					state = p.exactLUState
					defer func() { h.must(p.Close()) }()
				} else {
					r, err = NewRationalLU(a, basis, s)
					state = r.exactLUState
					defer func() { h.must(r.Close()) }()
				}
				wantErr := kind == "short" || kind == "outside" || (sourceDebugEnabled && (kind == "duplicate" || kind == "tableau"))
				if (err != nil) != wantErr {
					t.Fatalf("err=%v want error=%v", err, wantErr)
				}
				if err != nil {
					if state.ready() != ErrLUStopped || s.Stats.Factorizations != 0 {
						t.Fatal("failed constructor state/stats")
					}
				} else {
					if (state.GetStatus() == LUDegenerated) != (kind == "singular" || kind == "duplicate" || kind == "dependent-sparse" || kind == "dependent-dense") {
						t.Fatal("precise status", state.GetStatus())
					}
					if s.Stats.Factorizations != 1 {
						t.Fatal("missing factorization")
					}
					if kind == "empty" {
						v := NewIndexedRationalVector(h.m, 0)
						if pair {
							h.must(p.SolveBy(nil))
							h.must(p.SolveYB(nil))
							h.must(p.SolveYBIndexed(v))
						} else {
							h.must(r.SolveBy(nil))
							h.must(r.SolveYB(nil))
							h.must(r.SolveYBIndexed(v))
						}
						h.must(v.Close())
					}
					if kind == "tiny" {
						want := []*big.Rat{new(big.Rat).Set(b[0][0]), big.NewRat(0, 1)}
						rhs := lpExactMatrixVector(b, want, false)
						if pair {
							v := NewIndexedPairVector(h.m, 2)
							for i := range v.Data {
								h.must(h.m.SetString(&v.Data[i].X, rhs[i].RatString()))
								h.must(h.m.SetString(&v.Data[i].Y, rhs[i].RatString()))
							}
							h.must(p.SolveBy(v.Data))
							for i := range v.Data {
								h.check(&v.Data[i], want[i], want[i])
							}
							h.must(v.Close())
						} else {
							v := lpExactLUVector(h, rhs)
							h.must(r.SolveBy(v.Data))
							lpExactLUCheckVector(t, h, v.Data, want)
							h.must(v.Close())
						}
					}
					if kind == "dependent-sparse" && state.pendingEta == nil {
						t.Fatal("failed eta owner not retained")
					}
					if kind == "dependent-dense" && (!state.denseOwned || state.Dense == nil) {
						t.Fatal("failed dense owner not retained")
					}
				}
				state.SetStatus(LUOK)
				if state.GetStatus() != LUOK {
					t.Fatal("status")
				}
				state.RefactorCounter = 199
				if state.NeedToRefactor() {
					t.Fatal("refactor")
				}
				state.RefactorCounter = 200
				if !state.NeedToRefactor() {
					t.Fatal("refactor boundary")
				}
				if pair {
					h.must(p.Close())
					h.must(p.Close())
					if !errors.Is(p.SolveBy(nil), ErrLUClosed) || !errors.Is(p.SolveYB(nil), ErrLUClosed) || !errors.Is(p.SolveYBIndexed(nil), ErrLUClosed) {
						t.Fatal("closed pair")
					}
				} else {
					h.must(r.Close())
					h.must(r.Close())
					if !errors.Is(r.SolveBy(nil), ErrLUClosed) || !errors.Is(r.SolveYB(nil), ErrLUClosed) || !errors.Is(r.SolveYBIndexed(nil), ErrLUClosed) {
						t.Fatal("closed rational")
					}
				}
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
	}
}
