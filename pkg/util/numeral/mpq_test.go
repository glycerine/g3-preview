// Source assertions: src/test/mpq.cpp tst0, bug1, bug2. Remaining source
// coverage and lifecycle obligations: ../PORT_PROGRESS.md#audit-mpnum-port-status-md.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func rationalManager(t *testing.T, synch bool) (*RationalManager, *arena.Budget) {
	t.Helper()
	return rationalManagerWithLimit(t, synch, 1<<20)
}

func rationalManagerWithLimit(t *testing.T, synch bool, limit uint64) (*RationalManager, *arena.Budget) {
	t.Helper()
	budget := arena.NewBudget(limit)
	m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}, Synchronized: synch})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		m.Close()
		if budget.Used() != 0 {
			t.Errorf("rational close retained %d", budget.Used())
		}
	})
	return m, budget
}
func rationalString(t *testing.T, m *RationalManager, a *Rational) string {
	t.Helper()
	s, e := m.String(a)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func setRationalOracle(t *testing.T, m *RationalManager, a *Rational, n, d *big.Int) {
	t.Helper()
	var numerator, denominator Int
	setIntegerOracle(t, m.Integers(), &numerator, n)
	setIntegerOracle(t, m.Integers(), &denominator, d)
	defer m.Integers().Del(&numerator)
	defer m.Integers().Del(&denominator)
	if e := m.SetParts(a, &numerator, &denominator); e != nil {
		t.Fatal(e)
	}
}

func TestRationalSourceDivAndInverseAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, b := NewRational(), NewRational()
		// tst0's actual source assertion, dividing into the divisor destination.
		m.SetFraction(&a, 2, 3)
		m.SetFraction(&b, 4, 3)
		if e := m.Div(&a, &b, &b); e != nil || rationalString(t, m, &b) != "1/2" {
			t.Fatal("source tst0", synch, e)
		}
		// bug1 and bug2 exact source in-place reciprocal expectations.
		for _, n := range []int64{2, -2} {
			m.SetInt64(&a, n)
			if e := m.Inv(&a, &a); e != nil {
				t.Fatal(e)
			}
			want := "1/2"
			if n < 0 {
				want = "-1/2"
			}
			if rationalString(t, m, &a) != want {
				t.Fatal("source inverse", synch, n)
			}
		}
	}
}

func TestRationalConstructorLifecycleAndBudget(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a, b, c := NewRational(), NewRational(), NewRational()
		if rationalString(t, m, &a) != "0" || a.Den.Val != 1 || a.Num.Val != 0 {
			t.Fatal("source default0/1")
		}
		if e := arena.ValidateRecordType(reflect.TypeOf(Rational{})); e != nil {
			t.Fatal(e)
		}
		if m.HeaderBytes != uint64(reflect.TypeOf(RationalManager{}).Size()) || budget.Used() < m.HeaderBytes {
			t.Fatal("unaccounted manager header/scratch")
		}
		m.SetFraction(&a, math.MinInt64, math.MaxUint64)
		if e := m.Copy(&b, &a); e != nil {
			t.Fatal(e)
		}
		if a.Num.Cell == b.Num.Cell || a.Den.Cell == b.Den.Cell {
			t.Fatal("rational shallow copy")
		}
		if e := m.Neg(&a); e != nil {
			t.Fatal(e)
		}
		if rationalString(t, m, &b) != "-9223372036854775808/18446744073709551615" {
			t.Fatal("copy isolation")
		}
		if e := m.Move(&c, &b); e != nil || b.Num.Cell != 0 || b.Den.Cell != 0 {
			t.Fatal(e)
		}
		if e := m.Swap(&a, &c); e != nil {
			t.Fatal(e)
		}
		if e := m.Abs(&a); e != nil {
			t.Fatal(e)
		}
		if e := m.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e := m.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e := m.Reset(&a); e != nil || rationalString(t, m, &a) != "0" {
			t.Fatal(e)
		}
		m.Close()
		m.Close()
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
		if _, e := m.String(&a); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
	}
}

func TestRationalCheckedInitializationAndZero(t *testing.T) {
	m, _ := rationalManager(t, true)
	var raw Rational
	valid := NewRational()
	if _, e := m.String(&raw); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal("raw zero read", e)
	}
	if e := m.SetInt64(&raw, 1); e != nil || raw != (Rational{Num: NewInt(1), Den: NewInt(1)}) {
		t.Fatal("explicit scalar initialization", raw, e)
	}
	e := m.SetFraction(&valid, 1, 0)
	qopsAuditLiveOwnership(t, m, []*Int{&raw.Num, &raw.Den, &valid.Num, &valid.Den})
	if modularSourceDebug {
		if !errors.Is(e, ErrIntegerAssertion) || valid != NewRational() {
			t.Fatal(valid, e)
		}
	} else if e != nil || valid.Num != NewInt(1) || valid.Den != NewInt(0) {
		t.Fatal(valid, e)
	}
	// Reset the finite invalid output explicitly before the independent zero inverse.
	qopsMust(t, m.Reset(&valid))
	e = m.Inv(&valid, &valid)
	divPublicationAudit(t, m, &valid)
	if modularSourceDebug {
		if !errors.Is(e, ErrIntegerAssertion) || valid != NewRational() {
			t.Fatal(e)
		}
	} else {
		qopsMust(t, e)
		if valid.Num != NewInt(1) || valid.Den != NewInt(0) {
			t.Fatal(valid)
		}
		if _, e = m.String(&valid); !errors.Is(e, ErrUninitializedRational) {
			t.Fatal(e)
		}
	}
	// Explicit Reset owns the invalid finite result; it is not arithmetic retry.
	qopsMust(t, m.Reset(&valid))
	one := NewRational()
	qopsMust(t, m.SetInt64(&one, 1))
	qopsMust(t, m.Div(&one, &valid, &one))
	divPublicationAudit(t, m, &valid, &one)
	if one.Num != NewInt(1) || one.Den != NewInt(0) {
		t.Fatal(one)
	}
	if _, e = m.String(&one); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	qopsMust(t, m.Del(&one))
	qopsMust(t, m.Del(&valid))
	qopsMust(t, m.Del(&raw))
}

func TestRationalIndependentSignedArithmetic(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, b, out := NewRational(), NewRational(), NewRational()
		rng := rand.New(rand.NewSource(0x4d5051))
		// 250 signed exact fractions; at most eight numerator/five denominator limbs.
		for trial := 0; trial < 250; trial++ {
			an, bn, ad, bd := make([]uint32, 1+rng.Intn(8)), make([]uint32, 1+rng.Intn(8)), make([]uint32, 1+rng.Intn(5)), make([]uint32, 1+rng.Intn(5))
			for i := range an {
				an[i] = rng.Uint32()
			}
			for i := range bn {
				bn[i] = rng.Uint32()
			}
			for i := range ad {
				ad[i] = rng.Uint32()
			}
			for i := range bd {
				bd[i] = rng.Uint32()
			}
			ad[len(ad)-1] |= 1
			bd[len(bd)-1] |= 1
			bn[len(bn)-1] |= 1
			av, bv, dv, ev := oracle(an), oracle(bn), oracle(ad), oracle(bd)
			if trial%2 == 0 {
				av.Neg(av)
			}
			if trial%3 == 0 {
				bv.Neg(bv)
			}
			if trial%5 == 0 {
				dv.Neg(dv)
			}
			if trial%7 == 0 {
				ev.Neg(ev)
			}
			ar, br := new(big.Rat).SetFrac(av, dv), new(big.Rat).SetFrac(bv, ev)
			for op := 0; op < 4; op++ {
				for alias := 0; alias < 3; alias++ {
					setRationalOracle(t, m, &a, av, dv)
					setRationalOracle(t, m, &b, bv, ev)
					dst := &out
					if alias == 1 {
						dst = &a
					}
					if alias == 2 {
						dst = &b
					}
					want := new(big.Rat)
					var err error
					switch op {
					case 0:
						want.Add(ar, br)
						err = m.Add(&a, &b, dst)
					case 1:
						want.Sub(ar, br)
						err = m.Sub(&a, &b, dst)
					case 2:
						want.Mul(ar, br)
						err = m.Mul(&a, &b, dst)
					case 3:
						want.Quo(ar, br)
						err = m.Div(&a, &b, dst)
					}
					if err != nil || rationalString(t, m, dst) != want.RatString() {
						t.Fatal("rational arithmetic", synch, trial, op, alias, err)
					}
				}
			}
			setRationalOracle(t, m, &a, av, dv)
			setRationalOracle(t, m, &b, bv, ev)
			if cmp, e := m.Compare(&a, &b); e != nil || cmp != ar.Cmp(br) {
				t.Fatal("order", synch, trial, e)
			}
			if eq, e := m.Equal(&a, &b); e != nil || eq != (ar.Cmp(br) == 0) {
				t.Fatal("equal", e)
			}
			if e := m.Inv(&a, &out); e != nil || rationalString(t, m, &out) != new(big.Rat).Inv(ar).RatString() {
				t.Fatal("reciprocal", synch, trial, e)
			}
		}
	}
}

func TestRationalNormalizationAndIntegerBranches(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, b, out := NewRational(), NewRational(), NewRational()
		for _, pair := range [][2]int64{{0, 7}, {2, 4}, {-2, 4}, {7, -3}, {-7, -3}, {math.MinInt32, 1}} {
			n, d := NewInt(0), NewInt(0)
			m.Integers().SetInt64(&n, pair[0])
			m.Integers().SetInt64(&d, pair[1])
			if e := m.SetParts(&a, &n, &d); e != nil {
				t.Fatal(e)
			}
			want := new(big.Rat).SetFrac(big.NewInt(pair[0]), big.NewInt(pair[1]))
			if rationalString(t, m, &a) != want.RatString() {
				t.Fatal(pair)
			}
			m.Integers().Del(&n)
			m.Integers().Del(&d)
		}
		m.SetInt64(&a, 17)
		m.SetInt64(&b, -9)
		for op := 0; op < 3; op++ {
			var err error
			want := ""
			switch op {
			case 0:
				err = m.Add(&a, &b, &out)
				want = "8"
			case 1:
				err = m.Sub(&a, &b, &out)
				want = "26"
			case 2:
				err = m.Mul(&a, &b, &out)
				want = "-153"
			}
			if err != nil || rationalString(t, m, &out) != want || out.Den.Val != 1 {
				t.Fatal(op, err)
			}
		}
		for _, pair := range [][2]int64{{-1, 0}, {0, 1}, {1, -1}, {-1, -2}, {0, 0}} {
			m.SetInt64(&a, pair[0])
			m.SetInt64(&b, pair[1])
			cmp, e := m.Compare(&a, &b)
			if e != nil || cmp != big.NewInt(pair[0]).Cmp(big.NewInt(pair[1])) {
				t.Fatal(pair, e)
			}
		}
	}
}

func TestRationalSourceInheritedIntegerAssertionsAndSelfAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		z := m.Integers()
		var v, v2, v3 Int
		// src/test/mpq.cpp::tst1 uses inherited integer operations. Decimal parsing
		// remains independent fixture setup rather than a claimed parser port.
		source, _ := new(big.Int).SetString("1002034040050606089383838288182", 10)
		setIntegerOracle(t, z, &v, source)
		minusTwo := NewInt(-2)
		if e := z.Mul(&v, &minusTwo, &v2); e != nil {
			t.Fatal(e)
		}
		if e := z.Add(&v, &v2, &v3); e != nil {
			t.Fatal(e)
		}
		if e := z.Neg(&v3); e != nil {
			t.Fatal(e)
		}
		if cmp, e := z.Compare(&v, &v3); e != nil || cmp != 0 {
			t.Fatal("source eq/le/ge and !neq", e)
		}
		if cmp, e := z.Compare(&v2, &v); e != nil || cmp >= 0 {
			t.Fatal("source lt/le/gt/ge and neq", e)
		}
		z.Del(&v)
		z.Del(&v2)
		z.Del(&v3)
		a := NewRational()
		m.SetFraction(&a, -7, 3)
		if e := m.Copy(&a, &a); e != nil || rationalString(t, m, &a) != "-7/3" {
			t.Fatal(e)
		}
		if e := m.Move(&a, &a); e != nil {
			t.Fatal(e)
		}
		if e := m.Add(&a, &a, &a); e != nil || rationalString(t, m, &a) != "-14/3" {
			t.Fatal(e)
		}
		if e := m.Sub(&a, &a, &a); e != nil || rationalString(t, m, &a) != "0" {
			t.Fatal(e)
		}
		m.SetFraction(&a, -7, 3)
		if e := m.Mul(&a, &a, &a); e != nil || rationalString(t, m, &a) != "49/9" {
			t.Fatal(e)
		}
		if e := m.Div(&a, &a, &a); e != nil || rationalString(t, m, &a) != "1" {
			t.Fatal(e)
		}
	}
}

func TestRationalConstructorAllocationFailures(t *testing.T) {
	successes, failures := 0, 0
	for after := int64(0); after < 128; after++ {
		budget := arena.NewBudget(1 << 20)
		budget.FailAfter(after)
		m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 8, Descriptors: arena.Config{PageSize: 1, Budget: budget}}}})
		if e != nil {
			failures++
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			constructionFailureOwningCleanup(t, e, budget)
		} else {
			successes++
			m.Close()
		}
		if budget.Used() != 0 {
			t.Fatal("constructor header/base unwind", after, budget.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
}

func TestRationalArithmeticFailureCleanup(t *testing.T) {
	an, ad, bn, bd := new(big.Int).Lsh(big.NewInt(1), 300), new(big.Int).Lsh(big.NewInt(1), 170), new(big.Int).Lsh(big.NewInt(1), 280), new(big.Int).Lsh(big.NewInt(1), 180)
	an.Add(an, big.NewInt(7))
	an.Neg(an)
	ad.Add(ad, big.NewInt(3))
	bn.Add(bn, big.NewInt(13))
	bd.Add(bd, big.NewInt(5))
	ar, br := new(big.Rat).SetFrac(an, ad), new(big.Rat).SetFrac(bn, bd)
	for _, synch := range []bool{false, true} {
		for op := 0; op < 5; op++ {
			for alias := 0; alias < 3; alias++ {
				successes, failures := 0, 0
				want := new(big.Rat)
				switch op {
				case 0:
					want.Add(ar, br)
				case 1:
					want.Sub(ar, br)
				case 2:
					want.Mul(ar, br)
				case 3:
					want.Quo(ar, br)
				case 4:
					want.Inv(ar)
				}
				for after := int64(0); after < 1024; after++ {
					m, budget := rationalManager(t, synch)
					a, b, out := NewRational(), NewRational(), NewRational()
					setRationalOracle(t, m, &a, an, ad)
					setRationalOracle(t, m, &b, bn, bd)
					m.SetInt64(&out, 17)
					dst := &out
					if alias == 1 {
						dst = &a
					}
					if alias == 2 {
						dst = &b
					}
					call := func() error {
						switch op {
						case 0:
							return m.Add(&a, &b, dst)
						case 1:
							return m.Sub(&a, &b, dst)
						case 2:
							return m.Mul(&a, &b, dst)
						case 3:
							return m.Div(&a, &b, dst)
						default:
							return m.Inv(&a, dst)
						}
					}
					beforeNum, beforeDen := integerString(t, m.Z, &dst.Num), integerString(t, m.Z, &dst.Den)
					budget.FailAfter(after)
					err := call()
					qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
					budget.FailAfter(-1)
					if err != nil {
						failures++
						if !numericPhysicalReservationRefusal(err) {
							t.Fatal("unexpected rational failure", synch, op, alias, after, err)
						}
						if op == 2 {
							productNum := new(big.Int).Mul(ar.Num(), br.Num()).String()
							productDen := new(big.Int).Mul(ar.Denom(), br.Denom()).String()
							num := powerPublicationComponent(t, m.Z, &dst.Num, beforeNum, productNum, want.Num().String())
							powerPublicationComponent(t, m.Z, &dst.Den, beforeDen, productDen, want.Denom().String())
							if num == "failed" || num == beforeNum {
								if roundPublicationDecode(t, m.Z, &dst.Den).String() != beforeDen {
									t.Fatal("denominator acted before numerator Mul completed", synch, alias, after)
								}
							}
							if dst != &a && rationalIndependentString(t, m, &a) != ar.RatString() {
								t.Fatal("non-destination a changed", synch, alias, after)
							}
							if dst != &b && rationalIndependentString(t, m, &b) != br.RatString() {
								t.Fatal("non-destination b changed", synch, alias, after)
							}
						} else if op == 3 || op == 4 {
							if op == 3 {
								rawNum := new(big.Int).Mul(ar.Num(), br.Denom()).String()
								rawDen := new(big.Int).Mul(ar.Denom(), br.Num()).String()
								powerPublicationComponent(t, m.Z, &dst.Num, beforeNum, rawNum, want.Num().String())
								powerPublicationComponent(t, m.Z, &dst.Den, beforeDen, rawDen, want.Denom().String())
							} else {
								absNum := new(big.Int).Abs(new(big.Int).Set(ar.Num())).String()
								negDen := new(big.Int).Neg(new(big.Int).Set(ar.Denom())).String()
								powerPublicationComponent(t, m.Z, &dst.Num, beforeNum, ar.Num().String(), absNum, negDen)
								powerPublicationComponent(t, m.Z, &dst.Den, beforeDen, ar.Denom().String(), negDen, absNum)
							}
							if dst != &a && rationalIndependentString(t, m, &a) != ar.RatString() {
								t.Fatal("non-destination a changed", op, alias, after)
							}
							if dst != &b && rationalIndependentString(t, m, &b) != br.RatString() {
								t.Fatal("non-destination b changed", op, alias, after)
							}
						} else if op == 0 || op == 1 {
							pn := new(big.Int).Mul(ar.Num(), br.Denom())
							cross := new(big.Int).Mul(br.Num(), ar.Denom())
							if op == 0 {
								pn.Add(pn, cross)
							} else {
								pn.Sub(pn, cross)
							}
							pd := new(big.Int).Mul(ar.Denom(), br.Denom())
							addPublicationExpectRaw(t, m, dst, beforeNum, beforeDen, pn, pd)
							if dst != &a && rationalIndependentString(t, m, &a) != ar.RatString() {
								t.Fatal("non-destination a changed", op, alias, after)
							}
							if dst != &b && rationalIndependentString(t, m, &b) != br.RatString() {
								t.Fatal("non-destination b changed", op, alias, after)
							}
						} else if rationalIndependentString(t, m, &a) != ar.RatString() || rationalIndependentString(t, m, &b) != br.RatString() || rationalIndependentString(t, m, &out) != "17" {
							t.Fatal("failed rational operation changed destination/input", synch, op, alias, after, err)
						}
						// Source TMP can be transferred before its replacement allocation
						// fails. No implicit scratch repair or retry contract exists.
					} else {
						successes++
					}
					if err == nil && rationalString(t, m, dst) != want.RatString() {
						t.Fatal("successful rational result", synch, op, alias, after)
					}
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("failed rational owner leaked budget", budget.Used())
					}
					if err == nil {
						break
					}
				}
				// Inv may reuse all storage already reserved by fixture normalization.
				if successes == 0 || (failures == 0 && op != 4) {
					t.Fatal("fault sweep completion boundary", synch, op, alias, failures, successes)
				}
			}
		}
	}
}

func TestRationalCopySetterFailureCleanup(t *testing.T) {
	n, d := new(big.Int).Lsh(big.NewInt(1), 400), new(big.Int).Lsh(big.NewInt(1), 170)
	n.Add(n, big.NewInt(7))
	d.Add(d, big.NewInt(3))
	for _, synch := range []bool{false, true} {
		for op := 0; op < 3; op++ {
			failures, successes := 0, 0
			for after := int64(0); after < 512; after++ {
				m, budget := rationalPublicationDirectOwner(t, synch)
				a, out := NewRational(), NewRational()
				setRationalOracle(t, m, &a, n, d)
				m.SetInt64(&out, 17)
				var numerator, denominator Int
				setIntegerOracle(t, m.Z, &numerator, n)
				setIntegerOracle(t, m.Z, &denominator, d)
				call := func() error {
					switch op {
					case 0:
						return m.Copy(&out, &a)
					case 1:
						return m.SetParts(&out, &numerator, &denominator)
					default:
						return m.SetFraction(&out, math.MinInt64, math.MaxUint64)
					}
				}
				budget.FailAfter(after)
				err := call()
				qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den, &numerator, &denominator})
				if err != nil && op != 0 {
					setterPairPrefix(t, m, &out, "17", "1", wantSetterNumerator(op, n), wantSetterDenominator(op, d), op == 1)
				}
				budget.FailAfter(-1)
				if err != nil {
					failures++
					if !numericPhysicalReservationRefusal(err) {
						t.Fatal("unexpected setter failure", synch, op, after, err)
					}
					if op == 0 {
						num := powerPublicationComponent(t, m.Z, &out.Num, "17", n.String())
						powerPublicationComponent(t, m.Z, &out.Den, "1", d.String())
						if num == "failed" || num == "17" {
							if out.Den != NewInt(1) {
								t.Fatal("denominator acted before numerator Copy", synch, after, out)
							}
						}
						if rationalString(t, m, &a) != new(big.Rat).SetFrac(n, d).RatString() {
							t.Fatal("Copy source changed", synch, after)
						}
					}
				} else {
					successes++
				}
				want := new(big.Rat).SetFrac(n, d)
				if op == 2 {
					want.SetFrac(big.NewInt(math.MinInt64), new(big.Int).SetUint64(math.MaxUint64))
				}
				if err == nil && rationalString(t, m, &out) != want.RatString() {
					t.Fatal(op, after)
				}
				m.Close()
				if budget.Used() != 0 {
					t.Fatal(budget.Used())
				}
			}
			// The two-limb setter can reuse the larger fixture pools without reservations.
			if successes == 0 || (failures == 0 && op != 2) {
				t.Fatal("setter failure boundary", synch, op, failures, successes)
			}
		}
	}
}

func TestRationalWarmStorageAndScratchModes(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a, b, out := NewRational(), NewRational(), NewRational()
		m.SetFraction(&a, 1<<40, 7)
		m.SetFraction(&b, (1<<40)+1, 11)
		for i := 0; i < 2; i++ {
			m.Add(&a, &b, &out)
			m.Sub(&a, &b, &out)
			m.Mul(&a, &b, &out)
			m.Div(&a, &b, &out)
			m.Compare(&a, &b)
		}
		beforeWarm := numericWarmProfileSnapshot(t, m.Z, budget)
		var operationError error
		allocations := testing.AllocsPerRun(20, func() {
			operationError = m.Add(&a, &b, &out)
			if operationError == nil {
				operationError = m.Sub(&a, &b, &out)
			}
			if operationError == nil {
				operationError = m.Mul(&a, &b, &out)
			}
			if operationError == nil {
				operationError = m.Div(&a, &b, &out)
			}
			if operationError == nil {
				_, operationError = m.Compare(&a, &b)
			}
		})
		if operationError != nil {
			t.Fatal("warm rational storage allocates", synch, allocations, operationError)
		}
		numericWarmProfileAudit(t, m.Z, budget, beforeWarm, allocations, m.Z.Allocator.Stats().DirectObjects > 0)
		if synch {
			if m.AddTmp1.Cell != 0 || m.AddTmp2.Cell != 0 || m.LtTmp1.Num.Cell != 0 || m.LtTmp2.Num.Cell != 0 || m.NTmp.Cell != 0 {
				t.Fatal("local-temp mode retained caches")
			}
		} else {
			if m.AddTmp1.Cell == 0 || m.AddTmp2.Cell == 0 || m.LtTmp1.Num.Cell == 0 || m.LtTmp2.Num.Cell == 0 {
				t.Fatal("reusable scratch mode did not retain source cache cells")
			}
		}
	}
}

func TestRationalColdSetterFailureCleanup(t *testing.T) {
	for _, synch := range []bool{false, true} {
		failures, successes := 0, 0
		for after := int64(0); after < 128; after++ {
			m, budget := rationalPublicationDirectOwner(t, synch)
			out := NewRational()
			budget.FailAfter(after)
			err := m.SetFraction(&out, math.MinInt64, math.MaxUint64)
			qopsAuditLiveOwnership(t, m, []*Int{&out.Num, &out.Den})
			if err != nil {
				setterPairPrefix(t, m, &out, "0", "1", big.NewInt(math.MinInt64), new(big.Int).SetUint64(math.MaxUint64), false)
			}
			budget.FailAfter(-1)
			if err != nil {
				failures++
				if !numericPhysicalReservationRefusal(err) {
					t.Fatal(after, err)
				}
			} else {
				successes++
			}
			want := new(big.Rat).SetFrac(big.NewInt(math.MinInt64), new(big.Int).SetUint64(math.MaxUint64))
			if err == nil && rationalString(t, m, &out) != want.RatString() {
				t.Fatal(after)
			}
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
		if failures == 0 || successes == 0 {
			t.Fatal("cold setter failure boundary", failures, successes)
		}
	}
}

func TestRationalColdInverseFailureCleanup(t *testing.T) {
	for _, synch := range []bool{false, true} {
		failures, successes := 0, 0
		for after := int64(0); after < 128; after++ {
			m, budget := rationalPublicationDirectOwner(t, synch)
			a := Rational{Num: NewInt(math.MinInt32), Den: NewInt(1)}
			budget.FailAfter(after)
			err := m.Inv(&a, &a)
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
			budget.FailAfter(-1)
			if err != nil {
				failures++
				if !numericPhysicalReservationRefusal(err) || rationalString(t, m, &a) != "-2147483648" {
					t.Fatal(after, err)
				}
				// No arithmetic retry after failure: the original source state is independently checked.
			} else {
				successes++
			}
			if err == nil && rationalString(t, m, &a) != "-1/2147483648" {
				t.Fatal(after)
			}
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
		if failures == 0 || successes == 0 {
			t.Fatal("cold inverse boundary", failures, successes)
		}
	}
}

// Exact paired actions for SetParts/SetFraction: NUM acts before DEN, and
// normalization divides NUM before DEN. This is not a Cartesian prefix model.
func setterPairPrefix(t *testing.T, m *RationalManager, out *Rational, oldN, oldD string, n, d *big.Int, copyGrowth bool) {
	t.Helper()
	component := func(v *Int) string {
		if m.Z.IsFailureRef(v.Cell) {
			if !copyGrowth {
				t.Fatal("scalar initial allocation cannot publish failed Copy token")
			}
			return "failed"
		}
		return roundPublicationDecode(t, m.Z, v).String()
	}
	nn, dd := component(&out.Num), component(&out.Den)
	normalized := new(big.Rat).SetFrac(n, d)
	allowed := [][2]string{{oldN, oldD}, {n.String(), oldD}, {n.String(), d.String()}, {normalized.Num().String(), d.String()}, {normalized.Num().String(), normalized.Denom().String()}}
	if copyGrowth {
		allowed = append(allowed, [2]string{"failed", oldD}, [2]string{n.String(), "failed"})
	}
	for _, p := range allowed {
		if nn == p[0] && dd == p[1] {
			return
		}
	}
	t.Fatal("outside source setter paired actions", nn, dd, allowed)
}
func wantSetterNumerator(op int, n *big.Int) *big.Int {
	if op == 2 {
		return big.NewInt(math.MinInt64)
	}
	return n
}
func wantSetterDenominator(op int, d *big.Int) *big.Int {
	if op == 2 {
		return new(big.Int).SetUint64(math.MaxUint64)
	}
	return d
}

// Raw source component inspection after a failed action must not consume or
// allocate the formatter's numerical scratch. No normalization hides aliases.
func rationalIndependentString(t *testing.T, m *RationalManager, a *Rational) string {
	t.Helper()
	n := roundPublicationDecode(t, m.Z, &a.Num).String()
	d := roundPublicationDecode(t, m.Z, &a.Den).String()
	if d == "1" {
		return n
	}
	return n + "/" + d
}
