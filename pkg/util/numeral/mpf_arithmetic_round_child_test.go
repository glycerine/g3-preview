// Source: src/util/mpf.cpp:584,662,744; src/util/mpf.cpp:1963-2098.
package numeral

import "testing"

func TestArbitraryFloatArithmeticActualNonallocatingRoundChild(t *testing.T) {
	for _, name := range []string{"add", "sub", "mul", "div"} {
		t.Run(name, func(t *testing.T) {
			m, r, b := mpfNumeralHelpersFixture(t)
			var x, y, out ArbitraryFloat
			serviceMust(t, m.MkOne(8, 3, false, &x))
			serviceMust(t, m.MkOne(8, 3, false, &y))
			x.exponent = 1 // 2 op 1 reaches the numerical body in every case.
			operation, cleanup := m.Add, m.CleanupAddSub
			switch name {
			case "sub":
				operation = m.Sub
			case "mul":
				operation, cleanup = m.Mul, m.CleanupMul
			case "div":
				operation, cleanup = m.Div, m.CleanupDiv
			}
			serviceMust(t, operation(ArbitraryFloatNearestEven, &x, &y, &out))
			serviceMust(t, cleanup())
			before, used, state := m.RoundScope.Invocation, b.Used(), r.State()
			b.FailAfter(0)
			serviceMust(t, operation(ArbitraryFloatNearestEven, &x, &y, &out))
			var frame *ArbitraryFloatRoundScope
			var invocation uint64
			var receipt ArbitraryFloatRoundReceipt
			var active bool
			switch name {
			case "add", "sub":
				frame, invocation, receipt, active = m.addSub.roundFrame, m.addSub.roundInvocation, m.addSub.roundReceipt, m.addSub.active
			case "mul":
				frame, invocation, receipt, active = m.mul.roundFrame, m.mul.roundInvocation, m.mul.roundReceipt, m.mul.active
			case "div":
				frame, invocation, receipt, active = m.div.roundFrame, m.div.roundInvocation, m.div.roundReceipt, m.div.active
			}
			if frame != &m.RoundScope || invocation != before+1 || invocation != m.RoundScope.Invocation || receipt != (ArbitraryFloatRoundReceipt{}) || active || m.RoundScope.active {
				t.Fatal("actual nonallocating child tuple", frame, invocation, before, receipt, active)
			}
			if b.Used() != used || r.State() != state {
				t.Fatal("warm caller allocated metadata for an erased receipt")
			}
			b.FailAfter(-1)
			// A genuine later public Round reuses the child frame. The ended
			// arithmetic caller must erase its obsolete tuple during cleanup.
			serviceMust(t, m.Round(ArbitraryFloatNearestEven, &out))
			later, used, state := m.RoundScope.Invocation, b.Used(), r.State()
			if later != invocation+1 {
				t.Fatal("actual child frame was not reused")
			}
			serviceMust(t, cleanup())
			if m.RoundScope.Invocation != later || b.Used() != used || r.State() != state {
				t.Fatal("ended caller touched a later Round child")
			}
			integerPhysicalAudit(t, m.integer, &x.significand, &y.significand, &out.significand)
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.Del(&y))
		})
	}
}
