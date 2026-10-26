package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactBasisDispatchGuardsAndZeroDirection(t *testing.T) {
	for _, phase := range []string{"nil", "index", "base-stop", "unknown", "free-zero", "low-zero", "upper-zero", "boxed-zero"} {
		h := newLPPairHarness(t)
		s, e := NewLARSolver(h.m)
		h.must(e)
		_, e = s.AddVar(31)
		h.must(e)
		m := s.Core.RPrimal
		theta, d := h.pair("-2", "3"), h.pair("0", "0")
		multiplier := &d.X
		j := uint32(0)
		kind := ColumnFree
		want := error(nil)
		switch phase {
		case "nil":
			multiplier = nil
			want = ErrBounds
		case "index":
			j = 99
			want = ErrBounds
		case "base-stop":
			if e = m.UpdateColumnInInfSet(99); e != ErrBounds {
				t.Fatal(e)
			}
			want = ErrExactVectorPending
		case "unknown":
			kind = ColumnType(99)
			if sourceDebugEnabled {
				want = ErrAssertion
			}
		case "low-zero":
			kind = ColumnLowBound
		case "upper-zero":
			kind = ColumnUpperBound
		case "boxed-zero":
			kind = ColumnBoxed
			want = numeral.ErrDivideByZero
			if sourceDebugEnabled {
				want = ErrAssertion
			}
		}
		h.must(s.Bounds.Types.Set(0, kind))
		unlimited := false
		if e = m.LimitThetaOnBasisColumn(j, multiplier, theta, &unlimited); e != want {
			t.Fatal(phase, e, want)
		}
		if e == nil {
			h.check(theta, big.NewRat(0, 1), big.NewRat(0, 1))
			if unlimited {
				t.Fatal("changed limited flag")
			}
		} else if e = m.ZeroHarrisEps(); e != ErrExactVectorPending {
			t.Fatal("lost stop", e)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBasisDispatchMultiplierOutputAlias(t *testing.T) {
	for _, kind := range []ColumnType{ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed} {
		for _, feasible := range []bool{false, true} {
			for _, positive := range []bool{false, true} {
				h := newLPPairHarness(t)
				s, e := NewLARSolver(h.m)
				h.must(e)
				_, e = s.AddVar(31)
				h.must(e)
				m := s.Core.RPrimal
				h.must(s.Bounds.Types.Set(0, kind))
				h.must(m.InfSet.Resize(1))
				if !feasible {
					h.must(m.InfSet.Insert(0))
				}
				h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-1", "0")))
				h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("1", "0")))
				theta := h.pair("2/3", "1/13")
				if !positive {
					h.must(h.m.Neg(&theta.X))
				}
				want := big.NewRat(3, 2)
				if (kind == ColumnLowBound && positive) || (kind == ColumnUpperBound && !positive) {
					want = big.NewRat(2, 3)
					if !positive {
						want.Neg(want)
					}
				}
				unlimited := true
				h.must(m.LimitThetaOnBasisColumn(0, &theta.X, theta, &unlimited))
				wantY := big.NewRat(0, 1)
				if unlimited {
					wantY = big.NewRat(1, 13)
				}
				h.check(theta, want, wantY)
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPExactBasisDispatchIndependentPairOracle(t *testing.T) {
	for _, kind := range []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed} {
		for _, feasible := range []bool{false, true} {
			for _, positive := range []bool{false, true} {
				op := "none"
				switch kind {
				case ColumnLowBound:
					if feasible {
						if !positive {
							op = "feasible-negative"
						}
					} else if positive {
						op = "low-positive"
					} else {
						op = "low-negative"
					}
				case ColumnUpperBound:
					if feasible {
						if positive {
							op = "feasible-positive"
						}
					} else if positive {
						op = "upper-positive"
					} else {
						op = "upper-negative"
					}
				case ColumnBoxed, ColumnFixed:
					if feasible {
						if positive {
							op = "feasible-positive"
						} else {
							op = "feasible-negative"
						}
					} else if positive {
						op = "boxed-positive"
					} else {
						op = "boxed-negative"
					}
				}
				h := newLPPairHarness(t)
				for world := 0; world < 120; world++ {
					s, e := NewLARSolver(h.m)
					h.must(e)
					_, e = s.AddVar(31)
					h.must(e)
					m := s.Core.RPrimal
					h.must(s.Bounds.Types.Set(0, kind))
					h.must(m.InfSet.Resize(1))
					if !feasible {
						h.must(m.InfSet.Insert(0))
					}
					x := [2]*big.Rat{big.NewRat(int64(world%19-9), 7), big.NewRat(int64(world%13-6), 11)}
					low := [2]*big.Rat{big.NewRat(-3, 7), big.NewRat(-2, 11)}
					upper := [2]*big.Rat{big.NewRat(5, 7), big.NewRat(3, 11)}
					if kind == ColumnFixed {
						upper[0].Set(low[0])
						upper[1].Set(low[1])
					}
					if world%5 == 0 {
						x[0].Set(low[0])
						x[1].Set(low[1])
					}
					if world%7 == 0 {
						x[0].Set(upper[0])
						x[1].Set(upper[1])
					}
					cmp := func(a, b [2]*big.Rat) int {
						c := a[0].Cmp(b[0])
						if c == 0 {
							c = a[1].Cmp(b[1])
						}
						return c
					}
					set := func(dst *RationalPair, value [2]*big.Rat) {
						h.must(h.m.SetString(&dst.X, value[0].RatString()))
						h.must(h.m.SetString(&dst.Y, value[1].RatString()))
					}
					set(&s.Core.RX.Data[0], x)
					set(&(*m.LowBounds)[0], low)
					set(&(*m.UpperBounds)[0], upper)
					scalar := big.NewRat(int64(world%11+1), 13)
					if !positive {
						scalar.Neg(scalar)
					}
					d := h.pair(scalar.RatString(), "0")
					eps := big.NewRat(int64(world%9-4), 100)
					h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, eps.RatString()))
					old := [2]*big.Rat{big.NewRat(int64(world%17-8), 19), big.NewRat(int64(world%23-11), 29)}
					theta := h.pair(old[0].RatString(), old[1].RatString())
					if world%4 == 0 {
						theta = &s.Core.RX.Data[0]
						old = x
					}
					if world%4 == 1 {
						theta = &(*m.LowBounds)[0]
						old = low
					}
					if world%4 == 2 {
						theta = &(*m.UpperBounds)[0]
						old = upper
					}
					want := [2]*big.Rat{new(big.Rat).Set(old[0]), new(big.Rat).Set(old[1])}
					unlimited := world%2 == 0
					wantUnlimited := unlimited
					var target, epsilonBound [2]*big.Rat
					active, equality := false, false
					switch op {
					case "upper-positive":
						active = cmp(x, upper) < 0
						target, epsilonBound = upper, upper
						equality = cmp(x, upper) == 0
					case "low-negative":
						active = cmp(x, low) > 0
						target, epsilonBound = low, low
						equality = cmp(x, low) == 0
					case "low-positive":
						active = cmp(x, low) < 0
						target, epsilonBound = low, low
					case "none":
					case "upper-negative":
						active = cmp(x, upper) > 0
						target, epsilonBound = upper, upper
					case "boxed-positive":
						if cmp(x, low) < 0 {
							active = true
							target, epsilonBound = low, upper
						} else if cmp(x, upper) < 0 {
							active = true
							target, epsilonBound = upper, upper
						} else {
							equality = cmp(x, upper) == 0
						}
					case "boxed-negative":
						if cmp(x, upper) > 0 {
							active = true
							target, epsilonBound = upper, upper
						} else if cmp(x, low) > 0 {
							active = true
							target, epsilonBound = low, low
						} else {
							equality = cmp(x, low) == 0
						}
					case "feasible-positive":
						active = true
						target, epsilonBound = upper, upper
					default:
						active = true
						target, epsilonBound = low, low
					}
					if active {
						abs := [2]*big.Rat{new(big.Rat).Set(epsilonBound[0]), new(big.Rat).Set(epsilonBound[1])}
						if cmp(abs, [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}) < 0 {
							abs[0].Neg(abs[0])
							abs[1].Neg(abs[1])
						}
						margin := [2]*big.Rat{new(big.Rat).Mul(new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(abs[0], big.NewRat(10, 1))), eps), new(big.Rat).Mul(new(big.Rat).Quo(abs[1], big.NewRat(10, 1)), eps)}
						if op == "upper-positive" || op == "low-positive" || op == "low-negative" {
							margin[0].SetInt64(0)
							margin[1].SetInt64(0)
						}
						q := [2]*big.Rat{new(big.Rat), new(big.Rat)}
						for i := 0; i < 2; i++ {
							margin[i].Quo(margin[i], big.NewRat(3, 1))
							q[i].Sub(target[i], x[i])
							if positive {
								q[i].Add(q[i], margin[i])
							} else {
								q[i].Sub(q[i], margin[i])
							}
							q[i].Quo(q[i], scalar)
						}
						if unlimited || cmp(q, old) <= 0 {
							want = q
						}
						wantUnlimited = false
					} else if equality {
						want = [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}
						wantUnlimited = false
					}
					if !wantUnlimited && cmp(want, [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}) < 0 {
						want = [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}
					}
					h.must(m.LimitThetaOnBasisColumn(0, &d.X, theta, &unlimited))
					h.check(theta, want[0], want[1])
					if unlimited != wantUnlimited {
						t.Fatal(op, world, unlimited, wantUnlimited)
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}
			}
		}
	}
}

func TestLPExactBasisDispatchPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"free", "low-positive", "low-negative", "upper-positive", "upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		for _, direct := range []bool{false, true} {
			success, failures, own, child := false, 0, 0, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s, e := NewLARSolver(h.m)
					h.must(e)
					_, e = s.AddVar(31)
					h.must(e)
					m := s.Core.RPrimal
					kind := ColumnBoxed
					if strings.HasPrefix(op, "low") {
						kind = ColumnLowBound
					}
					if strings.HasPrefix(op, "upper") {
						kind = ColumnUpperBound
					}
					if op == "free" {
						kind = ColumnFree
					}
					h.must(s.Bounds.Types.Set(0, kind))
					h.must(m.InfSet.Resize(1))
					if !strings.HasPrefix(op, "feasible") {
						h.must(m.InfSet.Insert(0))
					}
					huge := strings.Repeat("1234567890", 12)
					h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(huge+"1/7", huge+"3/11")))
					h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-"+huge+"1/3", huge+"3/13")))
					h.must((*m.UpperBounds)[0].Copy(h.m, h.pair(huge+"1/2", huge+"3/17")))
					if op == "low-positive" {
						h.must(h.m.Neg(&(*m.LowBounds)[0].X))
					}
					if op == "upper-negative" {
						h.must(h.m.SetString(&s.Core.RX.Data[0].X, strings.Repeat("9876543210", 100)+"1/7"))
					}
					h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, huge+"9/19"))
					d, theta := h.pair(huge+"9/17", "0"), h.pair("23", "29")
					if !strings.HasSuffix(op, "positive") {
						h.must(h.m.Neg(&d.X))
					}
					unlimited := true
					h.budget.FailAfter(ordinal)
					err := m.LimitThetaOnBasisColumn(0, &d.X, theta, &unlimited)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() || m.basisMultiplier != nil {
							own++
						}
						if m.PairCoreSolverBase.calls.pending() {
							child++
							if m.calls.pending() {
								t.Fatal("adopted base comparison")
							}
						}
						if e = m.ZeroHarrisEps(); e != ErrExactVectorPending {
							t.Fatal("lost stopped owner", e)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || (op != "free" && (failures == 0 || own == 0)) {
				t.Fatal(op, direct, success, failures, own, child)
			}
		}
	}
}

func TestLPExactBasisDispatchPreservesOlderComparisonOwner(t *testing.T) {
	for _, op := range []string{"upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, e := NewLARSolver(h.m)
				h.must(e)
				m := s.Core.RPrimal
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				_, e = s.AddVar(31)
				h.must(e)
				kind := ColumnBoxed
				if op == "upper-negative" {
					kind = ColumnUpperBound
				}
				h.must(s.Bounds.Types.Set(0, kind))
				h.must(m.InfSet.Resize(1))
				if !strings.HasPrefix(op, "feasible") {
					h.must(m.InfSet.Insert(0))
				}
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("2/3", "1/11")))
				h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-1/7", "1/13")))
				h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("1/3", "1/17")))
				theta, d := h.pair("23", "29"), h.pair("2/3", "0")
				if !strings.HasSuffix(op, "positive") {
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
					err = m.LimitThetaOnBasisColumn(0, &d.X, theta, &unlimited)
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
