// Source: src/math/polynomial/polynomial.cpp:6436-6506;
// src/math/polynomial/polynomial.cpp:1298-1517.
package polynomial

import (
	"errors"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// An actual newly allocated partially initialized monomial exercises reached
// source fields. This is a guarded incomplete-placement domain, not a claim
// that its native constructor or the missing fields successfully completed.
func TestPolynomialEvaluationReachedDegreeAfterVariableBreak(t *testing.T) {
	m, base, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
	mm := m.core.monomials
	q := m.core.config.Rationals
	slot, e := mm.prepareLocation()
	serviceMust(t, e)
	raw, e := mm.allocator.Allocate(36)
	serviceMust(t, e)
	l := &mm.locations[slot-1]
	l.raw = raw
	l.requested = 36
	l.live = true
	l.sourceRaw, l.sourceOffset, e = mm.allocator.RawLocation(raw)
	serviceMust(t, e)
	term := Monomial{mm.origin, slot, l.generation}
	for _, store := range [][2]uint32{{0, 0}, {12, 2}, {20, 0}, {24, 1}, {28, 1}} {
		serviceMust(t, mm.allocator.WriteUint32(raw, uint64(store[0]), store[1]))
	}
	serviceMust(t, mm.IncRef(term))
	partial, e := m.core.MkCoreTake([]unumeral.Int{unumeral.NewInt(3)}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(partial))
	// The actual placed polynomial field now owns the preexisting source INC.
	t.Cleanup(func() {
		// Source DEC consumes the last reference before reaching the missing
		// hash header. The recorded last-drop frontier authorizes the next
		// explicit owning abort of this actual placed field, without filling
		// either the header or the deliberately unread second degree.
		e := m.core.CleanupValue(&partial)
		if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("partial source deletion header", e)
		}
		l, e := m.core.polynomialLocation(partial)
		serviceMust(t, e)
		if !l.monomialDropConsumed || !l.monomialDropLast {
			t.Fatal("actual consumed last drop")
		}
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{base, partial}, nil, 0)
		serviceMust(t, m.core.CleanupValue(&partial))
	})
	values := polynomialEvalRationalMap{manager: q}
	for i := range values.values {
		values.values[i] = unumeral.NewRational()
		serviceMust(t, q.SetInt32(&values.values[i], 2))
	}
	out := unumeral.NewRational()
	m.evaluation.rational = q
	m.evaluation.active = true
	m.evaluation.executing = true
	e = m.evaluationRationalCore(partial, &values, 0, 1, 0, &out)
	m.evaluation.executing = false
	serviceMust(t, e)
	if (*out.BorrowNumerator()) != unumeral.NewInt(6) || values.reads != 1 {
		t.Fatal("higher variable breaks before its uninitialized degree", out, values.reads)
	}
	inputs := []*unumeral.Int{&(*out.BorrowNumerator()), &(*out.BorrowDenominator())}
	for i := range values.values {
		inputs = append(inputs, &(*values.values[i].BorrowNumerator()), &(*values.values[i].BorrowDenominator()))
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{base, partial}, inputs, 0)
	serviceMust(t, m.CleanupEvaluation())
	m.evaluation.executing = true
	e = m.evaluationRationalCore(partial, &values, 0, 1, 1, &out)
	m.evaluation.executing = false
	if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("actually reached second degree", e)
	}
	want := int32(6)
	if polynomialSourceDebug {
		want = 3
	}
	if (*out.BorrowNumerator()) != unumeral.NewInt(want) {
		t.Fatal("source DEBUG degree assertion versus release loop prefix", (*out.BorrowNumerator()), want)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{base, partial}, inputs, 0)
	serviceMust(t, m.CleanupEvaluation())
	for i := range values.values {
		serviceMust(t, q.Del(&values.values[i]))
	}
	serviceMust(t, q.Del(&out))
}
