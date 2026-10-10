// Source: src/util/mpf.cpp:1174-1181.
package numeral

import (
	"errors"
	"testing"
)

func TestArbitraryFloatToMPZEveryColdReservationAndOwners(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			if foreign && alias {
				continue
			}
			for _, exp := range []int64{40, 160} {
				complete := false
				for ordinal := int64(0); ordinal < 128; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					owner, ownerBudget := m.integer, b
					if foreign {
						owner, ownerBudget = mpzPublicationDirectOwner(t)
					}
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2kInPlace(&n, 119))
					serviceMust(t, m.integer.Inc(&n))
					x := ArbitraryFloat{format: arbitraryFloatFormat(17, 120, true), exponent: exp, significand: n}
					// Transfer the actual constructor numeral into x; no second owner copy.
					separate := NewInt(1)
					serviceMust(t, owner.Mul2kInPlace(&separate, 200))
					out := &separate
					if alias {
						out = &x.significand
					}
					old := x
					ownerBudget.FailAfter(ordinal)
					body := m.ToMPZ(&x, owner, out)
					if foreign {
						integerPhysicalAudit(t, m.integer, &x.significand)
						integerPhysicalAudit(t, owner, &separate)
						if ownerBudget.Used() != integerActualOwnedBytes(owner) {
							t.Fatal("immediate foreign graph", ordinal, body, ownerBudget.Used(), integerActualOwnedBytes(owner))
						}
					} else {
						integerPhysicalAudit(t, owner, &x.significand, &separate)
					}
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate source graph", foreign, alias, exp, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if !alias && x != old {
						t.Fatal("borrowed input changed", ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected refusal", ordinal, body)
					}
					if m.toMPZ.active {
						if e := m.ToMPZ(&x, owner, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped invocation replay", e)
						}
					}
					ownerBudget.FailAfter(-1)
					serviceMust(t, m.CleanupToMPZ())
					serviceMust(t, owner.CleanupRawNumeral(&separate))
					serviceMust(t, m.CleanupValue(&x))
					serviceMust(t, m.Close())
					serviceMust(t, r.Close())
					if foreign {
						serviceMust(t, owner.CloseChecked())
						if ownerBudget.Used() != 0 {
							t.Fatal("foreign refund", ownerBudget.Used())
						}
					}
					if b.Used() != 0 {
						t.Fatal("source refund", b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("to_mpz reservation horizon", foreign, alias, exp)
				}
			}
		}
	}
}

func TestArbitraryFloatToMPZWarmInlineAndEndedContext(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 3, true), exponent: 1, significand: NewInt(6)}
	out := NewInt(7)
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	serviceMust(t, m.ToMPZ(&x, m.integer, &out))
	if out != NewInt(-3) || b.Used() != used || r.State() != state {
		t.Fatal("inline body allocated", out)
	}
	before := out
	if e := m.toMPZ.run(&x, &out); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("ended body authority", e)
	}
	copy := m.toMPZ
	copy.hooks.Policy = &copy
	if e := copy.run(&x, &out); !errors.Is(e, ErrNumericOwner) || out != before {
		t.Fatal("copied body authority", e, out)
	}
	b.FailAfter(-1)
}
