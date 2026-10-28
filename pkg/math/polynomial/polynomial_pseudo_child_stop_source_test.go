package polynomial

// Source: src/math/polynomial/polynomial.cpp:4939-5010;
// src/util/mpz.h:393-407; src/util/scoped_numeral.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Signed-host TLS/direct allocator profile isolates an actual end-only RAW
// release after ADD. This is not ordinary mathematical allocation OOM.
func TestPolynomialPseudoYoungerRAWSourceStopPreservesActualOlderLocals(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	rationals, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: rationals.Integers(), Rationals: rationals, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, modulus, warm, warmProduct unumeral.Int
	serviceMust(t, c.integer.SetLimbs(&large, []uint32{17, 1 << 30}))
	term, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]unumeral.Int{large, unumeral.NewInt(5)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	b, e := c.MkTerms([]unumeral.Int{large, unumeral.NewInt(3)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(b))
	digits := make([]uint32, 6)
	digits[0], digits[5] = 251, 1<<31
	serviceMust(t, c.integer.SetLimbs(&modulus, digits))
	serviceMust(t, c.modular.SetZP(&modulus))
	Q, e := NewPolynomialRef(c)
	serviceMust(t, e)
	R, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var d uint32
	serviceMust(t, c.PseudoDivide(p, b, 0, &d, Q, R))
	three := unumeral.NewInt(3)
	for i := 0; i < 2; i++ {
		serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
		serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	}
	var external, recovery umemory.Block
	var receipt polynomialPseudoDivisionInvocationReceipt
	defer func() {
		extra, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, c.CleanupPseudoDivision)
		contextTLSFinish(t, rt, Q.Close)
		contextTLSFinish(t, rt, R.Close)
		if x = c.DecRef(p); x != nil {
			if !errors.Is(x, arena.ErrBounds) {
				t.Fatal(x)
			}
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		}
		for _, v := range []*unumeral.Int{&large, &modulus, &warm, &warmProduct} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(v) })
		}
		if x = c.DecRef(b); x != nil {
			if !errors.Is(x, arena.ErrBounds) {
				t.Fatal(x)
			}
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(&b) })
		}
		contextTLSFinish(t, rt, full.CloseChecked)
		contextTLSFinish(t, rt, rationals.CloseChecked)
		for _, b := range []*umemory.Block{&external, &recovery, &extra} {
			if *b != (umemory.Block{}) {
				x := rt.Deallocate(b)
				if x != nil && (!errors.Is(x, arena.ErrBounds) || *b != (umemory.Block{})) {
					t.Fatal("actual named direct release", x, *b)
				}
			}
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
		if visits == 1 {
			contextTLSNearMinimum(t, rt)
			x := rt.Deallocate(&external)
			if !errors.Is(x, arena.ErrBounds) || external != (umemory.Block{}) {
				t.Fatal("actual arm release", x)
			}
		}
		return true
	}
	// The actual public source body binds its internal receipt at guarded begin.
	body := c.PseudoDivide(p, b, 0, &d, Q, R)
	receipt = c.pseudoDivision.directReceipt
	if body == nil || receipt.invocation == 0 || receipt.frame.sourceCause == nil {
		t.Fatal("actual younger source stop absent", body, visits, receipt.frame)
	}
	raw := receipt.frame.remainderBuffer.frame.addMul
	if raw.Frame == nil || raw.Frame.Stage != 3 || raw.Frame.BodyCause != nil || c.integer.AddMulInvocationSourceCause(&raw) != receipt.frame.sourceCause {
		t.Fatal("actual end-only RAW DEL not reached", body, visits, raw.Frame)
	}
	if receipt.frame.depth == 0 || receipt.frame.constructed&(uint16(1)<<pseudoNumeral) == 0 || receipt.frame.numeral == (unumeral.Int{}) || R.root.value != p {
		t.Fatal("younger source stop destroyed older NUM/output", receipt.frame.depth, receipt.frame.numeral, R.root.value)
	}
	source := receipt.frame.sourceCause
	cleanup := receipt.frame.cleanupCause
	if cleanup == nil {
		t.Fatal("actual stopped source cleanup history absent")
	}
	returnedText := body.Error()
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*unumeral.Int{&large, &modulus, &warm, &warmProduct}, Q.header+R.header)
	state := rt.State()
	depth := receipt.frame.depth
	if c.pseudoCleanupPhase(false, false) != source || rt.State() != state || receipt.frame.depth != depth {
		t.Fatal("source retry mutated stopped suffix")
	}
	recovery, e = rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.cleanupPseudoInvocation(&receipt, false) })
	contextTLSFinish(t, rt, func() error { return c.cleanupPseudoInvocation(&receipt, true) })
	if receipt.frame.cleanupCause != cleanup || c.pseudoCleanupPhase(false, false) != source {
		t.Fatal("own ACK erased stopped source cleanup continuation")
	}
	if c.pseudoInvocationSourceCause(&receipt) != source || c.integer.AddMulInvocationSourceCause(&raw) != source {
		t.Fatal("own acknowledgment erased child/history")
	}
	serviceMust(t, c.modular.SetZ())
	c.config.Checkpoint = func() bool { return true }
	serviceMust(t, c.PseudoDivide(p, b, 0, &d, Q, R))
	if c.pseudoInvocationSourceCause(&receipt) != nil || !errors.Is(body, source) || body.Error() != returnedText {
		t.Fatal("later actual call mutated prior error")
	}
}
