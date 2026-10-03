// Copyright (c) 2006 Microsoft Corporation
// Source mpq.h delegates to mpz and resets the output denominator afterward.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/big"
	"strings"
	"testing"
)

func TestRationalBitwiseForwardingIndependentAndAliases(t *testing.T) {
	m, _ := rationalManager(t, true)
	for _, text := range []string{"0", "1", "2147483647", "4294967295", "18446744073709551616", "1329227995784915872903807060280344577"} {
		for op := 0; op < 4; op++ {
			for alias := 0; alias < 4; alias++ {
				a, b, out := NewRational(), NewRational(), NewRational()
				qopsMust(t, m.SetString(&a, text))
				qopsMust(t, m.SetString(&b, "18446744073709551617"))
				qopsMust(t, m.SetFraction(&out, 3, 7))
				ap, bp, dst := &a, &b, &out
				if alias == 1 {
					dst = ap
				}
				if alias == 2 {
					dst = bp
				}
				if alias == 3 {
					bp, dst = ap, ap
				}
				aa, _ := new(big.Int).SetString(text, 10)
				bb, _ := new(big.Int).SetString("18446744073709551617", 10)
				if alias == 3 {
					bb.Set(aa)
				}
				want := new(big.Int)
				var e error
				switch op {
				case 0:
					e = m.BitwiseOr(ap, bp, dst)
					want.Or(aa, bb)
				case 1:
					e = m.BitwiseAnd(ap, bp, dst)
					want.And(aa, bb)
				case 2:
					e = m.BitwiseXor(ap, bp, dst)
					want.Xor(aa, bb)
				case 3:
					e = m.BitwiseNot(129, ap, dst)
					mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 129), big.NewInt(1))
					want.Xor(aa, mask)
				}
				qopsMust(t, e)
				qopsEqual(t, m, dst, new(big.Rat).SetInt(want))
				qopsMust(t, m.Del(&a))
				qopsMust(t, m.Del(&b))
				qopsMust(t, m.Del(&out))
			}
		}
	}
	a, b, out := NewRational(), NewRational(), NewRational()
	m.SetFraction(&a, 1, 2)
	m.SetInt64(&b, 3)
	if e := m.BitwiseOr(&a, &b, &out); !errors.Is(e, ErrRationalNotInteger) {
		t.Fatal(e)
	}
	m.SetInt64(&a, -1)
	if e := m.BitwiseNot(32, &a, &out); !errors.Is(e, ErrBitwiseDomain) {
		t.Fatal(e)
	}
	m.Del(&a)
	m.Del(&b)
	m.Del(&out)
}

func TestRationalBitwiseFailureOwnershipAndDenominatorPublication(t *testing.T) {
	for op := 0; op < 4; op++ {
		for alias := 0; alias < 3; alias++ {
			failures, success := 0, false
			for fault := int64(0); fault < 256; fault++ {
				m, budget := rationalManager(t, true)
				a, b, out := NewRational(), NewRational(), NewRational()
				qopsMust(t, m.SetString(&a, strings.Repeat("1234567890", 25)+"7"))
				qopsMust(t, m.SetString(&b, strings.Repeat("9876543210", 20)+"9"))
				qopsMust(t, m.SetFraction(&out, 3, 7))
				dst := []*Rational{&out, &a, &b}[alias]
				before := rationalString(t, m, dst)
				budget.FailAfter(fault)
				var e error
				switch op {
				case 0:
					e = m.BitwiseOr(&a, &b, dst)
				case 1:
					e = m.BitwiseAnd(&a, &b, dst)
				case 2:
					e = m.BitwiseXor(&a, &b, dst)
				case 3:
					e = m.BitwiseNot(193, &a, dst)
				}
				budget.FailAfter(-1)
				qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
				if e != nil {
					failures++
					if !errors.Is(e, arena.ErrInjectedFailure) {
						t.Fatal(op, alias, fault, e)
					}
					if rationalString(t, m, dst) != before {
						t.Fatal("failed integer action reset denominator")
					}
				} else {
					success = true
					if !isOneInteger(&dst.den) {
						t.Fatal("missing source denominator reset")
					}
				}
				for _, v := range []*Rational{&a, &b, &out} {
					qopsMust(t, m.Del(v))
				}
				m.Close()
				if budget.Used() != 0 {
					t.Fatal(op, alias, fault, budget.Used())
				}
				if success {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal(op, alias, failures)
			}
		}
	}
}
