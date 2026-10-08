// Source: src/util/mpf.cpp:1474,1480-1481; actual nonallocating child entries.
package util

import "testing"

func TestArbitraryFloatRemWarmChildrenAndLaterInvocation(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x, y, out ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 3, 3))
	serviceMust(t, m.SetDouble(&y, 8, 3, 2))
	serviceMust(t, m.Rem(&x, &y, &out))
	serviceMust(t, m.CleanupRem())
	partial, round := m.partialRemainder.invocation, m.RoundScope.Invocation
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	serviceMust(t, m.Rem(&x, &y, &out))
	s := &m.rem
	if s.active || !s.partialDone || s.partialInvocation != partial+1 || s.roundFrame != &m.RoundScope || s.roundInvocation != round+1 || s.roundInvocation != m.RoundScope.Invocation || s.roundReceipt != (ArbitraryFloatRoundReceipt{}) || s.formatReceipt != (ArbitraryFloatFormattedSetReceipt{}) {
		t.Fatal("actual nonallocating descendant tuple", s.partialInvocation, s.roundInvocation, s.active, s.roundReceipt, s.formatReceipt)
	}
	if b.Used() != used || r.State() != state {
		t.Fatal("warm remainder allocated metadata")
	}
	b.FailAfter(-1)
	a := ArbitraryFloat{format: arbitraryFloatFormat(8, 3, false), significand: NewInt(4)}
	divisor := ArbitraryFloat{format: arbitraryFloatFormat(8, 3, false), significand: NewInt(4)}
	serviceMust(t, m.PartialRemainder(&a, &divisor, 0, false))
	serviceMust(t, m.SetFormat(&out, 8, 3, ArbitraryFloatTowardZero, &x))
	sig := NewInt(32)
	serviceMust(t, m.SetComponents(&out, 8, 3, false, 0, &sig))
	serviceMust(t, m.Round(ArbitraryFloatNearestEven, &out))
	laterPartial, laterFormat, laterRound := m.partialRemainder.invocation, m.formattedSet.invocation, m.RoundScope.Invocation
	used, state = b.Used(), r.State()
	serviceMust(t, m.CleanupRem())
	if m.partialRemainder.invocation != laterPartial || m.formattedSet.invocation != laterFormat || m.RoundScope.Invocation != laterRound || b.Used() != used || r.State() != state || s.partialInvocation != 0 || s.roundFrame != nil {
		t.Fatal("ended parent touched later descendants")
	}
	integerPhysicalAudit(t, m.integer, &x.significand, &y.significand, &out.significand, &a.significand, &divisor.significand)
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&divisor))
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
}
