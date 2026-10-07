// Source: src/math/polynomial/polynomial.cpp:4600-4614,1988-2039;
// src/util/mpzzp.h:58-68; src/util/mpz.h:393-407; src/util/scoped_numeral.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// A real signed-host accounting guard isolates the reached AddMul RAW end DEL.
// The output ADD completes before that source explicit release. This is an owner-primitive profile, not normal mathematical OOM.
func TestPolynomialPowerYoungerAddMulSourceStopPreservesOlderResultAndHistory(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, warm util.Int
	digits := make([]uint32, 2)
	digits[0] = 17
	digits[1] = 1 << 30
	serviceMust(t, c.integer.SetLimbs(&large, digits))
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{large, util.NewInt(3)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var modulus util.Int
	modulusDigits := make([]uint32, 6)
	modulusDigits[0] = 251
	modulusDigits[5] = 1 << 31
	serviceMust(t, c.integer.SetLimbs(&modulus, modulusDigits))
	serviceMust(t, c.modular.SetZP(&modulus))

	var warmPolynomial polynomialReference
	serviceMust(t, c.powerInto(p, 2, &warmPolynomial))
	three := util.NewInt(3)
	var warmProduct util.Int
	serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	// Ordinary warm cycles retain the actual shared multiplication/remainder
	// scratch. The direct allocator profile isolates actual RAW tmp release.
	serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	var external util.Block
	var receipt polynomialPowerInvocationReceipt
	var output polynomialReference
	defer func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, func() error { return c.cleanupPowerInvocation(&receipt, true) })
		contextTLSFinish(t, rt, func() error { return output.closeValue(c, true) })
		contextTLSFinish(t, rt, func() error { return output.abort(c) })
		contextTLSFinish(t, rt, func() error { return warmPolynomial.closeValue(c, true) })
		contextTLSFinish(t, rt, func() error { return warmPolynomial.abort(c) })
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
		if external != (util.Block{}) {
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&external) })
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("retained owner charge", budget.Used())
		}
	}()
	external, e = rt.Allocate(200000)
	serviceMust(t, e)
	visits := 0
	c.config.Checkpoint = func() bool {
		visits++
		if visits == 2 {
			if !c.powerLocal.active || c.powerLocal.result.value != p {
				t.Fatal("actual older result declaration")
			}
			contextTLSNearMinimum(t, rt)
			x := rt.Deallocate(&external)
			if !errors.Is(x, arena.ErrBounds) || external != (util.Block{}) {
				t.Fatal("actual external TLS release", x)
			}
		}
		return true
	}
	e = c.powerIntoWithReceipt(p, 2, &output, &receipt)

	if e == nil || receipt.frame.executing || receipt.frame.sourceCause == nil {
		t.Fatal("actual AddMul end DEL stop absent", e, receipt.frame, "actual child", receipt.frame.arithmetic.frame)
	}
	if receipt.frame.result.value != p {
		t.Fatal("younger explicit DEL stop destroyed older Power result", receipt.frame.result)
	}
	oldReturnedError := e
	oldErrorText := e.Error()
	oldArithmetic := receipt.frame.arithmetic
	oldRAW := oldArithmetic.frame.addMul
	source := receipt.frame.sourceCause
	cleanup := receipt.frame.cleanupCause
	if cleanup == nil {
		t.Fatal("actual source stop omitted cleanup history")
	}
	if c.integer.AddMulInvocationSourceCause(&oldRAW) != source || oldRAW.Frame.Stage != 3 || oldRAW.Frame.BodyCause != nil {
		t.Fatal("matching source end-only DEL cause", source)
	}
	row, x := c.som.as.coefficient(c, 0)
	serviceMust(t, x)
	termRow, x := c.som.ms.monomial(c, 0)
	serviceMust(t, x)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, output.value, output.incoming, warmPolynomial.value, warmPolynomial.incoming}, []*util.Int{&large, &warm, &warmProduct, &modulus}, 0)
	state := rt.State()
	if c.finishPolynomialPowerSource() != source || rt.State() != state || receipt.frame.result.value != p {
		t.Fatal("source retry changed stopped older result")
	}
	if c.powerInvocationSourceCause(&receipt) != source || state != rt.State() {
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
	contextTLSFinish(t, rt, func() error { return c.cleanupPowerInvocation(&receipt, false) })
	if receipt.frame.cleanupCause != cleanup {
		t.Fatal("false own acknowledgment erased cleanup history")
	}
	contextTLSFinish(t, rt, func() error { return c.cleanupPowerInvocation(&receipt, true) })
	if receipt.frame.cleanupCause != cleanup {
		t.Fatal("true own acknowledgment erased cleanup history")
	}
	if receipt.frame.sourceCause != source || c.powerInvocationSourceCause(&receipt) != source || c.integer.AddMulInvocationSourceCause(&oldRAW) != source {
		t.Fatal("own ack erased source history")
	}
	if row == (util.Int{}) || termRow == (Monomial{}) {
		t.Fatal("stopped source lacked actual persistent row", row, termRow)
	}
	serviceMust(t, c.modular.SetZ())
	c.config.Checkpoint = func() bool { return true }
	var later polynomialPowerInvocationReceipt
	serviceMust(t, c.powerIntoWithReceipt(p, 2, &output, &later))
	serviceMust(t, c.cleanupPowerInvocation(&receipt, true))
	if later.invocation == receipt.invocation || later.frame.sourceCause != nil {
		t.Fatal("old child history poisoned new actual Power invocation")
	}
	if !errors.Is(oldReturnedError, source) || oldReturnedError.Error() != oldErrorText {
		t.Fatal("new source changed previous exception/history")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p, output.value, output.incoming, warmPolynomial.value, warmPolynomial.incoming}, []*util.Int{&large, &warm, &warmProduct, &modulus}, 0, []util.Block{recovery})
	serviceMust(t, c.cleanupPowerInvocation(&later, true))
	serviceMust(t, output.closeValue(c, true))
	serviceMust(t, output.abort(c))
}
