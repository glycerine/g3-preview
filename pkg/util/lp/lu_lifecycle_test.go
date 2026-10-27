package lp

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestLPLUFactorizationReplacementIndependentOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 12; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			n := 1 + world%6
			b := lpExactIdentity(n)
			basis := make([]uint32, n)
			for i := range b {
				basis[i] = uint32(i)
				for j := range b[i] {
					if i == j {
						b[i][j].SetInt64(int64(40*n + i))
					} else {
						b[i][j].SetFrac64(int64(i+j+1), 7)
					}
				}
			}
			a := lpExactLUStatic(t, h, b, basis)
			fa := lpFloatLUSource(t, b, basis)
			s := lpFloatLUSettings(math.Inf(1))
			if world >= 6 {
				s.DensityThreshold = 0
			}
			var f *FloatLU
			var r *RationalLU
			var p *PairLU
			lpStaticMust(t, InitFloatFactorization(&f, fa, basis, s))
			h.must(InitRationalFactorization(&r, a, basis, s))
			h.must(InitPairFactorization(&p, a, basis, s))
			defer func() { lpStaticMust(t, f.Close()) }()
			defer func() { h.must(r.Close()); h.must(p.Close()) }()
			oldF, oldR, oldP := f, r, p
			x, y := make([]*big.Rat, n), make([]*big.Rat, n)
			for i := range x {
				x[i] = big.NewRat(int64(i+2), 11)
				y[i] = big.NewRat(int64(i-3), 13)
			}
			lpFloatLUCheckSolutions(t, f, b, x)
			lpExactLUUpdateCheckSolutions(t, h, r, nil, b, x, y)
			lpExactLUUpdateCheckSolutions(t, h, nil, p, b, x, y)
			reverse := append([]uint32(nil), basis...)
			for i := range reverse {
				reverse[i] = uint32(n - i - 1)
			}
			lpStaticMust(t, InitFloatFactorization(&f, fa, reverse, s))
			h.must(InitRationalFactorization(&r, a, reverse, s))
			h.must(InitPairFactorization(&p, a, reverse, s))
			if !oldF.closed || !oldR.closed || !oldP.closed || f == oldF || r == oldR || p == oldP {
				t.Fatal("old owner not retired before replacement")
			}
			if err := oldF.SolveYB(nil); err != ErrLUClosed {
				t.Fatal(err)
			}
			if err := oldR.SolveYB(nil); err != ErrLUClosed {
				t.Fatal(err)
			}
			if err := oldP.SolveYB(nil); err != ErrLUClosed {
				t.Fatal(err)
			}
			reversed := lpExactIdentity(n)
			for i := range b {
				for j := range b[i] {
					reversed[i][j].Set(b[i][n-j-1])
				}
			}
			lpFloatLUCheckSolutions(t, f, reversed, x)
			lpExactLUUpdateCheckSolutions(t, h, r, nil, reversed, x, y)
			lpExactLUUpdateCheckSolutions(t, h, nil, p, reversed, x, y)
			// Constructor failure retains the attempted replacement as a stopped owner.
			previousF, previousR, previousP := f, r, p
			errF := InitFloatFactorization(&f, fa, nil, s)
			errR := InitRationalFactorization(&r, a, nil, s)
			errP := InitPairFactorization(&p, a, nil, s)
			if errF != ErrBounds || errR != ErrBounds || errP != ErrBounds {
				t.Fatal(errF, errR, errP)
			}
			if !previousF.closed || !previousR.closed || !previousP.closed || !f.stopped || !r.stopped || !p.stopped {
				t.Fatal("failed replacement ownership")
			}
			lpStaticMust(t, f.Close())
			h.must(r.Close())
			h.must(p.Close())
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			lpExactLUCheckSource(t, h, a, b, basis)
		})
	}
}

func TestLPExactLUFactorizationReplacementAllocationCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, pair := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(pair, "/", direct), func(t *testing.T) {
				success := false
				failures := 0
				for ordinal := int64(0); ordinal < 800 && !success; ordinal++ {
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						b := lpExactIdentity(2)
						basis := []uint32{0, 1}
						a := lpExactLUStatic(t, h, b, basis)
						s := lpFloatLUSettings(math.Inf(1))
						var r *RationalLU
						var p *PairLU
						var old *exactLUState
						var err error
						if pair {
							h.must(InitPairFactorization(&p, a, basis, s))
							old = p.exactLUState
						} else {
							h.must(InitRationalFactorization(&r, a, basis, s))
							old = r.exactLUState
						}
						defer func() {
							if pair {
								h.must(p.Close())
							} else {
								h.must(r.Close())
							}
						}()
						v := h.pair("1"+strings.Repeat("0", 31)+"1/7", "2"+strings.Repeat("0", 47)+"3/13")
						for i := range b {
							for j := range b[i] {
								value := &v.X
								if i != j {
									value = &v.Y
								}
								if i == j {
									// Source operator() replaces a coefficient; Set appends.
									h.must(h.m.Copy(&a.Rows[i][0].Value, value))
								} else {
									h.must(a.Set(uint32(i), uint32(j), value))
								}
								text, e := h.m.String(value)
								h.must(e)
								b[i][j] = mustLPRat(t, text)
							}
						}
						h.budget.FailAfter(ordinal)
						if pair {
							err = InitPairFactorization(&p, a, basis, s)
						} else {
							err = InitRationalFactorization(&r, a, basis, s)
						}
						h.budget.FailAfter(-1)
						if !old.closed {
							t.Fatal("old factorization survives constructor attempt")
						}
						var state *exactLUState
						if pair {
							state = p.exactLUState
						} else {
							state = r.exactLUState
						}
						if state == old {
							t.Fatal("old slot not replaced")
						}
						if err == nil {
							success = true
							x := []*big.Rat{big.NewRat(3, 7), big.NewRat(-2, 11)}
							lpExactLUUpdateCheckSolutions(t, h, r, p, b, x, x)
						} else {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) || !state.stopped {
								t.Fatal(err, state.stopped)
							}
							if pair {
								err = InitPairFactorization(&p, a, basis, s)
							} else {
								err = InitRationalFactorization(&r, a, basis, s)
							}
							if err != nil {
								// A source Add/SubMul refusal can consume manager scratch.
								// Cleanup owns the failed call; it cannot repair that pointer.
								// See numeral/TestIntegerRawExecutingOwnerRefusesFinalizeAndClose.
								if !errors.Is(err, arena.ErrInvalidSlot) || (h.m.Z.Tmp[0] != 0 && h.m.Z.Tmp[1] != 0) {
									t.Fatal("unexpected re-entry error", err)
								}
								t.Logf("source-invalid scratch after refusal: tmp=%v", h.m.Z.Tmp)
							}
							if !state.closed {
								t.Fatal("stopped replacement was not retired on recovery")
							}
						}
						if pair {
							h.must(p.Close())
						} else {
							h.must(r.Close())
						}
						lpExactLUCheckSource(t, h, a, b, basis)
						lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
					})
				}
				if !success || (direct && failures == 0) {
					t.Fatal("incomplete replacement walk", success, failures)
				}
				t.Logf("physical refusals=%d", failures)
			})
		}
	}
}
