package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"strings"
	"testing"
)

type polynomialEvaluationResultWriter struct {
	m       *MultivariatePolynomialManager
	t       *testing.T
	out     bytes.Buffer
	results int
}

func (w *polynomialEvaluationResultWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	count := strings.Count(w.out.String(), ":6504 ")
	if count > w.results {
		w.results = count
		live := false
		for _, f := range w.m.evaluation.rationalFrames {
			live = live || f.constructed
		}
		for _, f := range w.m.evaluation.integerFrames {
			live = live || f.constructed
		}
		for _, f := range w.m.evaluation.intervalFrames {
			live = live || f.constructed
		}
		for _, f := range w.m.algebraicEvaluation.frames {
			live = live || f.constructed
		}
		if !live {
			w.t.Fatal("result record ran after auxiliary destruction")
		}
	}
	return n, err
}
func polynomialEvaluationResultOutput(t *testing.T, tags *diagnostic.TraceTags, w *polynomialEvaluationResultWriter, want string) {
	t.Helper()
	if !tags.ShouldTrace("eval_bug") {
		if w.out.Len() != 0 {
			t.Fatal("erased result TRACE emitted")
		}
		return
	}
	if w.results == 0 || !strings.Contains(w.out.String(), "result for start: 0, end: 1, x: 0\nr: "+want+"\n") {
		t.Fatal("source result formatter", w.out.String())
	}
}
func TestPolynomialEvaluationResultTraceConcreteFormatters(t *testing.T) {
	terms := []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}}
	t.Run("rational-fraction", func(t *testing.T) {
		m, p, _ := polynomialEvalFixture(t, terms)
		q := m.core.config.Rationals
		values := polynomialEvalRationalMap{manager: q}
		for i := range values.values {
			values.values[i] = unumeral.NewRational()
		}
		out := unumeral.NewRational()
		serviceMust(t, q.SetFraction(&values.values[0], 2, 3))
		w := &polynomialEvaluationResultWriter{m: m, t: t}
		tags := diagnostic.NewTraceTagsWithWriter(w)
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
		polynomialEvaluationResultOutput(t, tags, w, "2/3")
	})
	t.Run("integer", func(t *testing.T) {
		m, p, _ := polynomialEvalFixture(t, terms)
		var value, out unumeral.Int
		serviceMust(t, m.core.integer.SetString(&value, "-18446744073709551616"))
		w := &polynomialEvaluationResultWriter{m: m, t: t}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupEvaluation())
			serviceMust(t, m.core.integer.Del(&out))
			serviceMust(t, m.core.integer.Del(&value))
		}()
		serviceMust(t, m.UnivariateEvaluate(p, 0, &value, &out))
		polynomialEvaluationResultOutput(t, tags, w, "-18446744073709551616")
		if tags.ShouldTrace("eval_bug") && m.evaluation.trace.insertions[0].Character == nil {
			t.Fatal("actual BIG integer result insertion not captured")
		}
	})
	t.Run("interval-binary-fractions", func(t *testing.T) {
		svc, _ := serviceNew(t)
		q := svc.Manager
		m, p, _ := polynomialEvalFixtureWithRationals(t, q, terms)
		bounds, e := unumeral.NewBinaryRationalManager(q.Z, unumeral.BinaryRationalConfig{Diagnostics: svc})
		serviceMust(t, e)
		t.Cleanup(bounds.Close)
		vm, e := unumeral.NewBinaryIntervalManager(bounds)
		serviceMust(t, e)
		t.Cleanup(vm.Close)
		values := polynomialEvalIntervalMap{manager: vm}
		out := unumeral.NewBinaryInterval()
		lo, hi := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(3)
		serviceMust(t, bounds.SetInt32Power(&lo, 1, 1))
		serviceMust(t, bounds.SetInt32Power(&hi, 3, 1))
		serviceMust(t, vm.SetBounds(&values.value, &lo, &hi))
		w := &polynomialEvaluationResultWriter{m: m, t: t}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupEvaluation())
			serviceMust(t, vm.Del(&values.value))
			serviceMust(t, vm.Del(&out))
			serviceMust(t, bounds.Del(&lo))
			serviceMust(t, bounds.Del(&hi))
		}()
		serviceMust(t, m.EvaluateInterval(p, &values, &out))
		polynomialEvaluationResultOutput(t, tags, w, "(1/2, 3/2)")
	})
	t.Run("algebraic-decimal-default", func(t *testing.T) {
		vm, _, _ := algebraicControlledFixture(t)
		m, p, _ := polynomialEvalFixtureWithRationals(t, vm.config.Rationals, terms)
		values := algebraicSignTestValues{}
		q := unumeral.NewRational()
		var out Algebraic
		serviceMust(t, vm.config.Rationals.SetFraction(&q, 2, 3))
		serviceMust(t, vm.SetRational(&values.values[0], &q))
		w := &polynomialEvaluationResultWriter{m: m, t: t}
		tags := diagnostic.NewTraceTagsWithWriter(w)
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
		polynomialEvaluationResultOutput(t, tags, w, "0.6666666666?")
		if tags.ShouldTrace("eval_bug") && (!vm.matchesDecimalDisplayInvocation(&m.algebraicEvaluation.trace.decimal) || !vm.config.Rationals.MatchesDecimalInvocation(&vm.decimalDisplay.basic)) {
			t.Fatal("actual basic decimal invocation not captured")
		}
	})
	t.Run("algebraic-irrational-decimal", func(t *testing.T) {
		vm, _, _ := algebraicCompareFixture(t)
		m, p, _ := polynomialEvalFixtureWithRationals(t, vm.config.Rationals, terms)
		values := algebraicSignTestValues{}
		values.values[0] = algebraicCompareRoot(t, vm, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
		var out Algebraic
		w := &polynomialEvaluationResultWriter{m: m, t: t}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("eval_bug")
		m.core.monomials.config.Trace = tags
		defer func() {
			tags.Finalize()
			serviceMust(t, m.CleanupAlgebraicEvaluation())
			serviceMust(t, vm.Del(&out))
			for i := range values.values {
				serviceMust(t, vm.Del(&values.values[i]))
			}
		}()
		serviceMust(t, m.EvaluateAlgebraic(p, vm, &values, &out))
		polynomialEvaluationResultOutput(t, tags, w, "1.4142135623?")
	})
}

type polynomialEvaluationDecimalFaultWriter struct {
	m     *MultivariatePolynomialManager
	t     *testing.T
	fault int64
	text  bytes.Buffer
	armed bool
}

func (w *polynomialEvaluationDecimalFaultWriter) Write(p []byte) (int, error) {
	n, e := w.text.Write(p)
	if !w.armed && strings.Contains(w.text.String(), ":6504 ") {
		w.armed = true
		w.m.originalBudget.FailAfter(w.fault)
	}
	return n, e
}
func TestPolynomialEvaluationResultTraceActualDecimalReservationPrefixes(t *testing.T) {
	probe := diagnostic.NewTraceTagsWithWriter(&bytes.Buffer{})
	probe.Enable("eval_bug")
	enabled := probe.ShouldTrace("eval_bug")
	probe.Finalize()
	if !enabled {
		t.Skip("TRACE compiled out")
	}
	success := false
	pending := 0
	compiled := false
	for fault := int64(0); fault < 128 && !success; fault++ {
		t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			vm, _, _ := algebraicControlledFixture(t)
			m, p, _ := polynomialEvalFixtureWithRationals(t, vm.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
			values := algebraicSignTestValues{}
			q := unumeral.NewRational()
			var out Algebraic
			serviceMust(t, vm.config.Rationals.Z.SetString(q.BorrowNumerator(), "18446744073709551616"))
			serviceMust(t, vm.SetRational(&values.values[0], &q))
			writer := &polynomialEvaluationDecimalFaultWriter{m: m, t: t, fault: fault}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("eval_bug")
			m.core.monomials.config.Trace = tags
			defer func() {
				tags.Finalize()
				m.originalBudget.FailAfter(-1)
				serviceMust(t, m.CleanupAlgebraicEvaluation())
				serviceMust(t, vm.Del(&out))
				for i := range values.values {
					serviceMust(t, vm.Del(&values.values[i]))
				}
				serviceMust(t, vm.config.Rationals.Del(&q))
			}()
			if !tags.ShouldTrace("eval_bug") {
				t.Skip("TRACE compiled out")
			}
			compiled = true
			body := m.EvaluateAlgebraic(p, vm, &values, &out)
			m.originalBudget.FailAfter(-1)
			if !writer.armed {
				t.Fatal("real result insertion not reached")
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual decimal result refusal", body)
			}
			receipt := m.algebraicEvaluation.trace.decimal
			if vm.decimalDisplayInvocationPending(&receipt) {
				pending++
				if !m.algebraicEvaluation.trace.active || !vm.config.Rationals.DecimalInvocationPending(&vm.decimalDisplay.basic) {
					t.Fatal("actual basic driver not retained by evaluator")
				}
				copied := m.algebraicEvaluation.trace
				serviceMust(t, m.cleanupEvaluationTrace(&copied))
				if !vm.decimalDisplayInvocationPending(&receipt) {
					t.Fatal("copied evaluator metadata acquired decimal child")
				}
			}
			if body == nil {
				success = true
				if !strings.Contains(writer.text.String(), "r: 18446744073709551616\n") {
					t.Fatal("independent decimal result", writer.text.String())
				}
			}
			serviceMust(t, m.CleanupAlgebraicEvaluation())
			if vm.decimalDisplayInvocationPending(&receipt) {
				t.Fatal("owning evaluator cleanup retained actual decimal")
			}
		})
	}
	if compiled && !success {
		t.Fatal("actual decimal formatter never reached success")
	}
	// Compiled-out profiles skip every subcase; the combined profile must enter
	// a real retained driver, rather than merely reject the incoming call.
	if success && pending == 0 {
		t.Fatal("sweep never captured an actual decimal refusal")
	}
}

func TestPolynomialEvaluationResultTraceSkippedAfterBodyError(t *testing.T) {
	m, p, control := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	control.limit = 0
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
	body := m.UnivariateEvaluate(p, 0, &value, &out)
	if !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("actual source checkpoint did not cancel", body)
	}
	if strings.Contains(text.String(), ":6504 ") || strings.Contains(text.String(), "result for start:") {
		t.Fatal("result record ran after failed body", text.String())
	}
}
