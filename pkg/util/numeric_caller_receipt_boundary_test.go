package util

import (
	"errors"
	"testing"
)

type numericReceiptTestOwner struct{ numericPolicyTestOwner }

func (p *numericReceiptTestOwner) NumericBound(executing bool) bool { return p.active }

func TestNumericCallerReceiptBoundaryRetainsActualVerificationGeneration(t *testing.T) {
	z, _ := integerManager(t)
	n := z.Naturals()
	owner := numericReceiptTestOwner{numericPolicyTestOwner{manager: z, active: true}}
	hooks := NumericInvocationHooks{Policy: &owner}
	var buffer NaturalBufferReceipt
	if e := n.BindCallerBuffer(&buffer, 0, 1, &hooks); e != nil {
		t.Fatal(e)
	}
	identity := buffer.Identity()
	if identity == (NaturalBufferIdentity{}) || buffer.Pending() != (Block{}) {
		t.Fatal("missing buffer identity or invented pending allocation")
	}
	if e := n.BindCallerBuffer(&buffer, 0, 2, &hooks); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("stale buffer rebound", e)
	}
	if identity != buffer.Identity() {
		t.Fatal("failed rebind changed source buffer")
	}
	frame, e := n.BeginCallerVerification(&hooks)
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := frame.Receipt(n)
	if e != nil || !receipt.Matches(n) {
		t.Fatal("real verification receipt missing", e)
	}
	copied := *frame
	if _, e := copied.Receipt(n); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied verification holder accepted", e)
	}
	frame.executing = false
	if e := receipt.Cleanup(true); e != nil {
		t.Fatal(e)
	}
	if receipt.Matches(n) {
		t.Fatal("ended verification retained authority")
	}
	next, e := n.BeginCallerVerification(&hooks)
	if e != nil {
		t.Fatal(e)
	}
	current, e := next.Receipt(n)
	if e != nil || current == receipt || receipt.Matches(n) {
		t.Fatal("verification generation reused", e)
	}
	next.executing = false
	if e := current.Cleanup(true); e != nil {
		t.Fatal(e)
	}
	owner.active = false
	if _, e := n.BeginCallerVerification(&hooks); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("retired caller constructed verification", e)
	}
}
