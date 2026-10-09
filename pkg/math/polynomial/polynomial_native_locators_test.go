package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Locator directories contain nonowning object references. Growing those Go
// slices must not reserve simulated container storage or change real objects.
func TestPolynomialNativeLocatorGrowthAndGeneration(t *testing.T) {
	m := polynomialPSCFullFixtureWithLimit(t, 1<<25).core
	mm := m.monomials
	budget := m.integer.Budget()
	const count = 4097
	polys := make([]Polynomial, count)
	terms := make([]Monomial, count)
	t.Cleanup(func() {
		budget.FailAfter(-1)
		for i := range polys {
			serviceMust(t, m.CleanupValue(&polys[i]))
		}
	})
	polynomialGrowth, monomialGrowth := 0, 0
	for i := range polys {
		// All preceding slots are actual live objects. At each capacity boundary,
		// grow just the directory with reservations disabled; allocation of the
		// next real object follows separately with reservations enabled.
		if len(mm.locations) == cap(mm.locations) {
			used, before := budget.Used(), mm.allocator.Stats()
			old := len(mm.locations)
			budget.FailAfter(0)
			slot, e := mm.prepareLocation()
			serviceMust(t, e)
			if slot != uint32(old+1) || len(mm.locations) != old+1 || budget.Used() != used || mm.allocator.Stats() != before {
				t.Fatal("native monomial locator growth changed actual allocation", slot, old)
			}
			budget.FailAfter(-1)
			monomialGrowth++
		}
		var e error
		terms[i], e = mm.MkVariable(0, uint32(i+1))
		serviceMust(t, e)
		if len(m.locations) == cap(m.locations) {
			used, before := budget.Used(), mm.allocator.Stats()
			old := len(m.locations)
			budget.FailAfter(0)
			slot, e := m.prepareLocation()
			serviceMust(t, e)
			if slot != uint32(old+1) || len(m.locations) != old+1 || budget.Used() != used || mm.allocator.Stats() != before {
				t.Fatal("native polynomial locator growth changed actual allocation", slot, old)
			}
			budget.FailAfter(-1)
			polynomialGrowth++
		}
		polys[i], e = m.MkTerms([]util.Int{util.NewInt(int32(i + 3))}, []Monomial{terms[i]})
		serviceMust(t, e)
		serviceMust(t, m.IncRef(polys[i]))
	}
	if polynomialGrowth < 5 || monomialGrowth < 5 {
		t.Fatal("locator growth boundaries not exercised", polynomialGrowth, monomialGrowth)
	}
	for i, p := range polys {
		info, e := m.Info(p)
		serviceMust(t, e)
		a, e := m.Coefficient(p, 0)
		serviceMust(t, e)
		term, e := m.TermMonomial(p, 0)
		serviceMust(t, e)
		power, e := mm.ReadPower(term, 0)
		serviceMust(t, e)
		if info.Size != 1 || info.RefCount != 1 || a != util.NewInt(int32(i+3)) || term != terms[i] || power != (MonomialPower{0, uint32(i + 1)}) {
			t.Fatal("native growth changed retained object", i, info, a, term, power)
		}
	}
	stalePoly, staleTerm := polys[0], terms[0]
	serviceMust(t, m.CleanupValue(&polys[0]))
	term, e := mm.MkVariable(0, count+1)
	serviceMust(t, e)
	polys[0], e = m.MkTerms([]util.Int{util.NewInt(17)}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(polys[0]))
	if polys[0].slot != stalePoly.slot || polys[0].generation == stalePoly.generation || term.slot != staleTerm.slot || term.generation == staleTerm.generation {
		t.Fatal("recycled locator did not advance generation")
	}
	if _, e := m.Info(stalePoly); e == nil {
		t.Fatal("stale polynomial accepted")
	}
	if _, e := mm.Info(staleTerm); e == nil {
		t.Fatal("stale monomial accepted")
	}
}
