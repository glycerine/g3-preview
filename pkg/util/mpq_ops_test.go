// Source: src/util/mpq.h; src/util/mpq.cpp.
// Source: mpq.h mixed/fused overloads; rational.cpp active tst9 assertions.
package util

import (
	"errors"
	"math/big"
	"testing"
)

func qopsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func qopsEqual(t *testing.T, m *RationalManager, q *Rational, want *big.Rat) {
	t.Helper()
	got, ok := new(big.Rat).SetString(rationalString(t, m, q))
	if !ok || got.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", rationalString(t, m, q), want)
	}
}
func TestRationalOpsSourceTst9All24Assertions(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for _, tc := range [][2]int64{{41000000000000, -7000000000000}, {-41000000000000, -7000000000000}, {-41000000000000, 7000000000000}, {41000000000000, 7000000000000}, {41, -7}, {-41, -7}, {-41, 7}, {41, 7}} {
			a, b, q, r := NewRational(), NewRational(), NewRational(), NewRational()
			qopsMust(t, m.SetInt64(&a, tc[0]))
			qopsMust(t, m.SetInt64(&b, tc[1]))
			qopsMust(t, m.IDiv(&a, &b, &q))
			qopsMust(t, m.Mod(&a, &b, &r))
			av, bv := big.NewInt(tc[0]), big.NewInt(tc[1])
			qv, _ := new(big.Int).SetString(rationalString(t, m, &q), 10)
			rv, _ := new(big.Int).SetString(rationalString(t, m, &r), 10)
			// Each active tstmod call has precisely these three ENSURE assertions.
			if av.Cmp(new(big.Int).Add(new(big.Int).Mul(bv, qv), rv)) != 0 {
				t.Fatal("source reconstruction", tc)
			}
			if rv.Sign() < 0 {
				t.Fatal("source nonnegative remainder", tc)
			}
			if rv.Cmp(new(big.Int).Abs(bv)) >= 0 {
				t.Fatal("source remainder bound", tc)
			}
			for _, n := range []*Rational{&a, &b, &q, &r} {
				qopsMust(t, m.Del(n))
			}
		}
	}
}
func TestRationalOpsMixedFiniteOraclesAndAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		var z Int
		for n := int64(-7); n <= 7; n++ {
			for d := uint64(1); d <= 5; d++ {
				for coefficient := int64(-3); coefficient <= 3; coefficient++ {
					qopsMust(t, m.Z.SetInt64(&z, coefficient))
					oracle := new(big.Rat).SetFrac(big.NewInt(n), new(big.Int).SetUint64(d))
					for _, alias := range []bool{false, true} {
						dst := &out
						if alias {
							dst = &a
						}
						qopsMust(t, m.SetFraction(&a, n, d))
						qopsMust(t, m.AddInt(&a, &z, dst))
						qopsEqual(t, m, dst, new(big.Rat).Add(oracle, new(big.Rat).SetInt64(coefficient)))
						qopsMust(t, m.SetFraction(&a, n, d))
						qopsMust(t, m.MulInt(&z, &a, dst))
						qopsEqual(t, m, dst, new(big.Rat).Mul(oracle, new(big.Rat).SetInt64(coefficient)))
						if coefficient != 0 {
							qopsMust(t, m.SetFraction(&a, n, d))
							qopsMust(t, m.DivInt(&a, &z, dst))
							qopsEqual(t, m, dst, new(big.Rat).Quo(oracle, new(big.Rat).SetInt64(coefficient)))
						}
					}
					if coefficient != 0 {
						qopsMust(t, m.SetFraction(&a, n, d))
						qopsMust(t, m.AccDiv(&a, &z))
						qopsEqual(t, m, &a, new(big.Rat).Quo(oracle, new(big.Rat).SetInt64(coefficient)))
					}
				}
			}
		}
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&out))
		qopsMust(t, m.Z.Del(&z))
	}
}
func TestRationalOpsFusedAllOutputAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
		var z Int
		for _, coefficient := range []int64{-3, -1, 0, 1, 4} {
			for _, denominator := range []uint64{1, 3} {
				for alias := 0; alias < 4; alias++ {
					for _, subtract := range []bool{false, true} {
						reset := func() {
							qopsMust(t, m.SetFraction(&a, 7, 5))
							qopsMust(t, m.SetFraction(&b, coefficient, denominator))
							qopsMust(t, m.SetFraction(&c, -11, 7))
							qopsMust(t, m.Z.SetInt64(&z, coefficient))
						}
						reset()
						dst := []*Rational{&out, &a, &b, &c}[alias]
						product := new(big.Rat).Mul(new(big.Rat).SetFrac(big.NewInt(coefficient), new(big.Int).SetUint64(denominator)), big.NewRat(-11, 7))
						want := new(big.Rat)
						if subtract {
							want.Sub(big.NewRat(7, 5), product)
							qopsMust(t, m.SubMul(&a, &b, &c, dst))
						} else {
							want.Add(big.NewRat(7, 5), product)
							qopsMust(t, m.AddMul(&a, &b, &c, dst))
						}
						qopsEqual(t, m, dst, want)
						reset()
						product.Mul(new(big.Rat).SetInt64(coefficient), big.NewRat(-11, 7))
						if subtract {
							want.Sub(big.NewRat(7, 5), product)
							qopsMust(t, m.SubMulInt(&a, &z, &c, dst))
						} else {
							want.Add(big.NewRat(7, 5), product)
							qopsMust(t, m.AddMulInt(&a, &z, &c, dst))
						}
						qopsEqual(t, m, dst, want)
					}
				}
			}
		}
		for _, n := range []*Rational{&a, &b, &c, &out} {
			qopsMust(t, m.Del(n))
		}
		qopsMust(t, m.Z.Del(&z))
	}
}
func TestRationalOpsIntegerWrappersOracleAndSourceAliasQuirks(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, b, out := NewRational(), NewRational(), NewRational()
		var zi Int
		for av := int64(-19); av <= 19; av++ {
			for bv := int64(-7); bv <= 7; bv++ {
				if bv == 0 {
					continue
				}
				qopsMust(t, m.SetInt64(&a, av))
				qopsMust(t, m.SetInt64(&b, bv))
				eq, er := new(big.Int), new(big.Int)
				eq.DivMod(big.NewInt(av), big.NewInt(bv), er)
				tq, tr := new(big.Int), new(big.Int)
				tq.QuoRem(big.NewInt(av), big.NewInt(bv), tr)
				cases := []struct {
					q    func(*Rational, *Rational, *Rational) error
					z    func(*Rational, *Rational, *Int) error
					want *big.Int
				}{{m.IDiv, m.IDivToInt, eq}, {m.MachineIDiv, m.MachineIDivToInt, tq}, {m.Rem, m.RemToInt, tr}, {m.Mod, m.ModToInt, er}}
				for _, tc := range cases {
					qopsMust(t, tc.q(&a, &b, &out))
					qopsEqual(t, m, &out, new(big.Rat).SetInt(tc.want))
					qopsMust(t, tc.z(&a, &b, &zi))
					if integerString(t, m.Z, &zi) != tc.want.String() {
						t.Fatal("integer output")
					}
				}
			}
		}
		// mpz.h source rereads the output-aliased divisor. mpq wrappers delegate
		// directly and preserve these permanent control-flow fixtures.
		for _, tc := range []struct{ b, want int64 }{{7, -2}, {-7, 2}} {
			qopsMust(t, m.SetInt64(&a, -26))
			qopsMust(t, m.SetInt64(&b, tc.b))
			qopsMust(t, m.IDiv(&a, &b, &b))
			qopsEqual(t, m, &b, new(big.Rat).SetInt64(tc.want))
			qopsMust(t, m.SetInt64(&b, tc.b))
			qopsMust(t, m.Mod(&a, &b, &b))
			qopsEqual(t, m, &b, new(big.Rat))
		}
		for _, n := range []*Rational{&a, &b, &out} {
			qopsMust(t, m.Del(n))
		}
		qopsMust(t, m.Z.Del(&zi))
	}
}
func TestRationalOpsIntegerProductsRatDivQuirksAndGuards(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		q := NewRational()
		a, b := NewInt(6), NewInt(-9)
		qopsMust(t, m.MulInts(&a, &b, &q))
		qopsEqual(t, m, &q, big.NewRat(-54, 1))
		qopsMust(t, m.RatDiv(&a, &b, &q))
		if rationalString(t, m, &q) != "2/-3" {
			t.Fatal("rat_div must not repair source negative denominator")
		}
		zero := NewInt(0)
		qopsMust(t, m.DivInt(&q, &zero, &q))
		divPublicationAudit(t, m, &q)
		if q.Num != NewInt(1) || q.Den != NewInt(0) {
			t.Fatal("source finite division zero", q)
		}
		qopsMust(t, m.Reset(&q))
		qopsMust(t, m.RatDiv(&a, &zero, &q))
		divPublicationAudit(t, m, &q)
		if q.Num != NewInt(1) || q.Den != NewInt(0) {
			t.Fatal("source finite rat_div zero", q)
		}
		qopsMust(t, m.Reset(&q))
		x, y := NewRational(), NewRational()
		qopsMust(t, m.SetFraction(&x, 1, 2))
		qopsMust(t, m.SetInt64(&y, 2))
		e := m.IDiv(&x, &y, &q)
		divPublicationAudit(t, m, &q, &x, &y)
		if modularSourceDebug {
			if !errors.Is(e, ErrRationalNotInteger) || q != NewRational() {
				t.Fatal(e)
			}
		} else {
			qopsMust(t, e)
			qopsEqual(t, m, &q, new(big.Rat))
		}

		for _, n := range []*Rational{&q, &x, &y} {
			qopsMust(t, m.Del(n))
		}
	}
}

func TestRationalOpsSourceComponentAliasActionOrder(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		six := NewInt(6)
		qopsMust(t, m.SetInt64(&out, -9))
		qopsMust(t, m.RatDiv(&six, &out.Num, &out))
		if rationalString(t, m, &out) != "1" {
			t.Fatal("rat_div rereads output numerator divisor", rationalString(t, m, &out))
		}
		qopsMust(t, m.SetFraction(&a, 2, 3))
		qopsMust(t, m.SetInt64(&out, -5))
		qopsMust(t, m.DivInt(&a, &out.Num, &out))
		if rationalString(t, m, &out) != "1/3" {
			t.Fatal("mixed div rereads output numerator divisor", rationalString(t, m, &out))
		}
		minusNine := NewInt(-9)
		qopsMust(t, m.RatDiv(&six, &minusNine, &a))
		qopsMust(t, m.AccDiv(&a, &a.Den))
		if rationalString(t, m, &a) != "2/9" {
			t.Fatal("acc_div sign reads overwritten denominator", rationalString(t, m, &a))
		}
		// Source mixed add/mul cross-products consume the integer before
		// output denominator assignment. Freeze both same-q and foreign-output
		// component borrows rather than assuming their numerical equivalence.
		qopsMust(t, m.SetFraction(&a, 2, 3))
		qopsMust(t, m.AddInt(&a, &a.Den, &a))
		if rationalString(t, m, &a) != "11/3" {
			t.Fatal("mixed add same denominator")
		}
		qopsMust(t, m.SetFraction(&a, 2, 3))
		qopsMust(t, m.SetFraction(&out, 1, 5))
		qopsMust(t, m.AddInt(&a, &out.Den, &out))
		if rationalString(t, m, &out) != "17/3" {
			t.Fatal("mixed add foreign denominator")
		}
		qopsMust(t, m.SetFraction(&a, 2, 3))
		qopsMust(t, m.MulInt(&a.Den, &a, &a))
		if rationalString(t, m, &a) != "2" {
			t.Fatal("mixed multiply same denominator")
		}
		qopsMust(t, m.SetFraction(&a, 2, 3))
		qopsMust(t, m.SetFraction(&out, 1, 5))
		qopsMust(t, m.MulInt(&out.Den, &a, &out))
		if rationalString(t, m, &out) != "10/3" {
			t.Fatal("mixed multiply foreign denominator")
		}
		qopsMust(t, m.SetFraction(&out, 2, 3))
		qopsMust(t, m.MulInts(&out.Num, &out.Den, &out))
		if rationalString(t, m, &out) != "6" {
			t.Fatal("integer product components")
		}
		two, minusThree, minusFive := NewInt(2), NewInt(-3), NewInt(-5)
		qopsMust(t, m.RatDiv(&two, &minusThree, &a))
		qopsMust(t, m.RatDiv(&two, &minusFive, &out))
		qopsMust(t, m.DivInt(&a, &out.Den, &out))
		if rationalString(t, m, &out) != "2/15" {
			t.Fatal("mixed div sign reads output denominator", rationalString(t, m, &out))
		}
		qopsMust(t, m.RatDiv(&two, &minusThree, &a))
		qopsMust(t, m.SetFraction(&out, 1, 5))
		qopsMust(t, m.DivInt(&a, &out.Den, &out))
		if rationalString(t, m, &out) != "-2/15" {
			t.Fatal("mixed div output denominator sign repair")
		}
		qopsMust(t, m.RatDiv(&two, &minusFive, &out))
		qopsMust(t, m.RatDiv(&out.Den, &out.Den, &out))
		if rationalString(t, m, &out) != "-1/-1" {
			t.Fatal("rat_div shared negative denominator")
		}
		qopsMust(t, m.SetFraction(&out, 2, 3))
		qopsMust(t, m.RatDiv(&out.Den, &out.Num, &out))
		if rationalString(t, m, &out) != "1" {
			t.Fatal("rat_div both output component inputs")
		}
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&out))
	}
}

func TestRationalOpsLargeLimbScratchBranchesAndWarm(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
		var z Int
		an := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 120), big.NewInt(7))
		ad := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 70), big.NewInt(5))
		bn := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 130), big.NewInt(9))
		cn := new(big.Int).Neg(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(3)))
		cd := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 80), big.NewInt(13))
		setRationalOracle(t, m, &a, an, ad)
		setRationalOracle(t, m, &b, bn, big.NewInt(17))
		setRationalOracle(t, m, &c, cn, cd)
		setIntegerOracle(t, m.Z, &z, bn)
		ar, br, cr := new(big.Rat).SetFrac(an, ad), new(big.Rat).SetFrac(bn, big.NewInt(17)), new(big.Rat).SetFrac(cn, cd)
		qopsMust(t, m.AddMul(&a, &b, &c, &out))
		qopsEqual(t, m, &out, new(big.Rat).Add(ar, new(big.Rat).Mul(br, cr)))
		qopsMust(t, m.SubMulInt(&a, &z, &c, &out))
		qopsEqual(t, m, &out, new(big.Rat).Sub(ar, new(big.Rat).Mul(new(big.Rat).SetInt(bn), cr)))
		if synch {
			if rationalString(t, m, &m.AddMulTmp) != "0" {
				t.Fatal("synchronized touched cached rational")
			}
		} else {
			qopsEqual(t, m, &m.AddMulTmp, new(big.Rat).Mul(new(big.Rat).SetInt(bn), cr))
		}
		cached := rationalString(t, m, &m.AddMulTmp)
		unit := NewRational()
		qopsMust(t, m.SetInt64(&unit, 1))
		qopsMust(t, m.AddMul(&a, &unit, &c, &out))
		qopsEqual(t, m, &out, new(big.Rat).Add(ar, cr))
		if rationalString(t, m, &m.AddMulTmp) != cached {
			t.Fatal("unit fast branch used multiplication scratch")
		}
		qopsMust(t, m.AddInt(&a, &z, &out))
		qopsEqual(t, m, &out, new(big.Rat).Add(ar, new(big.Rat).SetInt(bn)))
		if !synch {
			want := new(big.Int).Mul(bn, new(big.Rat).SetFrac(an, ad).Denom())
			if integerString(t, m.Z, &m.AddTmp1) != want.String() {
				t.Fatal("mixed add cached source cross-product")
			}
		}
		// Warm existing scalar/cell/word workspaces before measuring. The manager
		// retains source nonsynch scratch values; no per-numeral Go object is boxed.
		for i := 0; i < 20; i++ {
			qopsMust(t, m.AddMulInt(&a, &z, &c, &out))
		}
		beforeWarm := numericWarmProfileSnapshot(t, m.Z, budget)
		allocs := testing.AllocsPerRun(20, func() {
			if e := m.AddMulInt(&a, &z, &c, &out); e != nil {
				panic(e)
			}
		})
		numericWarmProfileAudit(t, m.Z, budget, beforeWarm, allocs, m.Z.Allocator.Stats().DirectObjects > 0)
		qopsMust(t, m.Del(&unit))
		for _, q := range []*Rational{&a, &b, &c, &out} {
			qopsMust(t, m.Del(q))
		}
		qopsMust(t, m.Z.Del(&z))
	}
}
func TestRationalOpsAllPredictableFailurePrefixes(t *testing.T) {
	// Every operation eventually succeeds as FailAfter advances; each earlier
	// failure preserves public inputs/output. Retained manager scratch is allowed
	// to change like source; closing the complete manager refunds every byte.
	for _, synch := range []bool{false, true} {
		for operation := 0; operation < 12; operation++ {
			success := false
			for prefix := int64(0); prefix < 512; prefix++ {
				m, budget := rationalManager(t, synch)
				a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
				var z Int
				n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 185), big.NewInt(29))
				d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 95), big.NewInt(11))
				setRationalOracle(t, m, &a, n, d)
				setRationalOracle(t, m, &b, n, big.NewInt(19))
				setRationalOracle(t, m, &c, new(big.Int).Neg(n), d)
				setIntegerOracle(t, m.Z, &z, n)
				qopsMust(t, m.SetInt64(&out, 17))
				if operation == 7 {
					qopsMust(t, m.Copy(&out, &a))
				}
				if operation >= 8 {
					setRationalOracle(t, m, &a, new(big.Int).Neg(n), big.NewInt(1))
					setRationalOracle(t, m, &b, d, big.NewInt(1))
				}
				originalOutput := rationalString(t, m, &out)
				sa, sb, sc := rationalString(t, m, &a), rationalString(t, m, &b), rationalString(t, m, &c)
				budget.FailAfter(prefix)
				var e error
				switch operation {
				case 0:
					e = m.AddInt(&a, &z, &out)
				case 1:
					e = m.MulInt(&z, &a, &out)
				case 2:
					e = m.DivInt(&a, &z, &out)
				case 3:
					e = m.RatDiv(&z, &z, &out)
				case 4:
					e = m.AddMul(&a, &b, &c, &out)
				case 5:
					e = m.SubMulInt(&a, &z, &c, &out)
				case 6:
					e = m.MulInts(&z, &z, &out)
				case 7:
					e = m.AccDiv(&out, &z)
				case 8:
					e = m.IDiv(&a, &b, &out)
				case 9:
					e = m.MachineIDiv(&a, &b, &out)
				case 10:
					e = m.Rem(&a, &b, &out)
				case 11:
					e = m.Mod(&a, &b, &out)
				}
				// Inspect ownership before formatting, cleanup or owner Close can
				// allocate scratch or mask an orphaned temporary.
				if e != nil {
					qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &c.Num, &c.Den, &out.Num, &out.Den, &z})
				}
				budget.FailAfter(-1)
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal("unexpected failure", operation, prefix, e)
					}
					if operation == 2 || operation == 3 || operation == 7 {
						old, _ := new(big.Rat).SetString(originalOutput)
						var pn, pd *big.Int
						switch operation {
						case 2:
							pn = new(big.Int).Set(new(big.Rat).SetFrac(n, d).Num())
							pd = new(big.Int).Mul(new(big.Rat).SetFrac(n, d).Denom(), n)
						case 3:
							pn = new(big.Int).Set(n)
							pd = new(big.Int).Set(n)
						case 7:
							pn = new(big.Int).Set(old.Num())
							pd = new(big.Int).Mul(old.Denom(), n)
						}
						reduced := new(big.Rat).SetFrac(pn, pd)
						powerPublicationComponent(t, m.Z, &out.Num, old.Num().String(), pn.String(), reduced.Num().String())
						powerPublicationComponent(t, m.Z, &out.Den, old.Denom().String(), pd.String(), reduced.Denom().String())
					} else if operation == 0 || operation == 1 || operation == 4 || operation == 5 || operation == 6 {
						ar, _ := new(big.Rat).SetString(sa)
						br, _ := new(big.Rat).SetString(sb)
						cr, _ := new(big.Rat).SetString(sc)
						old, _ := new(big.Rat).SetString(originalOutput)
						var pn, pd *big.Int
						switch operation {
						case 0:
							pn = new(big.Int).Add(ar.Num(), new(big.Int).Mul(n, ar.Denom()))
							pd = new(big.Int).Set(ar.Denom())
						case 1:
							pn = new(big.Int).Mul(n, ar.Num())
							pd = new(big.Int).Set(ar.Denom())
						case 6:
							pn = new(big.Int).Mul(n, n)
							pd = big.NewInt(1)
						default:
							tmp := new(big.Rat)
							if operation == 4 {
								tmp.Mul(br, cr)
							} else {
								tmp.Mul(new(big.Rat).SetInt(n), cr)
							}
							pn = new(big.Int).Mul(ar.Num(), tmp.Denom())
							cross := new(big.Int).Mul(tmp.Num(), ar.Denom())
							if operation == 4 {
								pn.Add(pn, cross)
							} else {
								pn.Sub(pn, cross)
							}
							pd = new(big.Int).Mul(ar.Denom(), tmp.Denom())
						}
						addPublicationExpectRaw(t, m, &out, old.Num().String(), old.Denom().String(), pn, pd)
					} else if operation >= 8 {
						// Raw forwarding acts on NUM first; source Div/Mod can publish
						// machine output before their signed correction fails.
						dividend := new(big.Int).Neg(new(big.Int).Set(n))
						quotient, remainder := new(big.Int), new(big.Int)
						quotient.QuoRem(dividend, d, remainder)
						allowed := []string{"17"}
						if operation == 8 || operation == 9 {
							allowed = append(allowed, quotient.String())
							if operation == 8 && remainder.Sign() != 0 {
								allowed = append(allowed, new(big.Int).Sub(quotient, big.NewInt(1)).String())
							}
						} else {
							allowed = append(allowed, remainder.String())
							if operation == 11 && remainder.Sign() < 0 {
								allowed = append(allowed, new(big.Int).Add(remainder, d).String())
							}
						}
						powerPublicationComponent(t, m.Z, &out.Num, allowed...)
						if out.Den != NewInt(1) {
							t.Fatal("DEN reset only after integer success", operation, prefix, out.Den)
						}
					} else if rationalString(t, m, &out) != originalOutput {
						t.Fatal("partial output", operation, prefix)
					}
				} else {
					success = true
				}
				if rationalString(t, m, &a) != sa || rationalString(t, m, &b) != sb || rationalString(t, m, &c) != sc {
					t.Fatal("mutated input", operation, prefix)
				}
				for _, q := range []*Rational{&a, &b, &c, &out} {
					qopsMust(t, m.Del(q))
				}
				qopsMust(t, m.Z.Del(&z))
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("failed prefix graph leaked", operation, prefix, budget.Used())
				}
				if success {
					break
				}
			}
			if !success {
				t.Fatal("no successful failure bound", operation)
			}
		}
	}
}

// Known roots, rather than scanning all live allocations, form the expected
// ownership graph. No unreferenced temporary can justify its own presence.
// The owner's failed allocation token is charged, permanently dead metadata,
// not a live cell or payload. Every other invalid known root is still rejected.
func qopsAuditLiveOwnership(t *testing.T, m *RationalManager, public []*Int) {
	t.Helper()
	roots := append([]*Int(nil), public...)
	scratch := []*Int{&m.NTmp, &m.AddTmp1, &m.AddTmp2, &m.AddMulTmp.Num, &m.AddMulTmp.Den, &m.LtTmp1.Num, &m.LtTmp1.Den, &m.LtTmp2.Num, &m.LtTmp2.Den}
	for _, n := range scratch {
		if m.Synchronized && n.Cell != 0 && n.Cell != m.Z.FailedRef {
			t.Fatal("local mode retained rational scratch")
		}
		if !m.Synchronized {
			roots = append(roots, n)
		}
	}
	raw := &m.RawArithmetic
	if raw.PowerActive {
		roots = append(roots, &raw.Power.Num, &raw.Power.Den)
	}
	if raw.AddActive {
		roots = append(roots, &raw.AddOne, &raw.AddTwo)
	}
	if raw.NormActive {
		roots = append(roots, &raw.Norm)
	}
	if raw.DivActive {
		roots = append(roots, &raw.Div)
	}

	if raw.CompareActive {
		roots = append(roots, &raw.CompareOne.Num, &raw.CompareOne.Den, &raw.CompareTwo.Num, &raw.CompareTwo.Den)
	}
	integerPhysicalAudit(t, m.Z, roots...)
}
