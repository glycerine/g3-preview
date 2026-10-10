package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpColumnIndexCheck(t *testing.T, indices []uint32, want []*big.Rat, descending bool) {
	t.Helper()
	seen := make(map[uint32]bool)
	var order []uint32
	for i := len(want); i > 0; {
		i--
		if want[i].Sign() != 0 {
			order = append(order, uint32(i))
		}
	}
	for _, i := range indices {
		if int(i) >= len(want) || seen[i] || want[i].Sign() == 0 {
			t.Fatalf("invalid output index %v", indices)
		}
		seen[i] = true
	}
	if len(seen) != len(order) {
		t.Fatalf("missing output indices %v want %v", indices, order)
	}
	if descending && !reflect.DeepEqual(append([]uint32(nil), indices...), order) {
		t.Fatalf("dense index order %v want %v", indices, order)
	}
}

func TestLPLUEnteringColumnIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(633113, 219113))
	for world := 0; world < 48; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			n := 1 + world%8
			b := lpExactIdentity(n)
			basis := make([]uint32, n)
			want := make([]*big.Rat, n)
			order, columns := rng.Perm(n), rng.Perm(2*n+1)
			entering := uint32(columns[n])
			for i := range b {
				basis[i] = uint32(columns[i])
				want[i] = big.NewRat(int64(rng.IntN(9)-4), 8)
				for j := range b[i] {
					v := int64(rng.IntN(5) - 2)
					if j == order[i] {
						v = int64(20*n + i)
					}
					b[i][j].SetFrac64(v, 4)
				}
			}
			rhs := lpExactMatrixVector(b, want, false)
			a := lpFloatLUSource(t, b, basis)
			exact := lpExactLUStatic(t, h, b, basis)
			tmp := numeral.NewRational()
			defer func() { h.must(h.m.Del(&tmp)) }()
			for i := range rhs {
				v, _ := rhs[i].Float64()
				lpStaticMust(t, a.Set(uint32(i), entering, v))
				h.must(h.m.SetString(&tmp, rhs[i].RatString()))
				h.must(exact.Set(uint32(i), entering, &tmp))
			}
			density := math.Inf(1)
			if (world/8)%2 == 0 {
				density = 0
			}
			s := lpFloatLUSettings(density)
			f, err := NewFloatLU(a, basis, s)
			lpStaticMust(t, err)
			defer f.Close()
			r, err := NewRationalLU(exact, basis, s)
			h.must(err)
			defer func() { h.must(r.Close()) }()
			p, err := NewPairLU(exact, basis, s)
			h.must(err)
			defer func() { h.must(p.Close()) }()
			d, w := NewIndexedFloatVector(uint32(n)), NewIndexedFloatVector(uint32(n))
			lpStaticMust(t, d.SetValue(99, 0))
			lpStaticMust(t, w.SetValue(87, uint32(n-1)))
			lpStaticMust(t, f.SolveBd(entering, d, w))
			for i := range want {
				lpFloatBlockNear(t, d.Data[i], want[i])
			}
			lpColumnIndexCheck(t, d.Index, want, w.Size() > 0)
			// Preparation must leave U*R*solution in w, independently of the
			// selected column-solve implementation and graph traversal order.
			for i := range want {
				value := new(big.Rat)
				for j := range want {
					u, e := r.U.Get(uint32(i), uint32(j))
					h.must(e)
					text, e := h.m.String(u)
					h.must(e)
					coefficient, ok := new(big.Rat).SetString(text)
					if !ok {
						t.Fatal(text)
					}
					value.Add(value, new(big.Rat).Mul(coefficient, want[r.R.values[j]]))
				}
				lpFloatBlockNear(t, w.Data[i], value)
			}
			for _, pair := range []bool{false, true} {
				for _, fast := range []bool{false, true} {
					if fast && !pair {
						continue
					}
					d, w := NewIndexedRationalVector(h.m, uint32(n)), NewIndexedRationalVector(h.m, uint32(n))
					h.must(h.m.SetString(&tmp, "99"))
					h.must(d.SetValue(&tmp, 0))
					h.must(w.SetValue(&tmp, uint32(n-1)))
					if fast {
						h.must(p.SolveBdFaster(entering, d))
					} else if pair {
						h.must(p.SolveBd(entering, d, w))
					} else {
						h.must(r.SolveBd(entering, d, w))
					}
					lpExactLUCheckVector(t, h, d.Data, want)
					lpColumnIndexCheck(t, d.Index, want, !fast && w.Size() > 0)
					if !fast {
						for i := range want {
							value := new(big.Rat)
							for j := range want {
								u, e := r.U.Get(uint32(i), uint32(j))
								h.must(e)
								text, e := h.m.String(u)
								h.must(e)
								coefficient, ok := new(big.Rat).SetString(text)
								if !ok {
									t.Fatal(text)
								}
								value.Add(value, new(big.Rat).Mul(coefficient, want[r.R.values[j]]))
							}
							lpSparseRationalCheck(t, h, &w.Data[i], value)
						}
					}
					h.must(d.Close())
					h.must(w.Close())
				}
			}
			h.must(p.Close())
			h.must(r.Close())
			h.must(h.m.Del(&tmp))
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(exact))
		})
	}
}

func TestLPLUEnteringColumnSparseBoundaryAndAlias(t *testing.T) {
	for _, scalar := range []string{"float", "rational", "pair"} {
		ratio := 10
		if scalar == "float" {
			ratio = 120
		}
		for _, n := range []int{ratio, ratio + 1} {
			for _, alias := range []bool{false, true} {
				t.Run(fmt.Sprint(scalar, "/", n, "/alias=", alias), func(t *testing.T) {
					h := newLPPairHarness(t)
					basis := make([]uint32, n)
					a := NewFloatStaticMatrix(uint32(n), uint32(n+1))
					exact := NewRationalStaticMatrix(h.m, uint32(n), uint32(n+1))
					defer func() { h.must(exact.Close()) }()
					v := h.pair("2", "6")
					for j := range basis {
						basis[j] = uint32(j)
						row := uint32((j + 3) % n)
						lpStaticMust(t, a.Set(row, uint32(j), 2))
						h.must(exact.Set(row, uint32(j), &v.X))
					}
					row := uint32((n + 2) % n)
					lpStaticMust(t, a.Set(row, uint32(n), 6))
					h.must(exact.Set(row, uint32(n), &v.Y))
					s := lpFloatLUSettings(math.Inf(1))
					want := make([]*big.Rat, n)
					for i := range want {
						want[i] = new(big.Rat)
					}
					if !alias || n == ratio {
						want[n-1].SetInt64(3)
					}
					if scalar == "float" {
						m, e := NewFloatLU(a, basis, s)
						lpStaticMust(t, e)
						defer m.Close()
						d, w := NewIndexedFloatVector(uint32(n)), NewIndexedFloatVector(uint32(n))
						if alias {
							d = w
						}
						lpStaticMust(t, m.SolveBd(uint32(n), d, w))
						for i := range want {
							lpFloatBlockNear(t, d.Data[i], want[i])
						}
						lpColumnIndexCheck(t, d.Index, want, n == ratio)
					} else {
						d, w := NewIndexedRationalVector(h.m, uint32(n)), NewIndexedRationalVector(h.m, uint32(n))
						defer func() { h.must(d.Close()); h.must(w.Close()) }()
						if alias {
							h.must(d.Close())
							d = w
						}
						if scalar == "pair" {
							m, e := NewPairLU(exact, basis, s)
							h.must(e)
							defer func() { h.must(m.Close()) }()
							h.must(m.SolveBd(uint32(n), d, w))
							h.must(m.Close())
						} else {
							m, e := NewRationalLU(exact, basis, s)
							h.must(e)
							defer func() { h.must(m.Close()) }()
							h.must(m.SolveBd(uint32(n), d, w))
							h.must(m.Close())
						}
						lpExactLUCheckVector(t, h, d.Data, want)
						lpColumnIndexCheck(t, d.Index, want, n == ratio)
						h.must(d.Close())
						h.must(w.Close())
					}
					h.must(exact.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
		}
	}
	if !luSparseColumnRoute(1, 11, 10) || luSparseColumnRoute(1, 10, 10) || !luSparseColumnRoute(1<<31, 1, 120) {
		t.Fatal("strict or unsigned-wrapped column route")
	}
}

func TestLPLUEnteringColumnUnindexedDataAndDropPolicy(t *testing.T) {
	for _, scalar := range []string{"float", "rational", "pair"} {
		for _, sparse := range []bool{false, true} {
			t.Run(fmt.Sprint(scalar, "/", sparse), func(t *testing.T) {
				h := newLPPairHarness(t)
				b := lpExactIdentity(2)
				basis := []uint32{0, 1}
				s := lpFloatLUSettings(math.Inf(1))
				size := uint32(2)
				if sparse {
					size = 241
				}
				want := []*big.Rat{big.NewRat(1, 1), big.NewRat(7, 1)}
				wantIndex := []uint32{1, 0}
				if sparse {
					want[1].SetInt64(5)
					wantIndex = []uint32{0}
				}
				if scalar == "float" {
					a := lpFloatLUSource(t, b, basis)
					lpStaticMust(t, a.Set(0, 2, 1))
					m, e := NewFloatLU(a, basis, s)
					lpStaticMust(t, e)
					defer m.Close()
					d, w := NewIndexedFloatVector(size), NewIndexedFloatVector(2)
					d.Data[1] = 5
					w.Data[1] = 7
					lpStaticMust(t, m.SolveBd(2, d, w))
					for i := range want {
						lpFloatBlockNear(t, d.Data[i], want[i])
					}
					if !reflect.DeepEqual(d.Index, wantIndex) || !reflect.DeepEqual(w.Data, []float64{1, 7}) {
						t.Fatal("whole-data/indexed assignment distinction", d, w)
					}
				} else {
					a := lpExactLUStatic(t, h, b, basis)
					v := h.pair("1", "5")
					h.must(a.Set(0, 2, &v.X))
					d, w := NewIndexedRationalVector(h.m, size), NewIndexedRationalVector(h.m, 2)
					defer func() { h.must(d.Close()); h.must(w.Close()) }()
					h.must(h.m.Copy(&d.Data[1], &v.Y))
					h.must(h.m.SetString(&w.Data[1], "7"))
					if scalar == "pair" {
						m, e := NewPairLU(a, basis, s)
						h.must(e)
						defer func() { h.must(m.Close()) }()
						h.must(m.SolveBd(2, d, w))
						h.must(m.Close())
					} else {
						m, e := NewRationalLU(a, basis, s)
						h.must(e)
						defer func() { h.must(m.Close()) }()
						h.must(m.SolveBd(2, d, w))
						h.must(m.Close())
					}
					lpExactLUCheckVector(t, h, d.Data, want)
					if !reflect.DeepEqual(d.Index, wantIndex) {
						t.Fatal("exact index", d.Index)
					}
					h.must(d.Close())
					h.must(w.Close())
					lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
				}
			})
		}
	}
	for _, value := range []float64{0, math.Nextafter(1e-8, 0), 1e-8, -1e-8} {
		a := NewFloatStaticMatrix(1, 2)
		lpStaticMust(t, a.Set(0, 0, 1))
		lpStaticMust(t, a.Set(0, 1, value))
		s := lpFloatLUSettings(1)
		s.DropTolerance = 1e-8
		m, e := NewFloatLU(a, []uint32{0}, s)
		lpStaticMust(t, e)
		d, w := NewIndexedFloatVector(1), NewIndexedFloatVector(1)
		lpStaticMust(t, m.SolveBd(1, d, w))
		want := value
		if math.Abs(value) < 1e-8 {
			want = 0
		}
		if d.Data[0] != want || ((len(d.Index) == 0) != (want == 0)) {
			t.Fatal("strict configured drop", value, d)
		}
		lpStaticMust(t, m.Close())
	}
}

func TestLPLUEnteringColumnUpstreamSmallPreparation(t *testing.T) {
	// First retained prepare_entering(2,w) from test_small_lu, before its
	// still-open Suhl update sequence. Verify its independent basis solution.
	a := NewFloatStaticMatrix(3, 6)
	for _, c := range []struct {
		r, c uint32
		v    float64
	}{{0, 0, 1}, {0, 2, 3.9}, {2, 3, 11}, {0, 5, -3}, {1, 1, 4}, {1, 4, 7}, {2, 0, 1.8}, {2, 2, 5}, {2, 4, 2}, {2, 5, 8}} {
		lpStaticMust(t, a.Set(c.r, c.c, c.v))
	}
	m, e := NewFloatLU(a, []uint32{0, 1, 3}, lpFloatLUSettings(.7))
	lpStaticMust(t, e)
	defer m.Close()
	w := NewIndexedFloatVector(3)
	lpStaticMust(t, m.PrepareEntering(2, w))
	lpStaticMust(t, m.solveColumnIndexed(w))
	want := []*big.Rat{big.NewRat(39, 10), big.NewRat(0, 1), big.NewRat(-101, 550)}
	for i := range want {
		lpFloatBlockNear(t, w.Data[i], want[i])
	}
}
