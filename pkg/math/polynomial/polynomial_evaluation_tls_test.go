// Source: src/math/polynomial/polynomial.cpp:6436-6522; src/util/mpq.cpp;
// src/util/memory_manager.cpp. Licensed under the MIT license.
// Author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This public Runtime realloc recipe deliberately reaches the guarded native
// signed-host overflow boundary. It is neither valid exhaustion nor free OOM.
func TestPolynomialEvaluationStoppedPhysicalLeafAndReuse(t *testing.T) {
	budget := arena.NewBudget(1 << 25)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	t.Cleanup(func() {
		contextTLSFinish(t, rt, q.CloseChecked)
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("actual graph refund", budget.Used())
		}
	})
	m, p, _ := polynomialEvalFixtureWithRationals(t, q, []polynomialEvalTerm{{1, [3]uint32{1, 1, 0}}})
	v := polynomialEvalRationalMap{manager: q}
	for i := range v.values {
		v.values[i] = util.NewRational()
	}
	out := util.NewRational()
	m.evaluation.rational = q
	m.evaluation.active = true
	frame, e := m.evaluationRationalFrame()
	serviceMust(t, e)
	m.evaluation.depth--
	serviceMust(t, q.Integers().SetLimbs(&(*frame.aux.BorrowNumerator()), []uint32{1, 1}))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(&(*frame.aux.BorrowNumerator()), 40000))
	polynomialEvaluationAudit(t, m, p, &v, &out)
	// Ordinary source construction/deletion warms the two actual ID free vectors
	// before the signed-host fixture, isolating the reached lexical release.
	term, e := m.core.monomials.MkVariable(2, 1)
	serviceMust(t, e)
	warm, e := m.core.MkTerms([]util.Int{util.NewInt(3)}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(warm))
	serviceMust(t, m.core.DecRef(warm))
	contextTLSNearMinimum(t, rt)
	e = m.finishEvaluationRationalFrame(frame)
	if !errors.Is(e, arena.ErrBounds) || !frame.constructed || !frame.stopped || frame.stage != 0 || !errors.Is(frame.sourceCause, arena.ErrBounds) {
		t.Fatal("actual postphysical source destructor frontier", e, frame)
	}
	polynomialEvaluationAudit(t, m, p, &v, &out)
	before := rt.State()
	cause := frame.sourceCause
	if e = m.finishEvaluationRationalFrame(frame); e != cause || rt.State() != before {
		t.Fatal("source retry acknowledged physical release", e)
	}
	contextTLSFinish(t, rt, m.CleanupEvaluation)
	polynomialEvaluationAudit(t, m, p, &v, &out)
	if frame.constructed || m.evaluation.active || frame.sourceCause != cause {
		t.Fatal("owning completion altered source history")
	}
	// A subsequent lexical default constructor may reuse the now fully drained
	// concrete frame object. It does not replay the old source destructor.
	serviceMust(t, m.EvaluateRational(p, &v, &out))
	if len(m.evaluation.rationalFrames) != 1 || m.evaluation.rationalFrames[0] != frame || frame.constructed || frame.stopped || m.evaluation.sourceCause != nil {
		t.Fatal("fresh public invocation and completed lexical reuse")
	}
	for i := range v.values {
		index := i
		contextTLSFinish(t, rt, func() error { return q.Del(&v.values[index]) })
	}
	contextTLSFinish(t, rt, func() error { return q.Del(&out) })
	e = m.core.DecRef(p)
	if e != nil {
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual later caller source drop", e)
		}
		// The intentionally endangered TLS profile also permits a later caller
		// destructor to stop. Audit its exact frontier, then own-abort that
		// caller token; never replay the source DEC through contextTLSFinish.
		polynomialEvaluationAudit(t, m, p, &v, &out)
		contextTLSFinish(t, rt, func() error { return m.core.CleanupValue(&p) })
	}
	contextTLSFinish(t, rt, m.CloseChecked)
}
