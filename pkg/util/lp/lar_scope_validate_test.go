package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
)

func TestLARScopeEquationIndependentPairOracle(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(73182, 17))
	for world := 0; world < 160; world++ {
		a := big.NewRat(int64(rng.IntN(13)-6), int64(rng.IntN(7)+1))
		b := big.NewRat(int64(rng.IntN(17)-8), int64(rng.IntN(11)+1))
		s := lpNewRemovalModel(t, h, a.RatString(), b.RatString(), true)
		c := s.Core
		values := [2][2]*big.Rat{}
		for j := 0; j < 2; j++ {
			for k := 0; k < 2; k++ {
				values[j][k] = big.NewRat(int64(rng.IntN(19)-9), int64(rng.IntN(13)+1))
			}
			h.must(c.RX.Data[j].Copy(h.m, h.pair(values[j][0].RatString(), values[j][1].RatString())))
		}
		want := [2]*big.Rat{}
		for k := 0; k < 2; k++ {
			want[k] = new(big.Rat).Add(new(big.Rat).Mul(a, values[0][k]), new(big.Rat).Mul(b, values[1][k]))
		}
		for j := 2; j < 4; j++ {
			h.must(c.RX.Data[j].Copy(h.m, h.pair(want[0].RatString(), want[1].RatString())))
		}
		for j := range c.RightSides.Data {
			h.must(c.RightSides.Data[j].Copy(h.m, h.pair("17", "19")))
		}
		ok, err := s.AxIsCorrect()
		h.must(err)
		if !ok {
			t.Fatal("independent homogeneous equation", world)
		}
		h.must(h.m.SetString(&c.RX.Data[3].Y, new(big.Rat).Add(want[1], big.NewRat(1, 17)).RatString()))
		ok, err = s.RowIsCorrect(0)
		h.must(err)
		if !ok {
			t.Fatal("unrelated row changed")
		}
		ok, err = s.AxIsCorrect()
		h.must(err)
		if ok {
			t.Fatal("epsilon residual accepted")
		}
		h.must(h.m.SetString(&c.RX.Data[3].Y, want[1].RatString()))
		h.must(h.m.SetString(&c.RX.Data[3].X, new(big.Rat).Add(want[0], big.NewRat(1, 19)).RatString()))
		ok, err = s.RowIsCorrect(1)
		h.must(err)
		if ok {
			t.Fatal("rational residual accepted")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARScopeInfSetIndependentLexicographicOracle(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(71823, 381))
	lex := func(a, b [2]*big.Rat) int {
		if c := a[0].Cmp(b[0]); c != 0 {
			return c
		}
		return a[1].Cmp(b[1])
	}
	for world := 0; world < 160; world++ {
		f := lpNewPairCoreFixture(t, h, 0, 4)
		m, err := NewPairCoreSolverBase(f.input())
		h.must(err)
		m.Names = lpFeasibleNames{}
		h.must(m.InfSet.Resize(6))
		h.must(m.InfSet.Insert(5))
		feasible := [4]bool{}
		for j := 0; j < 4; j++ {
			kind := []ColumnType{ColumnFree, ColumnFixed, ColumnBoxed, ColumnLowBound, ColumnUpperBound}[(world+j)%5]
			low := [2]*big.Rat{big.NewRat(int64(rng.IntN(11)-5), 3), big.NewRat(int64(rng.IntN(7)-3), 5)}
			up := [2]*big.Rat{new(big.Rat).Add(low[0], big.NewRat(int64(rng.IntN(4)+1), 7)), big.NewRat(int64(rng.IntN(7)-3), 5)}
			if kind == ColumnFixed {
				up = [2]*big.Rat{new(big.Rat).Set(low[0]), new(big.Rat).Set(low[1])}
			}
			x := [2]*big.Rat{new(big.Rat).Set(low[0]), big.NewRat(int64(rng.IntN(11)-5), 5)}
			if world%3 == 0 {
				x[0].Set(up[0])
			}
			for k, p := range []*RationalPair{&f.low.Data[j], &f.up.Data[j], &f.x.Data[j]} {
				v := [][2]*big.Rat{low, up, x}[k]
				h.must(p.Copy(h.m, h.pair(v[0].RatString(), v[1].RatString())))
			}
			f.types[j] = kind
			feasible[j] = true
			if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnLowBound) && lex(x, low) < 0 {
				feasible[j] = false
			}
			if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnUpperBound) && lex(x, up) > 0 {
				feasible[j] = false
			}
			if !feasible[j] {
				h.must(m.InfSet.Insert(uint32(j)))
			}
		}
		ok, err := m.InfSetIsCorrect()
		h.must(err)
		if !ok {
			t.Fatal("independent membership oracle", world)
		}
		wrong := world % 4
		if feasible[wrong] {
			h.must(m.InfSet.Insert(uint32(wrong)))
		} else {
			m.InfSet.Erase(uint32(wrong))
		}
		// Also corrupt a lower column: the first diagnostic must still be the higher index.
		if wrong > 0 {
			if feasible[0] {
				h.must(m.InfSet.Insert(0))
			} else {
				m.InfSet.Erase(0)
			}
		}
		output := lpCaptureFeasibleStdout(t, func() { ok, err = m.InfSetIsCorrect() })
		h.must(err)
		if ok || !strings.HasPrefix(output, fmt.Sprintf("column_index = %d,", wrong)) {
			t.Fatal(output)
		}
		suffix := "belongs_to_set = 0\ninf\n"
		if feasible[wrong] {
			suffix = "belongs_to_set = 1\nfeas\n"
		}
		if !strings.HasSuffix(output, suffix) {
			t.Fatal(output, suffix)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARScopeValidatorMetadataAndStoppedGuards(t *testing.T) {
	for _, which := range []string{"row", "column", "empty", "inf-x", "inf-types", "inf-print", "inf-stdout"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			savedX := c.RX.Data
			var ok bool
			var err error
			switch which {
			case "row":
				ok, err = s.RowIsCorrect(99)
			case "column":
				c.RA.Rows[0][0].Column = 99
				ok, err = s.AxIsCorrect()
			case "empty":
				empty, e := NewLARSolver(h.m)
				h.must(e)
				ok, err = empty.AxIsCorrect()
				h.must(empty.Close())
			case "inf-x":
				c.RX.Data = nil
				ok, err = c.RPrimal.InfSetIsCorrect()
			case "inf-types":
				c.Bounds.Types.values = nil
				ok, err = c.RPrimal.InfSetIsCorrect()
			case "inf-print":
				h.must(c.RPrimal.InfSet.Insert(2))
				c.RPrimal.Names = nil
				ok, err = c.RPrimal.InfSetIsCorrect()
			case "inf-stdout":
				h.must(c.RPrimal.InfSet.Insert(2))
				old := os.Stdout
				closed, e := os.CreateTemp("/mnt/oldrog/tmp", "g3-closed-inf-")
				h.must(e)
				h.must(closed.Close())
				os.Stdout = closed
				ok, err = c.RPrimal.InfSetIsCorrect()
				os.Stdout = old
				h.must(os.Remove(closed.Name()))
			}
			if which == "empty" {
				h.must(err)
				if !ok {
					t.Fatal("empty model")
				}
			} else {
				if err == nil || ok {
					t.Fatal(ok, err)
				}
				if strings.HasPrefix(which, "inf-") {
					if _, e := c.RPrimal.InfSetIsCorrect(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
				} else {
					if _, e := s.RowIsCorrect(0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if _, e := s.AxIsCorrect(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
				}
			}
			c.RX.Data = savedX
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARScopeValidatorPhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"row", "ax", "inf"} {
			success := false
			failures, owned := 0, 0
			for ordinal := int64(0); ordinal < 300; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					a := strings.Repeat("9876543210", 100) + "/7"
					b := strings.Repeat("1234567890", 100) + "/13"
					s := lpNewRemovalModel(t, h, a, "-5/7", false)
					c := s.Core
					h.must(h.m.SetString(&c.RX.Data[0].X, b))
					if op == "inf" {
						h.must(s.Bounds.Low.Set(0, h.pair(a, "0")))
						h.must(s.Bounds.Types.Set(0, ColumnLowBound))
						h.must(c.RPrimal.InfSet.Insert(0))
					} else {
						h.must(h.m.SetString(&c.RX.Data[2].X, new(big.Rat).Mul(mustLPRat(t, a), mustLPRat(t, b)).RatString()))
						h.must(h.m.SetString(&c.RX.Data[0].Y, a))
						h.must(h.m.SetString(&c.RX.Data[2].Y, new(big.Rat).Mul(mustLPRat(t, a), mustLPRat(t, a)).RatString()))
					}
					h.budget.FailAfter(ordinal)
					var ok bool
					var err error
					switch op {
					case "row":
						ok, err = s.RowIsCorrect(0)
					case "ax":
						ok, err = s.AxIsCorrect()
					case "inf":
						ok, err = c.RPrimal.InfSetIsCorrect()
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if !ok {
							t.Fatal("independent oracle rejected")
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if op == "inf" {
							if s.calls.pending() {
								t.Fatal("outer adopted core comparison")
							}
							if c.RPrimal.PairCoreSolverBase.calls.pending() {
								owned++
							}
							if _, e := c.RPrimal.InfSetIsCorrect(); e != ErrExactVectorPending {
								t.Fatal(e)
							}
						} else {
							if s.calls.pending() || s.work != nil || s.rationalWork != nil {
								owned++
							}
							if _, e := s.AxIsCorrect(); e != ErrExactVectorPending {
								t.Fatal(e)
							}
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 || owned == 0 {
				t.Fatal("incomplete campaign", direct, op, success, failures, owned)
			}
		}
	}
}

func TestLARScopeOperationsLeaveOlderComparisonOwned(t *testing.T) {
	for _, op := range []string{"pop", "row", "inf"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				a := strings.Repeat("9876543210", 100) + "/7"
				b := strings.Repeat("1234567890", 100) + "/13"
				s := lpNewRemovalModel(t, h, "2/3", "5/7", false)
				c := s.Core
				v := h.pair(a, b)
				h.must(h.m.Copy(&c.RX.Data[0].X, &v.Y))
				h.must(s.Bounds.Low.Set(0, v))
				h.must(s.Bounds.Types.Set(0, ColumnLowBound))
				h.must(c.RPrimal.InfSet.Insert(0))
				if op == "pop" {
					h.must(s.Push())
				}
				var caller exactEtaCalls
				before, err := caller.begin(h.m)
				h.must(err)
				h.budget.FailAfter(ordinal)
				_, err = h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					switch op {
					case "pop":
						err = s.Pop(1)
					case "row":
						_, err = s.RowIsCorrect(0)
					case "inf":
						_, err = c.RPrimal.InfSetIsCorrect()
					}
					if err == nil || s.calls.comparison != 0 || c.RPrimal.PairCoreSolverBase.calls.comparison != 0 {
						t.Fatal("adopted older comparison", err)
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(s.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}
