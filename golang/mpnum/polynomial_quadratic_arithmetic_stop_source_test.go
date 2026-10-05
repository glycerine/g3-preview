// Source: src/math/polynomial/polynomial.cpp:1988-2039,6600-6679;
// src/util/mpzzp.h:58-68; src/util/scoped_numeral.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// A real regular primitive PN destructor is bound to the actual quadratic
// DEBUG temporary receipt. This is a signed-host owner-primitive proof of the
// younger/older source suffix, not ordinary field quadratic factorization.
func TestPolynomialQuadraticDEBUGArithmeticSourceStopPreservesOlderLocals(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	budget := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, warm Int
	digits := make([]uint32, 16)
	digits[0] = 17
	digits[15] = 1 << 30
	serviceMust(t, c.integer.SetLimbs(&large, digits))
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{large}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var modulus Int
	modulusDigits := make([]uint32, 16)
	modulusDigits[0] = 251
	modulusDigits[15] = 1 << 31
	serviceMust(t, c.integer.SetLimbs(&modulus, modulusDigits))
	serviceMust(t, c.modular.SetZP(&modulus))
	two := NewInt(3)
	var warmProduct Int
	serviceMust(t, c.integer.Mul(&large, &two, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &c.modular.p, &warm))
	// Second ordinary cycle leaves the actual source result scratch BIG after
	// the first output SWAP replaced it with the default six-limb member.
	serviceMust(t, c.integer.Mul(&large, &two, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &c.modular.p, &warm))
	parent, e := m.beginGCDFunction()
	serviceMust(t, e)
	parent.constructNumeral(0)
	serviceMust(t, q.z.Copy(&parent.nums[0], &large))
	parent.constructReference(c, 11)
	receipt := &parent.quadraticArithmetic[3]
	older, depth := parent.nums[0], parent.depth
	defer func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, func() error { return m.CleanupGCD() })
		if x = c.DecRef(p); x != nil {
			if !errors.Is(x, arena.ErrBounds) {
				t.Fatal(x)
			}
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		}
		for _, a := range []*Int{&large, &warm, &warmProduct, &modulus} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(a) })
		}
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		x = rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal(x)
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("retained owner charge", budget.Used())
		}
	}()
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, rt)
	e = rt.Deallocate(&external)
	if e == nil || external != (memory.Block{}) {
		t.Fatal("actual consumed external signed guard", e)
	}
	serviceMust(t, c.som.acquireArithmeticInvocation(receipt))
	three := NewInt(3)
	e = c.som.addMulWithArithmeticReceipt(&three, c.monomials.unit, p, receipt)
	if e == nil || receipt.frame.executing || receipt.frame.sourceCause == nil {
		t.Fatal("actual PN source destructor stop absent", e, receipt.frame)
	}
	// Receipt copies may name the same actual child; they duplicate no NUM roots.
	receiptCopy := *receipt
	receipt = &receiptCopy
	oldReturnedError := e
	oldPN := receipt.frame.normalize
	source := receipt.frame.sourceCause
	if c.modular.pNormalizeInvocationSourceCause(&oldPN) != source {
		t.Fatal("matching child source cause", source)
	}
	row, x := c.som.as.coefficient(c, 0)
	serviceMust(t, x)
	termRow, x := c.som.ms.monomial(c, 0)
	serviceMust(t, x)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*Int{&large, &warm, &warmProduct, &modulus}, 0)
	m.gcdDepth--
	outer := m.finishGCDFunctionSource(parent)
	c.pscLeave()
	if !errors.Is(outer, source) || parent.cleanupCause != source || parent.depth != depth || parent.nums[0] != older {
		t.Fatal("younger arithmetic source stop drained older quadratic locals", outer, parent.depth, depth, parent.nums[0], older)
	}
	state := rt.State()
	again := m.finishGCDFunctionSource(parent)
	if !errors.Is(again, source) || parent.depth != depth || parent.nums[0] != older || rt.State() != state {
		t.Fatal("source retry replayed older quadratic declarations", again)
	}
	if c.arithmeticInvocationSourceCause(receipt) != source || state != rt.State() {
		t.Fatal("readonly history changed source frontier")
	}
	// Restore accounting by a real named allocation only after immediate proof.
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	defer func() {
		x := rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal("actual recovery release", x, recovery)
		}
	}()
	contextTLSFinish(t, rt, func() error { return m.cleanupQuadraticChildren(parent, false) })
	contextTLSFinish(t, rt, func() error { return m.CleanupGCD() })
	if receipt.frame.sourceCause != source || c.arithmeticInvocationSourceCause(receipt) != source || c.modular.pNormalizeInvocationSourceCause(&oldPN) != source {
		t.Fatal("own ack erased source history")
	}
	if row == (Int{}) || termRow == (Monomial{}) {
		t.Fatal("stopped source lacked actual persistent row", row, termRow)
	}
	if parent.active || parent.depth != 0 || parent.cleanupCause != source || !errors.Is(outer, source) || !errors.Is(oldReturnedError, source) {
		t.Fatal("owning acknowledgment erased parent source history", parent.active, parent.depth, parent.cleanupCause)
	}
	serviceMust(t, c.modular.SetZ())
	oldInvocation := parent.invocation
	primitive, x := m.IsPrimitive(c.unit, 0)
	serviceMust(t, x)
	if !primitive || parent.invocation == oldInvocation || parent.cleanupCause != nil || !errors.Is(outer, source) || !errors.Is(oldReturnedError, source) {
		t.Fatal("new real cached invocation changed previous source exception", primitive, parent.invocation, oldInvocation, parent.cleanupCause)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &warm, &warmProduct, &modulus}, 0, []memory.Block{recovery})
}
