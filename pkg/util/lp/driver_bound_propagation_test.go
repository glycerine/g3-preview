package lp

// Source: src/test/lp.cpp:2759-2967.

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func lpBoundPropagationReport(c BoundPropagationCase) string {
	switch c {
	case BoundPropagationTotalUpper:
		return "test_total_case_u\n"
	case BoundPropagationTotalLower:
		return "test_total_case_l\n"
	case BoundPropagationTriangle:
		if !sourceDebugEnabled {
			return " bound ev from test_bound_propagation_one_small_sample1\n"
		}
	}
	return ""
}

func TestLPBoundPropagationActualSourceCases(t *testing.T) {
	// Independent literal source inventory: general constraints do not install
	// rows. Only triangle AddTerm creates rows/extra columns during Solve.
	counts := []uint32{3, 1, 1, 3, 2, 1, 3}
	columns := []uint32{3, 5, 2, 2, 3, 2, 3}
	for c := BoundPropagationTotalUpper; c <= BoundPropagationTotalLower; c++ {
		h := newLPPairHarness(t)
		out := &lpDriverFlushWriter{}
		d := NewBoundPropagationExample(h.m, out)
		err := d.Run(c)
		if err != sourceAssertionFailure() || out.String() != lpBoundPropagationReport(c) {
			t.Fatal(c, err, out.String())
		}
		if sourceDebugEnabled {
			wantTerms := 0
			if c == BoundPropagationTriangle {
				wantTerms = 2
			}
			if d.Solver == nil || d.Propagator != nil || d.closed || d.Solver.NumberOfConstraints() != 0 || len(d.Solver.OriginalTerms) != wantTerms || out.flushes != 0 {
				t.Fatal("debug first general constraint", c, d)
			}
		} else {
			wantRows, wantTerms := uint32(0), uint32(0)
			flushes := 0
			if c == BoundPropagationTriangle {
				wantRows, wantTerms, flushes = 2, 2, 1
			}
			if d.Status != StatusOptimal || d.Constraints != counts[c] || d.Columns != columns[c] || d.Rows != wantRows || d.Terms != wantTerms || d.Found != 0 || !d.PresolveWithDouble || !d.closed || d.Solver != nil || d.Propagator != nil || out.flushes != flushes {
				t.Fatal("native source snapshot", c, d, out.flushes)
			}
		}
		if d.Run(c) != ErrExactVectorPending {
			t.Fatal("repeated case", c)
		}
		h.must(d.Close())
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	h := newLPPairHarness(t)
	out := &lpDriverFlushWriter{}
	d := NewBoundPropagationDriver(h.m, out)
	err := d.Run()
	want := "test_total_case_u\n"
	if sourceDebugEnabled {
		if err != ErrAssertion || len(d.Completed) != 0 || d.Current == nil || d.closed {
			t.Fatal("first case assertion", err, d)
		}
	} else {
		want += " bound ev from test_bound_propagation_one_small_sample1\ntest_total_case_l\n"
		h.must(err)
		if len(d.Completed) != 7 || d.Current != nil || !d.closed || out.flushes != 1 {
			t.Fatal("completed source sequence", d, out.flushes)
		}
		for i, c := range d.Completed {
			if c.Constraints != counts[i] || c.Columns != columns[i] || !c.closed || c.Solver != nil || c.Propagator != nil {
				t.Fatal("completed child", i, c)
			}
		}
	}
	if out.String() != want || d.Run() != ErrExactVectorPending {
		t.Fatal("source sequence report", out.String(), want)
	}
	h.must(d.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPBoundPropagationOutputPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for c := BoundPropagationTotalUpper; c <= BoundPropagationTotalLower; c++ {
		want := lpBoundPropagationReport(c)
		for cut := 0; cut < len(want); cut++ {
			h := newLPPairHarness(t)
			out := &larConstraintPrefixWriter{remaining: cut, err: io.ErrClosedPipe}
			d := NewBoundPropagationExample(h.m, out)
			err := d.Run(c)
			if !errors.Is(err, io.ErrClosedPipe) || out.output.String() != want[:cut] {
				t.Fatal(c, cut, err, out.output.String())
			}
			if c == BoundPropagationTriangle {
				if d.Solver == nil || d.Propagator == nil || d.Status != StatusOptimal || d.Found != 0 {
					t.Fatal("triangle completed solve prefix", d)
				}
			} else if d.Solver != nil || d.literals != nil {
				t.Fatal("initial report must precede construction", d)
			}
			h.must(d.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
	if !sourceDebugEnabled {
		h := newLPPairHarness(t)
		out := &lpDriverFlushWriter{failAt: 1}
		d := NewBoundPropagationDriver(h.m, out)
		err := d.Run()
		if !errors.Is(err, lpErrResidualWrite) || len(d.Completed) != 1 || d.Current == nil || d.Current.Solver == nil || d.Current.Propagator == nil || d.Current.Status != StatusOptimal || out.String() != "test_total_case_u\n"+lpBoundPropagationReport(BoundPropagationTriangle) {
			t.Fatal("batch triangle endl refusal", err, d, out.String())
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	// Full parent write prefix: later children must not be constructed when an
	// earlier report stops. TotalLower report follows all six completed cases.
	want := "test_total_case_u\n"
	if !sourceDebugEnabled {
		want += lpBoundPropagationReport(BoundPropagationTriangle) + "test_total_case_l\n"
	}
	for cut := 0; cut < len(want); cut++ {
		h := newLPPairHarness(t)
		out := &larConstraintPrefixWriter{remaining: cut, err: io.ErrClosedPipe}
		d := NewBoundPropagationDriver(h.m, out)
		err := d.Run()
		completed := 0
		if cut >= len("test_total_case_u\n") {
			completed = 1
		}
		if cut >= len(want)-len("test_total_case_l\n") && !sourceDebugEnabled {
			completed = 6
		}
		if !errors.Is(err, io.ErrClosedPipe) || out.output.String() != want[:cut] || len(d.Completed) != completed || d.Current == nil {
			t.Fatal("parent prefix", cut, err, len(d.Completed), completed, out.output.String())
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPBoundPropagationSmallNumeralOwnership(t *testing.T) {
	for c := BoundPropagationTotalUpper; c <= BoundPropagationTotalLower; c++ {
		for _, direct := range []bool{false, true} {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			d := NewBoundPropagationExample(h.m, io.Discard)
			used := h.budget.Used()
			h.budget.FailAfter(0)
			err := d.Run(c)
			h.budget.FailAfter(-1)
			if err != sourceAssertionFailure() || h.budget.Used() != used {
				t.Fatal(c, direct, err, h.budget.Used(), used)
			}
			h.must(d.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
	for _, which := range []BoundPropagationCase{BoundPropagationOneRow, BoundPropagationCase(255)} {
		d := NewBoundPropagationExample(nil, io.Discard)
		if d.Run(which) != ErrBounds {
			t.Fatal("missing manager/invalid case")
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
		if d.Run(which) != ErrExactVectorPending {
			t.Fatal("closed case")
		}
	}
	d := NewBoundPropagationDriver(nil, io.Discard)
	if d.Run() != ErrBounds {
		t.Fatal("nil parent")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if d.Run() != ErrExactVectorPending {
		t.Fatal("closed parent")
	}
}

func TestLPBoundPropagationPreservesOlderComparison(t *testing.T) {
	for _, c := range []BoundPropagationCase{BoundPropagationTotalUpper, BoundPropagationTriangle, BoundPropagationTotalLower} {
		found := false
		for ordinal := int64(0); ordinal < 40 && !found; ordinal++ {
			h := newLPPairHarness(t)
			v := h.pairExactLarge(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			d := NewBoundPropagationExample(h.m, io.Discard)
			if older.Active {
				found = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_ = d.Run(c)
				if d.calls.comparison != 0 || d.Solver != nil && d.Solver.calls.comparison != 0 || d.Propagator != nil && d.Propagator.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			}
			h.must(d.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !found {
			t.Fatal("no actual older comparison", c)
		}
	}
}
