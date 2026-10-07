// Source: src/math/polynomial/polynomial.cpp actual four imp SOM members.
package polynomial

import (
	"testing"
)

// Invalid Go owner metadata guard proof, not source invocation/acquisition credit.
func TestPolynomialArithmeticReceiptRejectsExternalBufferAndCopiedCore(t *testing.T) {
	c := polynomialCellFixture(t)
	borrowed := polynomialSOM{owner: c}
	s := &borrowed.arithmeticScope
	*s = polynomialArithmeticScope{buffer: &borrowed, invocation: 1, active: true, executing: true}
	r := polynomialArithmeticInvocationReceipt{manager: c, frame: s, invocation: 1}
	if c.matchesArithmeticInvocation(&r) || (numericIntegerManager_arithmeticAllocationRoots(c.integer, &r)) != nil {
		t.Fatal("external fabricated buffer acquired actual core numeric authority")
	}
	copied := *c
	owned := copied.cheap
	owned.owner = &copied
	owned.arithmeticScope = polynomialArithmeticScope{buffer: &owned, invocation: 1, active: true, executing: true}
	r = polynomialArithmeticInvocationReceipt{manager: &copied, frame: &owned.arithmeticScope, invocation: 1}
	if copied.matchesArithmeticInvocation(&r) || (numericIntegerManager_arithmeticAllocationRoots(c.integer, &r)) != nil {
		t.Fatal("copied core acquired numeric authority")
	}
}
