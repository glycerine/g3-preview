// Source: src/math/polynomial/polynomial.cpp:5193-5272,4939-5010;
// src/util/mpz.h:393-407; src/util/scoped_numeral.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Signed-host TLS/direct allocator profile isolates an actual end-only RAW
// release after ADD. This is not ordinary mathematical allocation OOM.
func TestPolynomialQuasiResultantSuccessfulChildOlderNUMStopAndLaterHistory(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	rationals, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: rationals.Integers(), Rationals: rationals, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, modulus, warm, warmProduct, older util.Int
	serviceMust(t, c.integer.SetLimbs(&large, []uint32{17, 1 << 30}))
	term, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{large, util.NewInt(5)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	b, e := c.MkTerms([]util.Int{large, util.NewInt(3)}, []Monomial{term, c.monomials.unit})
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
	three := util.NewInt(3)
	for i := 0; i < 2; i++ {
		serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
		serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	}
	var external, recovery util.Block
	defer func() {
		extra, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, full.CleanupQuasiResultant)
		contextTLSFinish(t, rt, c.CleanupPseudoDivision)
		contextTLSFinish(t, rt, Q.Close)
		contextTLSFinish(t, rt, R.Close)
		if x = c.DecRef(p); x != nil {
			if !errors.Is(x, arena.ErrBounds) {
				t.Fatal(x)
			}
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		}
		for _, v := range []*util.Int{&large, &modulus, &warm, &warmProduct, &older} {
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
		for _, b := range []*util.Block{&external, &recovery, &extra} {
			if *b != (util.Block{}) {
				x := rt.Deallocate(b)
				if x != nil && (!errors.Is(x, arena.ErrBounds) || *b != (util.Block{})) {
					t.Fatal("actual named direct release", x, *b)
				}
			}
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("retained owner charge", budget.Used())
		}
	}()

	serviceMust(t, full.QuasiResultant(p, b, 0, R))
	parent := &c.quasiResultant
	old := polynomialQuasiResultantInvocationReceipt{c, parent, parent.invocation}
	diagnostic, ok := c.quasiResultantDiagnostic(&old)
	if !ok || parent.active {
		t.Fatal("successful child did not end source frame")
	}
	serviceMust(t, c.integer.Copy(&older, &large))
	external, e = rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, rt)
	if x := rt.Deallocate(&external); !errors.Is(x, arena.ErrBounds) || external != (util.Block{}) {
		t.Fatal("actual arm", x)
	}
	guard, stopped := upolynomialSourceNumeralDel(c.integer, &older)
	if guard || !errors.Is(stopped, arena.ErrBounds) || (older).Cell == 0 || !c.integer.Cell((older).Cell).Released || !c.integer.Cell((older).Cell).CleanupPending {
		t.Fatal("actual older scoped NUM source stop", stopped)
	}
	current, ok := c.quasiResultantDiagnostic(&old)
	if !ok || current != diagnostic {
		t.Fatal("older source stop changed completed child diagnostics")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, Q.root.value, Q.root.incoming, R.root.value, R.root.incoming}, []*util.Int{&large, &modulus, &warm, &warmProduct, &older}, Q.header+R.header)
	recovery, e = rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.integer.Del(&older) })
	var later polynomialQuasiResultantInvocationReceipt
	serviceMust(t, c.quasiResultantWithReceipt(p, b, 0, R, &later))
	current, ok = c.quasiResultantDiagnostic(&old)
	if !ok || current != diagnostic || old.invocation == later.invocation {
		t.Fatal("later begin erased older-context diagnostic tuple")
	}
	before := R.root.value
	serviceMust(t, c.cleanupQuasiResultantInvocation(&old, true))
	if R.root.value != before || !c.matchesQuasiResultantInvocation(&later) {
		t.Fatal("stale own ACK changed later caller/output")
	}
}
