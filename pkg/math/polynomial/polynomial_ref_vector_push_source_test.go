package polynomial

// Source: src/util/ref_vector.h:67-71,96-100,128-132.

import "testing"

func TestPolynomialReferenceVectorNativeGrowthResetAndReuse(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{1, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { m.originalBudget.FailAfter(-1); serviceMust(t, out.Close()) })
	initialP, e := c.readWord(p, 0)
	serviceMust(t, e)
	initialQ, e := c.readWord(q, 0)
	serviceMust(t, e)
	before, used := c.memory.Stats(), m.originalBudget.Used()
	var ps, qs uint32
	m.originalBudget.FailAfter(0)
	for i := 0; i < 4097; i++ {
		var v Polynomial
		if i%3 == 0 {
			v = p
			ps++
		} else if i%3 == 1 {
			v = q
			qs++
		}
		serviceMust(t, out.Push(v))
		got, e := out.Get(uint32(i))
		serviceMust(t, e)
		if got != v {
			t.Fatal("native reference association", i)
		}
	}
	size, e := out.Size()
	serviceMust(t, e)
	if size != 4097 || c.memory.Stats() != before || m.originalBudget.Used() != used {
		t.Fatal("native reference collection charged Runtime", size)
	}
	// Both have other references, so SET has no final-reference destruction.
	var receipt polynomialReferenceVectorSetReceipt
	serviceMust(t, out.setWithReceipt(0, q, &receipt))
	ps--
	qs++
	for _, pair := range []struct {
		p    Polynomial
		want uint32
	}{{p, initialP + ps}, {q, initialQ + qs}} {
		count, e := c.readWord(pair.p, 0)
		serviceMust(t, e)
		if count != pair.want {
			t.Fatal("native growth exact INC/DEC", count, pair.want)
		}
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p, q}, nil)
	serviceMust(t, out.Reset())
	if len(out.state.values) != 0 {
		t.Fatal("native reset left logical entries")
	}
	serviceMust(t, out.resizeZeroRange(4097))
	for i := uint32(0); i < 4097; i++ {
		v, e := out.Get(i)
		serviceMust(t, e)
		if v != (Polynomial{}) {
			t.Fatal("native reset/resize exposed stale handle", i)
		}
	}
	serviceMust(t, out.Reset())
	serviceMust(t, out.Push(p))
	serviceMust(t, receipt.cleanup()) // Completed SET cannot acquire reused storage.
	count, e := c.readWord(p, 0)
	serviceMust(t, e)
	if count != initialP+1 {
		t.Fatal("native reuse changed owning count", count)
	}
	if c.memory.Stats() != before || m.originalBudget.Used() != used {
		t.Fatal("native reuse charged Runtime")
	}
	serviceMust(t, out.Reset())
	m.originalBudget.FailAfter(-1)
	serviceMust(t, out.Close())
	if out.state.values != nil {
		t.Fatal("owning close retained native storage")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, q})
}
