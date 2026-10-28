package numeral

// Source: src/util/mpf.cpp:1695-1711,316-333.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatToRationalRetainsInitialCopyChild(t *testing.T) {
	for ordinal := int64(0); ordinal < 24; ordinal++ {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
		serviceMust(t, e)
		qm, e := NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
		serviceMust(t, e)
		var sig Int
		digits := make([]uint32, 1024)
		digits[1023] = 1
		serviceMust(t, m.integer.SetLimbs(&sig, digits))
		var x ArbitraryFloat
		serviceMust(t, m.SetComponents(&x, 11, 32768, false, 0, &sig))
		var out Rational
		b.FailAfter(ordinal)
		body := m.ToRational(&x, qm, &out)
		s := &m.rationalConversion
		retained := s.numericPhase == 1 && s.copy.frame != nil && s.copy.Child.Frame != nil && s.copy.Child.Frame.active
		integerPhysicalAudit(t, m.integer, &sig, &x.significand, &s.a.significand, &s.n, &s.d)
		b.FailAfter(-1)
		if retained {
			old := s.copy
			if !s.active {
				t.Fatal("completed caller source suffix erased reached unjoined Copy child", ordinal)
			}
			zero := ArbitraryFloat{}
			if err := m.ToRational(&zero, qm, &out); err != ErrNumericPending {
				t.Fatal("retained child accepted another call", ordinal, err)
			}
			if s.copy != old || s.bodyCause != body {
				t.Fatal("rejected begin changed exact child/history")
			}
		}
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, qm.Del(&out))
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.integer.Del(&sig))
		// The literal source owner may encounter the inherited null scratch guard;
		// acknowledge that exact source frontier separately from own teardown.
		if err := m.SourceFinalize(); err != nil && !m.integer.sourceGuard {
			t.Fatal(err)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if retained {
			return
		}
	}
	t.Fatal("no reached initial Copy child")
}
