package polynomial

// Source: src/math/polynomial/polynomial.cpp:5904-5940;
// src/util/memory_manager.cpp:274-285.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Returning Go ErrorTransport profile: allocation refusal invokes a supported
// owner callback; it is distinct from source mathematical GCD behavior.
func TestPolynomialNormalizeActualTransportOwnerSourceRefusal(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	var full *MultivariatePolynomialManager
	active, seen := false, false
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			active = false
			seen = true
			s := &full.normalization
			state, used := rtStateForNormalize(full), budget.Used()
			started, stage := full.sourceStarted, full.bodyStage
			cursor, cause := s.cursor, s.sourceCause
			own := full.CleanupNormalization()
			source := full.SourceFinalize()
			prefix := full.core.SourceFinalizePrefix()
			reset := full.core.cheap.reset()
			if own != ErrPolynomialPending || source != ErrPolynomialPending || prefix != ErrPolynomialPending || reset != ErrPolynomialPending || state != rtStateForNormalize(full) || used != budget.Used() || started != full.sourceStarted || stage != full.bodyStage || cursor != s.cursor || cause != s.sourceCause {
				t.Errorf("actual executing owner callback changed prefix own=%v source=%v prefix=%v reset=%v", own, source, prefix, reset)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	var a unumeral.Int
	limbs := make([]uint32, 65536)
	limbs[0], limbs[len(limbs)-1] = 3, 1
	serviceMust(t, q.Integers().SetLimbs(&a, limbs))
	p, e := c.MkTerms([]unumeral.Int{a}, []Monomial{c.monomials.unit})
	serviceMust(t, e)
	var caller polynomialReference
	serviceMust(t, caller.assign(c, p))
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	_, returned := full.Normalize(p)
	active = false
	if !seen || !numericPhysicalReservationRefusal(returned) {
		t.Fatal("real numeric Copy returning transport absent", seen, returned)
	}
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p}, []*unumeral.Int{&a}, 0, nil)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, full.CleanupNormalization())
	serviceMust(t, q.Integers().Del(&a))
	serviceMust(t, caller.closeValue(c, true))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, rt.Close())
	if budget.Used() != 0 {
		t.Error("named owners retained", budget.Used())
	}
}
func rtStateForNormalize(m *MultivariatePolynomialManager) umemory.RuntimeState {
	return m.core.memory.State()
}
