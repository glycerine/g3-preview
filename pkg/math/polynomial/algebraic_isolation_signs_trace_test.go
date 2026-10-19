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
		if c.isolationSigns.constructed {
			t.Fatal("input/root trace must precede sample construction")
		}
		if e := roots.Close(); !errors.Is(e, ErrAlgebraicPending) || roots.closing || roots.closed {
			t.Fatal("trace callback changed borrowed root vector", e, roots.closing, roots.closed)
		}
		for i, action := range []func() error{pm.CloseChecked, pm.SourceFinalize, pm.core.CloseChecked, func() error { return pm.core.DecRef(p) }} {
			if e := action(); !errors.Is(e, ErrPolynomialPending) {
				t.Fatalf("trace callback polynomial action %d: %v", i, e)
			}
		}
		if pm.closing || pm.sourceStarted || pm.core.closing || pm.core.sourceStarted {
			t.Fatal("trace callback changed borrowed polynomial manager")
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
	serviceMust(t, c.traceIsolationSignsRoots(nil, Polynomial{}, nil, nil))
	// A foreign sample would fail display; an erased trace never reads it.
	other, _, _ := algebraicControlledFixture(t)
	a := algebraicCompareRoot(t, other, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	c.isolationSigns.sample = a
	serviceMust(t, c.traceIsolationSignsSample())
	c.isolationSigns.sample = Algebraic{}
	serviceMust(t, other.Del(&a))
}

func TestAlgebraicIsolationSignsRootsTraceSourceRecordsAndReads(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, _, _ := algebraicControlledFixture(t)
		pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {1, [3]uint32{1, 0, 0}}})
		values := &algebraicIsolationAssignments{assigned: [3]bool{true, false, false}}
		roots, e := c.NewVector()
		serviceMust(t, e)
		var zero Algebraic
		serviceMust(t, roots.PushCopy(&zero))
		root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
		serviceMust(t, roots.PushCopy(&root))
		var output strings.Builder
		broken := &linearTraceFailWriter{}
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if fail {
			tags = diagnostic.NewTraceTagsWithWriter(broken)
		}
		tags.Enable("isolate_roots_bug")
		serviceMust(t, c.config.Rationals.BindTrace(tags))
		serviceMust(t, c.traceIsolationSignsRoots(pm, p, values, roots))
		if tags.ShouldTrace("isolate_roots_bug") {
			if !reflect.DeepEqual(values.contains, []uint32{2, 0}) || !reflect.DeepEqual(values.reads, []uint32{0, 0}) {
				t.Fatal("source variable traversal and repeated getter", values.contains, values.reads)
			}
			if fail {
				if broken.calls != 1 {
					t.Fatal("sticky failure", broken.calls)
				}
			} else {
				want := "-------- [isolate_roots_bug] isolate_roots src/math/polynomial/algebraic_numbers.cpp:2578 ---------\np: x2 + x0\nx0 -> (#, 1) [0, 0]\nroot[i]: (#, 1)\nroot[i]: (#^2 - 2, 2)\n------------------------------------------------\n"
				if output.String() != want {
					t.Fatalf("literal source roots trace: %q", output.String())
				}
			}
		} else if len(values.contains) != 0 || len(values.reads) != 0 || output.Len() != 0 || broken.calls != 0 {
			t.Fatal("erased trace evaluated inputs")
		}
		if c.isolationSigns.traceRunning || pm.core.algebraicTraceRunning {
			t.Fatal("trace execution pin leaked")
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, roots.Close())
		serviceMust(t, c.Del(&root))
	}
}

type algebraicIsolationTraceReadFailure struct {
	algebraicIsolationAssignments
	marker error
}

func (v *algebraicIsolationTraceReadFailure) Algebraic(x uint32) (*Algebraic, error) {
	a, e := v.algebraicIsolationAssignments.Algebraic(x)
	if e == nil && len(v.reads) == 2 {
		return nil, v.marker
	}
	return a, e
}

func TestAlgebraicIsolationSignsRootsTraceSecondGetterFailure(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
	values := &algebraicIsolationTraceReadFailure{marker: errors.New("interval getter Throw")}
	values.assigned[0] = true
	roots, e := c.NewVector()
	serviceMust(t, e)
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("isolate_roots_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	e = c.traceIsolationSignsRoots(pm, p, values, roots)
	if tags.ShouldTrace("isolate_roots_bug") {
		if !errors.Is(e, values.marker) || !reflect.DeepEqual(values.reads, []uint32{0, 0}) || !strings.Contains(output.String(), "x0 -> (#, 1) ") || strings.Contains(output.String(), "[0, 0]") {
			t.Fatal("second lookup failure frontier", e, values.reads, output.String())
		}
	} else {
		serviceMust(t, e)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	// The trace's native local vector and manager-owned found bits must have
	// completed their source reset, even when a later getter throws.
	var xs PolynomialVariableDegrees
	variableEmbedded(&xs, pm.core.memory, true)
	serviceMust(t, pm.Vars(p, &xs))
	if !reflect.DeepEqual(xs.values, []uint32{0}) {
		t.Fatal("trace failure poisoned subsequent variable collection", xs.values)
	}
	serviceMust(t, xs.CloseChecked())
	serviceMust(t, roots.Close())
}

func TestAlgebraicIsolationSignsNoRootsErasesTraceBlock(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{-7, [3]uint32{}}})
	roots, e := c.NewVector()
	serviceMust(t, e)
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("isolate_roots_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var signs []int
	serviceMust(t, c.IsolateMultivariateRootsWithSigns(pm, p, &algebraicIsolationAssignments{}, roots, &signs))
	if output.Len() != 0 || !reflect.DeepEqual(signs, []int{-7}) {
		t.Fatal("no-root branch reached root/sample trace", output.String(), signs)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, roots.Close())
}
