// Source: src/math/polynomial/polynomial.cpp:5356-5379.
package polynomial

import (
	"errors"
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialResultantActualConstantPowerChildPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
			q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 3}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, out.Close())
			})
			c.integer.Budget().FailAfter(prefix)
			body := full.Resultant(p, q, 0, out)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("reached source category", prefix, body)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, full.CleanupGCD())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if c.powerLocal.active {
				t.Fatal("Resultant omitted actual constant Power child", prefix, body)
			}
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 8})
				t.Log("actual first success", prefix)
			}
		})
	}
	if !success {
		t.Fatal("actual first success absent")
	}
}

func TestPolynomialResultantActualContentPowerChildPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 3}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, out.Close())
			})
			c.integer.Budget().FailAfter(prefix)
			body := full.Resultant(p, q, 0, out)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("reached source category", prefix, body)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, full.CleanupGCD())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if c.powerLocal.active {
				t.Fatal("Resultant omitted actual constant Power child", prefix, body)
			}
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{0, 3, 0}: 1, {0, 2, 0}: 3, {0, 1, 0}: 3, {}: 1})
				t.Log("actual first success", prefix)
			}
		})
	}
	if !success {
		t.Fatal("actual first success absent")
	}
}

func TestPolynomialResultantActualPseudoChildInvocationPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 3}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, out.Close())
			})
			visits := 0
			c.config.Checkpoint = func() bool {
				if c.pseudoDivision.executing && full.gcdDepth == 1 {
					visits++
					frame := full.gcdFrames[0]
					if frame.resultantPseudo.manager != c || frame.resultantPseudo.frame != &c.pseudoDivision || frame.resultantPseudo.invocation != c.pseudoDivision.invocation {
						t.Fatal("actual Resultant Pseudo guarded checkpoint tuple", prefix, frame.resultantPseudo)
					}
				}
				return true
			}
			c.integer.Budget().FailAfter(prefix)
			body := full.Resultant(p, q, 0, out)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("reached source category", prefix, body)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, full.CleanupGCD())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if c.powerLocal.active {
				t.Fatal("Resultant omitted actual constant Power child", prefix, body)
			}
			if body == nil {
				if visits == 0 {
					t.Fatal("actual Resultant Pseudo source checkpoint not reached")
				}
				success = true
				polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{0, 3, 0}: 1, {0, 2, 0}: 3, {0, 1, 0}: 3, {}: 1})
				t.Log("actual first success", prefix)
			}
		})
	}
	if !success {
		t.Fatal("actual first success absent")
	}
}

func TestPolynomialResultantActualArithmeticChildRetirementPrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 3}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, out.Close())
			})
			c.integer.Budget().FailAfter(prefix)
			body := full.Resultant(p, q, 0, out)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("reached source category", prefix, body)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, full.CleanupGCD())
			for _, b := range []*polynomialSOM{&c.som, &c.som2, &c.cheap, &c.cheap2} {
				as, e := b.as.size(c)
				serviceMust(t, e)
				ms, e := b.ms.size(c)
				serviceMust(t, e)
				if as != ms || b.hasPendingReference || b.arithmeticScope.active || b.appendScope.active {
					t.Fatal("Resultant omitted actual arithmetic child retirement", prefix, body, as, ms, b.hasPendingReference)
				}
				// Successful source Divides hands R rows to its persistent member.
				// Their zero tags identify completed transfer; current receipts must
				// retire every actual tagged slot without sweeping these member rows.
				for column, tags := range [][]uint64{b.coefficientOwners, b.monomialOwners} {
					published := as
					if column == 1 {
						published = ms
					}
					// Legacy source rows need not have adapter tag storage. Do not
					// invent ownership for those rows or inspect unused capacity.
					for slot := uint32(0); slot < published && int(slot) < len(tags); slot++ {
						if tags[slot] != 0 {
							t.Fatal("retained actual row acquisition", prefix, slot, tags[slot])
						}
					}
				}
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if c.powerLocal.active {
				t.Fatal("Resultant omitted actual constant Power child", prefix, body)
			}
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{0, 3, 0}: 1, {0, 2, 0}: 3, {0, 1, 0}: 3, {}: 1})
				t.Log("actual first success", prefix)
			}
		})
	}
	if !success {
		t.Fatal("actual first success absent")
	}
}

// Source Divides returns false without resetting R (polynomial.cpp:5154-5194).
// This actual successful return transfers rows to the persistent member; a later
// failed caller has no authority to retire them. The next literal entry resets R.
func TestPolynomialResultantLaterFailurePreservesSuccessfulDividesMemberRows(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, full.CleanupGCD())
		serviceMust(t, out.Close())
	})
	var earlier polynomialDividesInvocationReceipt
	ok, e := c.dividesWithReceipt(q, p, &earlier)
	serviceMust(t, e)
	if ok || c.som.arithmeticScope.active {
		t.Fatal("source false return did not hand off persistent R", ok)
	}
	as, e := c.som.as.size(c)
	serviceMust(t, e)
	ms, e := c.som.ms.size(c)
	serviceMust(t, e)
	if as != 2 || ms != 2 {
		t.Fatal("literal R.add(p) row construction", as, ms)
	}
	var terms [2]Monomial
	var values [2]unumeral.Int
	var rc [2]uint32
	for j := uint32(0); j < 2; j++ {
		terms[j], e = c.som.ms.monomial(c, j)
		serviceMust(t, e)
		values[j], e = c.som.as.coefficient(c, j)
		serviceMust(t, e)
		rc[j], e = c.monomialRefCount(terms[j])
		serviceMust(t, e)
		if c.som.monomialOwners[j] != 0 || c.som.coefficientOwners[j] != 0 {
			t.Fatal("actual successful member transfer retained slot authority")
		}
	}
	rawAs, rawMs := append([]unumeral.Int(nil), c.som.as.values...), append([]Monomial(nil), c.som.ms...)
	c.integer.Budget().FailAfter(0)
	body := full.Resultant(p, q, 0, out)
	if !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual later caller refusal absent", body)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, full.CleanupGCD())
	serviceMust(t, c.cleanupDividesInvocation(&earlier, true))
	if !polynomialEqualNumerals(c.som.as.values, rawAs) || !polynomialEqualMonomials(c.som.ms, rawMs) {
		t.Fatal("later caller swept persistent raw members")
	}
	for j := uint32(0); j < 2; j++ {
		term, e := c.som.ms.monomial(c, j)
		serviceMust(t, e)
		value, e := c.som.as.coefficient(c, j)
		serviceMust(t, e)
		count, e := c.monomialRefCount(term)
		serviceMust(t, e)
		if term != terms[j] || value != values[j] || count != rc[j] {
			t.Fatal("actual persistent row/NUM/INC changed", j)
		}
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
	var later polynomialDividesInvocationReceipt
	ok, e = c.dividesWithReceipt(q, p, &later)
	serviceMust(t, e)
	if ok || later.invocation == earlier.invocation {
		t.Fatal("actual subsequent reset/rebuild missing", ok)
	}
	for j := uint32(0); j < 2; j++ {
		count, e := c.monomialRefCount(terms[j])
		serviceMust(t, e)
		if count != rc[j] {
			t.Fatal("source reset/rebuild leaked incoming INC", j)
		}
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
	// The fixture's actual manager Close consumes the persistent vectors and
	// their NUM/INC ownership after caller references, with its cold graph check.
}

func TestPolynomialResultantCompletedSourceBodyHistorySurvivesOwningACK(t *testing.T) {
	witnessed := false
	for prefix := int64(0); prefix < 32 && !witnessed; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 3}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, out.Close())
			})
			c.integer.Budget().FailAfter(prefix)
			body := full.Resultant(p, q, 0, out)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out.root.value, out.root.incoming}, nil, out.header)
			if body == nil || len(full.gcdFrames) == 0 || full.gcdFrames[0].invocation == 0 || full.gcdFrames[0].active {
				return
			}
			witnessed = true
			s := full.gcdFrames[0]
			first := s.bodyCause
			if first == nil || !errors.Is(body, first) {
				t.Fatal("completed source destruction erased actual Resultant body history", prefix, body, first)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, full.CleanupGCD())
			if s.bodyCause != first {
				t.Fatal("own acknowledgment erased first body cause")
			}
			invocation := s.invocation
			serviceMust(t, full.Resultant(p, q, 0, out))
			if s.invocation == invocation || !errors.Is(body, first) {
				t.Fatal("new invocation retargeted original returned failure")
			}
		})
	}
	if !witnessed {
		t.Fatal("actual completed-source failed body profile absent")
	}
}
