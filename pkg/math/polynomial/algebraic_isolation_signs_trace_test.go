package polynomial

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAlgebraicIsolationSignsSampleTraceCallbackAndAllocationFailure(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}})
	roots, e := c.NewVector()
	serviceMust(t, e)
	output := &linearOperationProbeWriter{probe: func() {
		before := c.isolationSigns.sample
		for i, action := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&c.isolationSigns.sample) }} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("trace callback action %d: %v", i, e)
			}
		}
		if before != c.isolationSigns.sample || c.closed || c.closing {
			t.Fatal("trace callback changed live sample or owner")
		}
		c.config.Rationals.Budget().FailAfter(0)
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("isolate_roots_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var signs []int
	e = c.IsolateMultivariateRootsWithSigns(pm, p, &algebraicIsolationAssignments{}, roots, &signs)
	c.config.Rationals.Budget().FailAfter(-1)
	if tags.ShouldTrace("isolate_roots_bug") {
		if !output.called || !numericPhysicalReservationRefusal(e) || len(signs) != 0 || len(roots.values) != 1 {
			t.Fatal("trace formatter failure lost roots or published signs", output.called, e, signs)
		}
	} else {
		serviceMust(t, e)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, roots.Close())
}

func TestAlgebraicIsolationSignsLeftSampleTrace(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, _, _ := algebraicControlledFixture(t)
		pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}})
		roots, e := c.NewVector()
		serviceMust(t, e)
		var output strings.Builder
		broken := &linearTraceFailWriter{}
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if fail {
			tags = diagnostic.NewTraceTagsWithWriter(broken)
		}
		tags.Enable("isolate_roots_bug")
		serviceMust(t, c.config.Rationals.BindTrace(tags))
		var signs []int
		serviceMust(t, c.IsolateMultivariateRootsWithSigns(pm, p, &algebraicIsolationAssignments{}, roots, &signs))
		if !reflect.DeepEqual(signs, []int{-1, 1}) || len(roots.values) != 1 || c.isolationSigns.active {
			t.Fatal("trace changed isolated roots/signs or retained sample", signs)
		}
		if tags.ShouldTrace("isolate_roots_bug") {
			if fail {
				if broken.calls != 1 {
					t.Fatal("sticky stream failure retried", broken.calls)
				}
			} else if !strings.Contains(output.String(), "w: (# + 1, 1)\n") {
				t.Fatal("literal left sample", output.String())
			}
		} else if output.Len() != 0 || broken.calls != 0 {
			t.Fatal("erased trace performed output")
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, roots.Close())
	}
}

func TestAlgebraicIsolationSignsDisabledSampleTraceDoesNotRead(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	// A foreign sample would fail display; an erased trace never reads it.
	other, _, _ := algebraicControlledFixture(t)
	a := algebraicCompareRoot(t, other, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	c.isolationSigns.sample = a
	serviceMust(t, c.traceIsolationSignsSample())
	c.isolationSigns.sample = Algebraic{}
	serviceMust(t, other.Del(&a))
}
