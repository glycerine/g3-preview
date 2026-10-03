// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/big"
	"testing"
)

func TestNumberTheorySourceRationalGCDAndSignedSequence(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		r1, r2, a, b, g := NewRational(), NewRational(), NewRational(), NewRational(), NewRational()
		qopsMust(t, m.SetInt64(&r1, 12))
		qopsMust(t, m.SetInt64(&r2, 8))
		qopsMust(t, m.GCD(&r1, &r2, &g))
		if rationalString(t, m, &g) != "4" {
			t.Fatal("rational.cpp tst1 ENSURE gcd12,8=4")
		}
		// All calls in active tst6 (the source has no ENSURE there); independently
		// strengthen each call with its exact signed Bezout identity and gcd oracle.
		for _, pair := range [][2]int64{{5, 3}, {15, 25}, {-15, 25}, {-15, -25}, {15, -25}, {25, 15}, {-25, 15}, {-25, -15}, {25, -15}} {
			qopsMust(t, m.SetInt64(&r1, pair[0]))
			qopsMust(t, m.SetInt64(&r2, pair[1]))
			qopsMust(t, m.ExtendedGCD(&r1, &r2, &a, &b, &g))
			ax, _ := new(big.Int).SetString(rationalString(t, m, &a), 10)
			bx, _ := new(big.Int).SetString(rationalString(t, m, &b), 10)
			want := new(big.Int).GCD(nil, nil, big.NewInt(pair[0]), big.NewInt(pair[1]))
			got := new(big.Int).Add(new(big.Int).Mul(ax, big.NewInt(pair[0])), new(big.Int).Mul(bx, big.NewInt(pair[1])))
			if got.Cmp(want) != 0 || rationalString(t, m, &g) != want.String() {
				t.Fatal("signed source sequence", pair)
			}
		}
		for _, q := range []*Rational{&r1, &r2, &a, &b, &g} {
			qopsMust(t, m.Del(q))
		}
	}
}
func TestNumberTheorySourceInverse999Cases(t *testing.T) {
	m, _ := rationalManager(t, true)
	n, p, g, x, y, product, modulus, parity := NewRational(), NewRational(), NewRational(), NewRational(), NewRational(), NewRational(), NewRational(), NewRational()
	qopsMust(t, m.SetInt64(&p, 1<<32))
	two := NewRational()
	qopsMust(t, m.SetInt64(&two, 2))
	// src/test/rational.cpp tst7 active property for every i=1..999. Source uses
	// ext_gcd.h's wrapper algorithm; this independently checks the manager variant
	// against the same mathematical property, without claiming ext_gcd.h ported.
	for i := int64(1); i < 1000; i++ {
		qopsMust(t, m.SetInt64(&n, i))
		qopsMust(t, m.ExtendedGCD(&n, &p, &x, &y, &g))
		qopsMust(t, m.Mod(&n, &two, &parity))
		if rationalString(t, m, &parity) == "1" {
			qopsMust(t, m.Mul(&n, &x, &product))
			qopsMust(t, m.Mod(&product, &p, &modulus))
			if rationalString(t, m, &modulus) != "1" {
				t.Fatal("source inverse ENSURE", i)
			}
		}
	}
	for _, q := range []*Rational{&n, &p, &g, &x, &y, &product, &modulus, &parity, &two} {
		qopsMust(t, m.Del(q))
	}
}
func TestNumberTheoryIntegerFiniteOraclesAndSignedLCM(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, x, y, g, c Int
	for av := int64(-27); av <= 27; av++ {
		for bv := int64(-17); bv <= 17; bv++ {
			qopsMust(t, m.SetInt64(&a, av))
			qopsMust(t, m.SetInt64(&b, bv))
			qopsMust(t, m.ExtendedGCD(&a, &b, &x, &y, &g))
			gx, _ := new(big.Int).SetString(integerString(t, m, &g), 10)
			xx, _ := new(big.Int).SetString(integerString(t, m, &x), 10)
			yy, _ := new(big.Int).SetString(integerString(t, m, &y), 10)
			want := new(big.Int).GCD(nil, nil, big.NewInt(av), big.NewInt(bv))
			identity := new(big.Int).Add(new(big.Int).Mul(xx, big.NewInt(av)), new(big.Int).Mul(yy, big.NewInt(bv)))
			if gx.Cmp(want) != 0 || identity.Cmp(want) != 0 {
				t.Fatal("Bezout", av, bv)
			}
			divides, e := m.Divides(&a, &b)
			qopsMust(t, e)
			expected := av == 0 && bv == 0 || av != 0 && bv%av == 0
			if divides != expected {
				t.Fatal("divides", av, bv)
			}
		}
	}
	for _, tc := range [][3]int64{{-6, 1, -6}, {1, -6, -6}, {-6, -6, -6}, {-6, 3, -6}, {6, -3, -6}, {-6, -3, 6}, {0, 0, 0}, {0, -7, 0}, {-7, 0, 0}, {6, 9, 18}, {-6, 9, -18}} {
		for alias := 0; alias < 3; alias++ {
			qopsMust(t, m.SetInt64(&a, tc[0]))
			qopsMust(t, m.SetInt64(&b, tc[1]))
			out := []*Int{&c, &a, &b}[alias]
			qopsMust(t, m.LCM(&a, &b, out))
			if integerString(t, m, out) != big.NewInt(tc[2]).String() {
				t.Fatal("source signed LCM", tc, alias)
			}
		}
	}
	for _, z := range []*Int{&a, &b, &x, &y, &g, &c} {
		qopsMust(t, m.Del(z))
	}
}
func TestNumberTheoryArraySourceOrderAndRationalSingleton(t *testing.T) {
	m, _ := integerManager(t)
	a := []Int{NewInt(6), NewInt(10), NewInt(9)}
	var out Int
	qopsMust(t, m.GCDMany(a, &a[2]))
	if integerString(t, m, &a[2]) != "2" {
		t.Fatal("future array output overwrite")
	}
	qopsMust(t, m.GCDMany(nil, &out))
	if out != NewInt(0) {
		t.Fatal("empty gcd")
	}
	single := []Int{NewInt(-14)}
	qopsMust(t, m.GCDMany(single, &out))
	if integerString(t, m, &out) != "14" {
		t.Fatal("single absolute gcd")
	}
	for _, synch := range []bool{false, true} {
		q, _ := rationalManager(t, synch)
		as := []Rational{NewRational(), NewRational(), NewRational()}
		g := NewRational()
		qopsMust(t, q.SetFraction(&as[0], -2, 3))
		qopsMust(t, q.GCDMany(as[:1], &g))
		if rationalString(t, q, &g) != "2/3" {
			t.Fatal("fractional singleton source gcd")
		}
		qopsMust(t, q.SetInt64(&as[0], 2))
		qopsMust(t, q.SetInt64(&as[1], 3))
		qopsMust(t, q.SetFraction(&as[2], 1, 2))
		qopsMust(t, q.GCDMany(as, &g))
		if rationalString(t, q, &g) != "1" {
			t.Fatal("early one must not inspect later fractional item")
		}
		qopsMust(t, q.SetInt64(&as[0], 6))
		qopsMust(t, q.SetInt64(&as[1], 10))
		qopsMust(t, q.SetInt64(&as[2], 9))
		qopsMust(t, q.GCDMany(as, &as[2]))
		if rationalString(t, q, &as[2]) != "2" {
			t.Fatal("rational future output alias")
		}
		qopsMust(t, q.SetFraction(&as[2], 1, 2))
		e := q.GCDMany(as, &g)
		qopsAuditLiveOwnership(t, q, []*Int{&as[0].num, &as[0].den, &as[1].num, &as[1].den, &as[2].num, &as[2].den, &g.num, &g.den})
		if modularSourceDebug {
			if !errors.Is(e, ErrRationalNotInteger) || rationalString(t, q, &g) != "2" {
				t.Fatal("source debug visited integrality prefix", e)
			}
		} else if e != nil || g.num != NewInt(1) || g.den != NewInt(1) {
			t.Fatal("source release visited numerator gcd", e, g)
		}
		qopsMust(t, q.GCDMany(nil, &g))
		if rationalString(t, q, &g) != "0" {
			t.Fatal("empty rational gcd")
		}
		for i := range as {
			qopsMust(t, q.Del(&as[i]))
		}
		qopsMust(t, q.Del(&g))
	}
}
func TestNumberTheoryFrozenExtendedGCDAliases(t *testing.T) {
	m, _ := integerManager(t)
	// Fixed release-source control-flow results; aliased coefficient identities
	// need not satisfy the ordinary Bezout postcondition after source overwrites.
	for _, tc := range []struct {
		a, b, g int
		want    [5]int64
	}{{2, 3, 4, [5]int64{-26, 7, -3, -11, 1}}, {0, 3, 4, [5]int64{3, 7, 0, -11, 1}}, {2, 1, 4, [5]int64{-26, 11, -3, 0, 1}}, {2, 2, 4, [5]int64{-26, 7, 5, 0, 1}}, {2, 3, 2, [5]int64{-26, 7, 1, -11, 0}}, {2, 3, 0, [5]int64{1, 7, -3, -11, 0}}} {
		values := [5]Int{NewInt(-26), NewInt(7), NewInt(0), NewInt(0), NewInt(0)}
		e := m.ExtendedGCD(&values[0], &values[1], &values[tc.a], &values[tc.b], &values[tc.g])
		// These three aliases fail the source debug Bezout assertion before
		// the final gcd copy. The other identities still reach publication.
		debugFailure := sourceExtendedGCDDebug && (tc.a == 0 || tc.b == 1 || tc.a == tc.b)
		if debugFailure {
			if !errors.Is(e, errExtendedGCDInvariant) {
				t.Fatal("source debug assertion", tc, e)
			}
			tc.want[tc.g] = 0
		} else {
			qopsMust(t, e)
		}
		for i := range values {
			if integerString(t, m, &values[i]) != big.NewInt(tc.want[i]).String() {
				t.Fatal("source alias fixture", tc, i, integerString(t, m, &values[i]))
			}
		}
	}
}

func TestNumberTheoryCanonicalSourceCoefficients(t *testing.T) {
	m, _ := integerManager(t)
	var r1, r2, a, b, g Int
	for _, tc := range [][5]int64{{5, 3, -1, 2, 1}, {15, 25, 2, -1, 5}, {-15, 25, -2, -1, 5}, {-15, -25, -2, 1, 5}, {15, -25, 2, 1, 5}, {25, 15, -1, 2, 5}, {-25, 15, 1, 2, 5}, {-25, -15, 1, -2, 5}, {25, -15, -1, -2, 5}, {0, 0, 1, 0, 0}, {0, 7, 0, 1, 7}, {7, 0, 1, 0, 7}, {7, 7, 0, 1, 7}} {
		qopsMust(t, m.SetInt64(&r1, tc[0]))
		qopsMust(t, m.SetInt64(&r2, tc[1]))
		qopsMust(t, m.ExtendedGCD(&r1, &r2, &a, &b, &g))
		for i, z := range []*Int{&a, &b, &g} {
			if integerString(t, m, z) != big.NewInt(tc[i+2]).String() {
				t.Fatal("source canonical coefficient sequence", tc, i)
			}
		}
	}
}
func TestNumberTheoryFailurePrefixRootsBeforeFormatting(t *testing.T) {
	for operation := 0; operation < 5; operation++ {
		succeeded := false
		for prefix := int64(0); prefix < 160; prefix++ {
			m, budget := rationalManager(t, true)
			as := []Rational{NewRational(), NewRational(), NewRational()}
			a, b, g := NewRational(), NewRational(), NewRational()
			n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 201), big.NewInt(17))
			d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 135), big.NewInt(9))
			setRationalOracle(t, m, &as[0], n, big.NewInt(1))
			setRationalOracle(t, m, &as[1], d, big.NewInt(1))
			setRationalOracle(t, m, &as[2], new(big.Int).Mul(n, d), big.NewInt(1))
			qopsMust(t, m.SetFraction(&a, 17, 3))
			qopsMust(t, m.SetFraction(&b, 19, 5))
			qopsMust(t, m.SetFraction(&g, 23, 7))
			budget.FailAfter(prefix)
			var e error
			switch operation {
			case 0:
				e = m.ExtendedGCD(&as[0], &as[1], &a, &b, &g)
			case 1:
				e = m.LCM(&as[0], &as[1], &g)
			case 2:
				_, e = m.Divides(&as[0], &as[1])
			case 3:
				e = m.GCDMany(as, &g)
			case 4:
				e = m.GCD(&as[0], &as[1], &g)
			}
			// Legitimately published source-prefix outputs are roots. Independent root
			// accounting cannot be satisfied by an orphaned local, even before Close.
			if e != nil {
				qopsAuditLiveOwnership(t, m, []*Int{&as[0].num, &as[0].den, &as[1].num, &as[1].den, &as[2].num, &as[2].den, &a.num, &a.den, &b.num, &b.den, &g.num, &g.den})
			}
			budget.FailAfter(-1)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("unexpected source prefix failure", operation, prefix, e)
			}
			if e == nil {
				succeeded = true
			}
			for i := range as {
				qopsMust(t, m.Del(&as[i]))
			}
			for _, q := range []*Rational{&a, &b, &g} {
				qopsMust(t, m.Del(q))
			}
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("number theory graph leak", operation, prefix)
			}
			if succeeded {
				break
			}
		}
		if !succeeded {
			t.Fatal("no successful injection bound", operation)
		}
	}
}

func TestNumberTheoryLargeCanonicalCoefficientsAndLCM(t *testing.T) {
	m, _ := integerManager(t)
	var r1, r2, a, b, g, c Int
	factor := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 165), big.NewInt(3))
	n, d := new(big.Int).Mul(big.NewInt(15), factor), new(big.Int).Mul(big.NewInt(25), factor)
	setIntegerOracle(t, m, &r1, n)
	setIntegerOracle(t, m, &r2, d)
	qopsMust(t, m.ExtendedGCD(&r1, &r2, &a, &b, &g))
	if integerString(t, m, &a) != "2" || integerString(t, m, &b) != "-1" || integerString(t, m, &g) != new(big.Int).Mul(big.NewInt(5), factor).String() {
		t.Fatal("large scaled canonical source quotients")
	}
	qopsMust(t, m.LCM(&r1, &r2, &c))
	if integerString(t, m, &c) != new(big.Int).Mul(big.NewInt(75), factor).String() {
		t.Fatal("large lcm")
	}
	// Fibonacci/Cassini gives independent finite exact coefficients for the
	// source quotient sequence (248 unit quotients and a final quotient2).
	fib := make([]big.Int, 251)
	fib[1].SetInt64(1)
	for i := 2; i < len(fib); i++ {
		fib[i].Add(&fib[i-1], &fib[i-2])
	}
	setIntegerOracle(t, m, &r1, &fib[250])
	setIntegerOracle(t, m, &r2, &fib[249])
	qopsMust(t, m.ExtendedGCD(&r1, &r2, &a, &b, &g))
	if integerString(t, m, &a) != fib[247].String() || integerString(t, m, &b) != new(big.Int).Neg(&fib[248]).String() || integerString(t, m, &g) != "1" {
		t.Fatal("large finite coefficient recurrence")
	}
	for _, z := range []*Int{&r1, &r2, &a, &b, &g, &c} {
		qopsMust(t, m.Del(z))
	}
}
func TestNumberTheoryRationalResetPrefixAndOutputAliasing(t *testing.T) {
	m, budget := rationalManager(t, true)
	r1, r2, a, b, g := NewRational(), NewRational(), NewRational(), NewRational(), NewRational()
	huge := new(big.Int).Lsh(big.NewInt(1), 300)
	setRationalOracle(t, m, &r1, huge, big.NewInt(1))
	qopsMust(t, m.SetInt64(&r2, 7))
	qopsMust(t, m.SetFraction(&a, 17, 3))
	qopsMust(t, m.SetFraction(&b, 19, 5))
	qopsMust(t, m.SetFraction(&g, 23, 7))
	budget.FailAfter(0)
	e := m.ExtendedGCD(&r1, &r2, &a, &b, &g)
	qopsAuditLiveOwnership(t, m, []*Int{&r1.num, &r1.den, &r2.num, &r2.den, &a.num, &a.den, &b.num, &b.den, &g.num, &g.den})
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || a.den != NewInt(1) || b.den != NewInt(1) || g.den != NewInt(1) {
		t.Fatal("source denominator reset precedes local allocation", e)
	}
	qopsMust(t, m.SetInt64(&r1, -26))
	qopsMust(t, m.SetInt64(&r2, 7))
	e = m.ExtendedGCD(&r1, &r2, &r1, &b, &g)
	qopsAuditLiveOwnership(t, m, []*Int{&r1.num, &r1.den, &r2.num, &r2.den, &a.num, &a.den, &b.num, &b.den, &g.num, &g.den})
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("destroyed source scratch is not repaired on continuation", e)
	}
	for _, q := range []*Rational{&r1, &r2, &a, &b, &g} {
		qopsMust(t, m.Del(q))
	}
	// The source alias arithmetic remains tested independently of the owner
	// whose failed allocation destroyed a scratch buffer.
	m, _ = rationalManager(t, true)
	r1, r2, a, b, g = NewRational(), NewRational(), NewRational(), NewRational(), NewRational()
	qopsMust(t, m.SetInt64(&r1, -26))
	qopsMust(t, m.SetInt64(&r2, 7))
	qopsMust(t, m.SetInt64(&g, 23))
	e = m.ExtendedGCD(&r1, &r2, &r1, &b, &g)
	wantG := "1"
	if sourceExtendedGCDDebug {
		if !errors.Is(e, errExtendedGCDInvariant) {
			t.Fatal("source debug assertion", e)
		}
		wantG = "23"
	} else {
		qopsMust(t, e)
	}
	if rationalString(t, m, &r1) != "3" || rationalString(t, m, &b) != "-11" || rationalString(t, m, &g) != wantG {
		t.Fatal("rational input-sign reread")
	}
	qopsMust(t, m.SetInt64(&r1, -6))
	qopsMust(t, m.SetInt64(&r2, -6))
	qopsMust(t, m.LCM(&r1, &r2, &r2))
	if rationalString(t, m, &r2) != "-6" {
		t.Fatal("rational signed shortcut")
	}
	ok, e := m.Divides(&r1, &r2)
	qopsMust(t, e)
	if !ok {
		t.Fatal("rational divides")
	}
	for _, q := range []*Rational{&r1, &r2, &a, &b, &g} {
		qopsMust(t, m.Del(q))
	}
}
