package polynomial

// Source: src/math/polynomial/polynomial.cpp:5356-5421,4600-4614;
// src/util/mpzzp.h:58-68; src/util/mpz.h:393-407; src/util/scoped_numeral.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// A real signed-host accounting guard isolates the reached AddMul RAW end DEL.
// The output ADD completes before that source explicit release. This is an owner-primitive profile, not normal mathematical OOM.
func TestPolynomialResultantYoungerPowerSourceStopPreservesOlderAB(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, warm unumeral.Int
	digits := make([]uint32, 2)
	digits[0] = 17
	digits[1] = 1 << 30
	serviceMust(t, c.integer.SetLimbs(&large, digits))
	term, e := c.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	p, e := c.MkTerms([]unumeral.Int{large, unumeral.NewInt(3)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var modulus unumeral.Int
	modulusDigits := make([]uint32, 6)
	modulusDigits[0] = 251
	modulusDigits[5] = 1 << 31
	serviceMust(t, c.integer.SetLimbs(&modulus, modulusDigits))
	serviceMust(t, c.modular.SetZP(&modulus))

	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	y, e := c.monomials.MkVariable(1, 1)
	serviceMust(t, e)
	xy, e := c.monomials.MkMonomial([]MonomialPower{{0, 3}, {1, 1}})
	serviceMust(t, e)
	a, e := c.MkTerms([]unumeral.Int{large, large, unumeral.NewInt(3), unumeral.NewInt(3)}, []Monomial{xy, term, y, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(a))
	y2, e := c.monomials.MkVariable(1, 2)
	serviceMust(t, e)
	b, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}, []Monomial{y2, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(b))
	var warmPolynomial polynomialReference
	serviceMust(t, m.resultant(a, b, 1, &warmPolynomial))
	three := unumeral.NewInt(3)
	var warmProduct unumeral.Int
	serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	// Ordinary warm cycles retain the actual shared multiplication/remainder
	// scratch. The direct allocator profile isolates actual RAW tmp release.
	serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
	serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	var external umemory.Block
	var receipt polynomialPowerInvocationReceipt
	var output polynomialReference
	defer func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, m.CleanupGCD)
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
		for _, v := range []*Polynomial{&a, &b} {
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(v) })
		}
		for _, a := range []*unumeral.Int{&large, &warm, &warmProduct, &modulus} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(a) })
		}
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		x = rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
			t.Fatal(x)
		}
		if external != (umemory.Block{}) {
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
		if c.powerLocal.executing {
			visits++
			if visits == 2 {
				contextTLSNearMinimum(t, rt)
				x := rt.Deallocate(&external)
				if !errors.Is(x, arena.ErrBounds) || external != (umemory.Block{}) {
					t.Fatal("actual external release", x)
				}
			}
		}
		return true
	}
	e = m.resultant(a, b, 1, &output)
	// Read-only diagnosis of the actual child; this tuple supplies no parent acquisition.
	receipt = polynomialPowerInvocationReceipt{manager: c, frame: &c.powerLocal, invocation: c.powerLocal.generation}
	if e == nil || c.powerLocal.sourceCause == nil {
		t.Fatal("actual reached Power child stop absent", e, "visits", visits)
	}
	child := c.powerLocal.sourceCause
	frame := m.gcdFrames[0]
	if frame.resultantPower.manager != c || frame.resultantPower.frame != &c.powerLocal || frame.resultantPower.invocation != c.powerLocal.generation {
		t.Fatal("missing exact actual Resultant Power invocation", frame.resultantPower)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, a, b, output.value, output.incoming, warmPolynomial.value, warmPolynomial.incoming}, []*unumeral.Int{&large, &warm, &warmProduct, &modulus}, 0)
	if c.integer.AddMulInvocationSourceCause(&c.powerLocal.arithmetic.frame.addMul) != child || c.powerLocal.arithmetic.frame.addMul.Frame.Stage != 3 {
		t.Fatal("actual end DEL source phase", e)
	}
	if frame.depth == 0 || frame.refs[0].root.value != a || frame.refs[1].root.value != b {
		t.Fatal("Resultant destroyed older A/B before actual Power child source DT completion", frame.depth, frame.refs[0].root.value, frame.refs[1].root.value)
	}
	oldReturned, oldText := e, e.Error()
	current := frame.resultantPower
	state := rt.State()
	if m.finishGCDFunctionSource(frame) != child || rt.State() != state || frame.refs[0].root.value != a || frame.refs[1].root.value != b {
		t.Fatal("matched source retry changed older source prefix")
	}
	recovery, x := rt.Allocate(400000)
	serviceMust(t, x)
	defer func() {
		x := rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
			t.Fatal("actual recovery release", x)
		}
	}()
	contextTLSFinish(t, rt, m.CleanupGCD)
	if frame.resultantPower != current || c.powerInvocationSourceCause(&frame.resultantPower) != child || frame.cleanupCause != child {
		t.Fatal("own acknowledgment erased actual parent child/source history")
	}
	serviceMust(t, c.modular.SetZ())
	c.config.Checkpoint = func() bool { return true }
	serviceMust(t, m.resultant(a, b, 1, &output))
	if !errors.Is(oldReturned, child) || oldReturned.Error() != oldText {
		t.Fatal("new Resultant invocation changed original returned cause")
	}

}
