// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/ext_gcd.h and src/test/rational.cpp::tst7.
package mpnum

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestExtendedGCDWrapperSource999Cases(t *testing.T) {
	s, _ := serviceNew(t)
	two, err := s.NewValue()
	serviceMust(t, err)
	serviceMust(t, s.SetInt64(&two, 2))
	p, err := s.NewValue()
	serviceMust(t, err)
	power, err := s.Power(&two, 32)
	serviceMust(t, err)
	serviceMust(t, s.Copy(&p, &power))
	serviceMust(t, s.Del(&power))
	n, _ := s.NewValue()
	x, _ := s.NewValue()
	y, _ := s.NewValue()
	g, _ := s.NewValue()
	checks, assertions := 0, 0
	for i := int64(1); i < 1000; i++ {
		serviceMust(t, s.SetInt64(&n, i))
		serviceMust(t, s.ExtendedGCD(&n, &p, &g, &x, &y))
		parity, err := s.Mod(&n, &two)
		serviceMust(t, err)
		odd, err := s.IsOne(&parity)
		serviceMust(t, err)
		assertions++
		if odd {
			checks++
			product, err := s.Mul(&n, &x)
			serviceMust(t, err)
			residue, err := s.Mod(&product, &p)
			serviceMust(t, err)
			one, err := s.IsOne(&residue)
			serviceMust(t, err)
			if !one {
				t.Fatalf("source ENSURE case %d", i)
			}
			// Full-expression objects die after the condition is checked.
			serviceMust(t, s.Del(&residue))
			serviceMust(t, s.Del(&product))
			text, err := s.String(&x)
			serviceMust(t, err)
			coefficient, ok := new(big.Int).SetString(text, 10)
			if !ok || new(big.Int).Mod(new(big.Int).Mul(big.NewInt(i), coefficient), big.NewInt(4294967296)).Cmp(big.NewInt(1)) != 0 {
				t.Fatalf("source inverse case %d: x=%s", i, text)
			}
		}
		serviceMust(t, s.Del(&parity))
	}
	if checks != 500 || assertions != 999 {
		t.Fatal(checks, assertions)
	}
	for _, v := range []*OwnedRational{&p, &n, &x, &y, &g, &two} {
		serviceMust(t, s.Del(v))
	}
}

func TestExtendedGCDWrapperIndependentSignedBezout(t *testing.T) {
	s, _ := serviceNew(t)
	a, _ := s.NewValue()
	b, _ := s.NewValue()
	g, _ := s.NewValue()
	x, _ := s.NewValue()
	y, _ := s.NewValue()
	for i := int64(-12); i <= 12; i++ {
		for j := int64(-12); j <= 12; j++ {
			serviceMust(t, s.SetInt64(&a, i))
			serviceMust(t, s.SetInt64(&b, j))
			serviceMust(t, s.ExtendedGCD(&a, &b, &g, &x, &y))
			values := [3]*big.Int{}
			for k, v := range []*OwnedRational{&g, &x, &y} {
				text, err := s.String(v)
				serviceMust(t, err)
				var ok bool
				values[k], ok = new(big.Int).SetString(text, 10)
				if !ok {
					t.Fatal(text)
				}
			}
			lhs := new(big.Int).Add(new(big.Int).Mul(big.NewInt(i), values[1]), new(big.Int).Mul(big.NewInt(j), values[2]))
			want := new(big.Int).GCD(nil, nil, big.NewInt(i), big.NewInt(j))
			if lhs.Cmp(values[0]) != 0 || new(big.Int).Abs(values[0]).Cmp(want) != 0 {
				t.Fatalf("(%d,%d): g,x,y=%v", i, j, values)
			}
		}
	}
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
}

func TestExtendedGCDWrapperImmediateFailureOwnership(t *testing.T) {
	failures, successes := 0, 0
	sawInitializedPrefix := false
	for prefix := int64(0); prefix < 600; prefix++ {
		s, budget := serviceNew(t)
		a := serviceValue(t, s, "123456789012345678901234567891")
		b := serviceValue(t, s, "99887766554433221101")
		g := serviceValue(t, s, "17")
		x := serviceValue(t, s, "19")
		y := serviceValue(t, s, "23")
		budget.FailAfter(prefix)
		err := s.ExtendedGCD(&a, &b, &g, &x, &y)
		budget.FailAfter(-1)
		if err != nil {
			failures++
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatalf("prefix %d: %v", prefix, err)
			}
			if g.raw.IsSmall() && x.raw.IsSmall() && y.raw.IsSmall() &&
				g.raw.num.val == 17 && g.raw.den.val == 1 &&
				x.raw.num.val == 0 && x.raw.den.val == 1 &&
				y.raw.num.val == 1 && y.raw.den.val == 1 {
				if !sawInitializedPrefix {
					t.Logf("source initialized-output failure prefix: %d", prefix)
				}
				sawInitializedPrefix = true
			}
		} else {
			successes++
		}
		// Outputs may publish source prefixes. Count actual reachable owners
		// before String, caller Del, or Close can hide an intermediate leak.
		var roots []*Int
		for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
			roots = append(roots, &v.raw.num, &v.raw.den)
		}
		// This independent graph includes the source manager's fixed argument,
		// result and constant cells, plus legitimately retained natural scratch.
		qopsAuditLiveOwnership(t, s.manager, roots)
		serviceText(t, s, &a, "123456789012345678901234567891")
		serviceText(t, s, &b, "99887766554433221101")
		serviceMust(t, s.Close())
		if budget.Used() != 0 {
			t.Fatal(prefix, budget.Used())
		}
		if successes == 2 {
			break
		}
	}
	if failures == 0 || successes == 0 || !sawInitializedPrefix {
		t.Fatal(failures, successes, sawInitializedPrefix)
	}
}

func TestExtendedGCDWrapperSourceOutputAliases(t *testing.T) {
	s, _ := serviceNew(t)
	a := serviceValue(t, s, "12")
	b := serviceValue(t, s, "8")
	g := serviceValue(t, s, "0")
	x := serviceValue(t, s, "0")
	y := serviceValue(t, s, "0")
	serviceMust(t, s.ExtendedGCD(&a, &b, &g, &a, &b))
	serviceText(t, s, &g, "4")
	serviceText(t, s, &a, "1")
	serviceText(t, s, &b, "-1")
	serviceMust(t, s.SetInt64(&a, 12))
	serviceMust(t, s.SetInt64(&b, 8))
	serviceMust(t, s.ExtendedGCD(&a, &b, &g, &x, &x))
	serviceText(t, s, &g, "4")
	serviceText(t, s, &x, "1")
	serviceMust(t, s.ExtendedGCD(&a, &b, &x, &x, &y))
	serviceText(t, s, &x, "1") // final coefficient overwrites gcd publication.
	serviceText(t, s, &y, "-1")
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
}

func TestExtendedGCDWrapperProvenanceGuards(t *testing.T) {
	s, budget := serviceNew(t)
	other, _ := serviceNew(t)
	a := serviceValue(t, s, "12")
	b := serviceValue(t, s, "8")
	g := serviceValue(t, s, "17")
	x := serviceValue(t, s, "19")
	y := serviceValue(t, s, "23")
	foreign := serviceValue(t, other, "12")
	for index := 0; index < 5; index++ {
		used := budget.Used()
		args := [5]*OwnedRational{&a, &b, &g, &x, &y}
		args[index] = &foreign
		if err := s.ExtendedGCD(args[0], args[1], args[2], args[3], args[4]); !errors.Is(err, ErrRationalServiceOwner) {
			t.Fatal(index, err)
		}
		if budget.Used() != used {
			t.Fatal("foreign operand changed budget", index)
		}
		serviceText(t, s, &g, "17")
		serviceText(t, s, &x, "19")
		serviceText(t, s, &y, "23")
	}
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
	serviceMust(t, other.Del(&foreign))
}
