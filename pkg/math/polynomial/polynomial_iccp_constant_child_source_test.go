// Source: src/math/polynomial/polynomial.cpp:3461-3465.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialICCPConstantSETActualPendingAcquisition(t *testing.T) {
	success, acquired := false, false
	for prefix := int64(0); prefix < 64 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialExactDivDirectPageOneFixture(t)
			c := full.core
			var big, i, held util.Int
			words := make([]uint32, 80)
			words[0], words[79] = 7, 1
			serviceMust(t, c.integer.SetLimbs(&big, words))
			p, e := c.MkConstantTakeCopy(&big)
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			serviceMust(t, c.integer.SetLimbs(&held, words))
			var content, primitive polynomialReference
			var receipt polynomialICCPInvocationReceipt
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupContentPrimitiveParts())
				serviceMust(t, c.integer.CleanupPendingAllocation())
				serviceMust(t, content.close(c))
				serviceMust(t, primitive.close(c))
				serviceMust(t, c.integer.Del(&held))
				serviceMust(t, c.integer.Del(&big))
				serviceMust(t, c.integer.Del(&i))
			})
			c.integer.Budget().FailAfter(prefix)
			body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &receipt)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*util.Int{&big, &i, &held}, 0)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("literal constant SET category", body)
			}
			if c.integer.PendingAllocationObject().OwnerID() != 0 {
				acquired = true
				if receipt.manager != full || !full.hasICCPConstantHistory(&receipt.shortcutConstant) || receipt.shortcutConstant.frame.numeric.Cell != c.integer.PendingCell || receipt.shortcutConstant.frame.executing {
					t.Fatal("actual constant SET pending cell has no captured ICCP invocation", prefix, receipt)
				}
			}
			if len(full.iccpFrames) != 0 {
				t.Fatal("constant branch invented filter/powers/ci source frame")
			}
			if full.hasICCPConstantHistory(&receipt.shortcutConstant) {
				actual := receipt.shortcutConstant
				before := *actual.frame
				copied := before
				forged := actual
				forged.frame = &copied
				if full.hasICCPConstantHistory(&forged) {
					t.Fatal("copied constant frame acquired history authority")
				}
				serviceMust(t, full.cleanupICCPConstantInvocation(&forged, true))
				ownerCopy := *full
				forged = actual
				forged.manager = &ownerCopy
				if ownerCopy.hasICCPConstantHistory(&forged) {
					t.Fatal("copied full owner acquired constant authority")
				}
				serviceMust(t, ownerCopy.cleanupICCPConstantInvocation(&forged, true))
				if !testNumericScopeEqual(*actual.frame, before) {
					t.Fatal("copied constant cleanup mutated actual scope")
				}
				if c.integer.PendingAllocationObject() != (umemory.ObjectRef{}) && actual.frame.numeric.Cell != c.integer.PendingCell {
					t.Fatal("copied cleanup consumed exact actual pending cell")
				}
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, full.cleanupICCPInvocation(&receipt, false))
			serviceMust(t, full.cleanupICCPInvocation(&receipt, true))
			if full.hasICCPConstantHistory(&receipt.shortcutConstant) && !errors.Is(body, receipt.shortcutConstant.frame.bodyCause) {
				t.Fatal("own ACK changed actual SET body history", body, receipt.shortcutConstant.frame.bodyCause)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*util.Int{&big, &i, &held}, 0)
			if body == nil {
				success = true
				a, e := c.coefficientCore(p, 0)
				serviceMust(t, e)
				eq, e := c.integer.Equal(&a, &i)
				serviceMust(t, e)
				if !eq || content.value != c.unit || primitive.value != c.unit {
					t.Fatal("literal constant outputs", eq, content.value, primitive.value)
				}
			}
		})
	}
	if !success || !acquired {
		t.Fatal("constant SET actual pending/first-success horizon absent", acquired, success)
	}
}

func TestPolynomialICCPConstantModeSwitchSETAndNoLexicalLocals(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	seven := util.NewInt(7)
	p, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	serviceMust(t, c.modular.SetZPUint64(5))
	var i util.Int
	var content, primitive polynomialReference
	var receipt polynomialICCPInvocationReceipt
	t.Cleanup(func() {
		serviceMust(t, full.CleanupContentPrimitiveParts())
		serviceMust(t, content.close(c))
		serviceMust(t, primitive.close(c))
		serviceMust(t, c.integer.Del(&i))
	})
	serviceMust(t, full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &receipt))
	if i != util.NewInt(2) || content.value != c.unit || primitive.value != c.unit || len(full.iccpFrames) != 0 {
		t.Fatal("literal constant SET normalization/constructor order", i)
	}
	if !full.hasICCPConstantHistory(&receipt.shortcutConstant) {
		t.Fatal("actual SET invocation tuple absent")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, primitive.value}, []*util.Int{&i}, 0)
	serviceMust(t, full.cleanupICCPInvocation(&receipt, false))
	serviceMust(t, full.cleanupICCPInvocation(&receipt, true))
	old := receipt
	serviceMust(t, full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &receipt))
	if old.shortcutConstant.invocation == receipt.shortcutConstant.invocation || full.hasICCPConstantHistory(&old.shortcutConstant) {
		t.Fatal("old constant tuple retargeted later invocation")
	}
}

// An actual later allocation may replace an earlier physically consumed token.
// The two pending cells have the same structural cardinality; identity comes
// from their captured object/backing tokens, never from that cardinality.
func TestPolynomialICCPConstantOldReceiptPreservesLaterForeignPendingCell(t *testing.T) {
	full := polynomialExactDivDirectPageOneFixture(t)
	c := full.core
	var big, held, i, foreign util.Int
	words := make([]uint32, 80)
	words[0], words[79] = 7, 1
	serviceMust(t, c.integer.SetLimbs(&big, words))
	p, e := c.MkConstantTakeCopy(&big)
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	serviceMust(t, c.integer.SetLimbs(&held, words))
	var content, primitive polynomialReference
	var receipt polynomialICCPInvocationReceipt
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, full.CleanupContentPrimitiveParts())
		serviceMust(t, c.integer.CleanupPendingAllocation())
		serviceMust(t, content.close(c))
		serviceMust(t, primitive.close(c))
		for _, n := range []*util.Int{&big, &held, &i, &foreign} {
			serviceMust(t, c.integer.Del(n))
		}
	})
	c.integer.Budget().FailAfter(1)
	body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &receipt)
	old := c.integer.PendingCell
	if body == nil || old.Object == (umemory.ObjectRef{}) || receipt.shortcutConstant.frame.numeric.Cell != old {
		t.Fatal("actual source SET pending descriptor horizon", body, old)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, primitive.value}, []*util.Int{&big, &held, &i, &foreign}, 0)
	// Explicit owner consumption is separate from the stopped source call.
	// Keep its old private binding to exercise exact replacement refusal.
	serviceMust(t, c.integer.CleanupPendingAllocation())
	large := make([]uint32, 8192)
	large[0], large[len(large)-1] = 11, 1
	found := false
	for prefix := int64(0); prefix < 32 && !found; prefix++ {
		c.integer.Budget().FailAfter(prefix)
		e = c.integer.SetLimbs(&foreign, large)
		now := c.integer.PendingCell
		if now.Object != (umemory.ObjectRef{}) && now != old {
			found = true
			if e == nil {
				t.Fatal("later pending cell without actual refusal")
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, primitive.value}, []*util.Int{&big, &held, &i, &foreign}, 0)
			roots, x := c.integer.SourceAllocator().PendingRoots()
			serviceMust(t, x)
			serviceMust(t, full.cleanupICCPInvocation(&receipt, false))
			serviceMust(t, full.cleanupICCPInvocation(&receipt, true))
			after, x := c.integer.SourceAllocator().PendingRoots()
			serviceMust(t, x)
			if c.integer.PendingCell != now || roots != after || receipt.shortcutConstant.frame.bodyCause == nil || !errors.Is(body, receipt.shortcutConstant.frame.bodyCause) {
				t.Fatal("old SET acknowledgment consumed or retargeted foreign pending roots")
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, primitive.value}, []*util.Int{&big, &held, &i, &foreign}, 0)
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.integer.CleanupPendingAllocation())
		serviceMust(t, c.integer.Del(&foreign))
	}
	if !found {
		t.Fatal("real later pending-cell replacement horizon absent")
	}
}
