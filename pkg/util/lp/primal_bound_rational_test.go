package lp

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpScalarBoundValue(h *lpPairHarness, s string) *numeral.Rational { return &h.pair(s, "0").X }

func lpScalarHarrisOracle(bound, epsilon *big.Rat) *big.Rat {
	abs := new(big.Rat).Abs(bound)
	return new(big.Rat).Quo(new(big.Rat).Mul(new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(abs, big.NewRat(10, 1))), epsilon), big.NewRat(3, 1))
}

func TestLPRationalPrimalHarrisIndependentScalarOracle(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 0, 0)
		b := big.NewRat(int64(world%23-11), 17)
		eps := big.NewRat(int64(world%11-5), 10000000)
		bound, out := lpScalarBoundValue(h, b.RatString()), lpScalarBoundValue(h, "23/29")
		h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, eps.RatString()))
		if world%3 == 0 {
			out = bound
		}
		if world%3 == 1 {
			out = &m.ConvertedHarrisEpsilon
		}
		want := lpScalarHarrisOracle(b, eps)
		h.must(m.HarrisEpsForBound(bound, out))
		lpSparseRationalCheck(t, h, out, want)
		h.must(m.ZeroHarrisEps())
		h.must(m.HarrisEpsForBound(bound, out))
		lpSparseRationalCheck(t, h, out, new(big.Rat))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalBoundStepIndependentFlipsAndAliases(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, kind := range []ColumnType{ColumnFree, ColumnFixed, ColumnLowBound, ColumnUpperBound, ColumnBoxed} {
		for _, direction := range []int32{-1, 0, 1} {
			for world := 0; world < 32; world++ {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 0, 1)
				f.types[0] = kind
				m.SignOfEnteringDelta = direction
				lo, up := big.NewRat(-3, 7), big.NewRat(5, 11)
				x := new(big.Rat).Add(lo, new(big.Rat).Mul(new(big.Rat).Sub(up, lo), big.NewRat(int64(world%17), 16)))
				h.must(h.m.SetString(&f.low.Data[0], lo.RatString()))
				h.must(h.m.SetString(&f.up.Data[0], up.RatString()))
				h.must(h.m.SetString(&f.x.Data[0], x.RatString()))
				limit := big.NewRat(int64(world%9), 17)
				theta := lpScalarBoundValue(h, limit.RatString())
				out := lpScalarBoundValue(h, "37/41")
				allowed := kind == ColumnBoxed || (kind == ColumnUpperBound && direction > 0) || (kind == ColumnLowBound && direction < 0)
				step := big.NewRat(37, 41)
				if allowed {
					if direction > 0 {
						step.Sub(up, x)
					} else {
						step.Sub(x, lo)
					}
				}
				unlimited := world%2 == 0
				want := allowed && (unlimited || step.Cmp(limit) <= 0)
				got, e := m.TryJumpToAnotherBoundOnEntering(0, theta, out, unlimited)
				h.must(e)
				if got != want {
					t.Fatal(kind, direction, world, got, want)
				}
				lpSparseRationalCheck(t, h, out, step)
				h.must(h.m.SetString(out, "37/41"))
				got, e = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out)
				h.must(e)
				if got != (kind == ColumnBoxed) {
					t.Fatal("separate unlimited path")
				}
				if kind == ColumnBoxed {
					lpSparseRationalCheck(t, h, out, step)
				} else {
					lpSparseRationalCheck(t, h, out, big.NewRat(37, 41))
				}
				if allowed {
					h.must(h.m.SetInt32(theta, -100))
					got, e = m.TryJumpToAnotherBoundOnEntering(0, theta, theta, false)
					h.must(e)
					if !got {
						t.Fatal("live limit/output alias")
					}
					lpSparseRationalCheck(t, h, theta, step)
					// Output can also replace the bound or X after both operands are read.
					alias := &f.x.Data[0]
					if world%2 == 0 {
						alias = &f.up.Data[0]
					}
					got, e = m.TryJumpToAnotherBoundOnEntering(0, nil, alias, true)
					h.must(e)
					if !got {
						t.Fatal("unlimited bound alias")
					}
					lpSparseRationalCheck(t, h, alias, step)
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPRationalPrimalLimitThetaIndependentMinimum(t *testing.T) {
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 0, 0)
		a, b := big.NewRat(int64(world%17-8), 11), big.NewRat(int64(world%13-6), 17)
		lim, theta := lpScalarBoundValue(h, a.RatString()), lpScalarBoundValue(h, b.RatString())
		unlimited := world%3 == 0
		want := b
		if unlimited || a.Cmp(b) <= 0 {
			want = a
		}
		h.must(m.LimitTheta(lim, theta, &unlimited))
		if unlimited {
			t.Fatal("limit flag")
		}
		lpSparseRationalCheck(t, h, theta, want)
		h.must(m.LimitTheta(theta, theta, &unlimited))
		lpSparseRationalCheck(t, h, theta, want)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalInfLimitIndependentScalarOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"negative", "positive", "low-positive"} {
		for world := 0; world < 480; world++ {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 0, 0)
			xv, bv := big.NewRat(int64(world%17-8), 11), big.NewRat(int64(world%23-11), 17)
			if world%3 == 0 {
				xv.Set(bv)
			}
			tv := big.NewRat(int64(world%13-6), 23)
			d := big.NewRat(int64(world%17+1), int64(world%11+1))
			if op == "negative" {
				d.Neg(d)
			}
			x, bound, theta, mult := lpScalarBoundValue(h, xv.RatString()), lpScalarBoundValue(h, bv.RatString()), lpScalarBoundValue(h, tv.RatString()), lpScalarBoundValue(h, d.RatString())
			// Each borrowed argument may be the output; source reads precede publication.
			switch world % 4 {
			case 0:
				theta = x
				tv.Set(xv)
			case 1:
				theta = bound
				tv.Set(bv)
			case 2:
				theta = mult
				tv.Set(d)
			}
			unlimited := world%2 == 0
			wantUnlimited := unlimited
			want := new(big.Rat).Set(tv)
			cmp := xv.Cmp(bv)
			active := (op == "negative" && cmp > 0) || (op != "negative" && cmp < 0)
			wantLimited := (op == "negative" && cmp >= 0) || (op == "positive" && cmp <= 0)
			if active {
				q := new(big.Rat).Quo(new(big.Rat).Sub(bv, xv), d)
				if unlimited || q.Cmp(tv) <= 0 {
					want = q
				}
				wantUnlimited = false
			} else if cmp == 0 && op != "low-positive" {
				want.SetInt64(0)
				wantUnlimited = false
			}
			var limited bool
			var err error
			switch op {
			case "negative":
				limited, err = m.LimitInfOnBoundMNeg(mult, x, bound, theta, &unlimited)
			case "positive":
				limited, err = m.LimitInfOnBoundMPos(mult, x, bound, theta, &unlimited)
			default:
				err = m.LimitInfOnLowBoundMPos(mult, x, bound, theta, &unlimited)
			}
			h.must(err)
			if (op != "low-positive" && limited != wantLimited) || unlimited != wantUnlimited {
				t.Fatal(op, world, limited, wantLimited, unlimited, wantUnlimited)
			}
			lpSparseRationalCheck(t, h, theta, want)
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

// Returns the source target/epsilon choice and forced-zero state for the five
// bound helpers. This oracle uses only independent math/big rational values.
func lpScalarBasisLimitOracle(op string, x, low, upper *big.Rat) (target, epsBound *big.Rat, active, zero bool) {
	switch op {
	case "upper-negative":
		return upper, upper, x.Cmp(upper) > 0, false
	case "boxed-positive":
		if x.Cmp(low) < 0 {
			return low, upper, true, false
		}
		if x.Cmp(upper) < 0 {
			return upper, upper, true, false
		}
		return nil, nil, false, x.Cmp(upper) == 0
	case "boxed-negative":
		if x.Cmp(upper) > 0 {
			return upper, upper, true, false
		}
		if x.Cmp(low) > 0 {
			return low, low, true, false
		}
		return nil, nil, false, x.Cmp(low) == 0
	case "feasible-positive":
		return upper, upper, true, false
	default:
		return low, low, true, false
	}
}

func lpCallScalarBasisLimit(m *RationalPrimalCoreSolver, op string, d, theta *numeral.Rational, unlimited *bool) error {
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

func TestLPRationalPrimalBasisLimitIndependentScalarOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative"} {
		for world := 0; world < 240; world++ {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 0, 1)
			x, low, upper := big.NewRat(int64(world%19-9), 7), big.NewRat(-3, 7), big.NewRat(5, 7)
			if world%5 == 0 {
				x.Set(low)
			}
			if world%7 == 0 {
				x.Set(upper)
			}
			for k, v := range []*big.Rat{x, low, upper} {
				out := []*IndexedRationalVector{f.x, f.low, f.up}[k]
				h.must(h.m.SetString(&out.Data[0], v.RatString()))
			}
			eps := big.NewRat(int64(world%11-5), 10000000)
			h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, eps.RatString()))
			d := big.NewRat(int64(world%17+1), 19)
			positive := op == "boxed-positive" || op == "feasible-positive"
			if !positive {
				d.Neg(d)
			}
			tv := big.NewRat(int64(world%17-8), 23)
			theta := lpScalarBoundValue(h, tv.RatString())
			switch world % 4 {
			case 0:
				theta = &f.x.Data[0]
				tv.Set(x)
			case 1:
				theta = &f.low.Data[0]
				tv.Set(low)
			case 2:
				theta = &f.up.Data[0]
				tv.Set(upper)
			}
			unlimited := world%2 == 0
			wantUnlimited := unlimited
			want := new(big.Rat).Set(tv)
			target, epsBound, active, zero := lpScalarBasisLimitOracle(op, x, low, upper)
			if active {
				num := new(big.Rat).Sub(target, x)
				e := lpScalarHarrisOracle(epsBound, eps)
				if positive {
					num.Add(num, e)
				} else {
					num.Sub(num, e)
				}
				q := new(big.Rat).Quo(num, d)
				if unlimited || q.Cmp(tv) <= 0 {
					want = q
				}
				wantUnlimited = false
			}
			if zero {
				want.SetInt64(0)
				wantUnlimited = false
			}
			if strings.HasPrefix(op, "feasible-") && want.Sign() < 0 {
				want.SetInt64(0)
			}
			err := lpCallScalarBasisLimit(m, op, lpScalarBoundValue(h, d.RatString()), theta, &unlimited)
			h.must(err)
			if unlimited != wantUnlimited {
				t.Fatal(op, world, "unlimited")
			}
			lpSparseRationalCheck(t, h, theta, want)
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalPrimalBasisDispatchIndependentScalarOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, kind := range []ColumnType{ColumnFree, ColumnFixed, ColumnLowBound, ColumnUpperBound, ColumnBoxed} {
		for _, feasible := range []bool{false, true} {
			for _, positive := range []bool{false, true} {
				for world := 0; world < 80; world++ {
					h := newLPPairHarness(t)
					f, m := lpRationalPricingFixture(t, h, 1, 1)
					f.types[0] = kind
					if !feasible {
						h.must(m.InfSet.Insert(0))
					}
					x, low, up := big.NewRat(int64(world%19-9), 7), big.NewRat(-3, 7), big.NewRat(5, 7)
					if kind == ColumnFixed {
						up.Set(low)
					}
					if kind == ColumnBoxed && world%11 == 0 {
						low.SetInt64(2)
						up.SetInt64(-2)
					}
					if world%5 == 0 {
						x.Set(low)
					}
					if world%7 == 0 {
						x.Set(up)
					}
					for k, v := range []*big.Rat{x, low, up} {
						out := []*IndexedRationalVector{f.x, f.low, f.up}[k]
						h.must(h.m.SetString(&out.Data[0], v.RatString()))
					}
					eps := big.NewRat(int64(world%7-3), 10000000)
					h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, eps.RatString()))
					dv := big.NewRat(int64(world%17+1), 19)
					if !positive {
						dv.Neg(dv)
					}
					d := lpScalarBoundValue(h, dv.RatString())
					tv := big.NewRat(int64(world%17-8), 23)
					theta := lpScalarBoundValue(h, tv.RatString())
					switch world % 5 {
					case 0:
						theta = d
						tv.Set(dv)
					case 1:
						theta = &f.x.Data[0]
						tv.Set(x)
					case 2:
						theta = &f.low.Data[0]
						tv.Set(low)
					case 3:
						theta = &f.up.Data[0]
						tv.Set(up)
					}
					unlimited := world%2 == 0
					wantUnlimited := unlimited
					want := new(big.Rat).Set(tv)
					var target, epsBound *big.Rat
					active, zero := false, false
					if feasible {
						switch kind {
						case ColumnUpperBound:
							if positive {
								target, epsBound, active = up, up, true
							}
						case ColumnLowBound:
							if !positive {
								target, epsBound, active = low, low, true
							}
						case ColumnBoxed, ColumnFixed:
							active = true
							if positive {
								target, epsBound = up, up
							} else {
								target, epsBound = low, low
							}
						}
					} else {
						switch kind {
						case ColumnUpperBound:
							target = up
							if positive {
								active = x.Cmp(up) < 0
								zero = x.Cmp(up) == 0
							} else {
								active = x.Cmp(up) > 0
								epsBound = up
							}
						case ColumnLowBound:
							target = low
							if positive {
								active = x.Cmp(low) < 0
							} else {
								active = x.Cmp(low) > 0
								zero = x.Cmp(low) == 0
							}
						case ColumnBoxed, ColumnFixed:
							op := "boxed-negative"
							if positive {
								op = "boxed-positive"
							}
							target, epsBound, active, zero = lpScalarBasisLimitOracle(op, x, low, up)
						}
					}
					if active {
						num := new(big.Rat).Sub(target, x)
						if epsBound != nil {
							v := lpScalarHarrisOracle(epsBound, eps)
							if positive {
								num.Add(num, v)
							} else {
								num.Sub(num, v)
							}
						}
						q := new(big.Rat).Quo(num, dv)
						if unlimited || q.Cmp(tv) <= 0 {
							want = q
						}
						wantUnlimited = false
					}
					if zero {
						want.SetInt64(0)
						wantUnlimited = false
					}
					if !wantUnlimited && want.Sign() < 0 {
						want.SetInt64(0)
					}
					h.must(m.LimitThetaOnBasisColumn(0, d, theta, &unlimited))
					if unlimited != wantUnlimited {
						t.Fatal(kind, feasible, positive, world, "limit flag")
					}
					lpSparseRationalCheck(t, h, theta, want)
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				}
			}
		}
	}
}

func lpCallScalarBoundOperation(m *RationalPrimalCoreSolver, op string, d, theta *numeral.Rational, unlimited *bool) error {
	switch op {
	case "harris":
		return m.HarrisEpsForBound(&(*m.UpperBounds)[0], theta)
	case "flip":
		_, e := m.TryJumpToAnotherBoundOnEntering(0, nil, theta, true)
		return e
	case "minimum":
		return m.LimitTheta(&(*m.UpperBounds)[0], theta, unlimited)
	case "inf-positive":
		_, e := m.LimitInfOnBoundMPos(d, &(*m.X)[0], &(*m.UpperBounds)[0], theta, unlimited)
		return e
	case "inf-negative":
		_, e := m.LimitInfOnBoundMNeg(d, &(*m.X)[0], &(*m.LowBounds)[0], theta, unlimited)
		return e
	case "inf-low":
		return m.LimitInfOnLowBoundMPos(d, &(*m.X)[0], &(*m.LowBounds)[0], theta, unlimited)
	case "dispatch-positive", "dispatch-negative", "dispatch-free":
		return m.LimitThetaOnBasisColumn(0, d, theta, unlimited)
	default:
		return lpCallScalarBasisLimit(m, op, d, theta, unlimited)
	}
}

func TestLPRationalPrimalBoundPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	ops := []string{"harris", "flip", "minimum", "inf-positive", "inf-negative", "inf-low", "upper-negative", "boxed-positive", "boxed-negative", "feasible-positive", "feasible-negative", "dispatch-positive", "dispatch-negative", "dispatch-free"}
	for _, direct := range []bool{false, true} {
		for _, op := range ops {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 800 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					h := newLPPairHarness(t)
					if direct {
						h = lpNewDirectSparseHarness(t)
					}
					f, m := lpRationalPricingFixture(t, h, 1, 1)
					f.types[0] = ColumnBoxed
					m.SignOfEnteringDelta = 1
					huge := strings.Repeat("1234567890", 20)
					negative := strings.Contains(op, "negative") || op == "upper-negative"
					ds := huge + "3/23"
					if negative {
						ds = "-" + ds
					}
					d := lpScalarBoundValue(h, ds)
					theta := lpScalarBoundValue(h, huge+"1/29")
					xs := huge + "1/31"
					if negative {
						xs = huge + "9/7"
					}
					h.must(h.m.SetString(&f.x.Data[0], xs))
					h.must(h.m.SetString(&f.low.Data[0], huge+"7/13"))
					h.must(h.m.SetString(&f.up.Data[0], huge+"3/19"))
					h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, "1/10000000"))
					if strings.HasPrefix(op, "dispatch-") {
						h.must(m.InfSet.Insert(0))
					}
					if op == "dispatch-free" {
						f.types[0] = ColumnFree
					}
					unlimited := op != "minimum"
					h.budget.FailAfter(ordinal)
					err := lpCallScalarBoundOperation(m, op, d, theta, &unlimited)
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted base comparison")
						}
						if e := m.LimitTheta(theta, theta, &unlimited); e != ErrExactVectorPending {
							t.Fatal("stopped owner", e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			// A free-column by-value copy can reuse warmed pooled storage entirely.
			if !done || (direct && failures == 0) {
				t.Fatal(direct, op, done, failures)
			}
		}
	}
}

func TestLPRationalPrimalBoundReachedGuards(t *testing.T) {
	for _, phase := range []string{"harris-bound", "harris-out", "minimum-limit", "minimum-theta", "minimum-flag", "inf-mult", "inf-x", "inf-bound", "inf-theta", "inf-flag", "inf-sign", "basis-mult", "basis-theta", "basis-flag", "basis-type", "basis-x", "basis-low", "basis-up", "unknown", "zero-direction", "jump-kind", "jump-x", "jump-bound", "negative-jump", "unlimited-negative", "unsupported-bypass"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 1)
			f.types[0] = ColumnBoxed
			m.SignOfEnteringDelta = 1
			h.must(h.m.SetInt32(&f.up.Data[0], 2))
			d, theta := lpScalarBoundValue(h, "1"), lpScalarBoundValue(h, "-7")
			unlimited := false
			var err error
			want := ErrBounds
			switch phase {
			case "harris-bound":
				err = m.HarrisEpsForBound(nil, theta)
			case "harris-out":
				err = m.HarrisEpsForBound(theta, nil)
			case "minimum-limit":
				err = m.LimitTheta(nil, theta, &unlimited)
			case "minimum-theta":
				err = m.LimitTheta(theta, nil, &unlimited)
			case "minimum-flag":
				err = m.LimitTheta(theta, theta, nil)
			case "inf-mult":
				_, err = m.LimitInfOnBoundMPos(nil, theta, theta, theta, &unlimited)
			case "inf-x":
				_, err = m.LimitInfOnBoundMPos(d, nil, theta, theta, &unlimited)
			case "inf-bound":
				_, err = m.LimitInfOnBoundMPos(d, theta, nil, theta, &unlimited)
			case "inf-theta":
				_, err = m.LimitInfOnBoundMPos(d, theta, theta, nil, &unlimited)
			case "inf-flag":
				_, err = m.LimitInfOnBoundMPos(d, theta, theta, theta, nil)
			case "inf-sign":
				_, err = m.LimitInfOnBoundMNeg(d, theta, theta, theta, &unlimited)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "basis-mult":
				err = m.LimitThetaOnBasisColumn(0, nil, theta, &unlimited)
			case "basis-theta":
				err = m.LimitThetaOnBasisColumn(0, d, nil, &unlimited)
			case "basis-flag":
				err = m.LimitThetaOnBasisColumn(0, d, theta, nil)
			case "basis-type":
				f.types = nil
				err = m.LimitThetaOnBasisColumn(0, d, theta, &unlimited)
			case "basis-x":
				h.must(f.x.resizeData(0))
				err = m.LimitThetaOnBasisColumn(0, d, theta, &unlimited)
			case "basis-low":
				h.must(h.m.Neg(d))
				h.must(f.low.resizeData(0))
				err = m.LimitThetaOnBasisColumn(0, d, theta, &unlimited)
			case "basis-up":
				h.must(f.up.resizeData(0))
				err = m.LimitThetaOnBasisColumn(0, d, theta, &unlimited)
			case "unknown":
				f.types[0] = ColumnType(99)
				err = m.LimitThetaOnBasisColumn(0, d, theta, &unlimited)
				want = nil
				value := big.NewRat(0, 1)
				if sourceDebugEnabled {
					want = ErrAssertion
					value.SetInt64(-7)
				}
				lpSparseRationalCheck(t, h, theta, value)
			case "zero-direction":
				h.must(h.m.SetInt32(d, 0))
				err = m.LimitThetaOnBasisColumn(0, d, theta, &unlimited)
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					if err == nil {
						t.Fatal("zero divisor accepted")
					}
					want = err
				}
			case "jump-kind":
				_, err = m.TryJumpToAnotherBoundOnEntering(99, theta, theta, false)
			case "jump-x":
				h.must(f.x.resizeData(0))
				_, err = m.TryJumpToAnotherBoundOnEntering(0, theta, theta, false)
			case "jump-bound":
				h.must(f.up.resizeData(0))
				_, err = m.TryJumpToAnotherBoundOnEntering(0, theta, theta, false)
			case "negative-jump":
				h.must(h.m.SetInt32(&f.x.Data[0], 3))
				_, err = m.TryJumpToAnotherBoundOnEntering(0, nil, theta, true)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				lpSparseRationalCheck(t, h, theta, big.NewRat(-1, 1))
			case "unlimited-negative":
				h.must(h.m.SetInt32(&f.x.Data[0], 3))
				var ok bool
				ok, err = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, theta)
				want = nil
				if !ok {
					t.Fatal("extra assertion on separate unlimited path")
				}
				lpSparseRationalCheck(t, h, theta, big.NewRat(-1, 1))
			case "unsupported-bypass":
				f.types[0] = ColumnFree
				h.must(f.x.resizeData(0))
				h.must(f.low.resizeData(0))
				h.must(f.up.resizeData(0))
				var ok bool
				ok, err = m.TryJumpToAnotherBoundOnEntering(0, nil, nil, false)
				want = nil
				if ok {
					t.Fatal("unsupported flip")
				}
			}
			if err != want {
				t.Fatal(err, want)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if e := m.ZeroHarrisEps(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.HarrisEpsForBound(theta, theta); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.LimitThetaOnBasisColumn(0, d, theta, &unlimited); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.LimitInfOnUpperBoundMNeg(d, theta, theta, theta, &unlimited); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.LimitThetaOnBasisColumnForInfCaseMPosBoxed(0, d, theta, &unlimited); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e := m.LimitThetaOnBasisColumnForFeasCaseMPosNoCheck(0, d, theta, &unlimited); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := m.TryJumpToAnotherBoundOnEntering(0, theta, theta, false); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := m.TryJumpToAnotherBoundOnEnteringUnlimited(0, theta); e != ErrExactVectorPending {
				t.Fatal(e)
			}
		})
	}
}

func TestLPRationalPrimalBoundPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"harris", "flip", "minimum", "inf-positive", "inf-negative", "boxed-positive", "feasible-positive", "dispatch-positive"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 1, 1)
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnBoxed
				m.SignOfEnteringDelta = 1
				h.must(h.m.Copy(&f.low.Data[0], &v.X))
				h.must(h.m.Copy(&f.up.Data[0], &v.X))
				h.must(h.m.Copy(&f.x.Data[0], &v.Y))
				d, theta := lpScalarBoundValue(h, "1"), lpScalarBoundValue(h, "-7")
				if op == "inf-negative" {
					h.must(h.m.Neg(d))
				}
				h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, "1/10000000"))
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
				h.budget.FailAfter(ordinal)
				_, err := h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				old := h.m.ComparisonInvocationState()
				if old.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					unlimited := false
					_ = lpCallScalarBoundOperation(m, op, d, theta, &unlimited)
					if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 || h.m.ComparisonInvocationState() != old {
						t.Fatal("adopted caller comparison")
					}
				}
				h.must(m.Close())
				if old.Active && h.m.ComparisonInvocationState() != old {
					t.Fatal("cleanup consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual older comparison", op)
		}
	}
}

func TestLPRationalPrimalBoundReturnedReferenceCopies(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, phase := range []string{"harris-product", "bound-quotient", "flip-result", "flip-assignment"} {
		t.Run(phase, func(t *testing.T) {
			done, witness := false, false
			for ordinal := int64(0); ordinal < 600 && !done; ordinal++ {
				h := lpNewDirectSparseHarness(t)
				f, m := lpRationalPricingFixture(t, h, 1, 1)
				huge := strings.Repeat("9876543210", 20)
				xs, bs, ds := huge+"1/31", huge+"7/13", huge+"3/23"
				x, bound, d, out := lpScalarBoundValue(h, xs), lpScalarBoundValue(h, bs), lpScalarBoundValue(h, ds), lpScalarBoundValue(h, "0")
				h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, ds))
				want := new(big.Rat)
				if phase == "harris-product" {
					want.Mul(new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(new(big.Rat).Abs(mustLPRat(t, bs)), big.NewRat(10, 1))), mustLPRat(t, ds))
				} else if phase == "bound-quotient" {
					want.Quo(new(big.Rat).Sub(mustLPRat(t, bs), mustLPRat(t, xs)), mustLPRat(t, ds))
				} else {
					want.Sub(mustLPRat(t, bs), mustLPRat(t, xs))
					f.types[0] = ColumnBoxed
					m.SignOfEnteringDelta = 1
					h.must(h.m.Copy(&f.x.Data[0], x))
					h.must(h.m.Copy(&f.up.Data[0], bound))
				}
				h.budget.FailAfter(ordinal)
				var err error
				if phase == "harris-product" {
					err = m.HarrisEpsForBound(bound, out)
				} else if phase == "bound-quotient" {
					unlimited := true
					_, err = m.LimitInfOnBoundMPos(d, x, bound, out, &unlimited)
				} else {
					_, err = m.TryJumpToAnotherBoundOnEntering(0, nil, out, true)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					gcd, div := h.m.Z.GCDInvocation(), h.m.Z.DivisionInvocation()
					if m.work != nil && h.m.RawInvocationState().Active == [3]bool{} && !gcd.Matches(h.m.Z) && !div.Matches(h.m.Z) {
						temporary := &m.work.Data[0]
						var result *numeral.Rational
						switch phase {
						case "harris-product":
							result = &m.work.Data[4]
						case "bound-quotient":
							result = &m.work.Data[3]
						case "flip-result":
							result = &m.work.Data[1]
						default:
							temporary = &m.work.Data[1]
							result = out
						}
						s, e := h.m.String(temporary)
						if e == nil {
							got, valid := new(big.Rat).SetString(s)
							if valid && got.Cmp(want) == 0 {
								r, readErr := h.m.String(result)
								if readErr != nil && !errors.Is(readErr, arena.ErrInvalidSlot) {
									t.Fatal(readErr)
								}
								returned, valid := new(big.Rat).SetString(r)
								if readErr != nil || !valid || returned.Cmp(want) != 0 {
									witness = true
								}
							}
						} else if !errors.Is(e, arena.ErrInvalidSlot) {
							t.Fatal(e)
						}
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
			if !done || !witness {
				t.Fatal("missing successful completion or returned-reference copy refusal", done, witness)
			}
		})
	}
}
