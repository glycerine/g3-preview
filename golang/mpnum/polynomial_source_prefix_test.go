// Source: src/math/polynomial/polynomial.cpp:2346-2370. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestPolynomialSourceDestructorPrefixStopsAtRealMissingMember(t *testing.T) {
	m := polynomialCellFixture(t)
	term, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	for _, b := range [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		a := NewInt(7)
		serviceMust(t, b.add(&a, term, false))
	}
	if e = m.SourceFinalizePrefix(); e != ErrPolynomialSourceMember {
		t.Fatal("missing interpolator binding", e)
	}
	if m.sourceStage != 7 || !m.sourceStarted || m.closed {
		t.Fatal("source prefix history", m.sourceStage)
	}
	for _, b := range [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		n, e := b.ms.size(m)
		serviceMust(t, e)
		if n != 0 {
			t.Fatal("source reset order incomplete")
		}
	}
	polynomialPhysicalAudit(t, m)
	if _, e = m.Info(m.zero); e != arena.ErrClosed {
		t.Fatal("source destruction owner boundary", e)
	}
	count, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if e = m.SourceFinalizePrefix(); e != ErrPolynomialSourceMember {
		t.Fatal(e)
	}
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if after != count {
		t.Fatal("missing member replayed source actions")
	}
	serviceMust(t, m.CloseChecked())
	if m.sourceStage != 7 {
		t.Fatal("owning abort fabricated source completion")
	}
}
func TestPolynomialSourcePrefixRetainsActualCallerReference(t *testing.T) {
	m := polynomialCellFixture(t)
	p := m.zero
	serviceMust(t, m.IncRef(p))
	if e := m.SourceFinalizePrefix(); e != ErrPolynomialSourceMember {
		t.Fatal(e)
	}
	r, e := m.readWord(p, 0)
	serviceMust(t, e)
	if r != 1 {
		t.Fatal("literal source DEC must preserve caller reference", r)
	}
	if e = m.CloseChecked(); e != ErrPolynomialLive {
		t.Fatal("owning Close consumed caller root", e)
	}
	if m.closing {
		t.Fatal("caller refusal must precede owning stop")
	}
	serviceMust(t, m.CleanupValue(&p))
	serviceMust(t, m.CloseChecked())
}
func TestPolynomialSourcePrefixStoppedHandlerOwnAbort(t *testing.T) {
	m := polynomialCellFixture(t)
	h := polynomialCallbackHandler(t, m, 6)
	serviceMust(t, m.AddDeletionHandler(h))
	polynomialCallbackObservation.count = 0
	polynomialCallbackObservation.failure = ErrPolynomialPartial
	if e := m.SourceFinalizePrefix(); e != ErrPolynomialPartial {
		t.Fatal(e)
	}
	if m.sourceStage != 0 || !m.sourceStarted {
		t.Fatal("source action advanced after refusal")
	}
	polynomialPhysicalAuditWithHandlers(t, m, []*PolynomialDeletionHandler{h})
	if e := m.SourceFinalizePrefix(); e != ErrPolynomialPartial {
		t.Fatal(e)
	}
	if polynomialCallbackObservation.count != 1 {
		t.Fatal("source callback replay")
	}
	if e := m.CloseChecked(); e != ErrPolynomialPartial {
		t.Fatal("literal next private callback", e)
	}
	serviceMust(t, m.CloseChecked())
	serviceMust(t, h.CloseChecked())
	if m.sourceStage != 0 {
		t.Fatal("owning abort changed stopped source history")
	}
	polynomialCallbackObservation.failure = nil
}
