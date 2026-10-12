package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPDiagonalTailAllocationFailureCensus(t *testing.T) {
	receipts, pairPrefixes := 0, 0
	for _, op := range []string{"constructor", "pair-constructor", "left", "right", "left-indexed", "right-indexed", "pair"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", direct), func(t *testing.T) {
				failures, successes := 0, 0
				for ordinal := int64(0); ordinal < 650; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						const divisor = "123456789123456789123456789123456789/170141183460469231731687303715884105727"
						coefficient := h.pair(divisor, "0")
						pairs := []RationalPair{NewRationalPair()}
						h.must(h.m.SetInt64(&pairs[0].X, 1))
						h.must(h.m.SetInt64(&pairs[0].Y, 2))
						v := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1)})
						var m *RationalDiagonalMatrix
						var pm *PairDiagonalMatrix
						var err error
						if op != "constructor" && op != "pair-constructor" {
							if op == "pair" {
								pm, err = NewPairDiagonalMatrix(h.m, 0, &coefficient.X)
								m = pm.RationalDiagonalMatrix
							} else {
								m, err = NewRationalDiagonalMatrix(h.m, 0, &coefficient.X)
							}
							h.must(err)
						}
						h.budget.FailAfter(ordinal)
						switch op {
						case "constructor":
							m, err = NewRationalDiagonalMatrix(h.m, 0, &coefficient.X)
						case "pair-constructor":
							pm, err = NewPairDiagonalMatrix(h.m, 0, &coefficient.X)
							m = pm.RationalDiagonalMatrix
						case "left":
							err = m.ApplyLeft(v.Data)
						case "right":
							err = m.ApplyRight(v.Data)
						case "left-indexed":
							err = m.ApplyLeftToT(v, nil)
						case "right-indexed":
							err = m.ApplyRightIndexed(v)
						case "pair":
							err = pm.ApplyLeft(pairs)
						}
						h.budget.FailAfter(-1)
						if err == nil {
							done = true
							successes++
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if m.calls.pending() {
								receipts++
							}
							if e := m.Conjugate(NewPermutation(1)); e != ErrExactVectorPending {
								t.Fatalf("stopped tail %v", e)
							}
							if e := m.ApplyLeft(v.Data); e != ErrExactVectorPending {
								t.Fatalf("stopped dense %v", e)
							}
							if e := m.ApplyRightIndexed(v); e != ErrExactVectorPending {
								t.Fatalf("stopped indexed %v", e)
							}
							if v.calls.pending() {
								t.Fatal("caller adopted tail receipt")
							}
						}
						if err == nil {
							d, _ := new(big.Rat).SetString(divisor)
							lpSparseRationalCheck(t, h, &m.value, d)
							lpSparseRationalCheck(t, h, &coefficient.X, d)
							if sourceDebugTests {
								lpSparseRationalCheck(t, h, &m.inverse, new(big.Rat).Inv(d))
							}
						}
						h.must(m.Close())
						d, _ := new(big.Rat).SetString(divisor)
						want := new(big.Rat).Inv(d)
						if op == "pair" {
							if err == nil {
								h.check(&pairs[0], want, new(big.Rat).Mul(want, big.NewRat(2, 1)))
							}
							if err != nil && direct {
								x, ex := h.m.String(&pairs[0].X)
								y, ey := h.m.String(&pairs[0].Y)
								xr, okx := new(big.Rat).SetString(x)
								yr, oky := new(big.Rat).SetString(y)
								if ex == nil && ey == nil && okx && oky && xr.Cmp(want) == 0 && yr.Cmp(big.NewRat(2, 1)) == 0 {
									pairPrefixes++
								}
							}
						}
						if err == nil && op != "constructor" && op != "pair-constructor" && op != "pair" {
							lpSparseRationalCheck(t, h, &v.Data[0], want)
						}
						lpExactVectorNamedCensus(t, h, []*numeral.Rational{&v.Data[0], &pairs[0].X, &pairs[0].Y})
						h.must(pairs[0].Del(h.m))
						h.must(v.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				// A warm pooled native constructor can copy both coefficients
				// without requesting new backing. Do not invent a refusal there.
				warmConstructor := !direct && !sourceDebugTests && (op == "constructor" || op == "pair-constructor")
				if (failures == 0 && !warmConstructor) || successes != 1 {
					t.Fatalf("incomplete campaign %d %d", failures, successes)
				}
			})
		}
	}
	if receipts == 0 {
		t.Fatal("no actual stopped numerical receipt")
	}
	if pairPrefixes == 0 {
		t.Fatal("no real pair X-before-Y failure prefix")
	}
}

func TestLPDiagonalTailBorrowedStoppedInput(t *testing.T) {
	for _, direct := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("direct=%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				d := h.pair("2", "0")
				m, err := NewRationalDiagonalMatrix(h.m, 0, &d.X)
				h.must(err)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				v := NewIndexedRationalVector(h.m, 1)
				h.must(v.SetValue(&p.X, 0))
				h.budget.FailAfter(ordinal)
				err = v.Add(0, &p.Y)
				h.budget.FailAfter(-1)
				if v.calls.pending() {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					matched = true
					before := h.m.RawInvocationState()
					if err := m.ApplyRightIndexed(v); err != ErrExactVectorPending {
						t.Fatalf("right stopped input %v", err)
					}
					if err := m.ApplyLeftToT(v, nil); err != ErrExactVectorPending {
						t.Fatalf("left stopped input %v", err)
					}
					if m.work != nil || m.calls.pending() {
						t.Fatal("tail adopted caller receipt")
					}
					h.must(m.Close())
					if h.m.RawInvocationState() != before {
						t.Fatal("tail consumed caller receipt")
					}
				} else {
					h.must(m.Close())
				}
				h.must(v.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual stopped input")
		}
	}
}
