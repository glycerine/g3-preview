package polynomial

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func upolyIsolationBinaryRat(t *testing.T, m *UPolynomialManager, b *util.BinaryRationalManager, a *util.BinaryRational) *big.Rat {
	t.Helper()
	var n util.Int
	if e := b.Numerator(a, &n); e != nil {
		t.Fatal(e)
	}
	neg, limbs, e := m.IntegerManager().Decompose(&n, nil)
	if e != nil {
		t.Fatal(e)
	}
	z := new(big.Int)
	for i := len(limbs) - 1; i >= 0; i-- {
		z.Lsh(z, 32)
		z.Add(z, new(big.Int).SetUint64(uint64(limbs[i])))
	}
	if neg {
		z.Neg(z)
	}
	if e = m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
	return new(big.Rat).SetFrac(z, new(big.Int).Lsh(big.NewInt(1), uint(a.K())))
}
func TestUPolynomialRefinableNineRegisteredCalls(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	counts := upolyIsolationCounts{}
	for _, f := range []struct {
		middle int32
		a      int32
		ak     uint32
		b      int32
		bk     uint32
	}{{4, 1, 0, 3, 0}, {4, 2, 0, 3, 0}, {4, 5, 1, 3, 0}, {5, 1, 0, 3, 0}, {5, 2, 0, 3, 0}, {5, 3, 1, 3, 0}, {5, 1, 0, 5, 1}, {5, 3, 1, 5, 1}, {1, 1, 0, 3, 0}} {
		factors := [][]int32{{-1, 1}, {-11, f.middle}, {-3, 1}}
		if f.middle == 1 {
			factors[1] = []int32{-2, 1}
		}
		p := upolyIsolationProduct(t, m, factors)
		v := mustUPolyView(t, m, p)
		a, b := util.NewBinaryIntPower(f.a, f.ak), util.NewBinaryIntPower(f.b, f.bk)
		interval, e := m.IsolatingToRefinable(v, bqm, &a, &b)
		if e != nil {
			t.Fatal(f, e)
		}
		if interval {
			sa, sb := upolyIsolationSign(t, m, v, &a), upolyIsolationSign(t, m, v, &b)
			upolyIsolationAssert(t, &counts, true, sa != 0 && sb != 0 && sa == -sb)
		} else {
			upolyIsolationAssert(t, &counts, false, upolyIsolationSign(t, m, v, &a) == 0)
		}
		if e = bqm.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e = bqm.Del(&b); e != nil {
			t.Fatal(e)
		}
	}
	if counts.ensure+counts.verify != 9 {
		t.Fatal(counts)
	}
	t.Logf("source nine calls ENSURE=%d VERIFY=%d", counts.ensure, counts.verify)
}
func TestUPolynomialRefineTwoRegisteredPrecisionInputs(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	for _, f := range []struct {
		p []int32
		k uint32
	}{{[]int32{-1, -1, 0, 0, 0, 1}, 20}, {[]int32{-2, 0, 1}, 200}} {
		p := upolySourceNew(t, m, f.p)
		v := mustUPolyView(t, m, p)
		a, b := util.NewBinaryInt(1), util.NewBinaryInt(2)
		interval, e := m.RefineToPrecision(v, bqm, &a, &b, f.k)
		if e != nil {
			t.Fatal(f.k, e)
		}
		if !interval {
			if upolyIsolationSign(t, m, v, &a) != 0 {
				t.Fatal("precise root")
			}
		} else {
			left, right := upolyIsolationBinaryRat(t, m, bqm, &a), upolyIsolationBinaryRat(t, m, bqm, &b)
			width := new(big.Rat).Sub(right, left)
			limit := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), uint(f.k)))
			if width.Sign() <= 0 || width.Cmp(limit) >= 0 || upolyIsolationSign(t, m, v, &a) != -upolyIsolationSign(t, m, v, &b) {
				t.Fatal("independent strict width/sign oracle", f.k, left, right)
			}
		}
		if e = bqm.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e = bqm.Del(&b); e != nil {
			t.Fatal(e)
		}
	}
	// Source print-only calls have zero registered assertions; the width checks
	// above are independent tests and do not increase the source ENSURE count.
}
func TestUPolynomialQ2BQNineRegisteredIntervals(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	p := upolyIsolationProduct(t, m, [][]int32{{-1, 1}, {-2, 1}, {-3, 1}, {-4, 1}})
	v := mustUPolyView(t, m, p)
	for _, f := range []struct{ a, b upolyIsolationRational }{{upolyIsolationRational{1, 3}, upolyIsolationRational{7, 5}}, {upolyIsolationRational{1, 2}, upolyIsolationRational{7, 5}}, {upolyIsolationRational{3, 7}, upolyIsolationRational{3, 2}}, {upolyIsolationRational{0, 1}, upolyIsolationRational{3, 2}}, {upolyIsolationRational{0, 1}, upolyIsolationRational{23, 21}}, {upolyIsolationRational{7, 2}, upolyIsolationRational{5, 1}}, {upolyIsolationRational{999, 1000}, upolyIsolationRational{1001, 1000}}, {upolyIsolationRational{9999, 10000}, upolyIsolationRational{10001, 10000}}, {upolyIsolationRational{39999, 10000}, upolyIsolationRational{40001, 10000}}} {
		a, b := util.NewRational(), util.NewRational()
		if e := q.SetFraction(&a, f.a.n, f.a.d); e != nil {
			t.Fatal(e)
		}
		if e := q.SetFraction(&b, f.b.n, f.b.d); e != nil {
			t.Fatal(e)
		}
		c, d := util.NewBinaryRational(), util.NewBinaryRational()
		interval, e := m.ConvertRationalToBinaryInterval(v, q, &a, &b, bqm, &c, &d)
		if e != nil {
			t.Fatal(f, e)
		}
		if !interval {
			if upolyIsolationSign(t, m, v, &c) != 0 {
				t.Fatal("root", f)
			}
		} else {
			left, right := upolyIsolationBinaryRat(t, m, bqm, &c), upolyIsolationBinaryRat(t, m, bqm, &d)
			originalLeft := new(big.Rat).SetFrac(big.NewInt(f.a.n), new(big.Int).SetUint64(f.a.d))
			originalRight := new(big.Rat).SetFrac(big.NewInt(f.b.n), new(big.Int).SetUint64(f.b.d))
			if left.Cmp(right) >= 0 || left.Cmp(originalLeft) < 0 || right.Cmp(originalRight) > 0 || upolyIsolationSign(t, m, v, &c) != -upolyIsolationSign(t, m, v, &d) {
				t.Fatal("independent bracketing", f, left, right)
			}
		}
		if e = q.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e = q.Del(&b); e != nil {
			t.Fatal(e)
		}
		if e = bqm.Del(&c); e != nil {
			t.Fatal(e)
		}
		if e = bqm.Del(&d); e != nil {
			t.Fatal(e)
		}
	}
}
func TestUPolynomialRefineAndNormalizeAllPublicRoutes(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	v := mustUPolyView(t, m, p)
	a, b := util.NewBinaryInt(1), util.NewBinaryInt(2)
	for route := 0; route < 3; route++ {
		sign := upolyIsolationSign(t, m, v, &a)
		var interval bool
		var e error
		switch route {
		case 0:
			interval, e = m.Refine(v, bqm, &a, &b)
		case 1:
			interval, e = m.RefineCore(v, sign, bqm, &a, &b)
		case 2:
			interval, e = m.RefineCoreToPrecision(v, sign, bqm, &a, &b, 10)
		}
		if e != nil || !interval || upolyIsolationSign(t, m, v, &a) != -upolyIsolationSign(t, m, v, &b) {
			t.Fatal(route, interval, e)
		}
	}
	if e := bqm.Del(&a); e != nil {
		t.Fatal(e)
	}
	if e := bqm.Del(&b); e != nil {
		t.Fatal(e)
	}
	positive := upolySourceNew(t, m, []int32{-1, 1})
	pv := mustUPolyView(t, m, positive)
	for _, known := range []bool{false, true} {
		a, b = util.NewBinaryInt(-1), util.NewBinaryInt(2)
		var yes bool
		var e error
		if known {
			yes, e = m.NormalizeIntervalCore(pv, -1, bqm, &a, &b)
		} else {
			yes, e = m.NormalizeInterval(pv, bqm, &a, &b)
		}
		zero, e2 := bqm.IsZero(&a)
		if e != nil || e2 != nil || !yes || !zero {
			t.Fatal(known, yes, e, e2)
		}
		if e = bqm.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e = bqm.Del(&b); e != nil {
			t.Fatal(e)
		}
	}
	zeroPoly := upolySourceNew(t, m, []int32{0, 1})
	a, b = util.NewBinaryInt(-1), util.NewBinaryInt(1)
	yes, e := m.NormalizeInterval(mustUPolyView(t, m, zeroPoly), bqm, &a, &b)
	if e != nil || yes {
		t.Fatal(yes, e)
	}
	if e = bqm.Del(&a); e != nil {
		t.Fatal(e)
	}
	if e = bqm.Del(&b); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialQ2BQAliasedOutputSourceDebugSite(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	p := upolySourceNew(t, m, []int32{-1, 1})
	a, b := util.NewRational(), util.NewRational()
	if e := q.SetInt64(&b, 2); e != nil {
		t.Fatal(e)
	}
	out := util.NewBinaryInt(37)
	interval, e := m.ConvertRationalToBinaryInterval(mustUPolyView(t, m, p), q, &a, &b, bqm, &out, &out)
	if polynomialSourceDebug {
		if e != ErrUPolynomialAssertion {
			t.Fatal("actual final c<d assertion", interval, e)
		}
	} else if e != nil || !interval {
		t.Fatal("defined release alias", interval, e)
	}
	two := util.NewInt(2)
	same, e2 := bqm.EqualInteger(&out, &two)
	if e2 != nil || !same {
		t.Fatal("source second endpoint publication", e2, same)
	}
	if e = q.Del(&a); e != nil {
		t.Fatal(e)
	}
	if e = q.Del(&b); e != nil {
		t.Fatal(e)
	}
	if e = bqm.Del(&out); e != nil {
		t.Fatal(e)
	}
}

// Only reads actual scalar references; failedRef is retired/nonlive. No copies
// of owning numerals or numerical formatting disturb the immediate prefix.
func upolyIsolationCallerNumerators(m *UPolynomialManager, values ...*util.BinaryRational) uint32 {
	failed := testFieldBySourceName(reflect.ValueOf(m.IntegerManager()).Elem(), "failedRef").Uint()
	var n uint32
	for _, v := range values {
		cell := testFieldBySourceName(testFieldBySourceName(reflect.ValueOf(v).Elem(), "num"), "cell").Uint()
		if cell != 0 && cell != failed {
			n++
		}
	}
	return n
}
func TestUPolynomialRefinementFaultPrefixesImmediateRoots(t *testing.T) {
	for route := 0; route < 2; route++ {
		if !t.Run(fmt.Sprintf("%d", route), func(t *testing.T) {
			failures, successes := 0, 0
			// Exact fresh checked-debug calibration needs2594/2496 reservations
			// for128-bit bounded isolation, and16146 forprecision100 refinement.
			// These finite horizons retain every refusal through the actual success.
			for fault := int64(0); fault < 32768; fault++ {
				if !t.Run(fmt.Sprintf("%d", fault), func(t *testing.T) {
					m, q, budget := upolySourceOwner(t)
					bqm := upolyIsolationBinaryOwner(t, q, budget)
					p := upolySourceNew(t, m, []int32{-2, 0, 1})
					v := mustUPolyView(t, m, p)
					a, b := util.NewBinaryInt(1), util.NewBinaryInt(2)
					c, d := util.NewBinaryRational(), util.NewBinaryRational()
					qa, qb := util.NewRational(), util.NewRational()
					if route == 1 {
						if e := m.SetInts(p, []util.Int{util.NewInt(-1), util.NewInt(1)}); e != nil {
							t.Fatal(e)
						}
						v = mustUPolyView(t, m, p)
						var num, den util.Int
						if e := m.IntegerManager().SetLimbs(&den, []uint32{0, 0, 0, 16}); e != nil {
							t.Fatal(e)
						}
						one := util.NewInt(1)
						if e := m.IntegerManager().Sub(&den, &one, &num); e != nil {
							t.Fatal(e)
						}
						if e := q.SetParts(&qa, &num, &den); e != nil {
							t.Fatal(e)
						}
						if e := m.IntegerManager().Add(&den, &one, &num); e != nil {
							t.Fatal(e)
						}
						if e := q.SetParts(&qb, &num, &den); e != nil {
							t.Fatal(e)
						}
						if e := m.IntegerManager().Del(&num); e != nil {
							t.Fatal(e)
						}
						if e := m.IntegerManager().Del(&den); e != nil {
							t.Fatal(e)
						}
					}
					// Input rationals have exactly four big components, all valid before the
					// operation; this route never mutates their components.
					inputs := uint32(0)
					if route == 1 {
						inputs = 4
					}
					budget.FailAfter(fault)
					var e error
					if route == 0 {
						_, e = m.RefineToPrecision(v, bqm, &a, &b, 100)
					} else {
						_, e = m.ConvertRationalToBinaryInterval(v, q, &qa, &qb, bqm, &c, &d)
					}
					graph := m.Stats()
					scratch, e2 := bqm.Stats()
					callers := upolyIsolationCallerNumerators(m, &a, &b, &c, &d) + inputs
					if e2 != nil || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager())+scratch.LiveScratchNumerators+callers {
						t.Fatal("immediate refine/conversion graph", route, fault, e, e2, graph, scratch, callers)
					}

					upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Rationals: []*util.Rational{&qa, &qb}, RationalManagers: []*util.RationalManager{q}, BinaryValues: []*util.BinaryRational{&a, &b, &c, &d}, BinaryManagers: []*util.BinaryRationalManager{bqm}})
					budget.FailAfter(-1)
					if e != nil {
						failures++
						if !numericStorageReservationRefusal(e) {
							t.Fatal(route, fault, e)
						}
					} else {
						successes++
					}
					for _, value := range []*util.BinaryRational{&a, &b, &c, &d} {
						if e = bqm.Del(value); e != nil {
							t.Fatal(route, fault, e)
						}
					}
					if e = q.Del(&qa); e != nil {
						t.Fatal(e)
					}
					if e = q.Del(&qb); e != nil {
						t.Fatal(e)
					}
					graph = m.Stats()
					scratch, e2 = bqm.Stats()
					if e2 != nil || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager())+scratch.LiveScratchNumerators {
						t.Fatal("untracked lexical root", route, fault, e2, graph)
					}

				}) {
					t.FailNow()
				}
				// Each subtest's actual owner cleanup runs before the next fresh fixture.
				if successes != 0 {
					break
				}
			}
			if failures == 0 || successes == 0 {
				t.Fatal(route, failures, successes)
			}
			t.Log("real refine/conversion refusal ordinals", route, failures, successes)
		}) {
			t.FailNow()
		}
	}
}
