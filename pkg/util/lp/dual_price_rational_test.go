// Source: src/util/lp/lp_dual_core_solver.hpp:109-160,236-262,301-331,534-577.
package lp

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpRationalDualPriceFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, 1, 2)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
	can := []bool{false, true}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	h.must(m.InitBetas())
	return f, m
}
func TestLPRationalDualPriceIndependentExactOracle(t *testing.T) {
	for world := 0; world < 400; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalDualPriceFixture(t, h)
		kind := []ColumnType{ColumnFixed, ColumnBoxed, ColumnLowBound, ColumnUpperBound, ColumnFree}[world%5]
		f.types[0] = kind
		lo := big.NewRat(int64(world%17-8), 7)
		up := new(big.Rat).Add(lo, big.NewRat(int64(1+world%11), 13))
		// Independent large rational gaps exercise copied-left and compound-square
		// arithmetic beyond inline numerals, with both signs and nonunit betas.
		gap := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(1+world%19)), uint(81+world%37)), big.NewInt(23))
		x := new(big.Rat).Set(lo)
		switch world % 4 {
		case 0:
			x.Sub(lo, gap)
		case 1:
			x.Add(up, gap)
		case 2:
			x.Add(lo, up)
			x.Quo(x, big.NewRat(2, 1))
		case 3:
			x.Set(up)
		}
		beta := big.NewRat(int64(2+world%29), 31)
		for _, v := range []struct {
			out   *numeral.Rational
			value *big.Rat
		}{{&f.low.Data[0], lo}, {&f.up.Data[0], up}, {&f.x.Data[0], x}, {&m.Betas.Data[0], beta}} {
			h.must(h.m.Copy(v.out, lpScalarBoundValue(h, v.value.RatString())))
		}
		out := lpScalarBoundValue(h, "97")
		h.must(m.GetEdgeSteepnessForLowBound(0, out))
		want := new(big.Rat).Sub(x, lo)
		want.Mul(want, want)
		want.Quo(want, beta)
		lpSparseRationalCheck(t, h, out, want)
		h.must(m.GetEdgeSteepnessForUpperBound(0, out))
		want.Sub(x, up)
		want.Mul(want, want)
		want.Quo(want, beta)
		lpSparseRationalCheck(t, h, out, want)
		h.must(m.PricingForRow(0, out))
		want.SetInt64(0)
		if kind != ColumnFree {
			if kind != ColumnUpperBound && x.Cmp(lo) < 0 {
				want.Sub(x, lo)
			} else if kind != ColumnLowBound && x.Cmp(up) > 0 {
				want.Sub(x, up)
			}
			want.Mul(want, want)
			want.Quo(want, beta)
		}
		lpSparseRationalCheck(t, h, out, want)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualPriceDeltaAndSignSourceProfiles(t *testing.T) {
	for _, kind := range []ColumnType{ColumnBoxed, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnFree, ColumnType(99)} {
		for _, x := range []int64{-4, -2, 0, 3, 5} {
			for _, op := range []string{"delta", "sign"} {
				h := newLPPairHarness(t)
				f, m := lpRationalDualPriceFixture(t, h)
				f.types[0] = kind
				h.must(h.m.SetInt32(&f.low.Data[0], -2))
				h.must(h.m.SetInt32(&f.up.Data[0], 3))
				h.must(h.m.SetInt32(&f.x.Data[0], int32(x)))
				h.must(h.m.SetInt32(&m.Betas.Data[0], 2))
				out := lpScalarBoundValue(h, "97")
				var e error
				var sign int
				if op == "delta" {
					e = m.GetDelta(out)
				} else {
					sign, e = m.DefineSignOfAlphaR()
				}
				valid := false
				want := new(big.Rat)
				wantSign := 0
				switch kind {
				case ColumnBoxed:
					valid = x < -2 || x > 3
					if x < -2 {
						want.SetInt64(x + 2)
						wantSign = -1
					} else if x > 3 {
						want.SetInt64(x - 3)
						wantSign = 1
					}
				case ColumnLowBound:
					valid = x < -2
					want.SetInt64(x + 2)
					wantSign = -1
				case ColumnUpperBound:
					valid = x > 3
					want.SetFrac64((x-3)*(x-3), 2)
					wantSign = 1
				case ColumnFixed:
					valid = op == "delta" || x < -2 || x > 3
					want.SetInt64(x - 3)
					if x < -2 {
						wantSign = -1
					} else if x > 3 {
						wantSign = 1
					}
				}
				if !valid && sourceDebugEnabled {
					if e != ErrAssertion {
						t.Fatal(kind, x, op, e)
					}
				} else {
					h.must(e)
					if !valid { // Source assertions erase into later switch labels.
						if op == "sign" {
							wantSign = 0
							if kind == ColumnLowBound && x > 3 {
								wantSign = 1
							}
						} else {
							want.SetInt64(0)
							if kind == ColumnBoxed || kind == ColumnLowBound || kind == ColumnUpperBound {
								want.SetInt64(x - 3)
								if kind == ColumnLowBound && x > 3 {
									want.SetFrac64((x-3)*(x-3), 2)
								}
							}
						}
					}
					if op == "delta" {
						lpSparseRationalCheck(t, h, out, want)
					} else if sign != wantSign {
						t.Fatal(kind, x, sign, wantSign)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPRationalDualPriceRowBudgetAndStatus(t *testing.T) {
	for _, rows := range []uint32{0, 1, 17, 300, 301, 500} {
		for _, percent := range []uint32{0, 1, 37, 100, 200, math.MaxUint32} {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, rows, rows+1)
			f.settings.SimplexStrategy = StrategyLU
			for i := uint32(0); i < rows; i++ {
				h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
			}
			can := make([]bool, rows+1)
			m, e := NewRationalDualCoreSolver(f.input(), &can)
			h.must(e)
			f.settings.PercentOfEnteringToCheck = percent
			before := f.settings.random
			rng := before
			r := rng.Next()
			s := rows
			conversionInvalid := false
			if rows > 300 {
				v := float64(rows) / 100 * float64(percent)
				conversionInvalid = v >= float64(math.MaxUint32)+1
				if !conversionInvalid {
					s = uint32(v)
				}
			}
			got, e := m.GetNumberOfRowsToTryForLeaving()
			if conversionInvalid {
				if e != ErrBounds || f.settings.random != before {
					t.Fatal("conversion prefix", rows, percent, e)
				}
			} else if s == 0 {
				if e != ErrBounds || f.settings.random != rng {
					t.Fatal("modulo prefix", rows, percent, e)
				}
			} else if e != nil || got != r%s+1 || f.settings.random != rng {
				t.Fatal(rows, percent, got, e)
			}
			for _, status := range []Status{StatusUnknown, StatusTentativeDualUnbounded, StatusDualUnbounded} {
				m.Status = status
				h.must(m.SetStatusToTentativeDualUnboundedOrDualUnbounded())
				want := StatusTentativeDualUnbounded
				if status == StatusTentativeDualUnbounded {
					want = StatusDualUnbounded
				}
				if m.Status != want {
					t.Fatal(m.Status, want)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalDualPricePhysicalOwnership(t *testing.T) {
	for _, op := range []string{"low-edge", "upper-edge", "row", "delta", "upper-delta", "sign"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalDualPriceFixture(t, h)
					f.types[0] = ColumnBoxed
					huge := strings.Repeat("1234567890", 20)
					h.must(h.m.SetString(&f.low.Data[0], huge+"1/13"))
					h.must(h.m.SetString(&f.up.Data[0], huge+"3/11"))
					h.must(h.m.SetString(&f.x.Data[0], huge+"7/17"))
					h.must(h.m.SetString(&m.Betas.Data[0], huge+"5/19"))
					if op == "upper-delta" {
						f.types[0] = ColumnUpperBound
						h.must(h.m.SetString(&f.x.Data[0], huge+"7/7"))
					}
					out := lpScalarBoundValue(h, "97")
					h.budget.FailAfter(ordinal)
					var e error
					switch op {
					case "low-edge":
						e = m.GetEdgeSteepnessForLowBound(0, out)
					case "upper-edge":
						e = m.GetEdgeSteepnessForUpperBound(0, out)
					case "row":
						e = m.PricingForRow(0, out)
					case "delta", "upper-delta":
						e = m.GetDelta(out)
					case "sign":
						_, e = m.DefineSignOfAlphaR()
					}
					h.budget.FailAfter(-1)
					if e == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if retry := m.PricingForRow(0, out); retry != ErrExactVectorPending {
							t.Fatal("stopped retry", retry)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || direct && failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalDualPriceLiveOutputAliases(t *testing.T) {
	for _, op := range []string{"edge", "row", "delta"} {
		for _, alias := range []string{"x", "low", "upper", "beta"} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPriceFixture(t, h)
			f.types[0] = ColumnLowBound
			h.must(h.m.SetInt32(&f.x.Data[0], -5))
			h.must(h.m.SetInt32(&f.low.Data[0], -2))
			h.must(h.m.SetInt32(&f.up.Data[0], 3))
			h.must(h.m.SetInt32(&m.Betas.Data[0], 2))
			out := &f.x.Data[0]
			switch alias {
			case "low":
				out = &f.low.Data[0]
			case "upper":
				out = &f.up.Data[0]
			case "beta":
				out = &m.Betas.Data[0]
			}
			var e error
			want := big.NewRat(9, 2)
			switch op {
			case "edge":
				e = m.GetEdgeSteepnessForLowBound(0, out)
			case "row":
				e = m.PricingForRow(0, out)
			case "delta":
				e = m.GetDelta(out)
				want.SetInt64(-3)
			}
			h.must(e)
			lpSparseRationalCheck(t, h, out, want)
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalDualPriceSignAndDualFeasibility(t *testing.T) {
	for value := -3; value <= 3; value++ {
		for initial := -2; initial <= 2; initial++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPriceFixture(t, h)
			delta := lpScalarBoundValue(h, big.NewRat(int64(value), 1000000007).RatString())
			yes, e := m.DeltaKeepsTheSign(initial, delta)
			h.must(e)
			want := value > 0 && initial == 1 || value < 0 && initial == -1
			if yes != want {
				t.Fatal(value, initial, yes, want)
			}
			f.types[1] = ColumnLowBound
			h.must(h.m.SetInt32(&m.D.Data[1], int32(value)))
			yes, e = m.ProblemIsDualFeasible()
			h.must(e)
			if yes != (value >= 0) {
				t.Fatal("problem feasibility", yes, value)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalDualPriceMetadataAndAssertionPrefixes(t *testing.T) {
	for _, phase := range []string{"nil-out", "heading", "negative-heading", "beta", "x", "low", "upper", "row", "types", "free-d", "unknown", "pending-local", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPriceFixture(t, h)
			f.types[0] = ColumnBoxed
			h.must(h.m.SetInt32(&f.x.Data[0], -5))
			h.must(h.m.SetInt32(&f.low.Data[0], -2))
			h.must(h.m.SetInt32(&f.up.Data[0], 3))
			xs, lo, up, head, betas, types := f.x.Data, f.low.Data, f.up.Data, f.heading, m.Betas.Data, f.types
			out := lpScalarBoundValue(h, "97")
			var e error
			switch phase {
			case "nil-out":
				e = m.GetEdgeSteepnessForLowBound(0, nil)
			case "heading":
				f.heading = nil
				e = m.GetEdgeSteepnessForLowBound(0, out)
			case "negative-heading":
				f.heading[0] = -1
				e = m.GetEdgeSteepnessForLowBound(0, out)
				f.heading[0] = 0
			case "beta":
				m.Betas.Data = nil
				e = m.GetEdgeSteepnessForLowBound(0, out)
			case "x":
				f.x.Data = nil
				e = m.PricingForRow(0, out)
			case "low":
				f.low.Data = nil
				e = m.GetDelta(out)
			case "upper":
				f.types[0] = ColumnUpperBound
				f.up.Data = nil
				_, e = m.DefineSignOfAlphaR()
			case "row":
				e = m.PricingForRow(2, out)
			case "types":
				f.types = nil
				e = m.PricingForRow(0, out)
			case "free-d":
				f.types[0] = ColumnFree
				h.must(h.m.SetInt32(&m.D.Data[0], 1))
				e = m.PricingForRow(0, out)
			case "unknown":
				f.types[0] = ColumnType(99)
				e = m.PricingForRow(0, out)
			case "pending-local":
				m.pricingValue = NewIndexedRationalVector(h.m, 1)
				e = m.PricingForRow(0, out)
			case "stopped":
				m.stopped = true
				e = m.PricingForRow(0, out)
			}
			erased := (phase == "free-d" || phase == "unknown") && !sourceDebugEnabled
			if erased {
				h.must(e)
				lpSparseRationalCheck(t, h, out, new(big.Rat))
			} else {
				if e == nil {
					t.Fatal("missing rejection")
				}
				lpSparseRationalCheck(t, h, out, big.NewRat(97, 1))
			}
			f.x.Data, f.low.Data, f.up.Data, f.heading, m.Betas.Data, f.types = xs, lo, up, head, betas, types
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualPricePreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"edge", "row", "delta"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualPriceFixture(t, h)
			f.types[0] = ColumnBoxed
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&f.low.Data[0], &v.X))
			h.must(h.m.Copy(&f.up.Data[0], &v.X))
			h.must(h.m.Copy(&f.x.Data[0], &v.Y))
			out := lpScalarBoundValue(h, "97")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, before, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				switch op {
				case "edge":
					e = m.GetEdgeSteepnessForLowBound(0, out)
				case "row":
					e = m.PricingForRow(0, out)
				case "delta":
					e = m.GetDelta(out)
				}
				if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			} else if e != nil && !lpExactPriorityPhysicalRefusal(e) {
				t.Fatal(e)
			}
			h.must(m.Close())
			f.close()
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !matched {
			t.Fatal("no actual older comparison", op)
		}
	}
}
