package ast

// Source: src/ast/rewriter/poly_rewriter_def.h:928.

import "testing"

func TestASTBitVectorPolynomialZeroAssignmentLiteralLifetime(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(65)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	out := x
	var call ASTBitVectorNumeralInvocation
	astTestMust(t, m.AssignBitVectorPolynomialZeroWithInvocation(s, &out, &call))
	astTestPin(t, m, out)
	yes, e := m.IsBitVectorOperation(out, 0)
	astTestMust(t, e)
	if !yes || out == x {
		t.Fatal("zero not published")
	}
	f := call.receipt.frame
	if !f.polynomialLiteral || !f.consumed[5] || !f.sourceDone {
		t.Fatal("outer literal did not end after assignment")
	}
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
}
