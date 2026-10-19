package polynomial

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBinaryNamedTraceEveryReservationPrefix(t *testing.T) {
	succeeded, refused := false, false
	for prefix := int64(0); prefix < 4096 && !succeeded; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, up, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a, b := algebraicResultantInputs(t, c, up)
			var out Algebraic
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("anum_mk_binary")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Add(&a, &b, &out)
			budget.FailAfter(-1)
			if body == nil {
				succeeded = true
				algebraicExpectBetween(t, c, &out, 3, 4)
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("binary diagnostic failed outside an actual reservation", body)
				}
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.config.Rationals.Integers().CleanupPendingAllocation())
			serviceMust(t, c.CleanupRetained())
			if c.binary.active || c.binary.traceRootPending || c.rootDisplay.active || c.rootTracePending() {
				t.Fatal("binary diagnostic child or locals survived owning cleanup")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !succeeded || !refused {
		t.Fatal("binary diagnostic reservation frontier did not reach refusal and success", succeeded, refused)
	}
}

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
	serviceMust(t, c.traceBinaryOperands(nil, nil))
	serviceMust(t, c.traceBinaryFactor(^uint32(0)))
	serviceMust(t, c.traceBinarySequence(^uint32(0), nil))
	serviceMust(t, c.traceBinaryTarget(^uint32(0)))
}

func TestAlgebraicBinaryNamedTraceSourceFrontiers(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	c.factor = false
	a, b := algebraicResultantInputs(t, c, up)
	var out Algebraic
	var stages []string
	writer := &algebraicBinaryNamedProbeWriter{probe: func(stage string) {
		stages = append(stages, stage)
		s := &c.binary
		if !s.running || s.p.slot == 0 || s.f.slot == 0 || out != (Algebraic{}) {
			t.Fatal("named trace lost enclosing binary owners", stage)
		}
		switch stage {
		case "a: ":
			if s.factors != nil || len(s.sequences) != 0 {
				t.Fatal("operand trace followed factors construction")
			}
		case "factor 0\n":
			if s.factors == nil || s.pendingSequence != nil || len(s.sequences) != 0 || s.intervalConstructed {
				t.Fatal("factor trace followed sequence construction")
			}
		case "sequence 0\n", "target_i: 0\n":
			if len(s.sequences) != 1 || s.sequences[0] == nil || !s.intervalConstructed || !s.saved[0].owned || !s.saved[1].owned || s.saved[0].restored || s.saved[1].restored {
				t.Fatal("selection trace moved past source saved-interval restoration", stage)
			}
		}
		for i, action := range []func() error{c.Close, c.CleanupRetained, c.cleanupBinary} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("named trace %q callback %d: %v", stage, i, e)
			}
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("anum_mk_binary")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Add(&a, &b, &out))
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if tags.ShouldTrace("anum_mk_binary") {
		if strings.Join(stages, "|") != "a: |factor 0\n|sequence 0\n|target_i: 0\n" {
			t.Fatal("source named-trace frontier order", stages, writer.output.String())
		}
		for _, text := range []string{
			"-------- [anum_mk_binary] mk_binary src/math/polynomial/algebraic_numbers.cpp:1012 ---------\na: (#^2 - 2, 2)\nb: (#^2 - 3, 2)\np: ",
			"-------- [anum_mk_binary] mk_binary src/math/polynomial/algebraic_numbers.cpp:1019 ---------\nfactor 0\n",
			"-------- [anum_mk_binary] mk_binary src/math/polynomial/algebraic_numbers.cpp:1042 ---------\nsequence 0\n",
			"-------- [anum_mk_binary] mk_binary src/math/polynomial/algebraic_numbers.cpp:1068 ---------\ntarget_i: 0\n------------------------------------------------\n",
		} {
			if strings.Count(writer.output.String(), text) != 1 {
				t.Fatal("literal named binary diagnostic", text, writer.output.String())
			}
		}
	} else if len(stages) != 0 || writer.output.Len() != 0 {
		t.Fatal("erased named binary diagnostics evaluated writer")
	}
	for _, value := range []*Algebraic{&a, &b} {
		l, e := c.location(value)
		serviceMust(t, e)
		r, e := c.load(l)
		serviceMust(t, e)
		want := uint32(0)
		if tags.ShouldTrace("anum_mk_binary") {
			want = 2
		}
		if r.flags>>3 != want {
			t.Fatal("root display cache did not survive arithmetic", r.flags>>3, want)
		}
	}
	algebraicExpectBetween(t, c, &out, 3, 4)
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}

type algebraicBinaryNamedProbeWriter struct {
	output strings.Builder
	probe  func(string)
}

func (w *algebraicBinaryNamedProbeWriter) Write(p []byte) (int, error) {
	switch string(p) {
	case "a: ", "factor 0\n", "sequence 0\n", "target_i: 0\n":
		w.probe(string(p))
	}
	return w.output.Write(p)
}

func TestAlgebraicBinaryNamedTraceActualDisplayStops(t *testing.T) {
	for _, marker := range []string{"a: ", "\nb: ", "\np: ", ":1019 ", ":1042 "} {
		t.Run(marker, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicLargeDyadicPromotionRoot(t, c)
			var b, out Algebraic
			serviceMust(t, c.Copy(&b, &a))
			writer := &algebraicBinaryDisplayStopWriter{core: c, marker: marker}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("anum_mk_binary")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			body := c.Add(&a, &b, &out)
			var frame *upolyDisplayScope
			var stop error
			root := marker == "a: " || marker == "\nb: "
			if tags.ShouldTrace("anum_mk_binary") {
				if root {
					frame = c.rootDisplay.display.frame
					stop = c.rootDisplay.sourceStop
					if !c.binary.traceRootPending || !c.rootDisplay.active || stop != c.binary.traceRootStop {
						t.Fatal("binary did not retain its actual root-display call", body)
					}
				} else {
					frame = c.rootTrace.univariate.frame
					stop = c.rootTraceSourceCause()
				}
				if writer.ref == 0 || !errors.Is(body, arena.ErrBounds) || frame == nil || !frame.active || stop == nil || c.binary.sourceStop != stop || out != (Algebraic{}) || !c.binary.active || c.binary.p.slot == 0 || c.binary.f.slot == 0 {
					t.Fatal("actual named diagnostic source stop did not protect binary locals", body, writer.ref)
				}
				if marker == ":1019 " {
					if c.binary.factors == nil || c.binary.pendingSequence != nil || len(c.binary.sequences) != 0 {
						t.Fatal("stopped factor trace acquired a younger sequence")
					}
				} else if marker == ":1042 " {
					if !c.binary.intervalConstructed || len(c.binary.sequences) != 1 || !c.binary.saved[0].owned || !c.binary.saved[1].owned {
						t.Fatal("stopped sequence trace lost older selection locals")
					}
				} else if c.binary.factors != nil {
					t.Fatal("stopped operand trace acquired younger factors")
				}
				// Restore the actual NUM after its qualified requested-size guard.
				c.config.Rationals.Integers().Store(writer.ref, writer.original)
			} else {
				serviceMust(t, body)
				algebraicExpectBetween(t, c, &out, 2, 3)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.binary.active || c.binary.traceRootPending || c.rootDisplay.active || c.rootTracePending() || frame != nil && (frame.active || frame.sourceCause != stop || c.binary.sourceStop != stop || !errors.Is(body, stop)) {
				t.Fatal("owning named-trace cleanup changed source history")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		})
	}
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
	for _, tag := range []string{"algebraic", "anum_mk_binary"} {
		t.Run(tag, func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			c.factor = false
			a, b := algebraicResultantInputs(t, c, up)
			writer := &linearTraceFailWriter{}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable(tag)
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			var out Algebraic
			serviceMust(t, c.Add(&a, &b, &out))
			want := 0
			if tags.ShouldTrace(tag) {
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

		})
	}
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
