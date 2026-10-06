// Source: src/math/polynomial/polynomial.cpp:6970-6980;
// src/math/polynomial/polynomial.h:319-329. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestPolynomialVariableRecognitionExactGuardAndArchiveRefusal(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{0, 1}}, nil})
	n := NewInt(9)
	v := uint32(77)
	var old polynomialVariableNumeralInvocationReceipt
	yes, e := c.isVariableNumeralWithReceipt(p, &v, &n, &old)
	serviceMust(t, e)
	if !yes || n != NewInt(5) {
		t.Fatal("reached SET")
	}
	before := c.variableNumeral
	copied := before
	forged := old
	forged.frame = &copied
	if c.matchesVariableNumeralInvocation(&forged) {
		t.Fatal("copied source frame accepted")
	}
	serviceMust(t, c.cleanupVariableNumeralInvocation(&forged, true))
	owner := *c
	foreign := old
	foreign.manager = &owner
	if owner.matchesVariableNumeralInvocation(&foreign) {
		t.Fatal("copied core accepted")
	}
	serviceMust(t, owner.cleanupVariableNumeralInvocation(&foreign, true))
	if c.variableNumeral != before {
		t.Fatal("invalid authority mutated actual source frame")
	}
	// The source variable publication precedes the real diagnostic archive reserve.
	c.integer.budget.FailAfter(0)
	unchanged := old
	n = NewInt(9)
	v = 77
	yes, e = c.isVariableNumeralWithReceipt(p, &v, &n, &unchanged)
	if yes || !errors.Is(e, arena.ErrInjectedFailure) || unchanged != old || c.variableNumeral != before || n != NewInt(9) || v != 0 {
		t.Fatal("archive refusal lost older exact tuple or source prefix", yes, e, v, n)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*Int{&n}, 0)
	c.integer.budget.FailAfter(-1)
	// False shape paths read no caller NUM and do not acquire a frame.
	nonunit := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	v = 77
	yes, e = c.isVariableNumeralWithReceipt(nonunit, &v, nil, &unchanged)
	serviceMust(t, e)
	if yes || v != 1 || unchanged != old || c.variableNumeral != before {
		t.Fatal("unit-tail failure prefix/erased SET", v)
	}
	yes, e = c.isVariableNumeralWithReceipt(c.zero, nil, nil, &unchanged)
	serviceMust(t, e)
	if yes || unchanged != old || c.variableNumeral != before {
		t.Fatal("zero branch constructed adapter")
	}
	var later polynomialVariableNumeralInvocationReceipt
	yes, e = c.isVariableNumeralWithReceipt(p, &v, &n, &later)
	serviceMust(t, e)
	if !yes || later.invocation == old.invocation {
		t.Fatal("new guarded invocation")
	}
	history, ok := c.variableNumeralDiagnostic(&old)
	if !ok || history.invocation != old.invocation || history.bodyCause != before.bodyCause || history.copy != before.copy || history.normalize != before.normalize {
		t.Fatal("actual archived child history")
	}
	current := c.variableNumeral
	serviceMust(t, c.cleanupVariableNumeralInvocation(&old, true))
	if c.variableNumeral != current {
		t.Fatal("stale old owning ACK touched later source")
	}
	// Nonwrapping invocation refusal occurs only at the literal reached SET.
	c.variableNumeral.invocation = ^uint64(0)
	v = 77
	yes, e = c.isVariableNumeralWithReceipt(p, &v, &n, &later)
	if yes || !errors.Is(e, ErrPolynomialPending) || v != 0 {
		t.Fatal("nonwrap reached SET prefix", yes, e, v)
	}
	c.variableNumeral = current
	serviceMust(t, c.integer.Del(&n))
}

func TestPolynomialVariableRecognitionOldACKPreservesLaterForeignCopyAndPN(t *testing.T) {
	for _, field := range []bool{false, true} {
		found := false
		for prefix := int64(0); prefix < 80 && !found; prefix++ {
			full := polynomialExactDivDirectPageOneFixture(t)
			c := full.core
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
			p := polynomialQueryTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{0, 1}}, nil})
			n := NewInt(9)
			v := uint32(77)
			var old polynomialVariableNumeralInvocationReceipt
			_, e = c.isVariableNumeralWithReceipt(p, &v, &n, &old)
			serviceMust(t, e)
			words := make([]uint32, 500)
			words[0], words[499] = 17, 1
			var big, out Int
			serviceMust(t, c.integer.SetLimbs(&big, words))
			var copyReceipt integerCopyAllocationReceipt
			var pnReceipt integerPNormalizeInvocationReceipt
			if field {
				serviceMust(t, c.integer.Copy(&out, &big))
				modulusWords := make([]uint32, 17)
				for i := range modulusWords {
					modulusWords[i] = ^uint32(0)
				}
				modulusWords[16] = 511
				var modulus Int
				serviceMust(t, c.integer.SetLimbs(&modulus, modulusWords))
				serviceMust(t, c.modular.SetZP(&modulus))
				serviceMust(t, c.integer.Del(&modulus))
			}
			c.integer.budget.FailAfter(prefix)
			var body error
			if field {
				body = c.modular.pNormalizeWithReceipt(&out, &pnReceipt)
			} else {
				body = c.integer.copyFromOwnerWithReceipt(&out, c.integer, &big, &copyReceipt)
			}
			actual := c.integer.matchesCopyAllocationInvocation(&copyReceipt) || c.modular.matchesPNormalizeInvocation(&pnReceipt)
			if (body != nil && actual) || (field && !modularSourceDebug && body == nil) {
				found = true

				copyState, pnState := c.integer.copyAllocation, c.modular.normalization
				valueBefore := out
				used, stats := c.integer.budget.Used(), c.memory.Stats()
				serviceMust(t, c.cleanupVariableNumeralInvocation(&old, true))
				if c.integer.copyAllocation != copyState || c.modular.normalization != pnState || used != c.integer.budget.Used() || stats != c.memory.Stats() || out != valueBefore {
					t.Fatal("old helper ACK swept later foreign actual child")
				}
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*Int{&n, &big, &out}, 0)
			c.integer.budget.FailAfter(-1)
			if found && field && body == nil {
				// Native one-shot reservations publish shared storage atomically; none of
				// these prefixes retains an untransferred PN leaf. Preserve its actual
				// successful shared members and caller NUM instead of inventing a root.
				expectedWords := make([]uint32, 11)
				expectedWords[0], expectedWords[10] = 17, 1<<18
				var expected Int
				serviceMust(t, c.integer.SetLimbs(&expected, expectedWords))
				cmp, e := c.integer.Compare(&out, &expected)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("literal foreign modular remainder", cmp)
				}
				serviceMust(t, c.integer.Del(&expected))
			}
			serviceMust(t, c.modular.cleanupPNormalizeInvocation(&pnReceipt, false))
			serviceMust(t, c.modular.cleanupPNormalizeInvocation(&pnReceipt, true))
			serviceMust(t, c.integer.cleanupCopyAllocationInvocation(&copyReceipt))
			serviceMust(t, c.integer.Del(&out))
			serviceMust(t, c.integer.Del(&big))
			serviceMust(t, c.integer.Del(&n))
		}
		if !found {
			t.Fatal("no reached later foreign retained child", field)
		}
	}
}
