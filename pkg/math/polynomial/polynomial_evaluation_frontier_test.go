// Source: src/math/polynomial/polynomial.cpp:6436-6522;
// src/util/scoped_numeral.h; src/util/memory_manager.cpp.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialEvaluationLiveCallerClosePreservesCachedFrames(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{2, 1, 0}}, {3, [3]uint32{1, 0, 0}}})
	q := m.core.config.Rationals
	v := polynomialEvalRationalMap{manager: q}
	for i := range v.values {
		v.values[i] = util.NewRational()
		serviceMust(t, q.SetInt32(&v.values[i], 2))
	}
	out := util.NewRational()
	defer func() {
		serviceMust(t, m.CleanupEvaluation())
		for i := range v.values {
			serviceMust(t, q.Del(&v.values[i]))
		}
		serviceMust(t, q.Del(&out))
	}()
	serviceMust(t, m.EvaluateRational(p, &v, &out))
	before := m.originalBudget.Used()
	frames := append([]*polynomialEvalRationalFrame(nil), m.evaluation.rationalFrames...)
	if len(frames) < 2 {
		t.Fatal("actual recursive cached frames")
	}
	e := m.CloseChecked()
	if !errors.Is(e, ErrPolynomialLive) {
		t.Fatal("actual caller preguard", e)
	}
	if m.originalBudget.Used() != before || len(m.evaluation.rationalFrames) != len(frames) {
		t.Fatal("rejected close mutated cached owned graph")
	}
	for i, f := range frames {
		if m.evaluation.rationalFrames[i] != f {
			t.Fatal("rejected close replaced actual frame")
		}
	}
	polynomialEvaluationAudit(t, m, p, &v, &out)
}

// These actual nested lexical owner primitives use the evaluator's exact
// constructor/source-dtor methods and real Runtime frees. The bounded public
// counter recipe selects the guarded signed-host domain, not ordinary OOM.
func TestPolynomialEvaluationYoungerSourceStopPreservesOlderAuxiliary(t *testing.T) {
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
	defer func() {
		if !m.closed {
			contextTLSFinish(t, rt, m.CleanupEvaluation)
			contextTLSFinish(t, rt, func() error { return m.core.CleanupValue(&p) })
			contextTLSFinish(t, rt, m.CloseChecked)
		}
	}()
	m.evaluation.active = true
	m.evaluation.rational = q
	older, e := m.evaluationRationalFrame()
	serviceMust(t, e)
	younger, e := m.evaluationRationalFrame()
	serviceMust(t, e)
	serviceMust(t, q.Integers().SetLimbs(&(*older.aux.BorrowNumerator()), []uint32{3, 1}))
	serviceMust(t, q.Integers().SetLimbs(&(*younger.aux.BorrowNumerator()), []uint32{1, 1}))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(&(*younger.aux.BorrowNumerator()), 40000))
	polynomialEvaluationAudit(t, m, p, &v, &out)
	term, e := m.core.monomials.MkVariable(2, 1)
	serviceMust(t, e)
	warm, e := m.core.MkTerms([]util.Int{util.NewInt(3)}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(warm))
	serviceMust(t, m.core.DecRef(warm))
	contextTLSNearMinimum(t, rt)
	olderBefore := older.aux
	var stoppedState util.RuntimeState
	cause := func() (cause error) {
		defer func() {
			e := m.finishEvaluationRationalFrame(older)
			m.evaluation.depth--
			cause = errors.Join(cause, e)
		}()
		cause = m.finishEvaluationRationalFrame(younger)
		m.evaluation.depth--
		stoppedState = rt.State()
		return
	}()
	if !errors.Is(cause, arena.ErrBounds) || !younger.stopped || m.evaluation.sourceCause != younger.sourceCause {
		t.Fatal("actual first lexical physical stop", cause)
	}
	if !older.constructed || older.stage != 0 || older.aux != olderBefore || rt.State() != stoppedState {
		t.Fatal("older lexical destructor ran after younger source stop")
	}
	polynomialEvaluationAudit(t, m, p, &v, &out)
	first := m.evaluation.sourceCause
	if e = m.finishEvaluationRationalFrame(older); e != first || older.aux != olderBefore || rt.State() != stoppedState {
		t.Fatal("source retry must preserve older frontier", e)
	}
	contextTLSFinish(t, rt, m.CleanupEvaluation)
	polynomialEvaluationAudit(t, m, p, &v, &out)
	e = m.core.DecRef(p)
	if e != nil {
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		polynomialEvaluationAudit(t, m, p, &v, &out)
		contextTLSFinish(t, rt, func() error { return m.core.CleanupValue(&p) })
	}
	contextTLSFinish(t, rt, m.CloseChecked)
}
