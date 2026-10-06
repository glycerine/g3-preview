// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Exact source-slot replacement uses the real SET leaf; direct member access
// is a qualified Go adapter ownership guard fixture, not another native driver.
package mpnum

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUPolynomialRemoveOneHalfRootFailedRowForeignReplacement(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 40 && !reached; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			warm := algebraicPolynomial(t, p, -1, 2)
			wv, e := p.wholeView(warm)
			serviceMust(t, e)
			warmOut, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.RemoveOneHalfRoot(wv, warmOut))
			serviceMust(t, p.DelPolynomial(&warmOut))
			var b, twoB, neg Int
			serviceMust(t, z.SetLimbs(&b, []uint32{1, 2, 3, 4}))
			two := NewInt(2)
			serviceMust(t, z.Mul(&b, &two, &twoB))
			serviceMust(t, z.Copy(&neg, &b))
			serviceMust(t, z.Neg(&neg))
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []Int{neg, twoB}))
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, z.Del(&b))
			serviceMust(t, z.Del(&twoB))
			serviceMust(t, z.Del(&neg))
			v, e := p.wholeView(input)
			serviceMust(t, e)
			z.budget.FailAfter(prefix)
			body := p.RemoveOneHalfRoot(v, out)
			q := &p.removeOneHalfRoot
			r := p.removeOneHalfRootReceipt()
			extra := []*Int{&q.values[0], &p.oneHalfRoot.values[0], &p.oneHalfRoot.values[1]}
			if z.matchesAddMulInvocation(&q.addMul) {
				extra = append(extra, &q.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{warm, input, out}, extra, nil)
			reached = q.failedRowStore != nil && !q.failedRowAcknowledged
			z.budget.FailAfter(-1)
			if reached {
				if body == nil || q.failedRowCell == 0 {
					t.Fatal("not real failed SET token")
				}
				row := &q.failedRowStore.backings[q.failedRowBacking-1].values[q.failedRowIndex]
				serviceMust(t, z.Del(row))
				serviceMust(t, z.SetLimbs(row, []uint32{7, 8, 9, 10}))
				replacement := *row
				ref := row.cell
				serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
				serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
				if *row != replacement || !validNumericRef(z.cells, ref) {
					t.Fatal("old own ACK deleted later actual foreign SET")
				}
				beforeFrame := *q
				var history upolyRemoveOneHalfRootHistory
				if r.history != nil {
					history = *r.history
				}
				serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
				if !reflect.DeepEqual(beforeFrame, *q) || (r.history != nil && *r.history != history) {
					t.Fatal("old stopped diagnostic changed")
				}
				algebraicCellSourcePhysical(t, c, []UPolynomial{warm, input, out}, nil, nil)
			} else {
				serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
				serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
			}
			serviceMust(t, p.DelPolynomial(&input))
			serviceMust(t, p.DelPolynomial(&out))
		})
	}
	if !reached {
		t.Fatal("no actual failed row token")
	}
}
func TestUPolynomialRemoveOneHalfRootErasedAndInvalidContext(t *testing.T) {
	_, p := algebraicFixture(t)
	z := p.config.Integer
	zero := algebraicPolynomial(t, p)
	v, e := p.wholeView(zero)
	serviceMust(t, e)
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, p.DelPolynomial(&out)) }()
	serviceMust(t, p.RemoveOneHalfRoot(v, out))
	old := p.removeOneHalfRootReceipt()
	historyBytes := p.removeOneHalfRootHistoryBytes
	used, physical := z.budget.Used(), z.memory.Stats()
	z.budget.FailAfter(0)
	for i := 0; i < 3; i++ {
		serviceMust(t, p.RemoveOneHalfRoot(v, out))
	}
	z.budget.FailAfter(-1)
	if z.budget.Used() != used || z.memory.Stats() != physical || p.removeOneHalfRoot.constructed != 0 || p.removeOneHalfRootHistoryBytes != historyBytes {
		t.Fatal("zero div constructed am/history/reserved")
	}
	q := &p.removeOneHalfRoot
	before := *q
	serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&old, true))
	if !reflect.DeepEqual(before, *q) {
		t.Fatal("old zero receipt acquired later frame")
	}
	copyFrame := *q
	copyFrame.active = true
	copyFrame.executing = true
	copyFrame.owner = p
	context := integerAllocationInvocationContext{removeOneHalfRoot: &copyFrame, removeOneHalfRootInvocation: copyFrame.invocation}
	divisor := upolyDivisorSpan{immediate: true, pair: [2]Int{NewInt(-1), NewInt(2)}}
	var power uint32
	if e := p.divRemCoreSpanWithSourceContext(v, &divisor, &power, &p.div2, &p.div1, nil, nil, context); e != ErrUPolynomialOwner {
		t.Fatal("copied frame reached erased numeric engine", e)
	}
	if z.budget.Used() != used || z.memory.Stats() != physical || !reflect.DeepEqual(before, *q) {
		t.Fatal("invalid context mutated actual owner")
	}
}

// An erased SASSERT operand invocation must not adopt an older genuine public
// HasOneHalfRoot scope merely because it is the manager's current member.
func TestUPolynomialRemoveOneHalfRootErasedAssertionPreservesOlderChild(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 64 && !reached; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := p.config.Integer
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []Int{NewInt(1), NewInt(2147483647), NewInt(1)}))
			v, e := p.wholeView(input)
			serviceMust(t, e)
			empty, e := p.NewPolynomial()
			serviceMust(t, e)
			ev, e := p.wholeView(empty)
			serviceMust(t, e)
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			z.budget.FailAfter(prefix)
			_, body := p.HasOneHalfRoot(v)
			h := &p.oneHalfRoot
			old := p.oneHalfRootReceipt()
			algebraicCellSourcePhysical(t, c, []UPolynomial{input, empty, out}, []*Int{&h.values[0], &h.values[1]}, nil)
			reached = body != nil && h.active && polynomialNegNumericPending(&h.numeric)
			z.budget.FailAfter(-1)
			if reached {
				before := *h
				used, physical := z.budget.Used(), z.memory.Stats()
				z.budget.FailAfter(0)
				e = p.RemoveOneHalfRoot(ev, out)
				z.budget.FailAfter(-1)
				serviceMust(t, e)
				if p.removeOneHalfRoot.assertion.frame != nil || !reflect.DeepEqual(before, *h) || z.budget.Used() != used || z.memory.Stats() != physical {
					t.Fatal("erased assertion captured old foreign child")
				}
				serviceMust(t, p.CleanupRemoveOneHalfRoot())
				if !reflect.DeepEqual(before, *h) || z.budget.Used() != used || z.memory.Stats() != physical {
					t.Fatal("erased caller ACK drained older Has scope")
				}
			}
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&old, false))
			serviceMust(t, p.DelPolynomial(&input))
			serviceMust(t, p.DelPolynomial(&empty))
			serviceMust(t, p.DelPolynomial(&out))
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&old, true))
		})
	}
	if !reached {
		t.Fatal("genuine earlier allocating child not reached")
	}
}
