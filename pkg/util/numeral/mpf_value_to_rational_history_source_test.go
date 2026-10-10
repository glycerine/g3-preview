// Source: src/util/mpf.cpp:1695-1711; src/util/scoped_numeral.h:27-51.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatToRationalAllocatedHistorySurvivesInlineCall(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	m, e := NewArbitraryFloatManager(cfg)
	serviceMust(t, e)
	qm, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	var x ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
	_, e = m.powerTwoWrapper(3999, false)
	serviceMust(t, e)
	out := NewRational()
	b.FailAfter(7)
	body := m.ToRational(&x, qm, &out)
	if body == nil {
		t.Fatal("required real allocating refusal")
	}
	old := m.rationalConversion.invocation
	integerPhysicalAudit(t, m.integer, &x.significand, &m.rationalConversion.a.significand, &m.rationalConversion.n, &m.rationalConversion.d)
	b.FailAfter(-1)
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, qm.Del(&out))
	var zero ArbitraryFloat
	zeroSig := NewInt(0)
	bottom, e := m.bottomExponent(2)
	serviceMust(t, e)
	serviceMust(t, m.SetComponents(&zero, 2, 1, false, bottom, &zeroSig))
	fresh := NewRational()
	b.FailAfter(0)
	serviceMust(t, m.ToRational(&zero, qm, &fresh))
	b.FailAfter(-1)
	retained, _, _, ok := m.RationalConversionCauses(old)
	if !ok || retained != body {
		t.Fatal("later inline call erased allocated history", ok, retained, body)
	}
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, qm.Del(&fresh))
	serviceMust(t, qm.CloseChecked())
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&zero))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
