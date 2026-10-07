// Source: src/math/polynomial/polynomial.cpp:2757-2768;
// src/util/mpn.cpp:206-217; src/util/scoped_numeral.h;
// src/util/memory_manager.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestScalarDivDirectStoppedReturnedCauseSurvivesOwnAndNewInvocation(t *testing.T) {
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
	var coefficient util.Int
	digits := make([]uint32, 64)
	digits[0] = 7
	digits[63] = 1
	serviceMust(t, c.integer.SetLimbs(&coefficient, digits))
	term, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{coefficient}, []Monomial{term})
	serviceMust(t, e)
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, pr.Assign(p))
	// Actual source scratch helper growth avoids an older scratch deletion in
	// the selected signed release domain; no synthetic allocation is reserved.
	serviceMust(t, c.integer.EnsureTmp(0, 64))
	serviceMust(t, c.integer.Naturals().ResizeDiv(0, 65))
	serviceMust(t, c.integer.Naturals().ResizeDiv(1, 1))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual guarded realloc", e)
	}
	d := util.NewInt(2)
	var receipt polynomialScalarDivInvocationReceipt
	var appendReceipt polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	result, returned := c.scalarDivWithParentReceipts(p, &d, &receipt, &appendReceipt, &factory)
	s := &c.scalarDivision
	if !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("actual source error missing", returned)
	}
	if util.SourceExtendedGCDDebug {
		v := &c.integer.Naturals().VerificationInvocation
		if !c.integer.Naturals().MatchesPNVerification(&s.verification) || v.Constructed || v.Buffer.PendingField == (util.Block{}) || s.sourceCause != nil || !errors.Is(s.bodyCause, arena.ErrBounds) {
			t.Fatal("actual failed younger verification ctor prefix", v, s.bodyCause, s.sourceCause)
		}
		original := v.Buffer
		copied := original
		if copied.Bound(false) {
			t.Fatal("copied private buffer acquired actual cleanup authority")
		}
		serviceMust(t, copied.Cleanup())
		if !reflect.DeepEqual(original, v.Buffer) || !reflect.DeepEqual(original, copied) {
			t.Fatal("copied private buffer changed actual pending owner")
		}
	} else if s.sourceCause == nil {
		t.Fatal("actual reached older NUM source DT stop missing", returned, s.sourceCause)
	}
	t.Log("actual stop", "NUM", s.a, "tmp", c.integer.Tmp, "cheap", c.cheap.as, c.cheap.ms, "pending", c.pending, "factory", c.factoryInvocation)
	capturedText := returned.Error()
	source := s.sourceCause
	auditIndex := 0
	audit := func(recovery util.Block) {
		auditIndex++
		row, rowErr := c.cheap.as.coefficient(c, 0)
		t.Log("audit source frontier", auditIndex, "NUM", s.a, "row", row, rowErr, "tmp", c.integer.Tmp, "numeric", s.numeric, "returned", s.returned, "transport", s.transportBound, s.transport, s.transportSlot)
		raw := []util.Block{lane}
		if recovery != (util.Block{}) {
			raw = append(raw, recovery)
		}
		polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{pr.root.value, result, s.returned, c.factoryInvocation.result}, []*util.Int{&coefficient, &s.a}, pr.header, raw)
	}
	audit(util.Block{})
	// Preserve the exact stopped source cause even if owning acknowledgement
	// consumes its already released physical allocation before recovery.
	own := c.cleanupScalarDivInvocation(&receipt, false)
	if own != nil && !errors.Is(own, arena.ErrBounds) {
		t.Fatal("matched false own", own)
	}
	if c.scalarDivInvocationSourceCause(&receipt) != source {
		t.Fatal("false owning acknowledgement erased source cause")
	}
	audit(util.Block{})
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, c.cleanupScalarDivInvocation(&receipt, false))
	serviceMust(t, c.cleanupScalarDivInvocation(&receipt, true))
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
	if c.scalarDivInvocationSourceCause(&receipt) != source || returned.Error() != capturedText || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("owning acknowledgement mutated returned cause")
	}
	audit(recovery)
	oldInvocation := receipt.invocation
	next, e := c.scalarDiv(c.zero, nil)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(next))
	if c.scalarDivision.invocation == oldInvocation || c.scalarDivision.sourceCause != nil {
		t.Fatal("new guarded invocation did not retire source history")
	}
	if returned.Error() != capturedText || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("new success mutated earlier returned error")
	}
	serviceMust(t, c.DecRef(next))
	serviceMust(t, pr.Close())
	serviceMust(t, c.integer.Del(&coefficient))
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal("actual recovery consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("named owner graph remains", budget.Used())
	}
}
