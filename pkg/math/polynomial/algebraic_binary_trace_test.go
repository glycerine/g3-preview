package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBinarySetCoreSourceTraces(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(map[bool]string{false: "algebraic", true: "zero"}[zero], func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			c.factor = false
			a, b := algebraicResultantInputs(t, c, up)
			var out Algebraic
			var output strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			if zero {
				serviceMust(t, c.Sub(&a, &a, &out))
			} else {
				serviceMust(t, c.Add(&a, &b, &out))
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if tags.ShouldTrace("algebraic") {
				input := "-------- [algebraic] set_core src/math/polynomial/algebraic_numbers.cpp:945 ---------\nset_core p: "
				last := "-------- [algebraic] set_core src/math/polynomial/algebraic_numbers.cpp:984 ---------\nset_core...\n------------------------------------------------\n"
				if zero {
					last = "-------- [algebraic] set_core src/math/polynomial/algebraic_numbers.cpp:951 ---------\nreseting\nresult: (#, 1)\n------------------------------------------------\n"
				}
				if strings.Count(output.String(), input) != 1 || !strings.HasSuffix(output.String(), last) {
					t.Fatal("set_core source records", output.String())
				}
			} else if output.Len() != 0 {
				t.Fatal("erased set_core trace evaluated output")
			}
			if zero {
				if out != (Algebraic{}) {
					t.Fatal("self-subtraction retained a cell")
				}
			} else {
				algebraicExpectBetween(t, c, &out, 3, 4)
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		})
	}
}

func TestAlgebraicBinarySetCoreDisabledTracesDoNotRead(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	serviceMust(t, c.traceSetCorePolynomial(UPolynomialView{count: 99}))
	serviceMust(t, c.traceSetCoreReset(nil))
	serviceMust(t, c.traceSetCoreAlgebraic())
	serviceMust(t, c.traceBinaryIntervals(nil, nil, ^uint32(0), ^uint32(0)))
}

func TestAlgebraicBinaryIntervalTraceBeforeCandidateSelection(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	c.factor = false
	a, b := algebraicResultantInputs(t, c, up)
	var out Algebraic
	var output strings.Builder
	probed := false
	probe := &algebraicBinaryIntervalProbeWriter{output: &output, probe: func() {
		probed = true
		if !c.binary.active || !c.binary.running || !c.binary.intervalConstructed || !c.binary.saved[0].owned || !c.binary.saved[1].owned || len(c.binary.sequences) != 1 || c.binary.sequences[0] == nil || out != (Algebraic{}) {
			t.Fatal("interval trace moved past candidate selection")
		}
		for i, action := range []func() error{c.Close, c.CleanupRetained, c.CleanupIsolation, c.cleanupBinary, func() error { return c.Del(&a) }} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("interval trace callback %d: %v", i, e)
			}
		}
		if e := up.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Fatal("interval trace released active UP owner", e)
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(probe)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Add(&a, &b, &out))
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if tags.ShouldTrace("algebraic") {
		want := "-------- [algebraic] mk_binary src/math/polynomial/algebraic_numbers.cpp:1046 ---------\nr_i: (2, 4)\nlV: 1, uV: 0\na.m_interval: (1, 2)\nb.m_interval: (1, 2)\n------------------------------------------------\n"
		if !probed || strings.Count(output.String(), want) != 1 {
			t.Fatal("interval selection source record", output.String())
		}
	} else if probed || output.Len() != 0 {
		t.Fatal("erased interval trace called writer")
	}
	algebraicExpectBetween(t, c, &out, 3, 4)
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}

type algebraicBinaryIntervalProbeWriter struct {
	output *strings.Builder
	probe  func()
}

func (w *algebraicBinaryIntervalProbeWriter) Write(p []byte) (int, error) {
	if w.probe != nil && string(p) == "r_i: " {
		probe := w.probe
		w.probe = nil
		probe()
	}
	return w.output.Write(p)
}

func TestAlgebraicBinarySetCoreStickyTraceFailure(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	c.factor = false
	a, b := algebraicResultantInputs(t, c, up)
	writer := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	serviceMust(t, c.Add(&a, &b, &out))
	want := 0
	if tags.ShouldTrace("algebraic") {
		want = 1
	}
	if writer.calls != want || c.rootTracePending() || c.binary.active {
		t.Fatal("failed trace stream changed binary execution", writer.calls)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	algebraicExpectBetween(t, c, &out, 3, 4)
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}

type algebraicBinaryDisplayStopWriter struct {
	core     *AlgebraicRootCore
	marker   string
	output   strings.Builder
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
}

func (w *algebraicBinaryDisplayStopWriter) Write(p []byte) (int, error) {
	_, _ = w.output.Write(p)
	if q := w.core.config.Polynomials.currentDisplay; w.ref == 0 && strings.Contains(w.output.String(), w.marker) && q != nil && q.constructed && q.numeral.Cell != 0 {
		z := w.core.config.Rationals.Integers()
		w.ref = q.numeral.Cell
		w.original = z.Cell(w.ref)
		guard := w.original
		guard.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(w.ref, guard)
	}
	return len(p), nil
}

func TestAlgebraicBinaryTraceRetainsActualDisplayBeforeLocals(t *testing.T) {
	for _, marker := range []string{":549 ", ":945 "} {
		t.Run(marker, func(t *testing.T) {
			c, up, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicLargeDyadicPromotionRoot(t, c)
			b, unused := algebraicResultantInputs(t, c, up)
			serviceMust(t, c.Del(&unused))
			var out Algebraic
			writer := &algebraicBinaryDisplayStopWriter{core: c, marker: marker}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			body := c.Add(&a, &b, &out)
			var frame *upolyDisplayScope
			var stop error
			if tags.ShouldTrace("algebraic") {
				frame = c.rootTrace.univariate.frame
				if writer.ref == 0 || !errors.Is(body, arena.ErrBounds) || frame == nil || !c.rootTracePending() || !c.binary.active || c.binary.f.slot == 0 || c.binary.p.slot == 0 || c.binary.factors == nil {
					t.Fatal("display consumed older binary owners", body, writer.ref)
				}
				stop = frame.sourceCause
				if stop == nil || c.binary.sourceStop != stop || out != (Algebraic{}) {
					t.Fatal("binary did not capture actual source display stop", body)
				}
				// Qualified signed-host size guard on the real formatter NUM.
				c.config.Rationals.Integers().Store(writer.ref, writer.original)
			} else {
				serviceMust(t, body)
				algebraicExpectBetween(t, c, &out, 2, 3)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.rootTracePending() || c.binary.active || frame != nil && (frame.active || frame.sourceCause != stop || c.binary.sourceStop != stop || !errors.Is(body, stop)) {
				t.Fatal("owning cleanup changed source display history")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		})
	}
}
