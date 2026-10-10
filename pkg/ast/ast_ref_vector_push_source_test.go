// Source: src/util/ref_vector.h:96-100; src/ast/ast.h:2273-2277.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTRefVectorPushExactAcquiredReferenceBeforeGrowthRefusal(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkFreshConst("unpublished", sort)
	astTestMust(t, e)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	var receipt ASTRefVectorPushReceipt
	held, e := receipt.Unpublished()
	astTestMust(t, e)
	if held {
		t.Fatal("empty receipt owns a reference")
	}
	m.BudgetField.FailAfter(0)
	cause := m.RefVectorPushTracked(v, h, &receipt)
	if !errors.Is(cause, arena.ErrInjectedFailure) {
		t.Fatal("real vector growth refusal", cause)
	}
	refs, e := m.GetRefCount(h)
	astTestMust(t, e)
	if refs != 1 {
		t.Fatal("actual source INC before reserve", refs)
	}
	size, e := m.RefVectorSize(v)
	astTestMust(t, e)
	if size != 0 {
		t.Fatal("unpublished pointer", size)
	}
	if !receipt.acquired || receipt.published {
		t.Fatal("real acquired-reference stamp", receipt)
	}
	held, e = receipt.Unpublished()
	astTestMust(t, e)
	if !held {
		t.Fatal("unpublished acquisition not reported")
	}
	copied := receipt
	if _, e = copied.Unpublished(); e != ErrASTHandle {
		t.Fatal("copied receipt accepted", e)
	}
	m.BudgetField.FailAfter(-1)
	// Source member DT ends the vector but cannot consume the unjoined reference.
	astTestMust(t, m.DelRefVector(&v))
	astTestMust(t, m.CompleteRefVectorPush(receipt.vector, h, &receipt))
	held, e = receipt.Unpublished()
	astTestMust(t, e)
	if held {
		t.Fatal("acknowledged receipt still owns a reference")
	}
	if _, e = m.GetRefCount(h); e == nil {
		t.Fatal("exact acquired ref not released")
	}
	if e = m.CompleteRefVectorPush(receipt.vector, h, &receipt); e == nil {
		t.Fatal("double own acknowledgment")
	}
}

func TestASTRefVectorPushForeignAndIndependentAcquisitions(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkFreshConst("a", sort)
	astTestMust(t, e)
	b, e := m.MkFreshConst("b", sort)
	astTestMust(t, e)
	av, e := m.NewRefVector()
	astTestMust(t, e)
	bv, e := m.NewRefVector()
	astTestMust(t, e)
	var ar, br ASTRefVectorPushReceipt
	m.BudgetField.FailAfter(0)
	ae := m.RefVectorPushTracked(av, a, &ar)
	be := m.RefVectorPushTracked(bv, b, &br)
	if ae == nil || be == nil || ar.invocation == 0 || br.invocation <= ar.invocation {
		t.Fatal("independent actual acquisitions", ae, be, ar, br)
	}
	before := m.Stats()
	copied := ar
	if e = m.CompleteRefVectorPush(av, a, &copied); e == nil {
		t.Fatal("copied receipt authority")
	}
	if e = m.CompleteRefVectorPush(bv, a, &ar); e == nil {
		t.Fatal("foreign vector tuple")
	}
	if e = m.CompleteRefVectorPush(av, b, &ar); e == nil {
		t.Fatal("foreign node tuple")
	}
	if e = m.RefVectorPushTracked(av, b, &ar); e == nil {
		t.Fatal("overwrite retained invocation")
	}
	if m.Stats() != before {
		t.Fatal("invalid receipt mutated physical owner")
	}
	refs, e := m.GetRefCount(a)
	astTestMust(t, e)
	if refs != 1 {
		t.Fatal("original leaf altered", refs)
	}
	refs, e = m.GetRefCount(b)
	astTestMust(t, e)
	if refs != 1 {
		t.Fatal("unrelated leaf altered", refs)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.CompleteRefVectorPush(av, a, &ar))
	refs, e = m.GetRefCount(b)
	astTestMust(t, e)
	if refs != 1 {
		t.Fatal("one cleanup adopted another acquired ref")
	}
	astTestMust(t, m.CompleteRefVectorPush(bv, b, &br))
	astTestMust(t, m.DelRefVector(&av))
	astTestMust(t, m.DelRefVector(&bv))
}
func TestASTRefVectorPushSuccessfulTransferAndNullReceiptErasure(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	var held ASTRefVectorPushReceipt
	astTestMust(t, m.RefVectorPushTracked(v, h, &held))
	unpublished, e := held.Unpublished()
	astTestMust(t, e)
	if unpublished {
		t.Fatal("published reference remains caller-owned")
	}
	if !held.acquired || !held.published {
		t.Fatal("actual pointer ownership transfer", held)
	}
	if e = m.CompleteRefVectorPush(v, h, &held); e == nil {
		t.Fatal("published vector reference adopted by frame")
	}
	var null ASTRefVectorPushReceipt
	astTestMust(t, m.RefVectorPushTracked(v, ASTHandle{}, &null))
	if null != (ASTRefVectorPushReceipt{}) {
		t.Fatal("NULL owns no extra acquired reference", null)
	}
	astTestMust(t, m.DelRefVector(&v))
	if _, e = m.GetRefCount(h); e == nil {
		t.Fatal("source vector DT did not own published reference")
	}
}

func TestASTRefVectorPushNativeWideCleanupAcknowledgesWithoutReplay(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(4000+i), sort)
		astTestMust(t, e)
	}
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	var receipt ASTRefVectorPushReceipt
	m.BudgetField.FailAfter(0)
	cause := m.RefVectorPushTracked(v, h, &receipt)
	if !errors.Is(cause, arena.ErrInjectedFailure) {
		t.Fatal("actual unjoined source INC", cause)
	}
	// Cleanup grows a native worklist even while the original vector allocation
	// refusal remains armed, and acknowledges the exact acquired reference.
	cleanup := m.CompleteRefVectorPush(v, h, &receipt)
	if cleanup != nil || !receipt.consumed || !receipt.acknowledged || receipt.cleanupCause != nil || receipt.cause != cause {
		t.Fatal("native cleanup acknowledgement", cleanup, receipt)
	}
	if m.pendingDeletionCurrent != (astNodeRef{}) || m.pendingDeletionHead != (astNodeRef{}) {
		t.Fatal("completed native deletion retained pending roots")
	}
	for _, leaf := range append(args[:], h) {
		if _, e = m.GetRefCount(leaf); e == nil {
			t.Fatal("physically consumed node remains live", leaf)
		}
	}
	before := m.Stats()
	if e = m.CompleteRefVectorPush(v, h, &receipt); e == nil || m.Stats() != before {
		t.Fatal("double acknowledgement", e)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.DelRefVector(&v))
}

func TestASTRefVectorPushStaleInvocationVectorAndNodeGeneration(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	first, e := m.MkVar(14000, sort)
	astTestMust(t, e)
	var receipt ASTRefVectorPushReceipt
	m.BudgetField.FailAfter(0)
	if e = m.RefVectorPushTracked(v, first, &receipt); e == nil {
		t.Fatal("actual first INC must stop before publication")
	}
	m.BudgetField.FailAfter(-1)
	oldSerial := receipt.invocation
	oldVector := v
	astTestMust(t, m.CompleteRefVectorPush(v, first, &receipt))
	astTestMust(t, m.DelRefVector(&v))
	v, e = m.NewRefVector()
	astTestMust(t, e)
	next, e := m.MkVar(14001, sort)
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	if e = m.RefVectorPushTracked(v, next, &receipt); e == nil {
		t.Fatal("actual second INC must stop before publication")
	}
	if receipt.invocation <= oldSerial {
		t.Fatal("invocation reused")
	}
	before := m.Stats()
	if e = m.CompleteRefVectorPush(oldVector, next, &receipt); e == nil || m.Stats() != before {
		t.Fatal("stale ended vector receipt tuple", e)
	}
	if e = m.CompleteRefVectorPush(v, first, &receipt); e == nil || m.Stats() != before {
		t.Fatal("stale destroyed node tuple/generation", e)
	}
	refs, e := m.GetRefCount(next)
	astTestMust(t, e)
	if refs != 1 {
		t.Fatal("foreign acknowledgement changed exact acquired node", refs)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.CompleteRefVectorPush(v, next, &receipt))
	astTestMust(t, m.DelRefVector(&v))
}

func TestASTRefVectorPushNonwrappingNonceGuardDomain(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(15000, sort)
	astTestMust(t, e)
	v, e := m.NewRefVector()
	astTestMust(t, e)
	// Explicit Go nonce exhaustion fixture, not native source arithmetic/OOM
	// evidence. The actual INC must remain unreached when identity cannot stamp.
	m.refVectorPushSerial = ^uint64(0)
	var receipt ASTRefVectorPushReceipt
	before := m.Stats()
	if e = m.RefVectorPushTracked(v, h, &receipt); !errors.Is(e, arena.ErrSlotLimit) || receipt != (ASTRefVectorPushReceipt{}) || m.Stats() != before {
		t.Fatal("nonce exhaustion changed acquisition", e)
	}
	refs, e := m.GetRefCount(h)
	astTestMust(t, e)
	if refs != 0 {
		t.Fatal("unstamped INC executed", refs)
	}
	astTestMust(t, m.DelRefVector(&v))
}
