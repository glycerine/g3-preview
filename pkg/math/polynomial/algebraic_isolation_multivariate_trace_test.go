package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func TestAlgebraicMultivariateIsolationTraceResultantFrontier(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "records", true: "sticky-stream"}[fail], func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}})
			values := algebraicIsolationAssignmentsFixture(t, c, up, false)
			roots, e := c.NewVector()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, c.CleanupRetained()); serviceMust(t, roots.Close()) })
			var output strings.Builder
			probes := 0
			writer := &linearOperationProbeWriter{probe: func() {
				probes++
				for _, action := range []func() error{c.Close, c.CleanupRetained, roots.Close} {
					if e := action(); !errors.Is(e, ErrAlgebraicPending) {
						t.Error("trace owner callback", e)
					}
				}
			}}
			// Keep the probe and literal stream together through the existing source writer.
			joined := &algebraicMultivariateTraceWriter{output: &output, probe: writer}
			broken := &linearTraceFailWriter{}
			tags := diagnostic.NewTraceTagsWithWriter(joined)
			if fail {
				tags = diagnostic.NewTraceTagsWithWriter(broken)
			}
			tags.Enable("isolate_roots")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			serviceMust(t, c.IsolateMultivariateRoots(pm, p, values, roots))
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if len(roots.values) != 1 {
				t.Fatal("linear root count", len(roots.values))
			}
			cmp, e := c.Compare(&roots.values[0], &values.values[0])
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("independent selected root", cmp)
			}
			if tags.ShouldTrace("isolate_roots") {
				if fail {
					if broken.calls != 1 {
						t.Fatal("sticky writer retried", broken.calls)
					}
				} else {
					records := []string{
						":2288 ---------\nisolating roots of: x2 - x0\n",
						":2306 ---------\np after applying (rational fragment of) x2v:\nx2 - x0\n",
						":2332 ---------\nthere are 1 variables assigned to nonbasic numbers...\n",
						":2360 ---------\nresultant loop i: 0, y: x0\np_y: x0^2 - 2\nq: x2^2 - 2\n",
					}
					previous := -1
					for _, record := range records {
						at := strings.Index(output.String(), record)
						if at < 0 || at <= previous {
							t.Fatalf("missing or reordered source record %q:\n%s", record, output.String())
						}
						previous = at
					}
					if probes == 0 {
						t.Fatal("source writer callback missing")
					}
				}
			} else if output.Len() != 0 || probes != 0 || broken.calls != 0 {
				t.Fatal("erased trace evaluated writer")
			}
		})
	}
}

type algebraicMultivariateTraceWriter struct {
	output *strings.Builder
	probe  *linearOperationProbeWriter
}

func (w *algebraicMultivariateTraceWriter) Write(p []byte) (int, error) {
	if _, e := w.probe.Write(p); e != nil {
		return 0, e
	}
	return w.output.Write(p)
}

func TestAlgebraicMultivariateIsolationTraceDisabledSkipsOperands(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	serviceMust(t, c.traceMultivariateIsolationPolynomial(nil, Polynomial{}, 2288, "unused"))
	serviceMust(t, c.traceMultivariateIsolationResultant(nil, 0, 0, Polynomial{}, Polynomial{}))
}

type algebraicMultivariateTraceStop struct {
	header strings.Builder
	base   algebraicRootTraceNumeralStop
	armed  bool
}

func (w *algebraicMultivariateTraceStop) Write(p []byte) (int, error) {
	if !w.armed {
		_, _ = w.header.Write(p)
	}
	if strings.Contains(w.header.String(), ":2306 ---------") {
		w.armed = true
	}
	if w.armed {
		return w.base.Write(p)
	}
	return len(p), nil
}
func TestAlgebraicMultivariateIsolationSubstitutionTraceOwnsStoppedDisplay(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, nil)
	z := c.config.Rationals.Integers()
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&n, "18446744073709551619"))
	x2, e := pm.core.monomials.MkMonomial([]MonomialPower{{Var: 2, Degree: 1}})
	serviceMust(t, e)
	x0, e := pm.core.monomials.MkMonomial([]MonomialPower{{Var: 0, Degree: 1}})
	serviceMust(t, e)
	p, e := pm.core.MkTerms([]unumeral.Int{n, unumeral.NewInt(1)}, []Monomial{x2, x0})
	serviceMust(t, e)
	p = polynomialCaller(t, pm.core, p)
	serviceMust(t, z.Del(&n))
	values := &algebraicIsolationAssignments{assigned: [3]bool{true, false, false}}
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&values.values[0], &one))
	t.Cleanup(func() { serviceMust(t, c.Del(&values.values[0])) })
	roots, e := c.NewVector()
	serviceMust(t, e)
	output := &algebraicMultivariateTraceStop{base: algebraicRootTraceNumeralStop{core: c}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("isolate_roots")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.IsolateMultivariateRoots(pm, p, values, roots)
	var stopped *polynomialDisplayFrame
	var sourceStop error
	if tags.ShouldTrace("isolate_roots") {
		if !errors.Is(body, arena.ErrBounds) || output.base.ref == 0 || !c.rootTracePending() || len(c.multivariateIsolationFrames) != 1 {
			t.Fatal("actual display guard not reached", body)
		}
		s := c.multivariateIsolationFrames[0]
		if !s.active || s.depth != 1 || s.substitution.prime.root.value == (Polynomial{}) || pm.algebraicIsolationBorrowers != 1 {
			t.Fatal("older substitution owner destroyed before display")
		}
		stopped = c.rootTrace.poly.frame
		sourceStop = stopped.sourceCause
		z.Store(output.base.ref, output.base.original)
	} else if body != nil || output.base.ref != 0 {
		t.Fatal("erased trace changed isolation", body)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, roots.Close())
	if c.rootTracePending() || pm.algebraicIsolationBorrowers != 0 || stopped != nil && (stopped.active || stopped.sourceCause != sourceStop) {
		t.Fatal("owning cleanup changed source history or retained borrow")
	}
}

func TestAlgebraicMultivariateIsolationTraceBranchRecords(t *testing.T) {
	for _, tc := range []struct {
		name   string
		terms  []polynomialEvalTerm
		basic  bool
		n      int32
		same   bool
		count  int
		record string
	}{
		{"zero", nil, false, 0, false, 0, ":2292 ---------\np is zero or the constant polynomial\n"},
		{"univariate", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-2, [3]uint32{}}}, false, 0, false, 2, ":2297 ---------\np is univariate, using univariate procedure\n"},
		{"rational-zero", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}}, true, 0, false, 0, ":2309 ---------\np is zero or the constant polynomial after applying (rational fragment of) x2v\n"},
		{"rational-univariate", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {-6, [3]uint32{}}}, true, 2, false, 1, ":2320 ---------\np is univariate after applying (rational fragment of) x2v... using univariate procedure\n"},
		{"constant-resultant", []polynomialEvalTerm{{1, [3]uint32{2, 0, 1}}, {-2, [3]uint32{0, 0, 1}}, {1, [3]uint32{}}}, false, 0, false, 0, ":2443 ---------\nq is the constant polynomial, so p does not contain any roots at x2v\n"},
		{"vanished-linear", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {1, [3]uint32{0, 1, 1}}, {-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 1, 0}}}, false, 0, true, 1, ":2380 ---------\np is linear after applying (rational fragment) of x2v\n"},
		{"vanished-linear-coefficient", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {-1, [3]uint32{0, 1, 1}}, {1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}}, false, 0, true, 0, ":2391 ---------\ncoefficient of degree 1 vanished, so p does not have roots at x2v\n"},
		{"vanished-all-coefficients", []polynomialEvalTerm{{1, [3]uint32{1, 0, 2}}, {-1, [3]uint32{0, 1, 2}}, {1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}}, false, 0, true, 0, ":2413 ---------\nall coefficients vanished... polynomial does not have roots\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
			values := algebraicIsolationAssignmentsFixture(t, c, up, tc.same)
			if tc.basic {
				n := unumeral.NewInt(tc.n)
				serviceMust(t, c.SetInteger(&values.values[0], &n))
			}
			roots, e := c.NewVector()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, c.CleanupRetained()); serviceMust(t, roots.Close()) })
			var output strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("isolate_roots")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			serviceMust(t, c.IsolateMultivariateRoots(pm, p, values, roots))
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if len(roots.values) != tc.count {
				t.Fatal("independent branch root count", len(roots.values), tc.count)
			}
			if tags.ShouldTrace("isolate_roots") {
				if !strings.Contains(output.String(), tc.record) {
					t.Fatalf("missing source branch record %q:\n%s", tc.record, output.String())
				}
				if strings.HasPrefix(tc.name, "vanished-") && !strings.Contains(output.String(), ":2369 ---------\nq vanished\n") {
					t.Fatal("missing vanished-resultant record")
				}
			} else if output.Len() != 0 {
				t.Fatal("erased trace emitted branch")
			}
		})
	}
}
