package numeral

import "testing"

func TestScopedIntegerBuffer16CopiesMutableSubpavingCoefficients(t *testing.T) {
	z, _ := mpzPublicationDirectOwner(t)
	var ns, ds ScopedIntegerBuffer16
	serviceMust(t, ns.Construct(z))
	serviceMust(t, ds.Construct(z))
	t.Cleanup(func() {
		serviceMust(t, ns.Cleanup())
		serviceMust(t, ds.Cleanup())
	})
	var numerator, denominator Int
	serviceMust(t, z.Copy(&numerator, &z.Two64))
	serviceMust(t, z.SetInt64(&denominator, 3))
	serviceMust(t, ns.Push(&numerator))
	serviceMust(t, ds.Push(&denominator))
	serviceMust(t, z.SetInt64(&numerator, -7))
	got, e := ns.At(0)
	serviceMust(t, e)
	text, e := z.String(got)
	serviceMust(t, e)
	if text != "18446744073709551616" {
		t.Fatal("subpaving numerator buffer must own a deep source copy", text)
	}
	serviceMust(t, z.Mul(got, &denominator, got))
	text, e = z.String(got)
	serviceMust(t, e)
	if text != "55340232221128654848" {
		t.Fatal("process_add must mutate the retained numerator in place", text)
	}
	if ns.Count() != 1 || ds.Count() != 1 {
		t.Fatal("two independent source buffers lost a coefficient")
	}
	view, e := ns.View()
	serviceMust(t, e)
	borrowed, e := view.At(0)
	serviceMust(t, e)
	text, e = z.String(&borrowed)
	serviceMust(t, e)
	if view.Count != 1 || text != "55340232221128654848" {
		t.Fatal("mk_sum must observe the mutated borrowed coefficient slice", view.Count, text)
	}
	serviceMust(t, z.Del(&numerator))
	serviceMust(t, z.Del(&denominator))
	serviceMust(t, ns.SourceClose())
	serviceMust(t, ds.SourceClose())
	if ns.Count() != 0 || ds.Count() != 0 {
		t.Fatal("source destructor retained numeral elements")
	}
}

func TestScopedIntegerBuffer16FailedCopyRetainsOwningCleanup(t *testing.T) {
	z, budget := mpzPublicationDirectOwner(t)
	var b ScopedIntegerBuffer16
	serviceMust(t, b.Construct(z))
	t.Cleanup(func() { serviceMust(t, b.Cleanup()) })
	var large Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
	budget.FailAfter(0)
	e := b.Push(&large)
	budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || b.Count() != 1 {
		t.Fatal("failed source set must leave its initialized destination row", e, b.Count())
	}
	serviceMust(t, b.SourceClose())
	serviceMust(t, b.Cleanup())
}
