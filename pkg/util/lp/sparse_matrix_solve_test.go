package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPSparseDenseTriangularSolvesIndependentResiduals(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewPCG(11419, 893))
	for world := 0; world < 220; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			coefficient := h.pair("0", "0")
			n := uint32(rng.IntN(8))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			dense := make([][]*big.Rat, n)
			rows, cols := make([]uint32, n), make([]uint32, n)
			for i := range dense {
				dense[i] = make([]*big.Rat, n)
				rows[i] = uint32(i)
				cols[i] = uint32(i)
				for j := range dense[i] {
					dense[i][j] = new(big.Rat)
				}
			}
			if n > 1 {
				for k := 0; k < 8; k++ {
					a := rng.Uint32N(n)
					b := (a + 1 + rng.Uint32N(n-1)) % n
					h.must(m.SwapRows(a, b))
					lpStaticMust(t, f.SwapRows(a, b))
					rows[a], rows[b] = rows[b], rows[a]
					a = rng.Uint32N(n)
					b = (a + 1 + rng.Uint32N(n-1)) % n
					h.must(m.SwapColumns(a, b))
					lpStaticMust(t, f.SwapColumns(a, b))
					cols[a], cols[b] = cols[b], cols[a]
				}
			}
			for _, off := range rng.Perm(int(n * n)) {
				i, j := uint32(off)/n, uint32(off)%n
				if i > j || (i != j && rng.IntN(3) == 0) {
					continue
				}
				v := float64(rng.IntN(17)-8) / 4
				if i == j {
					v = 1
				}
				if i != j && rng.IntN(7) == 0 {
					v = math.Ldexp(1, -80)
				}
				x := new(big.Rat).SetFloat64(v)
				dense[i][j] = x
				h.must(h.m.SetString(&coefficient.X, x.RatString()))
				h.must(m.AddNewElement(rows[i], cols[j], &coefficient.X))
				lpStaticMust(t, f.AddNewElement(rows[i], cols[j], v))
			}
			xs, ys := make([]*big.Rat, n), make([]*big.Rat, n)
			for i := range xs {
				xs[i] = big.NewRat(int64(rng.IntN(17)-8), 4)
				ys[i] = big.NewRat(int64(rng.IntN(17)-8), 4)
			}
			rhs := lpExactMatrixVector(dense, xs, false)
			rowRHS := lpExactMatrixVector(dense, xs, true)
			pairRHS := lpExactMatrixVector(dense, ys, false)
			uy := lpExactVectorFromOracle(h, append(rhs, big.NewRat(777, 1)))
			yu := lpExactVectorFromOracle(h, append(rowRHS, big.NewRat(777, 1)))
			pairs := make([]RationalPair, n+1)
			for i := range pairs {
				pairs[i] = NewRationalPair()
			}
			for i := range rhs {
				h.must(h.m.SetString(&pairs[i].X, rhs[i].RatString()))
				h.must(h.m.SetString(&pairs[i].Y, pairRHS[i].RatString()))
			}
			h.must(h.m.SetInt64(&pairs[n].X, 777))
			h.must(h.m.SetInt64(&pairs[n].Y, 888))
			h.must(m.SolveUY(uy.Data))
			h.must(m.SolveYU(yu.Data))
			h.must(m.SolveUYPair(pairs))
			fu, fr, fd := make([]float64, n+1), make([]float64, n+1), make([]float64, n+1)
			for i := range rhs {
				fu[i], _ = rhs[i].Float64()
				fr[i], _ = rowRHS[i].Float64()
				fd[i] = fu[i]
			}
			fu[n] = 777
			fr[n] = 777
			fd[n] = 777
			lpStaticMust(t, f.SolveUY(fu))
			lpStaticMust(t, f.SolveYU(fr))
			lpStaticMust(t, f.DoubleSolveUY(fd))
			for i := range xs {
				lpSparseRationalCheck(t, h, &uy.Data[i], xs[i])
				lpSparseRationalCheck(t, h, &yu.Data[i], xs[i])
				h.check(&pairs[i], xs[i], ys[i])
				want, _ := xs[i].Float64()
				for _, got := range []float64{fu[i], fr[i], fd[i]} {
					if math.Abs(got-want) > 1e-10 {
						t.Fatal("floating solution", i, got, want)
					}
				}
				dot, e := f.DotProductWithRow(uint32(i), fu)
				lpStaticMust(t, e)
				b, _ := rhs[i].Float64()
				if math.Abs(dot-b) > 1e-10 {
					t.Fatal("independent dense residual", i, dot, b)
				}
			}
			lpSparseRationalCheck(t, h, &uy.Data[n], big.NewRat(777, 1))
			lpSparseRationalCheck(t, h, &yu.Data[n], big.NewRat(777, 1))
			h.check(&pairs[n], big.NewRat(777, 1), big.NewRat(888, 1))
			if fu[n] != 777 || fr[n] != 777 || fd[n] != 777 {
				t.Fatal("trailing value changed")
			}
			owned := lpSparseEtaOwningCells(m, nil)
			for i := range uy.Data {
				owned = append(owned, &uy.Data[i], &yu.Data[i], &pairs[i].X, &pairs[i].Y)
			}
			lpExactVectorNamedCensus(t, h, owned)
			h.must(m.Close())
			h.must(uy.Close())
			h.must(yu.Close())
			for i := range pairs {
				h.must(pairs[i].Del(h.m))
			}
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseDenseSolveLiteralDiagonalDuplicatesAndIEEE(t *testing.T) {
	// These solves ignore stored diagonal coefficients. No implicit division or
	// structural repair is introduced for nonunit or nontriangular inputs.
	h := newLPPairHarness(t)
	p := h.pair("0", "0")
	m := lpNewExactSparse(t, h, 3)
	f := newFloatSparseMatrix(3)
	for _, e := range []struct {
		i, j uint32
		v    int64
	}{{0, 0, 17}, {1, 1, 19}, {2, 2, 23}, {0, 1, 2}, {0, 1, 3}, {1, 2, 4}, {0, 2, 0}} {
		h.must(h.m.SetInt64(&p.X, e.v))
		h.must(m.AddNewElement(e.i, e.j, &p.X))
		lpStaticMust(t, f.AddNewElement(e.i, e.j, float64(e.v)))
	}
	for _, right := range []bool{false, true} {
		w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), big.NewRat(2, 1), big.NewRat(3, 1)})
		fw := []float64{1, 2, 3}
		var want []int64
		if right {
			h.must(m.SolveYU(w.Data))
			lpStaticMust(t, f.SolveYU(fw))
			want = []int64{1, -3, 15}
		} else {
			h.must(m.SolveUY(w.Data))
			lpStaticMust(t, f.SolveUY(fw))
			want = []int64{51, -10, 3}
		}
		for i, x := range want {
			lpSparseRationalCheck(t, h, &w.Data[i], big.NewRat(x, 1))
			if fw[i] != float64(x) {
				t.Fatal("literal solve", right, fw, want)
			}
		}
		h.must(w.Close())
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
	// Unit size left solve never reads its input; a zero anchor skips malformed
	// coefficient storage. Stored zero times infinity still executes as NaN.
	one := newFloatSparseMatrix(1)
	lpStaticMust(t, one.SolveYU(nil))
	z := newFloatSparseMatrix(2)
	z.Rows = nil
	lpStaticMust(t, z.SolveYU([]float64{math.Copysign(0, -1)}))
	z = newFloatSparseMatrix(2)
	lpStaticMust(t, z.AddNewElement(0, 1, 0))
	out := []float64{1, math.Inf(1)}
	lpStaticMust(t, z.SolveUY(out))
	if !math.IsNaN(out[0]) {
		t.Fatal("stored zero product suppressed")
	}
}

func TestLPExactSparseDenseSolveAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	totalReceipts := 0
	for _, op := range []string{"uy", "yu", "pair"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", direct), func(t *testing.T) {
				failures, successes, pairPrefixes := 0, 0, 0
				for ordinal := int64(0); ordinal < 700; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						m := lpNewExactSparse(t, h, 2)
						xs, ys, zs := strings.Repeat("1234567890", 50)+"/7", strings.Repeat("9876543210", 100)+"/13", strings.Repeat("3141592653", 70)+"/17"
						p := h.pair(xs, ys)
						z := h.pair(zs, "1")
						x, y, zrat := mustLPRat(t, xs), mustLPRat(t, ys), mustLPRat(t, zs)
						h.must(m.AddNewElement(0, 0, &z.Y))
						h.must(m.AddNewElement(1, 1, &z.Y))
						h.must(m.AddNewElement(0, 1, &p.X))
						initial := []*big.Rat{big.NewRat(1, 1), new(big.Rat).Set(y)}
						target := 0
						if op == "yu" {
							initial[0], initial[1] = initial[1], initial[0]
							target = 1
						}
						w := lpExactVectorFromOracle(h, initial)
						pairs := []RationalPair{NewRationalPair(), NewRationalPair()}
						if op == "pair" {
							h.must(h.m.SetInt64(&pairs[0].X, 1))
							h.must(h.m.SetInt64(&pairs[0].Y, 2))
							h.must(h.m.Copy(&pairs[1].X, &p.Y))
							h.must(h.m.Copy(&pairs[1].Y, &z.X))
						}
						expected := new(big.Rat).Sub(big.NewRat(1, 1), new(big.Rat).Mul(x, y))
						expectedY := new(big.Rat).Sub(big.NewRat(2, 1), new(big.Rat).Mul(x, zrat))
						h.budget.FailAfter(ordinal)
						var err error
						switch op {
						case "uy":
							err = m.SolveUY(w.Data)
						case "yu":
							err = m.SolveYU(w.Data)
						case "pair":
							err = m.SolveUYPair(pairs)
						}
						h.budget.FailAfter(-1)
						if err == nil {
							successes++
							done = true
							if op == "pair" {
								h.check(&pairs[0], expected, expectedY)
							} else {
								lpSparseRationalCheck(t, h, &w.Data[target], expected)
							}
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if m.calls.pending() {
								totalReceipts++
							}
							if e := m.SolveUY(w.Data); e != ErrExactVectorPending {
								t.Fatal("stopped scalar solve", e)
							}
							if e := m.SolveYU(w.Data); e != ErrExactVectorPending {
								t.Fatal("stopped row solve", e)
							}
							if e := m.SolveUYPair(pairs); e != ErrExactVectorPending {
								t.Fatal("stopped pair solve", e)
							}
							if w.calls.pending() {
								t.Fatal("caller adopted matrix invocation")
							}
						}
						h.must(m.Close())
						if err != nil && direct && op == "pair" {
							sx, ex := h.m.String(&pairs[0].X)
							sy, ey := h.m.String(&pairs[0].Y)
							vx, okx := new(big.Rat).SetString(sx)
							vy, oky := new(big.Rat).SetString(sy)
							if ex == nil && ey == nil && okx && oky && vx.Cmp(expected) == 0 && vy.Cmp(big.NewRat(2, 1)) == 0 {
								pairPrefixes++
							}
						}
						var owned []*numeral.Rational
						for i := range w.Data {
							owned = append(owned, &w.Data[i], &pairs[i].X, &pairs[i].Y)
						}
						lpExactVectorNamedCensus(t, h, owned)
						h.must(w.Close())
						for i := range pairs {
							h.must(pairs[i].Del(h.m))
						}
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if failures == 0 || successes != 1 || (direct && op == "pair" && pairPrefixes == 0) {
					t.Fatal("incomplete solve campaign", failures, successes, pairPrefixes)
				}
			})
		}
	}
	if totalReceipts == 0 {
		t.Fatal("no actual solve numerical receipts")
	}
}

func TestLPSparseDenseSolveBoundsAndPublicationPrefixes(t *testing.T) {
	for _, direction := range []string{"uy", "yu", "pair"} {
		for _, phase := range []string{"rhs", "mapping", "storage", "inverse", "target", "prefix"} {
			t.Run(direction+"/"+phase, func(t *testing.T) {
				h := newLPPairHarness(t)
				m := lpNewExactSparse(t, h, 3)
				f := newFloatSparseMatrix(3)
				p := h.pair("1", "2")
				add := func(i, j uint32, v *numeral.Rational, x float64) {
					h.must(m.AddNewElement(i, j, v))
					lpStaticMust(t, f.AddNewElement(i, j, x))
				}
				for i := uint32(0); i < 3; i++ {
					add(i, i, &p.X, 1)
				}
				if direction == "yu" {
					add(0, 1, &p.Y, 2)
					add(0, 2, &p.Y, 2)
				} else {
					add(0, 2, &p.Y, 2)
					add(1, 2, &p.Y, 2)
				}
				w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1), big.NewRat(1, 1)})
				pairs := []RationalPair{NewRationalPair(), NewRationalPair(), NewRationalPair()}
				for i := range pairs {
					h.must(h.m.SetInt64(&pairs[i].X, 1))
					h.must(h.m.SetInt64(&pairs[i].Y, 1))
				}
				data := w.Data
				pairData := pairs
				fw := []float64{1, 1, 1}
				savedRows, savedCols := m.Rows, m.Columns
				switch phase {
				case "rhs":
					data = nil
					pairData = nil
					fw = nil
				case "mapping":
					if direction == "yu" {
						m.RowPermutation.values = nil
						f.RowPermutation.values = nil
					} else {
						m.ColumnPermutation.reverse = nil
						f.ColumnPermutation.reverse = nil
					}
				case "storage":
					if direction == "yu" {
						m.Rows = nil
						f.Rows = nil
					} else {
						m.Columns = nil
						f.Columns = nil
					}
				case "inverse":
					if direction == "yu" {
						m.Rows[0][1].Index = 3
						f.Rows[0][1].Index = 3
					} else {
						m.Columns[2].Values[1].Index = 3
						f.Columns[2].Values[1].Index = 3
					}
				case "target":
					if direction == "yu" {
						data = data[:1]
						fw = fw[:1]
					} else {
						m.RowPermutation.reverse[0] = 3
						f.RowPermutation.reverse[0] = 3
					}
				case "prefix":
					if direction == "yu" {
						m.Rows[0][2].Index = 3
						f.Rows[0][2].Index = 3
					} else {
						m.Columns[2].Values[2].Index = 3
						f.Columns[2].Values[2].Index = 3
					}
				}
				var err, ferr error
				switch direction {
				case "yu":
					err = m.SolveYU(data)
					ferr = f.SolveYU(fw)
				case "uy":
					err = m.SolveUY(data)
					ferr = f.SolveUY(fw)
				case "pair":
					err = m.SolveUYPair(pairData)
					ferr = f.SolveUY(fw)
				}
				if err != ErrBounds || ferr != ErrBounds {
					t.Fatal("solve bounds", direction, phase, err, ferr)
				}
				if phase == "prefix" {
					target := 0
					if direction == "yu" {
						target = 1
					}
					if direction == "pair" {
						h.check(&pairs[0], big.NewRat(-1, 1), big.NewRat(-1, 1))
					} else {
						lpSparseRationalCheck(t, h, &w.Data[target], big.NewRat(-1, 1))
					}
					if fw[target] != -1 {
						t.Fatal("floating publication prefix")
					}
				}
				m.Rows = savedRows
				m.Columns = savedCols
				h.must(m.Close())
				h.must(w.Close())
				for i := range pairs {
					h.must(pairs[i].Del(h.m))
				}
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
	for _, phase := range []string{"row", "storage", "column", "input", "partial", "double input", "double residual"} {
		t.Run("dot/"+phase, func(t *testing.T) {
			f := newFloatSparseMatrix(2)
			lpStaticMust(t, f.AddNewElement(0, 0, 1))
			lpStaticMust(t, f.AddNewElement(0, 1, 2))
			lpStaticMust(t, f.AddNewElement(1, 1, 1))
			y := []float64{1, 1}
			row := uint32(0)
			switch phase {
			case "row":
				row = 2
			case "storage":
				f.Rows = nil
			case "column":
				f.Rows[0][0].Index = 2
			case "input":
				y = nil
			case "partial":
				f.Rows[0][1].Index = 2
			case "double input":
				y = nil
			case "double residual":
				f.Rows = nil
			}
			var err error
			var sum float64
			if phase == "double input" || phase == "double residual" {
				err = f.DoubleSolveUY(y)
			} else {
				sum, err = f.DotProductWithRow(row, y)
			}
			if err != ErrBounds {
				t.Fatal("dot/refinement bounds", phase, err)
			}
			if phase == "partial" && sum != 1 {
				t.Fatal("partial dot value", sum)
			}
		})
	}
}
