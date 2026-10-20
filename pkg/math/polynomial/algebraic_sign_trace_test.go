package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"strings"
	"testing"
)

func TestAlgebraicEvalSignTraceRationalAndExactPhases(t *testing.T) {
	for _, exact := range []bool{false, true} {
		t.Run(map[bool]string{false: "rational", true: "exact"}[exact], func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			c.minMagnitude = 0
			terms := []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}}
			if exact {
				terms = []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}, {-3, [3]uint32{}}}
			}
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, terms)
			values := algebraicSignTestValues{}
			if exact {
				values.values[0], values.values[1] = algebraicResultantInputs(t, c, up)
			} else {
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
				serviceMust(t, c.SetRational(&values.values[0], &q))
				serviceMust(t, c.config.Rationals.Del(&q))
			}
			var output strings.Builder
			observer := &signPhaseTraceWriter{t: t, c: c, output: &output}
			tags := diagnostic.NewTraceTagsWithWriter(observer)
			tags.Enable("anum_eval_sign")
			defer func() {
				serviceMust(t, c.config.Rationals.BindTrace(nil))
				serviceMust(t, c.CleanupRetained())
				for i := range values.values {
					serviceMust(t, c.Del(&values.values[i]))
				}
			}()
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			sign, e := c.EvalSign(pm, p, &values)
			serviceMust(t, e)
			want := 5
			if exact {
				want = 1
			}
			if sign != want {
				t.Fatal("value", sign, want)
			}
			if tags.ShouldTrace("anum_eval_sign") {
				if strings.Count(output.String(), ":1981 ") != 1 {
					t.Fatal("outer entry repeated", output.String())
				}
				if exact && (!observer.resultant || !observer.exactInterval) {
					t.Fatal("source declaration frontiers not reached")
				}
				markers := []string{"evaluating sign of: ", "all variables are assigned to rationals, value of p: 5/2\n"}
				if exact {
					markers = []string{"p after eliminating rationals: ", "evaluating using intervals: ", "R: ", "\nk: ", "\nri: ", "zero lower bound is: ", "refined algebraic interval\n"}
				}
				for _, marker := range markers {
					if !strings.Contains(output.String(), marker) {
						t.Fatalf("missing %q in %s", marker, output.String())
					}
				}
			} else if output.Len() != 0 {
				t.Fatal("erased diagnostic executed")
			}
		})
	}
}

func TestAlgebraicEvalSignTraceBorrowBeforeLocals(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	w := &linearOperationProbeWriter{probe: func() {
		if c.rationalSign.active || c.signSubstitution.active || c.signInterval.active || c.signResultant.active {
			t.Fatal("entry constructed later locals")
		}
		for _, action := range []func() error{c.Close, c.CleanupRetained, func() error { _, e := c.EvalSign(pm, p, &values); return e }} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("executing trace owner", e)
			}
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("anum_eval_sign")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	sign, e := c.EvalSign(pm, p, &values)
	serviceMust(t, e)
	if sign != 1 {
		t.Fatal(sign)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
}

func TestAlgebraicEvalSignTraceActualEntryTLSRefusal(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	c, _, rt := algebraicNegTLSFixture(t, nil)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, nil)
	var big unumeral.Int
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, c.config.Rationals.Integers().SetLimbs(&big, digits))
	p, e := pm.core.MkConstantTake(&big)
	serviceMust(t, e)
	p = polynomialCaller(t, pm.core, p)
	tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
	tags.Enable("anum_eval_sign")
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, rt.SetMaxSize(1))
	values := algebraicSignTestValues{}
	_, body := c.EvalSign(pm, p, &values)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !c.rootTracePending() || c.rationalSign.active || c.signSubstitution.active || c.signInterval.active || c.signResultant.active {
		t.Fatal("entry refused after later locals", body)
	}
	roots, e := c.config.Rationals.Integers().Allocator.PendingRoots()
	serviceMust(t, e)
	if roots == ([3]umemory.Block{}) || roots != c.rootTrace.poly.frame.numeric.Allocator {
		t.Fatal("entry exact numeric child", roots)
	}
	if _, e = c.IsZero(&values.values[0]); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("pending display allowed new public work", e)
	}
	serviceMust(t, c.CleanupRetained())
	roots, e = c.config.Rationals.Integers().Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || c.rootTracePending() {
		t.Fatal("entry formatter retained", roots)
	}
}

type signResultBudgetWriter struct {
	pm *MultivariatePolynomialManager
	strings.Builder
	t      *testing.T
	c      *AlgebraicRootCore
	prefix int64
	armed  bool
}

func (w *signResultBudgetWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if !w.armed && strings.Contains(w.String(), ":1989 ") {
		if !w.c.rationalSign.active || !w.c.rationalSign.valueOwned {
			w.t.Fatal("result formatted after local destruction")
		}
		if w.pm != nil {
			if e := w.pm.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
				w.t.Fatal("result trace released borrowed polynomial manager", e)
			}
		}
		w.armed = true
		w.c.config.Rationals.Budget().FailAfter(w.prefix)
	}
	return n, e
}
func TestAlgebraicEvalSignTraceResultNativeScratchAndLifetime(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619/7"))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	w := &signResultBudgetWriter{t: t, c: c, pm: pm, prefix: 0}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("anum_eval_sign")
	defer func() {
		c.config.Rationals.Budget().FailAfter(-1)
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&values.values[0]))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	sign, body := c.EvalSign(pm, p, &values)
	c.config.Rationals.Budget().FailAfter(-1)
	serviceMust(t, body)
	if !w.armed || sign != 1 || !strings.Contains(w.String(), "all variables are assigned to rationals, value of p: 18446744073709551619/7\n") {
		t.Fatal("result output or source lifetime", sign, w.String())
	}
	if c.rootTracePending() || c.rationalSign.active {
		t.Fatal("result owner retained")
	}
}

func TestAlgebraicEvalSignTraceRefusingWriter(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	c.minMagnitude = 0
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-2, [3]uint32{}}})
	values := algebraicSignTestValues{}
	values.values[0], values.values[1] = algebraicResultantInputs(t, c, up)
	w := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("anum_eval_sign")
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&values.values[1]))
		serviceMust(t, c.Del(&values.values[0]))
	}()
	before := pm.core.display.next
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	sign, e := c.EvalSign(pm, p, &values)
	serviceMust(t, e)
	if sign != 0 {
		t.Fatal("writer failure changed exact value", sign)
	}
	if tags.ShouldTrace("anum_eval_sign") && (w.calls != 1 || pm.core.display.next <= before) {
		t.Fatal("sticky writer suppressed project formatter", w.calls)
	}
	if c.rootTracePending() || c.signResultant.active || c.signInterval.active || c.signSubstitution.active {
		t.Fatal("completed source locals retained")
	}
}

// Observe actual live source locals when each complete record header arrives.
type signPhaseTraceWriter struct {
	t                        *testing.T
	c                        *AlgebraicRootCore
	output                   *strings.Builder
	resultant, exactInterval bool
}

func (w *signPhaseTraceWriter) Write(p []byte) (int, error) {
	n, e := w.output.Write(p)
	text := w.output.String()
	if !w.resultant && strings.Contains(text, ":2135 ") {
		w.resultant = true
		if !w.c.signResultant.active || w.c.signResultant.limitsConstructed != 0 || !w.c.signInterval.constructed {
			w.t.Fatal("resultant record crossed source bound declaration")
		}
	}
	if !w.exactInterval && strings.Contains(text, ":2163 ") {
		w.exactInterval = true
		if !w.resultant || w.c.signResultant.limitsConstructed != 2 || !w.c.signInterval.constructed {
			w.t.Fatal("exact interval record lost live bounds")
		}
	}
	return n, e
}

func TestAlgebraicEvalSignTraceRationalRestart(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	c.minMagnitude = -100
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	values.values[0] = algebraicCompareRoot(t, c, []int32{-1, 0, 1}, 0, 0, 2, 0, false)
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("anum_eval_sign")
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&values.values[0]))
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	sign, e := c.EvalSign(pm, p, &values)
	serviceMust(t, e)
	if sign != 0 {
		t.Fatal("rational restart value", sign)
	}
	if tags.ShouldTrace("anum_eval_sign") && (strings.Count(output.String(), ":1981 ") != 1 || strings.Count(output.String(), ":2057 ") != 1 || !strings.Contains(output.String(), "all variables are assigned to rationals, value of p: 0\n")) {
		t.Fatal("restart record or outer-entry lifetime", output.String())
	}
}
