package util

import (
	"errors"
	"testing"
)

type numericReceiptTestOwner struct{ numericPolicyTestOwner }

func (p *numericReceiptTestOwner) NumericBound(executing bool) bool { return p.active }

func TestNumericCallerReceiptBoundaryRetainsActualBufferGeneration(t *testing.T) {
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

}
