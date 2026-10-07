// Licensed under the MIT license.
// Added source-action regressions for mpz.cpp/mpq.cpp Power and mpz.h shifts.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package util

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Checks a published component against concrete completed source action values.
// Only the owner's exact failed descriptor is a permitted unreadable prefix.
func powerPublicationComponent(t *testing.T, m *IntegerManager, a *Int, allowed ...string) string {
	t.Helper()
	if m.IsFailureRef(a.Cell) {
		if !errors.Is(m.check(a), arena.ErrInvalidSlot) {
			t.Fatal("failed component unexpectedly readable", *a)
		}
		return "failed"
	}
	s := roundPublicationDecode(t, m, a).String()
	for _, want := range allowed {
		if s == want {
			return s
		}
	}
	t.Fatal("component outside completed source prefixes", s, allowed)
	return ""
}

func TestPowerPublicationInPlaceShiftOverloads(t *testing.T) {
	m, _ := mpzPublicationOwner(t)
	a := NewInt(-3)
	qopsMust(t, m.Mul2kInPlace(&a, 40))
	mpzPublicationAudit(t, m, &a)
	qopsMust(t, m.MachineDiv2kInPlace(&a, 40))
	mpzPublicationAudit(t, m, &a)
	if a != NewInt(-3) {
		t.Fatal("actual in-place source shift roundtrip", a)
	}
}

func TestPowerPublicationWrappedShiftHeaderAccessOrder(t *testing.T) {
	m, _ := mpzPublicationOwner(t)
	var a Int
	qopsMust(t, m.SetUint64(&a, ^uint64(0)>>32))
	original := m.Cell(a.Cell)
	defer m.Store(a.Cell, original) // restore only this intentionally forged fixture
	forged := original
	forged.Size = ^uint32(0)
	m.Store(a.Cell, forged)
	// This is a bounded malformed encoded-header guard fixture, not a claim
	// that an ordinary public numeral can hold four billion live limbs.
	e := m.Mul2kInPlace(&a, 32)
	mpzPublicationAudit(t, m, &a)
	got := m.Cell(a.Cell)
	if !errors.Is(e, arena.ErrBounds) || got.Size != 1 || got.Capacity != original.Capacity {
		t.Fatal("source uint32 wrap publishes size before first unsafe backward access", got, e)
	}
}

func TestPowerPublicationIntegerZeroZeroProfile(t *testing.T) {
	m, _ := mpzPublicationOwner(t)
	zero, out := NewInt(0), NewInt(17)
	e := m.Power(&zero, 0, &out)
	mpzPublicationAudit(t, m, &zero, &out)
	if modularSourceDebug {
		if !errors.Is(e, ErrInvalidPower) || out != NewInt(17) {
			t.Fatal("source debug assertion before reset", out, e)
		}
	} else if e != nil || out != NewInt(0) {
		t.Fatal("release source reset", out, e)
	}
}

func TestPowerPublicationIntegerBudgetBeforeMaskWrap(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	a, out := NewInt(3), NewInt(17)
	b.FailAfter(0)
	e := m.Power(&a, 1<<31, &out)
	mpzPublicationAudit(t, m, &a, &out)
	if !numericPhysicalReservationRefusal(e) || out != NewInt(1) {
		t.Fatal("source finite square refusal after output1, before mask wrap", out, e)
	}
}

func TestPowerPublicationIntegerActualWrapPrefix(t *testing.T) {
	for _, p := range []uint32{1 << 31, ^uint32(0)} {
		for _, alias := range []bool{false, true} {
			m, _ := mpzPublicationOwner(t)
			a, out := NewInt(-1), NewInt(17)
			dst := &out
			if alias {
				dst = &a
			}
			e := m.Power(&a, p, dst)
			mpzPublicationAudit(t, m, &a, &out)
			want := int32(1)
			if p&1 != 0 {
				want = -1
			}
			var typed *PowerDomainError
			if !errors.As(e, &typed) || typed.Reason != PowerMaskOverflow || typed.Exponent != p || *dst != NewInt(want) {
				t.Fatal("completed source iterations at mask wrap", p, alias, *dst, e)
			}
		}
	}
}

func TestPowerPublicationRationalBudgetBeforeMaskWrap(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		qopsMust(t, m.SetInt64(&a, 3))
		qopsMust(t, m.SetFraction(&out, 17, 5))
		b.FailAfter(0)
		e := m.Power(&a, 1<<31, &out)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
		if !numericPhysicalReservationRefusal(e) || out.Num != NewInt(1) || out.Den != NewInt(1) {
			t.Fatal("rational source output1 precedes finite growth refusal", synch, out, e)
		}
	}
}

func TestPowerPublicationRationalActualWrapPrefix(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, base := range []int64{-1, 0, 1} {
			for _, p := range []uint32{1 << 31, ^uint32(0)} {
				for _, alias := range []bool{false, true} {
					m, _ := rationalManager(t, synch)
					a, out := NewRational(), NewRational()
					qopsMust(t, m.SetInt64(&a, base))
					qopsMust(t, m.SetFraction(&out, 17, 5))
					dst := &out
					if alias {
						dst = &a
					}
					e := m.Power(&a, p, dst)
					qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
					want := int32(1)
					if base == 0 {
						want = 0
					} else if base < 0 && p&1 != 0 {
						want = -1
					}
					var typed *PowerDomainError
					if !errors.As(e, &typed) || typed.Exponent != p || typed.Reason != PowerMaskOverflow || dst.Num != NewInt(want) || dst.Den != NewInt(1) {
						t.Fatal("rational completed iterations before real wrap", synch, base, p, alias, *dst, e)
					}
				}
			}
		}
	}
}

func TestPowerPublicationFinalUnusedSquareRefusal(t *testing.T) {
	for _, alias := range []bool{false, true} {
		m, budget := mpzPublicationDirectOwner(t)
		a, out := NewInt(46341), NewInt(17)
		dst := &out
		if alias {
			dst = &a
		}
		budget.FailAfter(0)
		e := m.Power(&a, 1, dst)
		mpzPublicationAudit(t, m, &a, &out)
		if !numericPhysicalReservationRefusal(e) || *dst != NewInt(46341) {
			t.Fatal("final unused integer square follows completed output", alias, *dst, e)
		}
	}
	for _, synch := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			m, budget := rationalPublicationDirectOwner(t, synch)
			a, out := NewRational(), NewRational()
			qopsMust(t, m.SetInt64(&a, 46341))
			qopsMust(t, m.SetFraction(&out, 17, 5))
			dst := &out
			if alias {
				dst = &a
			}
			budget.FailAfter(0)
			e := m.Power(&a, 1, dst)
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
			if !numericPhysicalReservationRefusal(e) || dst.Num != NewInt(46341) || dst.Den != NewInt(1) {
				t.Fatal("final unused rational square follows completed output", synch, alias, *dst, e)
			}
		}
	}
}

func TestPowerPublicationBigInputCopyPrecedesOutputOne(t *testing.T) {
	var words [33]uint32
	words[0], words[32] = 7, 1
	for _, alias := range []bool{false, true} {
		m, budget := mpzPublicationDirectOwner(t)
		var a Int
		qopsMust(t, m.SetLimbs(&a, words[:]))
		out := NewInt(17)
		dst := &out
		if alias {
			dst = &a
		}
		old := *dst // Nonowning scalar observation; no copied token is destroyed.
		budget.FailAfter(0)
		e := m.Power(&a, 0, dst)
		mpzPublicationAudit(t, m, &a, &out)
		if !numericPhysicalReservationRefusal(e) || *dst != old {
			t.Fatal("input copy refuses before output1", alias, *dst, e)
		}
	}
	for _, synch := range []bool{false, true} {
		m, budget := rationalPublicationDirectOwner(t, synch)
		a, out := NewRational(), NewRational()
		qopsMust(t, m.Z.SetLimbs(&a.Num, words[:]))
		qopsMust(t, m.SetFraction(&out, 17, 5))
		budget.FailAfter(0)
		e := m.Power(&a, 0, &out)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
		if !numericPhysicalReservationRefusal(e) || out.Num != NewInt(17) || out.Den != NewInt(5) {
			t.Fatal("rational local Copy refuses before output1", synch, out, e)
		}
	}
}

func TestPowerPublicationRawRationalShapesAndMulAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, pair := range [][4]int32{{2, 4, 3, 9}, {-2, 4, 5, -7}, {2, -4, -3, -9}, {0, -4, 7, 1}, {3, 1, 7, 1}} {
			for alias := 0; alias < 3; alias++ {
				m, _ := rationalManager(t, synch)
				a, b, out := NewRational(), NewRational(), NewRational()
				a.Num, a.Den, b.Num, b.Den = NewInt(pair[0]), NewInt(pair[1]), NewInt(pair[2]), NewInt(pair[3])
				dst := &out
				if alias == 1 {
					dst = &a
				}
				if alias == 2 {
					dst = &b
				}
				qopsMust(t, m.Mul(&a, &b, dst))
				qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
				want := new(big.Rat).Mul(new(big.Rat).SetFrac64(int64(pair[0]), int64(pair[1])), new(big.Rat).SetFrac64(int64(pair[2]), int64(pair[3])))
				got := new(big.Rat).SetFrac64(int64(dst.Num.Val), int64(dst.Den.Val))
				if got.Cmp(want) != 0 {
					t.Fatal(pair, alias, *dst, want)
				}
				// Source normalization divides by positive GCD; it does not fix denominator sign.
				if pair[1]*pair[3] < 0 && dst.Den.Val >= 0 {
					t.Fatal("source denominator sign", pair, alias, *dst)
				}
			}
			for _, alias := range []bool{false, true} {
				for p := uint32(0); p < 6; p++ {
					m, _ := rationalManager(t, synch)
					a, out := NewRational(), NewRational()
					a.Num, a.Den = NewInt(pair[0]), NewInt(pair[1])
					dst := &out
					if alias {
						dst = &a
					}
					qopsMust(t, m.Power(&a, p, dst))
					qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
					want := new(big.Rat).SetFrac(new(big.Int).Exp(big.NewInt(int64(pair[0])), new(big.Int).SetUint64(uint64(p)), nil), new(big.Int).Exp(big.NewInt(int64(pair[1])), new(big.Int).SetUint64(uint64(p)), nil))
					got := new(big.Rat).SetFrac64(int64(dst.Num.Val), int64(dst.Den.Val))
					if got.Cmp(want) != 0 {
						t.Fatal(pair, p, alias, *dst, want)
					}
				}
			}
		}
	}
}

func TestPowerPublicationLeftShiftCopyPrecedesPromotionRefusal(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	a, out := NewInt(-3), NewInt(17)
	b.FailAfter(0)
	e := m.Mul2k(&a, 1024, &out)
	mpzPublicationAudit(t, m, &a, &out)
	if !numericPhysicalReservationRefusal(e) || out != NewInt(-3) {
		t.Fatal("caller copy precedes preserving promotion refusal", out, e)
	}
}

func TestPowerPublicationRightShiftCopyRefusalPrefix(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	var a Int
	var words [33]uint32
	words[0], words[32] = 7, 1
	qopsMust(t, m.SetLimbs(&a, words[:]))
	out := NewInt(-17)
	b.FailAfter(0)
	e := m.MachineDiv2k(&a, 0, &out)
	mpzPublicationAudit(t, m, &a, &out)
	if !numericPhysicalReservationRefusal(e) || out.Val != a.Val || out.Cell != m.FailedNullRef {
		t.Fatal("copy-output setter failure prefix precedes k0 shortcut", out, e)
	}
	qopsMust(t, m.Del(&out))
	qopsMust(t, m.Del(&a))
}

func TestPowerPublicationRationalCopyComponentRefusal(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalPublicationDirectOwner(t, synch)
		a, out := NewRational(), NewRational()
		a.Num = NewInt(3)
		var words [33]uint32
		words[0], words[32] = 7, 1
		qopsMust(t, m.Z.SetLimbs(&a.Den, words[:]))
		qopsMust(t, m.SetFraction(&out, 17, 5))
		b.FailAfter(0)
		e := m.Copy(&out, &a)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
		if !numericPhysicalReservationRefusal(e) || out.Num != NewInt(3) || out.Den.Val != a.Den.Val || out.Den.Cell != m.Z.FailedNullRef {
			t.Fatal("numerator Copy completes before denominator destructive allocation refusal", synch, out, e)
		}
		qopsMust(t, m.Del(&out))
		qopsMust(t, m.Del(&a))
	}
}

func TestPowerPublicationRationalMulComponentRefusal(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a, b, out := NewRational(), NewRational(), NewRational()
		a.Num, b.Num = NewInt(3), NewInt(7)
		var words [33]uint32
		words[0], words[32] = 7, 1
		qopsMust(t, m.Z.SetLimbs(&a.Den, words[:]))
		qopsMust(t, m.SetFraction(&out, 17, 5))
		budget.FailAfter(0)
		e := m.Mul(&a, &b, &out)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
		if !numericPhysicalReservationRefusal(e) || out.Num != NewInt(21) || out.Den != NewInt(5) {
			t.Fatal("numerator Mul completes before denominator scratch growth refusal", synch, out, e)
		}
		qopsMust(t, m.Del(&out))
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&b))
	}
}
