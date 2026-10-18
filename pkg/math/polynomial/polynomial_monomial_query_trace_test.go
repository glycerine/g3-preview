package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func monomialPredecessorTraceObject(t *testing.T, m *MonomialManager, degrees bool) Monomial {
	t.Helper()
	a := monomialReadProfileObject(t, m, 8)
	for i, variable := range []uint32{0, 2, 4, 6, 11, 8, 10, 14} {
		monomialReadWrite(t, m, a, 20+uint64(i)*8, variable)
		if degrees {
			monomialReadWrite(t, m, a, 24+uint64(i)*8, 1)
		}
	}
	return a
}

func TestMonomialPredecessorSourceConditionalTrace(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			m := monomialFixture(t)
			polynomialAssertionActionsFixture(t, "C")
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			if enabled {
				tags.Enable("poly_bug")
			}
			m.config.Trace = tags
			a := monomialPredecessorTraceObject(t, m, true)
			v, e := m.MaxSmallerThan(a, 12)
			serviceMust(t, e)
			if v != 10 {
				t.Fatal("diagnostic changed source predecessor", v)
			}
			want := ""
			if polynomialSourceDebug && tags.ShouldTrace("poly_bug") {
				want = "-------- [poly_bug] max_smaller_than src/math/polynomial/polynomial.cpp:362 ---------\n" +
					"m: x0 x2 x4 x6 x11 x8 x10 x14\nx: 12\ny: 10\ni: 4\nget_var(i): 11\n" +
					"------------------------------------------------\n"
			}
			if output.String() != want {
				t.Fatalf("conditional source trace: got %q, want %q", output.String(), want)
			}
		})
	}
}

type monomialQueryRefusingWriter struct{ refusal error }

func (w monomialQueryRefusingWriter) Write([]byte) (int, error) { return 0, w.refusal }

func TestMonomialPredecessorTraceOperandGatingAndRefusedStream(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(map[bool]string{false: "open", true: "refused"}[refused], func(t *testing.T) {
			m := monomialFixture(t)
			polynomialAssertionActionsFixture(t, "C")
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			if refused {
				tags = diagnostic.NewTraceTagsWithWriter(monomialQueryRefusingWriter{errors.New("refused trace")})
			}
			m.config.Trace = tags
			a := monomialPredecessorTraceObject(t, m, false)
			// A disabled trace never evaluates display's degree operands.
			v, e := m.MaxSmallerThan(a, 12)
			serviceMust(t, e)
			if v != 10 {
				t.Fatal(v)
			}
			tags.Enable("poly_bug")
			polynomialAssertionActionsFixture(t, "C")
			v, e = m.MaxSmallerThan(a, 12)
			if polynomialSourceDebug && tags.ShouldTrace("poly_bug") {
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
					t.Fatal("trace display operand not evaluated after header/stream refusal", e)
				}
				if !refused && (!strings.HasSuffix(output.String(), "m: x0") || strings.Contains(output.String(), "------------------------------------------------\n")) {
					t.Fatal("failed operand emitted source suffix", output.String())
				}
				for i := uint64(0); i < 8; i++ {
					monomialReadWrite(t, m, a, 24+8*i, 1)
				}
				v, e = m.MaxSmallerThan(a, 12)
				serviceMust(t, e)
				if v != 10 {
					t.Fatal("sticky stream failure stopped the source query", v)
				}
			} else {
				serviceMust(t, e)
				if v != 10 || output.Len() != 0 {
					t.Fatal("erased source trace evaluated display", v, output.String())
				}
			}
		})
	}
}

type monomialQueryLifetimeWriter struct {
	output                              bytes.Buffer
	manager                             *MonomialManager
	value                               Monomial
	called                              bool
	source, close, cleanup, nestedQuery error
}

func (w *monomialQueryLifetimeWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		w.source = w.manager.SourceFinalize()
		w.close = w.manager.CloseChecked()
		w.cleanup = w.manager.CleanupValue(&w.value)
		_, w.nestedQuery = w.manager.MaxSmallerThan(w.value, 12)
	}
	return w.output.Write(p)
}

func TestMonomialPredecessorTraceOwnerReentry(t *testing.T) {
	m := monomialFixture(t)
	polynomialAssertionActionsFixture(t, "C")
	a := monomialPredecessorTraceObject(t, m, true)
	w := &monomialQueryLifetimeWriter{manager: m, value: a}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("poly_bug")
	m.config.Trace = tags
	v, e := m.MaxSmallerThan(a, 12)
	serviceMust(t, e)
	if v != 10 {
		t.Fatal(v)
	}
	if polynomialSourceDebug && tags.ShouldTrace("poly_bug") {
		if !w.called {
			t.Fatal("actual trace writer callback not reached")
		}
		for _, e := range []error{w.source, w.close, w.cleanup, w.nestedQuery} {
			if !errors.Is(e, ErrMonomialPending) {
				t.Fatal("trace callback consumed active source query", e)
			}
		}
		if w.value != a || m.closing || m.sourceFinalizing || m.queryTraceRunning {
			t.Fatal("refused callback mutated owner or executing state")
		}
	} else if w.called {
		t.Fatal("erased trace reached borrowed writer")
	}
	monomialPhysicalAudit(t, m)
}

func TestMonomialPredecessorTraceWithAssertionsDisabled(t *testing.T) {
	m := monomialFixture(t)
	polynomialAssertionActionsFixture(t, "T")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("poly_bug")
	m.config.Trace = tags
	a := monomialPredecessorTraceObject(t, m, true)
	v, e := m.MaxSmallerThan(a, 12)
	serviceMust(t, e)
	if v != 10 {
		t.Fatal(v)
	}
	if polynomialSourceDebug && tags.ShouldTrace("poly_bug") {
		if !strings.Contains(output.String(), "i: 4\nget_var(i): 11\n") {
			t.Fatal("runtime SASSERT switch erased compiled DEBUG_CODE trace", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased trace emitted bytes", output.String())
	}
}
