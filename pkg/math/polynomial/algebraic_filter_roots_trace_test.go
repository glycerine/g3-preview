package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strconv"
	"strings"
	"testing"
)

func TestAlgebraicFilterRootsTraceSourceOrderAndPublication(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(strconv.FormatBool(fail), func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}})
			a, b := algebraicResultantInputs(t, c, up)
			serviceMust(t, c.Del(&b))
			values := &algebraicSignTestValues{}
			values.values[1] = a
			candidates, e := c.NewVector()
			serviceMust(t, e)
			poly := algebraicPolynomial(t, up, -2, 0, 1)
			view, e := up.wholeView(poly)
			serviceMust(t, e)
			serviceMust(t, c.IsolateRoots(view, candidates))
			t.Cleanup(func() {
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, candidates.Close())
				serviceMust(t, c.Del(&values.values[1]))
			})
			var output strings.Builder
			probes := 0
			probe := &linearOperationProbeWriter{probe: func() {
				probes++
				if _, e := up.NewPolynomial(); !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Error("public UP construction crossed trace borrow", e)
				}
				before := poly
				if e := up.DelPolynomial(&poly); !errors.Is(e, ErrUPolynomialCleanupPending) || poly != before {
					t.Error("public UP deletion crossed trace borrow", e)
				}
				if !c.filterRootsRunning {
					t.Error("filter diagnostic escaped executing caller")
				}
				for _, action := range []func() error{c.Close, c.CleanupRetained, candidates.Close} {
					if e := action(); !errors.Is(e, ErrAlgebraicPending) {
						t.Error("filter trace callback", e)
					}
				}
			}}
			writer := &algebraicMultivariateTraceWriter{output: &output, probe: probe}
			broken := &linearTraceFailWriter{}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			if fail {
				tags = diagnostic.NewTraceTagsWithWriter(broken)
			}
			tags.Enable("isolate_roots")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			serviceMust(t, c.filterRoots(pm, p, values, 0, candidates))
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if len(candidates.values) != 1 {
				t.Fatal("spurious candidate retained")
			}
			cmp, e := c.Compare(&candidates.values[0], &values.values[1])
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("independent filter result", cmp)
			}
			if tags.ShouldTrace("isolate_roots") {
				if fail {
					if broken.calls != 1 {
						t.Fatal("sticky stream retried", broken.calls)
					}
				} else {
					records := []string{
						":2235 ---------\nbefore filtering roots, x: x0\n(#^2 - 2, 1)\n(#^2 - 2, 2)\n",
						":2248 ---------\nfilter_roots i: 0, ext_x2v: x0 -> (#^2 - 2, 1)\n",
						":2250 ---------\nfilter_roots i: 0, result sign: -1\n",
						":2248 ---------\nfilter_roots i: 1, ext_x2v: x0 -> (#^2 - 2, 2)\n",
						":2250 ---------\nfilter_roots i: 1, result sign: 0\n",
						":2262 ---------\nafter filtering roots:\n(#^2 - 2, 2)\n",
					}
					last := -1
					for _, record := range records {
						at := strings.Index(output.String(), record)
						if at < 0 || at <= last {
							t.Fatalf("missing/reordered source filter record %q:\n%s", record, output.String())
						}
						last = at
					}
					if probes == 0 {
						t.Fatal("missing writer callback")
					}
				}
			} else if output.Len() != 0 || broken.calls != 0 || probes != 0 {
				t.Fatal("erased trace evaluated writer")
			}
		})
	}
}
func TestAlgebraicFilterTraceDisabledSkipsOperands(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	serviceMust(t, c.traceFilterRoots(nil, 0, false))
	serviceMust(t, c.traceFilterRoots(nil, 0, true))
	serviceMust(t, c.traceFilterRootCandidate(0, 0, nil))
	serviceMust(t, c.traceMultivariateIsolationLinearRoot(nil))
	serviceMust(t, c.traceMultivariateIsolationRecovery(nil, Polynomial{}, 0, nil))
}

func TestAlgebraicFilterRootTraceRetainsActualDisplayAndVector(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, nil)
	a := algebraicLargeDyadicPromotionRoot(t, c)
	candidates, e := c.NewVector()
	serviceMust(t, e)
	serviceMust(t, candidates.PushCopy(&a))
	writer := &algebraicBinaryDisplayStopWriter{core: c, marker: ":2235 "}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("isolate_roots")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	values := &algebraicSignTestValues{}
	body := c.filterRoots(pm, pm.core.zero, values, 0, candidates)
	var stop error
	if tags.ShouldTrace("isolate_roots") {
		if !errors.Is(body, arena.ErrBounds) || writer.ref == 0 || !c.rootTracePending() || !c.rootDisplay.active || len(candidates.values) != 1 {
			t.Fatal("actual filter display guard", body)
		}
		stop = c.rootDisplay.sourceStop
		if stop == nil || !errors.Is(body, stop) {
			t.Fatal("missing source display stop", body)
		}
		if e := candidates.Close(); !errors.Is(e, ErrAlgebraicPending) || candidates.closing || candidates.closed {
			t.Fatal("pending display mutated vector", e)
		}
		c.config.Rationals.Integers().Store(writer.ref, writer.original)
	} else if body != nil || writer.ref != 0 {
		t.Fatal("erased filter trace changed result", body)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.rootTracePending() || c.rootDisplay.active || stop != nil && c.rootDisplay.sourceStop != stop {
		t.Fatal("filter display cleanup lost source history")
	}
	serviceMust(t, candidates.Close())
	serviceMust(t, c.Del(&a))
}
