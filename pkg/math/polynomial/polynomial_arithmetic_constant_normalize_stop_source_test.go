// Source: src/math/polynomial/polynomial.cpp:2530-2536,2692-2698;
// src/util/mpzzp.h:58-68; src/util/scoped_numeral.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// A real signed-host accounting guard isolates the reached DEBUG verification
// destructor. This is an owner-primitive profile, not normal mathematical OOM.
func TestPolynomialConstantYoungerPNStopPreservesOlderNUMSuffix(t *testing.T) {
	if !polynomialSourceDebug {
		return
	}
	budget := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, warm util.Int
	digits := make([]uint32, 16)
	digits[0] = 17
	digits[15] = 1 << 30
	serviceMust(t, c.integer.SetLimbs(&large, digits))
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{large}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var modulus util.Int
	modulusDigits := make([]uint32, 16)
	modulusDigits[0] = 251
	modulusDigits[15] = 1 << 31
	serviceMust(t, c.integer.SetLimbs(&modulus, modulusDigits))
	serviceMust(t, c.modular.SetZP(&modulus))
	two := util.NewInt(3)
	var warmProduct util.Int
	serviceMust(t, c.integer.Mul(&large, &two, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	// Second ordinary cycle leaves the actual source result scratch BIG after
	// the first output SWAP replaced it with the default six-limb member.
	serviceMust(t, c.integer.Mul(&large, &two, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	var receipt polynomialArithmeticInvocationReceipt
	defer func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, func() error { return c.cleanupArithmeticInvocation(&receipt, true) })
		if x = c.DecRef(p); x != nil {
			if !errors.Is(x, arena.ErrBounds) {
				t.Fatal(x)
			}
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		}
		for _, a := range []*util.Int{&large, &warm, &warmProduct, &modulus} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(a) })
		}
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		x = rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (util.Block{})) {
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
	if e == nil || external != (util.Block{}) {
		t.Fatal("actual consumed external signed guard", e)
	}
	out, body := c.mulAddWithReceipt(c.zero, Polynomial{}, &warmProduct, &receipt)
	e = body
	if out != (Polynomial{}) {
		t.Fatal("stopped younger source exposed result", out)
	}
	if e == nil || receipt.frame.executing || receipt.frame.sourceCause == nil {
		t.Fatal("actual PN source destructor stop absent", e, receipt.frame)
	}
	oldReturnedError := e
	oldPN := receipt.frame.normalize
	source := receipt.frame.sourceCause
	if c.modular.PNormalizeInvocationSourceCause(&oldPN) != source {
		t.Fatal("matching child source cause", source)
	}
	locals := [3]util.Int{c.constantScope.numeral, (*c.constantScope.rational.BorrowNumerator()), (*c.constantScope.rational.BorrowDenominator())}
	if !c.constantScope.numeralCompleted || !c.constantScope.rationalCompleted || c.constantScope.rationalStage != 0 || (locals[0]).Cell == 0 || !c.integer.IsLiveCellToken(uint64((locals[0]).Cell)) || c.integer.Cell((locals[0]).Cell).CleanupPending || c.integer.Cell((locals[0]).Cell).Released {
		t.Fatal("younger PN source stop consumed older NUM suffix", c.constantScope)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&large, &warm, &warmProduct, &modulus}, 0)
	state := rt.State()
	if again := c.finishConstantScope(nil); !errors.Is(again, source) || c.arithmeticInvocationSourceCause(&receipt) != source || state != rt.State() || locals != [3]util.Int{c.constantScope.numeral, (*c.constantScope.rational.BorrowNumerator()), (*c.constantScope.rational.BorrowDenominator())} {
		t.Fatal("readonly history changed source frontier")
	}
	// Restore accounting by a real named allocation only after immediate proof.
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	defer func() {
		x := rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (util.Block{})) {
			t.Fatal("actual recovery release", x, recovery)
		}
	}()
	contextTLSFinish(t, rt, func() error { return c.cleanupArithmeticInvocation(&receipt, false) })
	contextTLSFinish(t, rt, func() error { return c.cleanupArithmeticInvocation(&receipt, true) })
	if receipt.frame.sourceCause != source || c.arithmeticInvocationSourceCause(&receipt) != source || c.modular.PNormalizeInvocationSourceCause(&oldPN) != source {
		t.Fatal("own ack erased source history")
	}
	serviceMust(t, c.modular.SetZ())
	var later polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&later))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &later))
	serviceMust(t, c.cleanupArithmeticInvocation(&receipt, true))
	if !c.matchesArithmeticInvocation(&later) || later.frame.normalize != (util.IntegerNormalizeInvocationReceipt{}) {
		t.Fatal("old child history poisoned new no-PN invocation")
	}
	if c.modular.PNormalizeInvocationSourceCause(&oldPN) != source || !errors.Is(oldReturnedError, source) {
		t.Fatal("new source changed previous exception/history")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &warm, &warmProduct, &modulus}, 0, []util.Block{recovery})
	serviceMust(t, c.cleanupArithmeticInvocation(&later, true))
}
