package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"strings"
	"testing"
)

func polynomialEvaluationTraceOutput(t *testing.T, m *MultivariatePolynomialManager, out *bytes.Buffer, tags *diagnostic.TraceTags, root string, algebraic bool) {
	t.Helper()
	if !tags.ShouldTrace("eval_bug") {
		if out.Len() != 0 {
			t.Fatal("erased evaluation TRACE emitted")
		}
		return
	}
	if !strings.Contains(out.String(), ":6438 ") || !strings.Contains(out.String(), "p: ") || !strings.Contains(out.String(), root) {
		t.Fatal("source evaluation entry record", out.String())
	}
	s := &m.evaluation.trace
	if algebraic {
		s = &m.algebraicEvaluation.trace
	}
	if !m.evaluationTraceActual(s) || s.display.frame == nil || s.active {
		t.Fatal("actual completed evaluation diagnostic not retained")
	}
}
func TestPolynomialEvaluationTraceConcreteConsumers(t *testing.T) {
	terms := []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {3, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}}
	t.Run("rational-recursive", func(t *testing.T) {
		m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{2, 1, 0}}, {3, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
		q := m.core.config.Rationals
		values := polynomialEvalRationalMap{manager: q}
		out := unumeral.NewRational()
		for i := range values.values {
			values.values[i] = unumeral.NewRational()
			serviceMust(t, q.SetInt32(&values.values[i], int32(i+2)))
		}
		var text bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&text)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupEvaluation())
			serviceMust(t, q.Del(&out))
			for i := range values.values {
				serviceMust(t, q.Del(&values.values[i]))
			}
		}()
		serviceMust(t, m.EvaluateRational(p, &values, &out))
		if *out.BorrowNumerator() != unumeral.NewInt(19) || *out.BorrowDenominator() != unumeral.NewInt(1) {
			t.Fatal("recursive rational evaluation", out)
		}
		polynomialEvaluationTraceOutput(t, m, &text, tags, "start: 0, end: 3, x: 1\n", false)
		if tags.ShouldTrace("eval_bug") && m.evaluation.trace.invocation < 2 {
			t.Fatal("recursive entries not reached")
		}
	})
	t.Run("integer", func(t *testing.T) {
		m, p, _ := polynomialEvalFixture(t, terms)
		value := unumeral.NewInt(2)
		var out unumeral.Int
		var text bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&text)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupEvaluation())
			serviceMust(t, m.core.integer.Del(&out))
		}()
		serviceMust(t, m.UnivariateEvaluate(p, 0, &value, &out))
		if out != unumeral.NewInt(11) {
			t.Fatal("integer evaluation", out)
		}
		polynomialEvaluationTraceOutput(t, m, &text, tags, "start: 0, end: 3, x: 0\n", false)
	})
	t.Run("interval", func(t *testing.T) {
		service, _ := serviceNew(t)
		q := service.Manager
		m, p, _ := polynomialEvalFixtureWithRationals(t, q, terms)
		bounds, e := unumeral.NewBinaryRationalManager(q.Z, unumeral.BinaryRationalConfig{Diagnostics: service})
		serviceMust(t, e)
		t.Cleanup(bounds.Close)
		vm, e := unumeral.NewBinaryIntervalManager(bounds)
		serviceMust(t, e)
		t.Cleanup(vm.Close)
		values := polynomialEvalIntervalMap{manager: vm}
		intervalSet(t, vm, &values.value, 2, 2)
		out := unumeral.NewBinaryInterval()
		var text bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&text)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupEvaluation())
			serviceMust(t, vm.Del(&values.value))
			serviceMust(t, vm.Del(&out))
		}()
		serviceMust(t, m.EvaluateInterval(p, &values, &out))
		intervalText(t, vm, &out, "(11, 11)")
		polynomialEvaluationTraceOutput(t, m, &text, tags, "start: 0, end: 3, x: 0\n", false)
	})
	t.Run("algebraic", func(t *testing.T) {
		vm, _, _ := algebraicControlledFixture(t)
		m, p, _ := polynomialEvalFixtureWithRationals(t, vm.config.Rationals, terms)
		values := algebraicSignTestValues{}
		q := unumeral.NewRational()
		var out Algebraic
		serviceMust(t, vm.config.Rationals.SetInt32(&q, 2))
		serviceMust(t, vm.SetRational(&values.values[0], &q))
		var text bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&text)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupAlgebraicEvaluation())
			serviceMust(t, vm.Del(&out))
			for i := range values.values {
				serviceMust(t, vm.Del(&values.values[i]))
			}
			serviceMust(t, vm.config.Rationals.Del(&q))
		}()
		serviceMust(t, m.EvaluateAlgebraic(p, vm, &values, &out))
		serviceMust(t, vm.config.Rationals.SetInt32(&q, 11))
		cmp, e := vm.CompareRational(&out, &q)
		serviceMust(t, e)
		if cmp != 0 {
			t.Fatal("algebraic evaluation", cmp)
		}
		polynomialEvaluationTraceOutput(t, m, &text, tags, "start: 0, end: 3, x: 0\n", true)
	})
}

type polynomialEvaluationTraceWriter struct {
	m     *MultivariatePolynomialManager
	t     *testing.T
	calls int
}

func (w *polynomialEvaluationTraceWriter) Write(p []byte) (int, error) {
	w.calls++
	for _, e := range []error{w.m.CloseChecked(), w.m.SourceFinalize(), w.m.core.CloseChecked(), w.m.core.SourceFinalizePrefix(), w.m.core.CleanupDisplay(), w.m.CleanupEvaluation(), w.m.CleanupAlgebraicEvaluation(), w.m.CleanupGCD(), w.m.CleanupFactor()} {
		if !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatal("evaluation callback acquired owner", e)
		}
	}
	if e := w.m.core.integer.CloseChecked(); !errors.Is(e, unumeral.ErrNumericPending) {
		w.t.Fatal("evaluation callback closed integer", e)
	}
	if e := w.m.core.integer.Natural.CloseChecked(); !errors.Is(e, arena.ErrInvalidSlot) {
		w.t.Fatal("evaluation callback closed natural", e)
	}
	return 0, errors.New("ostream refused")
}
func TestPolynomialEvaluationTraceFailedStreamAndOwnerBorrow(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {1, [3]uint32{}}})
	value := unumeral.NewInt(2)
	var out unumeral.Int
	writer := &polynomialEvaluationTraceWriter{m: m, t: t}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("eval_bug")
	m.core.monomials.config.Trace = tags
	defer func() {
		tags.Finalize()
		serviceMust(t, m.CleanupEvaluation())
		serviceMust(t, m.core.integer.Del(&out))
	}()
	serviceMust(t, m.UnivariateEvaluate(p, 0, &value, &out))
	if out != unumeral.NewInt(5) {
		t.Fatal("refused stream changed evaluation", out)
	}
	if tags.ShouldTrace("eval_bug") && (writer.calls == 0 || m.evaluation.trace.display.frame == nil) {
		t.Fatal("refused writer suppressed actual polynomial formatting")
	}
}

func TestPolynomialEvaluationTraceEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 512 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{2, 1, 0}}, {3, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
			q := m.core.config.Rationals
			values := polynomialEvalRationalMap{manager: q}
			out := unumeral.NewRational()
			for i := range values.values {
				values.values[i] = unumeral.NewRational()
				serviceMust(t, q.SetInt32(&values.values[i], int32(i+2)))
			}
			var text bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			tags.Enable("eval_bug")
			m.core.monomials.config.Trace = tags
			defer func() {
				tags.Finalize()
				m.originalBudget.FailAfter(-1)
				serviceMust(t, m.CleanupEvaluation())
				serviceMust(t, q.Del(&out))
				for i := range values.values {
					serviceMust(t, q.Del(&values.values[i]))
				}
			}()
			m.originalBudget.FailAfter(ordinal)
			cause := m.EvaluateRational(p, &values, &out)
			m.originalBudget.FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual diagnostic/evaluation reservation", cause)
			}
			if cause == nil {
				success = true
				if *out.BorrowNumerator() != unumeral.NewInt(19) || *out.BorrowDenominator() != unumeral.NewInt(1) {
					t.Fatal("independent recursive result", out)
				}
				polynomialEvaluationTraceOutput(t, m, &text, tags, "start: 0, end: 3, x: 1\n", false)
			}
		})
	}
	if !success {
		t.Fatal("actual evaluation trace never reached success")
	}
}
