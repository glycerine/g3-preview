package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpCallExactBasisLimit(m *PairPrimalCoreSolver, op string, d *numeral.Rational, theta *RationalPair, unlimited *bool) error {
	switch op {
	case "upper-negative":
		return m.LimitInfOnUpperBoundMNeg(d, &(*m.X)[0], &(*m.UpperBounds)[0], theta, unlimited)
	case "boxed-positive":
		return m.LimitThetaOnBasisColumnForInfCaseMPosBoxed(0, d, theta, unlimited)
	case "boxed-negative":
		return m.LimitThetaOnBasisColumnForInfCaseMNegBoxed(0, d, theta, unlimited)
	case "feasible-positive":
		return m.LimitThetaOnBasisColumnForFeasCaseMPosNoCheck(0, d, theta, unlimited)
	default:
		return m.LimitThetaOnBasisColumnForFeasCaseMNegNoCheck(0, d, theta, unlimited)
	}
}

func TestLPExactBasisLimitIndependentPairOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		h := newLPPairHarness(t)
		for world := 0; world < 240; world++ {
			s, e := NewLARSolver(h.m)
			h.must(e)
			_, e = s.AddVar(31)
			h.must(e)
			m := s.Core.RPrimal
			x := [2]*big.Rat{big.NewRat(int64(world%19-9), 7), big.NewRat(int64(world%13-6), 11)}
			low := [2]*big.Rat{big.NewRat(-3, 7), big.NewRat(-2, 11)}
			upper := [2]*big.Rat{big.NewRat(5, 7), big.NewRat(3, 11)}
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
			positive := strings.HasSuffix(op, "positive")
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
			if strings.HasPrefix(op, "feasible") && cmp(want, [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}) < 0 {
				want = [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}
			}
			h.must(lpCallExactBasisLimit(m, op, &d.X, theta, &unlimited))
			h.check(theta, want[0], want[1])
			if unlimited != wantUnlimited {
				t.Fatal(op, world, unlimited, wantUnlimited)
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactBasisLimitPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		for _, direct := range []bool{false, true} {
			success, failures, own, child, comparisonStops := false, 0, 0, 0, 0
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
					huge := strings.Repeat("1234567890", 12)
					h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(huge+"1/7", huge+"3/11")))
					h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-"+huge+"1/3", huge+"3/13")))
					h.must((*m.UpperBounds)[0].Copy(h.m, h.pair(huge+"1/2", huge+"3/17")))
					if op == "upper-negative" {
						h.must(h.m.SetString(&s.Core.RX.Data[0].X, strings.Repeat("9876543210", 100)+"1/7"))
					}
					h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, huge+"9/19"))
					d, theta := h.pair("2/3", "0"), h.pair("23", "29")
					if !strings.HasSuffix(op, "positive") {
						h.must(h.m.Neg(&d.X))
					}
					unlimited := true
					h.budget.FailAfter(ordinal)
					err := lpCallExactBasisLimit(m, op, &d.X, theta, &unlimited)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						// A base comparison can refuse before retaining any raw
						// local. Count the actual stage as well as owned receipts.
						if m.work == nil && m.pairWork == nil {
							comparisonStops++
						}
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
			if !success || failures == 0 || own == 0 || (!strings.HasPrefix(op, "feasible") && comparisonStops == 0) {
				t.Fatal(op, direct, success, failures, own, child, comparisonStops)
			}
		}
	}
}

func TestLPExactBasisLimitPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
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
					err = lpCallExactBasisLimit(m, op, &d.X, theta, &unlimited)
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

func TestLPExactBasisLimitSourceGuards(t *testing.T) {
	for _, op := range []string{"upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		for _, phase := range []string{"nil", "base-stop", "wrong-sign"} {
			h := newLPPairHarness(t)
			s, e := NewLARSolver(h.m)
			h.must(e)
			_, e = s.AddVar(31)
			h.must(e)
			m := s.Core.RPrimal
			h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("2", "1/11")))
			h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-1", "1/13")))
			h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("1", "1/17")))
			d, theta := h.pair("2/3", "0"), h.pair("23", "29")
			if !strings.HasSuffix(op, "positive") {
				h.must(h.m.Neg(&d.X))
			}
			multiplier := &d.X
			want := ErrBounds
			switch phase {
			case "nil":
				multiplier = nil
			case "base-stop":
				if e = m.UpdateColumnInInfSet(99); e != ErrBounds {
					t.Fatal(e)
				}
				want = ErrExactVectorPending
			case "wrong-sign":
				h.must(h.m.Neg(&d.X))
				want = nil
				// The source boxed sign assertions are commented out.
				if sourceDebugEnabled && !strings.HasPrefix(op, "boxed") {
					want = ErrAssertion
				}
			}
			unlimited := true
			if e = lpCallExactBasisLimit(m, op, multiplier, theta, &unlimited); e != want {
				t.Fatal(op, phase, e, want)
			}
			if e != nil {
				if e = m.ZeroHarrisEps(); e != ErrExactVectorPending {
					t.Fatal("lost stopped owner", e)
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactBasisLimitScalarOutputAlias(t *testing.T) {
	for _, op := range []string{"upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		h := newLPPairHarness(t)
		s, e := NewLARSolver(h.m)
		h.must(e)
		_, e = s.AddVar(31)
		h.must(e)
		m := s.Core.RPrimal
		h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-1", "0")))
		h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("1", "0")))
		if op == "upper-negative" {
			h.must(h.m.SetInt32(&s.Core.RX.Data[0].X, 2))
		}
		h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, "1/10"))
		theta := h.pair("2/3", "1/13")
		if !strings.HasSuffix(op, "positive") {
			h.must(h.m.Neg(&theta.X))
		}
		unlimited := true
		h.must(lpCallExactBasisLimit(m, op, &theta.X, theta, &unlimited))
		h.check(theta, big.NewRat(311, 200), big.NewRat(0, 1))
		if unlimited {
			t.Fatal("lost scalar alias publication")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBasisLimitBoxedComparisonPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, profile := range []string{"positive-below-low", "positive-above-upper", "negative-inside", "negative-below-low"} {
		success, child := false, 0
		for ordinal := int64(0); ordinal < 500; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", profile, ordinal), func(t *testing.T) {
				h := lpNewDirectSparseHarness(t)
				s, e := NewLARSolver(h.m)
				h.must(e)
				_, e = s.AddVar(31)
				h.must(e)
				m := s.Core.RPrimal
				huge := strings.Repeat("1234567890", 12)
				xden, lowden := "7", "11"
				if profile == "positive-above-upper" {
					xden = "1"
				}
				if profile == "positive-below-low" || profile == "negative-below-low" {
					xden, lowden = "13", "7"
				}
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(huge+"1/"+xden, "0")))
				h.must((*m.LowBounds)[0].Copy(h.m, h.pair(huge+"1/"+lowden, "0")))
				h.must((*m.UpperBounds)[0].Copy(h.m, h.pair(huge+"1/2", "0")))
				d, theta := h.pair("2/3", "0"), h.pair("23", "29")
				op := "boxed-positive"
				if strings.HasPrefix(profile, "negative") {
					op = "boxed-negative"
					h.must(h.m.Neg(&d.X))
				}
				unlimited := true
				h.budget.FailAfter(ordinal)
				err := lpCallExactBasisLimit(m, op, &d.X, theta, &unlimited)
				h.budget.FailAfter(-1)
				if err == nil {
					success, done = true, true
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if m.PairCoreSolverBase.calls.pending() {
						child++
						if m.calls.pending() {
							t.Fatal("adopted base comparison")
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
		if !success || child == 0 {
			t.Fatal(profile, success, child)
		}
	}
}
