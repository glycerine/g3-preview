// Source: src/math/polynomial/polynomial.cpp:4939-4949,4989-5010;
// src/util/mpzzp.h:58-68,131; src/util/mpz.h:124-149.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialPseudoActualMinusANumericAcquisition(t *testing.T) {
	saw, success, foreignProof := false, false, false
	for prefix := int64(0); prefix < 96 && !success; prefix++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c := full.core
		var big util.Int
		words := make([]uint32, 80)
		words[0], words[79] = 3, 1
		serviceMust(t, c.integer.SetLimbs(&big, words))
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		term, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
		serviceMust(t, e)
		p, e := c.MkTerms([]util.Int{big}, []Monomial{term})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		q, e := c.MkTerms([]util.Int{util.NewInt(1)}, []Monomial{term})
		serviceMust(t, e)
		q = polynomialCaller(t, c, q)
		Q, R := polynomialPseudoOutputs(t, c)
		c.integer.Budget().FailAfter(prefix)
		body := c.ExactPseudoRemainder(p, q, 0, R)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*util.Int{&big}, Q.header+R.header)
		raw, e := c.integer.SourceAllocator().PendingRoots()
		serviceMust(t, e)
		cell := c.integer.PendingCell
		s := &c.pseudoDivision
		if body != nil && s.numericPhase == 1 && (cell.Object != (util.SourceCellBacking{}).Object || raw != ([3]util.Block{})) {
			saw = true
			if cell.Object != (util.SourceCellBacking{}).Object && s.numeric.Cell != cell {
				t.Fatal("actual minus_a Copy cell has no direct Pseudo authority", prefix, body, cell, s.numeric)
			}
			if raw != ([3]util.Block{}) && s.numeric.Allocator != raw {
				t.Fatal("actual minus_a Copy raw has no direct Pseudo authority", prefix, body, raw, s.numeric)
			}
			saved := body
			savedBody := s.bodyCause
			serviceMust(t, c.pseudoCleanupPhase(true, false))
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*util.Int{&big}, Q.header+R.header)
			serviceMust(t, c.pseudoCleanupPhase(true, true))
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*util.Int{&big}, Q.header+R.header)
			if body != saved || c.pseudoDivision.bodyCause != savedBody {
				t.Fatal("own ACK changed actual returned body history", prefix)
			}
			if !foreignProof && cell.Object != (util.SourceCellBacking{}).Object {
				oldReceipt := s.directReceipt
				for foreignPrefix := int64(0); foreignPrefix < 48 && !foreignProof; foreignPrefix++ {
					var foreign util.Int
					c.integer.Budget().FailAfter(foreignPrefix)
					foreignError := c.integer.Copy(&foreign, &big)
					replacement := c.integer.PendingCell
					replacementRaw, readErr := c.integer.SourceAllocator().PendingRoots()
					serviceMust(t, readErr)
					if foreignError != nil && replacement.Object != (util.SourceCellBacking{}).Object {
						if replacement.Object == cell.Object {
							t.Fatal("actual foreign leaf reused old object generation")
						}
						state := c.memory.State()
						serviceMust(t, c.cleanupPseudoInvocation(&oldReceipt, false))
						serviceMust(t, c.cleanupPseudoInvocation(&oldReceipt, true))
						actualRaw, readErr := c.integer.SourceAllocator().PendingRoots()
						serviceMust(t, readErr)
						if c.integer.PendingCell != replacement || actualRaw != replacementRaw || c.memory.State() != state {
							t.Fatal("old Pseudo receipt swept actual foreign replacement")
						}
						polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*util.Int{&big, &foreign}, Q.header+R.header)
						foreignProof = true
					}
					c.integer.Budget().FailAfter(-1)
					serviceMust(t, c.integer.CleanupPendingAllocation())
					serviceMust(t, c.integer.Del(&foreign))
				}
				if !foreignProof {
					t.Fatal("actual foreign pending replacement not reached")
				}
			}
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.CleanupPseudoDivision())
		if body == nil {
			success = true
			zero, e := c.IsZero(R.root.value)
			serviceMust(t, e)
			if !zero {
				t.Fatal("BIG*x divided by x has nonzero exact remainder")
			}
			t.Log("actual first success", prefix)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*util.Int{&big}, Q.header+R.header)
		serviceMust(t, c.integer.Del(&big))
	}
	if !saw || !success || !foreignProof {
		t.Fatal("actual numeric refusal/success horizon not reached", saw, success)
	}
}

// Invalid saved/copy metadata guard coverage uses only the actual embedded
// parent and a real default NUM constructor; it grants no source-prefix credit.
func TestPolynomialPseudoDirectContextActualFrameAndCapturedNonce(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	Q, R := polynomialPseudoOutputs(t, c)
	serviceMust(t, c.pseudoEntry(Q, R))
	c.pseudoDivision.construct(pseudoNumeral)
	old := numericIntegerModularManager_pseudoAllocationContext(c.modular, &c.pseudoDivision.directReceipt)
	if (old.roots(c.integer)) != (&c.pseudoDivision.numeric) {
		t.Fatal("actual constructed NUM context refused")
	}
	copied := c.pseudoDivision
	forged := c.pseudoDivision.directReceipt
	forged.frame = &copied
	if (numericIntegerManager_pseudoAllocationRoots(c.integer, &forged)) != nil {
		t.Fatal("copied scope acquired authority")
	}
	c.callbackDepth++
	if (old.roots(c.integer)) != nil {
		t.Fatal("callback context acquired authority")
	}
	c.callbackDepth--
	foreignReceiver, e := util.NewIntegerModularManager(c.integer)
	serviceMust(t, e)
	if numericIntegerModularManager_pseudoAllocationContext(foreignReceiver, &c.pseudoDivision.directReceipt) != (integerAllocationInvocationContext{}) {
		t.Fatal("foreign modular receiver acquired actual Pseudo frame")
	}
	serviceMust(t, foreignReceiver.Close())
	serviceMust(t, c.pseudoPop(false))
	if (old.roots(c.integer)) != nil {
		t.Fatal("completed actual NUM destructor retained allocating authority")
	}
	c.pseudoDivision.executing = false
	serviceMust(t, c.CleanupPseudoDivision())
	serviceMust(t, c.pseudoEntry(Q, R))
	c.pseudoDivision.construct(pseudoNumeral)
	fresh := numericIntegerModularManager_pseudoAllocationContext(c.modular, &c.pseudoDivision.directReceipt)
	if (old.roots(c.integer)) != nil || (fresh.roots(c.integer)) != (&c.pseudoDivision.numeric) {
		t.Fatal("saved embedded receipt context retargeted")
	}
	c.pseudoDivision.executing = false
	serviceMust(t, c.CleanupPseudoDivision())
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{Q.root.value, R.root.value}, nil, Q.header+R.header)
}
