// Source: src/math/polynomial/polynomial.cpp:1885-2214,2558-2633.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func polynomialEqualNumerals(a, b []unumeral.Int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPolynomialNativeNumeralGrowthResetAndBounds(t *testing.T) {
	m := polynomialCellFixture(t)
	vectors := []*polynomialNumerals{&m.linearTerms, &m.rationalTerms, &m.som.as, &m.som2.as, &m.cheap.as, &m.cheap2.as}
	m.integer.Budget().FailAfter(0)
	t.Cleanup(func() { m.integer.Budget().FailAfter(-1) })
	before, used := m.memory.Stats(), m.integer.Budget().Used()
	for _, v := range vectors {
		for _, n := range []uint32{4097, 0, 129, 1, 33} {
			for i := uint32(0); i < n; i++ {
				serviceMust(t, v.appendCoefficient(m, unumeral.NewInt(int32(37*i)-128)))
			}
			for i := uint32(0); i < n; i++ {
				got, e := v.coefficient(m, i)
				serviceMust(t, e)
				if got != unumeral.NewInt(int32(37*i)-128) {
					t.Fatal("independent scalar row", i, got)
				}
			}
			if _, e := v.coefficient(m, n); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("native read bounds", e)
			}
			if e := v.putCoefficient(m, n, unumeral.NewInt(1)); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("native write bounds", e)
			}
			if e := v.publishSize(m, n+1); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("native shrink bounds", e)
			}
			serviceMust(t, v.publishSize(m, n/2))
			if len(v.values) != int(n/2) {
				t.Fatal("native shrink", len(v.values))
			}
			serviceMust(t, v.reset(m))
			if len(v.values) != 0 || len(v.retired) != 0 {
				t.Fatal("scalar reset retained resources")
			}
		}
	}
	if m.memory.Stats() != before || m.integer.Budget().Used() != used {
		t.Fatal("native storage changed Runtime accounting")
	}
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialNativeNumeralResetRetainsActualBIG(t *testing.T) {
	m := polynomialCellFixture(t)
	v := &m.linearTerms
	var a unumeral.Int
	serviceMust(t, m.integer.SetInt64(&a, 1<<40))
	actual := a
	m.integer.Budget().FailAfter(0)
	before, used := m.memory.Stats(), m.integer.Budget().Used()
	serviceMust(t, v.appendCoefficient(m, a))
	a = unumeral.Int{}
	serviceMust(t, v.reset(m))
	if len(v.values) != 0 || len(v.retired) != 1 || v.retired[0] != actual {
		t.Fatal("actual discarded BIG ownership", v)
	}
	for i := 0; i < 4097; i++ {
		serviceMust(t, v.appendCoefficient(m, unumeral.NewInt(int32(i))))
	}
	serviceMust(t, v.reset(m))
	if len(v.retired) != 1 || v.retired[0] != actual {
		t.Fatal("regrowth lost discarded numeral")
	}
	if m.memory.Stats() != before || m.integer.Budget().Used() != used {
		t.Fatal("native retirement changed Runtime accounting")
	}
	got, e := m.integer.Int64(&v.retired[0])
	serviceMust(t, e)
	if got != 1<<40 {
		t.Fatal("actual retained BIG value", got)
	}
	polynomialPhysicalAudit(t, m)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.close(m))
	if v.values != nil || v.retired != nil {
		t.Fatal("owning cleanup retained slice")
	}
	polynomialPhysicalAudit(t, m)
}
