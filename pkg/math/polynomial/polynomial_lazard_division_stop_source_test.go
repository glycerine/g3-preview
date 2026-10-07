// Source: src/math/polynomial/polynomial.cpp:5659-5692,5114-5156;
// src/util/mpzzp.h:58-68,167-185; src/util/mpz.h:156-162.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The signed-host TLS release guard isolates a real reached lexical verification
// destructor. This is an owner-primitive domain, not ordinary mathematical OOM.
func TestPolynomialLazardYoungerDivisionSourceStopPreservesTwoReferences(t *testing.T) {
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
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var modulus, upper util.Int
	words := make([]uint32, 17)
	for i := range words {
		words[i] = ^uint32(0)
	}
	words[16] = 511
	serviceMust(t, c.integer.SetLimbs(&modulus, words))
	words[16] = 255
	serviceMust(t, c.integer.SetLimbs(&upper, words))
	p, e := c.MkConstantTakeCopy(&upper)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZP(&modulus))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var receipt polynomialLazardInvocationReceipt
	for i := 0; i < 2; i++ {
		serviceMust(t, c.seLazardWithReceipt(2, p, p, 0, &out.root, &receipt))
	}
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	t.Cleanup(func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, func() error { return c.cleanupLazardInvocation(&receipt, false) })
		contextTLSFinish(t, rt, func() error { return c.cleanupLazardInvocation(&receipt, true) })
		contextTLSFinish(t, rt, out.Close)
		contextTLSFinish(t, rt, func() error { return c.DecRef(p) })
		for _, n := range []*util.Int{&upper, &modulus} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(n) })
		}
		contextTLSFinish(t, rt, full.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		if external != (util.Block{}) {
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&external) })
		}
		x = rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (util.Block{})) {
			t.Fatal(x)
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("named owner budget", budget.Used())
		}
	})
	armed := true
	var release error
	c.config.Checkpoint = func() bool {
		if armed && c.lazardLocal.executing && c.lazardLocal.phase == 3 {
			armed = false
			if !c.exactDivision.active || c.exactDivision.kind != 1 {
				t.Fatal("checkpoint before actual NUM/ref declaration")
			}
			contextTLSNearMinimum(t, rt)
			release = rt.Deallocate(&external)
		}
		return true
	}
	body := c.seLazardWithReceipt(2, p, p, 0, &out.root, &receipt)
	parent := &c.lazardLocal
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, out.root.value, out.root.incoming, parent.x.value, parent.x.incoming, parent.c.value, parent.c.incoming}, []*util.Int{&upper, &modulus}, out.header, nil)
	s := &c.exactDivision
	if parent.constructed != 2 || parent.cursor != 0 || parent.x.value == (Polynomial{}) || parent.c.value == (Polynomial{}) || parent.sourceCause != s.sourceCause {
		t.Fatal("younger actual division stop did not protect two older refs", parent.cursor, parent.sourceCause, s.sourceCause)
	}
	if !errors.Is(release, arena.ErrBounds) || external != (util.Block{}) {
		t.Fatal("actual postphysical accounting setup", release)
	}
	if body == nil || s.sourceCause == nil || s.stage != 0 || !s.active {
		t.Fatal("actual younger PN stop did not preserve source NUM/ref", body, s.sourceCause, s.stage, s.active)
	}
	child := s.normalize
	cause := s.sourceCause
	old := body
	if c.modular.PNormalizeInvocationSourceCause(&child) != cause {
		t.Fatal("exact child source stop", cause)
	}
	state := rt.State()
	n := s.numeral
	ref := s.monomial
	if e = c.finishLazard(false); e != cause || state != rt.State() || s.numeral != n || s.monomial != ref || parent.cursor != 0 {
		t.Fatal("source retry changed older roots", e)
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.cleanupLazardInvocation(&receipt, false) })
	contextTLSFinish(t, rt, func() error { return c.cleanupLazardInvocation(&receipt, true) })
	if s.sourceCause != cause || c.modular.PNormalizeInvocationSourceCause(&child) != cause || !errors.Is(old, cause) {
		t.Fatal("own acknowledgment erased first source history")
	}
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, out.root.value, out.root.incoming}, []*util.Int{&upper, &modulus}, out.header, []util.Block{recovery})
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal(e)
	}
}
