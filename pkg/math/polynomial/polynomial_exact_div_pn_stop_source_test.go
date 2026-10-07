// Source: src/math/polynomial/polynomial.cpp:5114-5156;
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
func TestPolynomialExactDivYoungerPNSourceStopPreservesOlderNUMReference(t *testing.T) {
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
	warm, e := c.ExactDiv(p, p)
	serviceMust(t, e)
	serviceMust(t, c.CleanupValue(&warm))
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	t.Cleanup(func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, c.CleanupExactDivision)
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
		if armed {
			armed = false
			if !c.exactDivision.active || c.exactDivision.kind != 1 {
				t.Fatal("checkpoint before actual NUM/ref declaration")
			}
			contextTLSNearMinimum(t, rt)
			release = rt.Deallocate(&external)
		}
		return true
	}
	result, body := c.ExactDiv(p, p)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, []*util.Int{&upper, &modulus}, 0)
	s := &c.exactDivision
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
	if e = c.exactDivisionCleanup(false); e != cause || state != rt.State() || s.numeral != n || s.monomial != ref {
		t.Fatal("source retry changed older roots", e)
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.cleanupExactDivInvocation(&s.directReceipt, false) })
	contextTLSFinish(t, rt, func() error { return c.cleanupExactDivInvocation(&s.directReceipt, true) })
	if s.sourceCause != cause || c.modular.PNormalizeInvocationSourceCause(&child) != cause || !errors.Is(old, cause) {
		t.Fatal("own acknowledgment erased first source history")
	}
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p}, []*util.Int{&upper, &modulus}, 0, []util.Block{recovery})
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal(e)
	}
}
