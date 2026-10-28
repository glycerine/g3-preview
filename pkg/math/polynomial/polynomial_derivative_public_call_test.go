package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialDerivativeOwningCallValuesAliasAndFiniteField(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}, {1, 3}}, {{1, 1}}, nil})
	out, err := NewPolynomialRef(c)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	var call PolynomialDerivativeCall
	serviceMust(t, out.Assign(p))
	serviceMust(t, m.DerivativeInto(p, 1, out, &call))
	value, err := out.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, value, map[[3]uint32]int64{{2, 2, 0}: 9, {}: 5})
	serviceMust(t, call.CleanupRetained())
	input, err := out.Get()
	serviceMust(t, err)
	serviceMust(t, m.DerivativeInto(input, 0, out, &call))
	value, err = out.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, value, map[[3]uint32]int64{{1, 2, 0}: 18})
	serviceMust(t, call.CleanupRetained())
	serviceMust(t, c.modular.SetZPUint64(3))
	// Construct field-normalized coefficients rather than reinterpreting the
	// earlier integer polynomial under a different coefficient manager.
	field := polynomialQueryTerms(t, c, []int32{-1, -1, 1}, [][]MonomialPower{{{0, 2}, {1, 3}}, {{1, 1}}, nil})
	serviceMust(t, m.DerivativeInto(field, 1, out, &call))
	serviceMust(t, call.CleanupRetained())
	value, err = out.Get()
	serviceMust(t, err)
	polynomialQueryExpect(t, c, value, map[[3]uint32]int64{{}: -1})
	serviceMust(t, c.modular.SetZ())
	copied := call
	if !errors.Is(copied.CleanupRetained(), ErrPolynomialOwner) {
		t.Fatal("copied derivative call acquired cleanup")
	}
}
func TestPolynomialDerivativeOwningCallAllocationPrefixes(t *testing.T) {
	success, failures := false, 0
	for stop := int64(0); stop < 96 && !success; stop++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		var coefficient unumeral.Int
		words := make([]uint32, 8192)
		words[0] = 7
		words[len(words)-1] = 1
		serviceMust(t, c.integer.SetLimbs(&coefficient, words))
		t.Cleanup(func() { serviceMust(t, c.integer.Del(&coefficient)) })
		term, err := c.monomials.MkVariable(0, 2)
		serviceMust(t, err)
		p, err := c.MkTerms([]unumeral.Int{coefficient}, []Monomial{term})
		serviceMust(t, err)
		p = polynomialCaller(t, c, p)
		out, err := NewPolynomialRef(c)
		serviceMust(t, err)
		var call PolynomialDerivativeCall
		c.integer.Budget().FailAfter(stop)
		err = m.DerivativeInto(p, 0, out, &call)
		c.integer.Budget().FailAfter(-1)
		if err == nil {
			success = true
		} else {
			failures++
			if !numericPhysicalReservationRefusal(err) {
				t.Fatal(stop, err)
			}
		}
		serviceMust(t, call.FinishRetainedLocals())
		serviceMust(t, out.Close())
		serviceMust(t, call.CleanupRetained())
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*unumeral.Int{&coefficient}, 0)
	}
	if !success || failures < 1 {
		t.Fatal("actual derivative horizon incomplete", success, failures)
	}
	t.Log("consecutive failures", failures)
}
func TestPolynomialDerivativeCallStoppedNumeralSourceAndOwningPhases(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, err)
	q, err := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, err)
	m, err := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, err)
	_, err = m.core.monomials.MkVar()
	serviceMust(t, err)
	term, err := m.core.monomials.MkVariable(0, 2)
	serviceMust(t, err)
	words := make([]uint32, 40000)
	words[len(words)-1] = 1
	var a unumeral.Int
	serviceMust(t, q.Integers().SetLimbs(&a, words))
	p, err := m.core.MkTerms([]unumeral.Int{a}, []Monomial{term})
	serviceMust(t, err)
	serviceMust(t, m.core.IncRef(p))
	serviceMust(t, q.Integers().Del(&a))
	out, err := NewPolynomialRef(m.core)
	serviceMust(t, err)
	serviceMust(t, q.Integers().EnsureTmp(0, 40001))
	contextTLSNearMinimum(t, rt)
	var call PolynomialDerivativeCall
	b.FailAfter(0)
	body := m.DerivativeInto(p, 0, out, &call)
	if !numericPhysicalReservationRefusal(body) || !m.derivative.stopped || !errors.Is(m.derivative.sourceCause, arena.ErrBounds) {
		t.Fatal("actual stopped NUM not qualified", body, m.derivative.sourceCause)
	}
	before, stats, used := rt.State(), rt.Stats(), b.Used()
	beforeCleanup := m.derivative.cleanupCause
	err = call.FinishSource()
	if !errors.Is(err, arena.ErrBounds) || rt.State() != before || rt.Stats() != stats || b.Used() != used || m.derivative.cleanupCause != beforeCleanup {
		t.Fatal("source wrapper retried stopped owning NUM", err)
	}
	b.FailAfter(-1)
	// The NUM continuation is younger than the caller output. Raw children wait.
	contextTLSFinish(t, rt, call.FinishRetainedLocals)
	contextTLSFinish(t, rt, out.Close)
	contextTLSFinish(t, rt, call.CleanupRetained)
	contextTLSFinish(t, rt, func() error { return m.core.CleanupValue(&p) })
	contextTLSFinish(t, rt, m.CloseChecked)
	contextTLSFinish(t, rt, q.CloseChecked)
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("whole derivative caller retained bytes", b.Used())
	}
}

func TestPolynomialDerivativeCallOwnsOnlyAcceptedOrphanGeneration(t *testing.T) {
	m, p, _ := polynomialDerivativeEvenFixture(t, false)
	c := m.core
	out, err := NewPolynomialRef(c)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	var old, current PolynomialDerivativeCall
	serviceMust(t, m.DerivativeInto(p, 0, out, &old))
	before := c.monomials.publishedObjects
	if !old.active || len(m.derivative.orphans) != 3 {
		t.Fatal("actual zero-coefficient orphan cohort absent")
	}
	serviceMust(t, old.FinishSource())
	if c.monomials.publishedObjects != before {
		t.Fatal("source phase swept unjoined monomial cohort")
	}
	if err := m.DerivativeInto(p, 0, out, &old); !errors.Is(err, ErrPolynomialPending) {
		t.Fatal("reused caller lost pending cohort", err)
	}
	serviceMust(t, old.CleanupRetained())
	if c.monomials.publishedObjects != before-3 {
		t.Fatal("owning phase did not release accepted cohort")
	}
	serviceMust(t, m.DerivativeInto(p, 0, out, &current))
	before = c.monomials.publishedObjects
	used := c.integer.Budget().Used()
	invocation := m.derivative.invocation
	serviceMust(t, old.CleanupRetained())
	if c.monomials.publishedObjects != before || c.integer.Budget().Used() != used || m.derivative.invocation != invocation || !current.active {
		t.Fatal("stale call adopted newer monomial cohort")
	}
	serviceMust(t, current.CleanupRetained())
}
