package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactInfLimitIndependentPairOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	for _, op := range []string{"negative", "positive", "low-positive"} {
		for world := 0; world < 480; world++ {
			s, e := NewLARSolver(h.m)
			h.must(e)
			m := s.Core.RPrimal
			xx, xy := big.NewRat(int64(world%17-8), 11), big.NewRat(int64(world%19-9), 13)
			bx, by := big.NewRat(int64(world%23-11), 17), big.NewRat(int64(world%29-14), 19)
			if world%3 == 0 {
				xx.Set(bx)
			}
			if world%7 == 0 {
				xx.Set(bx)
				xy.Set(by)
			}
			tx, ty := big.NewRat(int64(world%13-6), 23), big.NewRat(int64(world%11-5), 29)
			scalar := big.NewRat(int64(world%17+1), int64(world%11+1))
			if op == "negative" {
				scalar.Neg(scalar)
			}
			x, bound, theta := h.pair(xx.RatString(), xy.RatString()), h.pair(bx.RatString(), by.RatString()), h.pair(tx.RatString(), ty.RatString())
			d := h.pair(scalar.RatString(), "0")
			unlimited := world%2 == 0
			wantUnlimited := unlimited
			wantX, wantY := new(big.Rat).Set(tx), new(big.Rat).Set(ty)
			cmp := xx.Cmp(bx)
			if cmp == 0 {
				cmp = xy.Cmp(by)
			}
			active := (op == "negative" && cmp > 0) || (op != "negative" && cmp < 0)
			wantLimited := (op == "negative" && cmp >= 0) || (op == "positive" && cmp <= 0)
			if active {
				qx := new(big.Rat).Quo(new(big.Rat).Sub(bx, xx), scalar)
				qy := new(big.Rat).Quo(new(big.Rat).Sub(by, xy), scalar)
				order := qx.Cmp(tx)
				if order == 0 {
					order = qy.Cmp(ty)
				}
				if unlimited || order <= 0 {
					wantX, wantY = qx, qy
				}
				wantUnlimited = false
			} else if cmp == 0 && op != "low-positive" {
				wantX.SetInt64(0)
				wantY.SetInt64(0)
				wantUnlimited = false
			}
			var limited bool
			var err error
			switch op {
			case "negative":
				limited, err = m.LimitInfOnBoundMNeg(&d.X, x, bound, theta, &unlimited)
			case "positive":
				limited, err = m.LimitInfOnBoundMPos(&d.X, x, bound, theta, &unlimited)
			default:
				err = m.LimitInfOnLowBoundMPos(&d.X, x, bound, theta, &unlimited)
			}
			h.must(err)
			if op != "low-positive" && limited != wantLimited {
				t.Fatal(op, world, limited, wantLimited)
			}
			if unlimited != wantUnlimited {
				t.Fatal("limit state", op, world, unlimited, wantUnlimited)
			}
			h.check(theta, wantX, wantY)
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactInfLimitLiveAliases(t *testing.T) {
	for _, op := range []string{"negative", "positive", "low-positive"} {
		for _, alias := range []string{"x", "bound", "scalar"} {
			h := newLPPairHarness(t)
			s, e := NewLARSolver(h.m)
			h.must(e)
			m := s.Core.RPrimal
			x, bound, theta, d := h.pair("1/7", "1/11"), h.pair("5/13", "-1/17"), h.pair("2/3", "3/5"), h.pair("2/3", "0")
			if op == "negative" {
				x, bound = bound, x
				h.must(h.m.SetString(&d.X, "-2/3"))
				h.must(h.m.SetString(&theta.X, "-2/3"))
			}
			qx := new(big.Rat).Mul(new(big.Rat).Sub(big.NewRat(5, 13), big.NewRat(1, 7)), big.NewRat(3, 2))
			qy := new(big.Rat).Mul(new(big.Rat).Sub(big.NewRat(-1, 17), big.NewRat(1, 11)), big.NewRat(3, 2))
			divisor := &d.X
			switch alias {
			case "x":
				theta = x
			case "bound":
				theta = bound
			case "scalar":
				divisor = &theta.X
			}
			unlimited := true
			switch op {
			case "negative":
				_, e = m.LimitInfOnBoundMNeg(divisor, x, bound, theta, &unlimited)
			case "positive":
				_, e = m.LimitInfOnBoundMPos(divisor, x, bound, theta, &unlimited)
			default:
				e = m.LimitInfOnLowBoundMPos(divisor, x, bound, theta, &unlimited)
			}
			h.must(e)
			h.check(theta, qx, qy)
			if unlimited {
				t.Fatal("alias state")
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactInfLimitPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		for _, op := range []string{"negative", "positive", "low-positive"} {
			success, failures, own, child := false, 0, 0, 0
			for ordinal := int64(0); ordinal < 400; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s, e := NewLARSolver(h.m)
					h.must(e)
					m := s.Core.RPrimal
					huge := strings.Repeat("1234567890", 12)
					x, bound, theta, d := h.pair(huge+"1/7", huge+"3/11"), h.pair(huge+"5/3", huge+"7/13"), h.pair("0", "0"), h.pair(huge+"9/17", "0")
					if op == "negative" {
						x, bound = bound, x
						h.must(h.m.Neg(&d.X))
					}
					unlimited := true
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "negative":
						_, err = m.LimitInfOnBoundMNeg(&d.X, x, bound, theta, &unlimited)
					case "positive":
						_, err = m.LimitInfOnBoundMPos(&d.X, x, bound, theta, &unlimited)
					default:
						err = m.LimitInfOnLowBoundMPos(&d.X, x, bound, theta, &unlimited)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() {
							own++
						}
						if m.PairCoreSolverBase.calls.pending() {
							child++
							if m.calls.pending() {
								t.Fatal("adopted base comparison")
							}
						}
						if e = m.LimitInfOnLowBoundMPos(&d.X, x, bound, theta, &unlimited); e != ErrExactVectorPending {
							t.Fatal("lost stopped primal", e)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 || own == 0 || child == 0 {
				t.Fatal("incomplete physical campaign", direct, op, success, failures, own, child)
			}
		}
	}
}

func TestLPExactInfLimitSourceSignAndPendingGuards(t *testing.T) {
	for _, op := range []string{"negative", "positive", "low-positive"} {
		for _, phase := range []string{"nil", "wrong-sign", "base-stop"} {
			h := newLPPairHarness(t)
			s, e := NewLARSolver(h.m)
			h.must(e)
			m := s.Core.RPrimal
			x, bound, theta, d := h.pair("0", "1/7"), h.pair("1", "1/11"), h.pair("17", "19"), h.pair("-2", "0")
			if op == "negative" {
				h.must(h.m.SetInt32(&d.X, 2))
			}
			unlimited := true
			want := ErrBounds
			if phase == "wrong-sign" {
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			if phase == "nil" {
				x = nil
			}
			if phase == "base-stop" {
				e = m.UpdateColumnInInfSet(99)
				if e != ErrBounds {
					t.Fatal(e)
				}
				want = ErrExactVectorPending
			}
			switch op {
			case "negative":
				_, e = m.LimitInfOnBoundMNeg(&d.X, x, bound, theta, &unlimited)
			case "positive":
				_, e = m.LimitInfOnBoundMPos(&d.X, x, bound, theta, &unlimited)
			default:
				e = m.LimitInfOnLowBoundMPos(&d.X, x, bound, theta, &unlimited)
			}
			if e != want {
				t.Fatal(op, phase, e, want)
			}
			if e != nil {
				if e = m.LimitInfOnLowBoundMPos(&d.X, bound, bound, theta, &unlimited); e != ErrExactVectorPending {
					t.Fatal("lost stop", e)
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactInfLimitPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"negative", "positive", "low-positive"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, e := NewLARSolver(h.m)
				h.must(e)
				m := s.Core.RPrimal
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				x, bound, theta, d := h.pair("1/7", "1/11"), h.pair("5/13", "-1/17"), h.pair("23", "29"), h.pair("2/3", "0")
				if op == "negative" {
					x, bound = bound, x
					h.must(h.m.Neg(&d.X))
				}
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
				h.budget.FailAfter(ordinal)
				_, err := h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					unlimited := true
					switch op {
					case "negative":
						_, err = m.LimitInfOnBoundMNeg(&d.X, x, bound, theta, &unlimited)
					case "positive":
						_, err = m.LimitInfOnBoundMPos(&d.X, x, bound, theta, &unlimited)
					default:
						err = m.LimitInfOnLowBoundMPos(&d.X, x, bound, theta, &unlimited)
					}
					if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
						t.Fatal("adopted older comparison", err)
					}
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
