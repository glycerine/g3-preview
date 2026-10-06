// Source: src/math/polynomial/polynomial.cpp:5473-5500; src/util/obj_ref.h;
// src/util/memory_manager.cpp. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// Actual returned-factory incoming INC precedes the old result DEC callback.
// Public TLS operations select the guarded signed domain; this is a Go source
// release-stop proof, not defined C++ exhaustion or a mathematical defect.
func TestPolynomialDiscriminantStrictLeadingNUMStopKeepsOlderPrimeAndCause(t *testing.T) {
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
	var leading Int
	digits := make([]uint32, 64)
	digits[0] = 7
	digits[63] = 1
	serviceMust(t, c.integer.SetLimbs(&leading, digits))
	x2, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}})
	serviceMust(t, e)
	unit, e := c.monomials.MkMonomial(nil)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{leading, NewInt(1)}, []Monomial{x2, unit})
	serviceMust(t, e)
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, pr.Assign(p))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var lane, recovery memory.Block
	var lcObject memory.ObjectRef
	var lcBefore Int
	var primeBefore Polynomial
	calls := 0
	h, e := NewPolynomialDeletionHandler(budget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, old Polynomial) error {
		s := &m.discriminant
		if calls != 0 || !s.active || !s.executing || s.depth != 2 || s.scalar.manager != core || !s.scalar.frame.ended || s.scalar.frame.active || old != out.root.value || !out.root.assigning || out.root.incoming == (Polynomial{}) {
			return nil
		}
		calls++
		if s.lc.IsSmall() || s.refs[0].value == (Polynomial{}) || core.factoryInvocation.result != (Polynomial{}) {
			t.Fatal("actual parent NUM/prime and incoming INC transfer prefix missing")
		}
		lcBefore = s.lc
		primeBefore = s.refs[0].value
		lcObject, e = core.integer.cellObject(s.lc.cell)
		serviceMust(t, e)
		contextTLSNearMinimum(t, rt)
		lane, e = rt.Allocate(8)
		serviceMust(t, e)
		delta := int64(-1000000) - rt.State().LocalSize
		request := uint64(16) + uint64(delta) - 8
		if e = rt.Reallocate(&lane, request); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual public realloc guard", e)
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	e = m.Discriminant(p, 0, out)
	returnedCause := e
	s := &m.discriminant
	if !errors.Is(e, arena.ErrBounds) || calls != 1 || s.depth != 2 || s.refs[0].value != primeBefore || s.lc != lcBefore || s.sourceCause == nil {
		t.Fatal("strict leading NUM stop did not retain older prime", e, calls, s.depth)
	}
	if !c.integer.cell(s.lc.cell).released || !c.integer.cell(s.lc.cell).cleanupPending {
		t.Fatal("actual LC allocation was not physically consumed")
	}
	if _, e = c.integer.allocator.ReadUint32(lcObject, 0); e == nil {
		t.Fatal("consumed exact LC object still readable")
	}
	values := func() []Polynomial {
		return []Polynomial{pr.root.value, out.root.value, out.root.incoming, s.refs[0].value, s.refs[1].value}
	}
	audit := func() {
		raws := []memory.Block{lane}
		if recovery != (memory.Block{}) {
			raws = append(raws, recovery)
		}
		polynomialImpPhysicalAuditNamedRaw(t, m, values(), []*Int{&leading, &s.lc}, pr.header+out.header+h.header, raws)
	}
	audit()
	cause := s.sourceCause
	scalar, constant, resultant, derivative := s.scalar, s.constant, s.resultant, s.derivative
	used, state, stats := budget.Used(), rt.State(), rt.Stats()
	if again := m.finishDiscriminantSource(); again != cause || used != budget.Used() || state != rt.State() || stats != rt.Stats() {
		t.Fatal("source retry replayed consumed leading NUM", again)
	}
	// The owning cursor can acknowledge consumed LC before older prime retirement.
	ownErr := m.CleanupDiscriminant()
	if ownErr != nil && !errors.Is(ownErr, arena.ErrBounds) {
		t.Fatal("exact parent owning prefix", ownErr)
	}
	if m.DiscriminantSourceCause() != cause || (ownErr != nil && s.cleanupCause != ownErr) {
		t.Fatal("owning prefix erased first source/cleanup cause")
	}
	audit()
	// Real external storage restores the guarded accounting lane only after
	// exact acquisition/consumption and unchanged source retry evidence above.
	recovery, e = rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupDiscriminant())
	if m.DiscriminantSourceCause() != cause || m.finishDiscriminantSource() != cause || (ownErr != nil && s.cleanupCause != ownErr) || s.scalar != scalar || s.constant != constant || s.resultant != resultant || s.derivative != derivative {
		t.Fatal("completed owning ack erased source history")
	}
	audit()
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	// A new actual degree-zero invocation retires old history, constructs its
	// p_prime reference, and writes zero without leading NUM construction.
	next, e := NewPolynomialRef(c)
	serviceMust(t, e)
	oldInvocation := s.invocation
	serviceMust(t, m.Discriminant(c.unit, 0, next))
	if s.invocation == oldInvocation || m.DiscriminantSourceCause() != nil || next.root.value != c.zero || !errors.Is(returnedCause, cause) {
		t.Fatal("new actual invocation did not reset source history")
	}
	serviceMust(t, next.Close())
	serviceMust(t, out.Close())
	serviceMust(t, pr.Close())
	serviceMust(t, c.integer.Del(&leading))
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal("real recovery block physical consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("named graph remains", budget.Used())
	}
}
