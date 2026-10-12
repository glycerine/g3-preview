package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPExactDenseBlockApplicationAllocationFailureCensus(t *testing.T) {
	receipts, pairPrefixes, assignPrefixes := 0, 0, 0
	for _, op := range []string{"left", "right", "left-indexed", "right-indexed", "pair", "pair-constructor", "left-prefix", "right-prefix", "left-indexed-prefix", "right-indexed-prefix", "pair-prefix"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", direct), func(t *testing.T) {
				failures, successes := 0, 0
				for ordinal := int64(0); ordinal < 1000; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						start := uint32(0)
						route := strings.TrimSuffix(op, "-prefix")
						if route != op {
							start = 1
						}
						n := start + 2
						xText, yText := "1"+strings.Repeat("0", 120)+"/7", "2"+strings.Repeat("0", 120)+"/13"
						p := h.pair(xText, yText)
						x, _ := new(big.Rat).SetString(xText)
						y, _ := new(big.Rat).SetString(yText)
						if x.Denom().Cmp(big.NewInt(7)) != 0 || y.Denom().Cmp(big.NewInt(13)) != 0 {
							t.Fatal("failure fixture lost its fractions")
						}
						parent := lpNewExactSparse(t, h, n)
						a := lpExactIdentity(int(n))
						for i := start; i < n; i++ {
							for j := start; j < n; j++ {
								value := &p.X
								a[i][j].Set(x)
								if i != j {
									value = &p.Y
									a[i][j].Set(y)
								}
								h.must(parent.AddNewElement(i, j, value))
							}
						}
						v := NewIndexedRationalVector(h.m, n)
						pw := NewIndexedPairVector(h.m, n)
						b, by := make([]*big.Rat, n), make([]*big.Rat, n)
						for i := uint32(0); i < n; i++ {
							b[i] = big.NewRat(int64(i+2), 1)
							by[i] = big.NewRat(-int64(i+3), 1)
							if i < start {
								b[i].Set(x)
								by[i].Set(y)
							}
							h.must(h.m.SetString(&v.Data[i], b[i].RatString()))
							h.must(h.m.SetString(&pw.Data[i].X, b[i].RatString()))
							h.must(h.m.SetString(&pw.Data[i].Y, by[i].RatString()))
						}
						for i := n; i > 0; {
							i--
							v.Index = append(v.Index, i)
						}
						var m *PairSquareDenseSubmatrix
						var err error
						if route != "pair-constructor" {
							m, err = NewPairSquareDenseSubmatrix(parent, start)
							h.must(err)
						}
						want := lpExactMatrixVector(a, b, strings.HasPrefix(route, "right"))
						wantY := lpExactMatrixVector(a, by, false)
						h.budget.FailAfter(ordinal)
						switch route {
						case "pair-constructor":
							m, err = NewPairSquareDenseSubmatrix(parent, start)
						case "left":
							err = m.RationalSquareDenseSubmatrix.ApplyLeft(v.Data)
						case "right":
							err = m.ApplyRight(v.Data)
						case "left-indexed":
							err = m.ApplyLeftToT(v, nil)
						case "right-indexed":
							err = m.ApplyRightIndexed(v)
						case "pair":
							err = m.ApplyLeft(pw.Data)
						}
						h.budget.FailAfter(-1)
						if m == nil {
							t.Fatal("missing owning result", err)
						}
						if err == nil {
							done = true
							successes++
							if route == "pair" {
								for i := range want {
									h.check(&pw.Data[i], want[i], wantY[i])
								}
							} else if route == "pair-constructor" {
								for i := start; i < n; i++ {
									for j := start; j < n; j++ {
										lpSparseRationalCheck(t, h, &m.Data[(i-start)*m.Dim+j-start], a[i][j])
									}
								}
							} else {
								for i := range want {
									lpSparseRationalCheck(t, h, &v.Data[i], want[i])
								}
							}
							owned := lpExactDenseCells(m.RationalSquareDenseSubmatrix)
							for i := range v.Data {
								owned = append(owned, &v.Data[i], &pw.Data[i].X, &pw.Data[i].Y)
							}
							lpExactVectorNamedCensus(t, h, owned)
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							owners := 0
							for _, pending := range []bool{m.calls.pending(), m.WorkVector.calls.pending(), v.calls.pending()} {
								if pending {
									owners++
									receipts++
								}
							}
							if owners > 1 {
								t.Fatal("duplicated invocation ownership", owners)
							}
							for _, f := range []func() error{
								func() error { return m.RationalSquareDenseSubmatrix.ApplyLeft(v.Data) },
								func() error { return m.ApplyRight(v.Data) }, func() error { return m.ApplyLeftToT(v, nil) },
								func() error { return m.ApplyRightIndexed(v) }, func() error { return m.ApplyLeft(pw.Data) },
								func() error { return m.Conjugate(NewPermutation(n)) },
							} {
								if e := f(); e != ErrExactVectorPending {
									t.Fatal("stopped application", e)
								}
							}
							if route == "right-indexed" && reflect.DeepEqual(v.Index, m.WorkVector.Index) && !reflect.DeepEqual(v.Index, []uint32{1, 0}) {
								assignPrefixes++
							}
						}
						h.must(m.Close())
						if err != nil && direct && route == "pair" {
							// Matrix Close releases only its receipts; caller values stay owned.
							xs, ex := h.m.String(&pw.Data[0].X)
							ys, ey := h.m.String(&pw.Data[0].Y)
							xx, okx := new(big.Rat).SetString(xs)
							yy, oky := new(big.Rat).SetString(ys)
							if ex == nil && ey == nil && okx && oky && xx.Cmp(want[0]) == 0 && yy.Cmp(by[0]) == 0 {
								pairPrefixes++
							}
						}
						h.must(v.Close())
						h.must(pw.Close())
						h.must(parent.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if successes != 1 || (direct && failures == 0) {
					t.Fatal("incomplete failure campaign", failures, successes)
				}
			})
		}
	}
	if receipts == 0 || pairPrefixes == 0 || assignPrefixes == 0 {
		t.Fatal("missing actual receipt/publication prefix", receipts, pairPrefixes, assignPrefixes)
	}
}

func TestLPExactDenseBlockLeavesStoppedIndexedCallerOwned(t *testing.T) {
	for _, direct := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 100 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("direct=%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				parent := lpNewExactSparse(t, h, 2)
				m, err := NewRationalSquareDenseSubmatrix(parent, 0)
				h.must(err)
				p := h.pair("1"+strings.Repeat("0", 120)+"/7", "2"+strings.Repeat("0", 120)+"/13")
				v := NewIndexedRationalVector(h.m, 2)
				h.must(h.m.Copy(&v.Data[0], &p.Y))
				v.Index = []uint32{0}
				h.budget.FailAfter(ordinal)
				err = v.Add(0, &p.X)
				h.budget.FailAfter(-1)
				if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if v.calls.pending() {
					matched = true
					before := h.m.RawInvocationState()
					if err = m.ApplyLeftToT(v, nil); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err = m.ApplyRightIndexed(v); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if m.work != nil || m.calls.pending() {
						t.Fatal("adopted stopped caller")
					}
					h.must(m.Close())
					if h.m.RawInvocationState() != before {
						t.Fatal("closed caller invocation")
					}
				}
				h.must(m.Close())
				h.must(v.Close())
				h.must(parent.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual stopped indexed caller")
		}
	}
}
