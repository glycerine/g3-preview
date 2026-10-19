package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicPolynomialRootTraceBeforeWorkspaceAndRangeError(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-1, [3]uint32{}}})
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	e := c.MkPolynomialRoot(pm, p, 3, &out)
	if e == nil || e.Error() != "invalid root object, polynomial does have sufficient roots" || out != (Algebraic{}) {
		t.Fatal("source range check/output frontier", e, out)
	}
	if tags.ShouldTrace("algebraic") {
		for _, record := range []string{
			"-------- [algebraic] mk_root src/math/polynomial/algebraic_numbers.cpp:668 ---------\nisolating roots of: x2^2 - 1\n------------------------------------------------\n",
			"-------- [algebraic] mk_root src/math/polynomial/algebraic_numbers.cpp:655 ---------\nnum-roots: 2\n[-1, -1]\n[1, 1]\n------------------------------------------------\n",
		} {
			if !strings.Contains(output.String(), record) {
				t.Fatal("missing literal source record", record, output.String())
			}
		}
		if strings.Index(output.String(), ":668 ---------") > strings.Index(output.String(), ":655 ---------") {
			t.Fatal("input trace followed selection trace")
		}
	} else if output.Len() != 0 {
		t.Fatal("erased source TRACE emitted output")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
}

func TestAlgebraicSexprRootTraceAndStickyStreamFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, _, sx, sy, q, _ := algebraicSexprRootFixture(t)
		x := sxPolySym(t, sx, sy, "x")
		one, two := sxPolyNum(t, sx, q, "1"), sxPolyNum(t, sx, q, "2")
		p := sxPolyApp(t, sx, sy, "-", sxPolyApp(t, sx, sy, "^", x, two), one)
		var output strings.Builder
		broken := &linearTraceFailWriter{}
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if fail {
			tags = diagnostic.NewTraceTagsWithWriter(broken)
		}
		tags.Enable("algebraic")
		serviceMust(t, c.config.Rationals.BindTrace(tags))
		var out Algebraic
		serviceMust(t, c.MkSexprRoot(sx, p, 2, &out))
		if tags.ShouldTrace("algebraic") {
			if fail {
				if broken.calls != 1 {
					t.Fatal("sticky source stream failure retried output", broken.calls)
				}
			} else if !strings.Contains(output.String(), "-------- [algebraic] mk_root src/math/polynomial/algebraic_numbers.cpp:678 ---------\nmk_root 2\nx^2 - 1\n------------------------------------------------\n") {
				t.Fatal("literal expression trace", output.String())
			}
		} else if output.Len() != 0 || broken.calls != 0 {
			t.Fatal("erased expression trace emitted output")
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		algebraicExpectBetween(t, c, &out, 0, 2)
		serviceMust(t, c.Del(&out))
	}
}

func TestAlgebraicRootConstructionTraceExecutingCallback(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-1, [3]uint32{}}})
	output := &linearOperationProbeWriter{probe: func() {
		if c.isolateTmp1.slot != 0 {
			t.Fatal("input trace must precede lazy Go workspace acquisition")
		}
		for i, action := range []func() error{c.Close, c.CleanupRetained, c.CleanupIsolation} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("trace callback algebraic action %d: %v", i, e)
			}
		}
		for i, action := range []func() error{pm.CloseChecked, pm.SourceFinalize, func() error { return pm.core.DecRef(p) }} {
			if e := action(); !errors.Is(e, ErrPolynomialPending) {
				t.Fatalf("trace callback polynomial action %d: %v", i, e)
			}
		}
		if e := c.config.Polynomials.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Fatal("trace callback univariate owner close", e)
		}
		if c.closed || c.closing || pm.closing || pm.sourceStarted {
			t.Fatal("trace callback changed enclosing owner lifetime")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	serviceMust(t, c.MkPolynomialRoot(pm, p, 2, &out))
	if output.called != tags.ShouldTrace("algebraic") || c.rootTrace.running || pm.core.algebraicTraceRunning {
		t.Fatal("trace callback or execution pin lifetime")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.Del(&out))
}

func TestAlgebraicRootConstructionDisabledTraceDoesNotRead(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	serviceMust(t, c.traceRootPolynomial(nil, Polynomial{}))
	serviceMust(t, c.traceRootExpression(UPolynomialView{}, ^uint32(0)))
	serviceMust(t, c.traceRootSelection(nil, ^uint32(0)))
}

type algebraicRootVectorTraceProbe struct {
	strings.Builder
	probe func()
}

func (w *algebraicRootVectorTraceProbe) Write(p []byte) (int, error) {
	if w.probe != nil && strings.Contains(string(p), "num-roots: ") {
		probe := w.probe
		w.probe = nil
		probe()
	}
	return w.Builder.Write(p)
}

func TestAlgebraicRootSelectionTraceKeepsActualVector(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	p := algebraicPolynomial(t, up, -1, 0, 1)
	called := false
	output := &algebraicRootVectorTraceProbe{probe: func() {
		called = true
		v := c.rootSelection
		if v == nil || len(v.values) != 2 {
			t.Fatal("selection trace did not borrow actual isolated roots")
		}
		before := v.values[0]
		for i, action := range []func() error{v.Close, c.Close, c.CleanupRetained, c.CleanupIsolation} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("selection trace callback action %d: %v", i, e)
			}
		}
		if v.closed || v.closing || v.values[0] != before || c.closed || c.closing {
			t.Fatal("selection trace callback changed borrowed owners")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	serviceMust(t, c.MkRoot(mustUPolyView(t, up, p), 1, &out))
	if called != tags.ShouldTrace("algebraic") || c.rootSelection != nil || c.rootTrace.running {
		t.Fatal("selection trace/vector lifetime", called)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.Del(&out))
}

type algebraicRootTraceNumeralStop struct {
	core     *AlgebraicRootCore
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
}

func (w *algebraicRootTraceNumeralStop) Write(p []byte) (int, error) {
	if f := w.core.rootTrace.poly.frame; w.ref == 0 && f != nil && f.constructed && f.numeral.Cell != 0 {
		z := w.core.config.Rationals.Integers()
		w.ref = f.numeral.Cell
		w.original = z.Cell(w.ref)
		guard := w.original
		guard.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(w.ref, guard)
	}
	return len(p), nil
}

func TestAlgebraicRootPolynomialTraceOwnsActualStoppedDisplay(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, nil)
	n := unumeral.NewInt(0)
	z := c.config.Rationals.Integers()
	serviceMust(t, z.SetString(&n, "18446744073709551619"))
	p, e := pm.core.MkConstantTake(&n)
	serviceMust(t, e)
	p = polynomialCaller(t, pm.core, p)
	vector, e := c.NewVector()
	serviceMust(t, e)
	output := &algebraicRootTraceNumeralStop{core: c}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	var stopped *polynomialDisplayFrame
	var sourceStop error
	body := c.MkPolynomialRoot(pm, p, 1, &out)
	if tags.ShouldTrace("algebraic") {
		// Qualified signed-host guard on the real reached NUM, not a claim
		// that ordinary source deallocation fails at this size.
		if output.ref == 0 || !errors.Is(body, arena.ErrBounds) || !c.rootTracePending() || c.isolateTmp1.slot != 0 || out != (Algebraic{}) {
			t.Fatal("actual display source stop frontier", output.ref, body, c.rootTracePending())
		}
		receipt := c.rootTrace.poly
		frame, stop := receipt.frame, receipt.frame.sourceCause
		stopped, sourceStop = frame, stop
		copied := *pm.core
		copied.self = &copied
		serviceMust(t, copied.cleanupDisplayReceipt(&receipt))
		stale := receipt
		stale.generation++
		serviceMust(t, pm.core.cleanupDisplayReceipt(&stale))
		if !frame.active || frame.sourceCause != stop || frame.numeral.Cell != output.ref {
			t.Fatal("copied owner or stale receipt consumed actual display")
		}
		if e = vector.Close(); !errors.Is(e, ErrAlgebraicPending) || vector.closing || vector.closed {
			t.Fatal("stopped display allowed older vector header mutation", e)
		}
		z.Store(output.ref, output.original)
	} else if body == nil || body.Error() != "invalid root object, polynomial does have sufficient roots" || output.ref != 0 {
		t.Fatal("erased trace changed constant root route", body, output.ref)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, vector.Close())
	if c.rootTracePending() || pm.algebraicIsolationBorrowers != 0 || c.rootConstruction.active {
		t.Fatal("owning cleanup retained trace or enclosing root constructor")
	}
	if stopped != nil && (stopped.sourceCause != sourceStop || c.rootConstruction.sourceStop != sourceStop || !errors.Is(body, sourceStop)) {
		t.Fatal("owning acknowledgement changed actual source-stop history")
	}
}
