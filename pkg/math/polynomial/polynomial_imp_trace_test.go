package polynomial

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func polynomialImpTraceFixture(t *testing.T, tags *diagnostic.TraceTags) (*MultivariatePolynomialManager, *MonomialManager) {
	t.Helper()
	q, budget := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: q.Integers().Memory, Trace: tags})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{
		Core: MultivariatePolynomialConfig{
			Integer: q.Integers(), Rationals: q, Monomials: mm,
			Checkpoint:  func() bool { panic("destructor does not checkpoint") },
			Cooperation: func() error { panic("destructor does not cooperate") },
		},
		Trace: tags,
	})
	serviceMust(t, e)
	return m, mm
}

func TestPolynomialImpDestructorLeakTrace(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("polynomial")
	m, mm := polynomialImpTraceFixture(t, tags)
	_, e := mm.MkVar()
	serviceMust(t, e)
	leaked, e := m.core.MkVariable(0, 2)
	serviceMust(t, e)
	serviceMust(t, m.SourceFinalize())
	want := ""
	if tags.ShouldTrace("polynomial") {
		want = "-------- [polynomial] del src/math/polynomial/polynomial.cpp:2421 ---------\n" +
			"deleting: 0\n------------------------------------------------\n" +
			"-------- [polynomial] del src/math/polynomial/polynomial.cpp:2421 ---------\n" +
			"deleting: 1\n------------------------------------------------\n"
		if polynomialSourceDebug {
			want += "-------- [polynomial] ~imp src/math/polynomial/polynomial.cpp:2357 ---------\n" +
				"leaked polynomials\nx0^2\n------------------------------------------------\n"
		}
	}
	if output.String() != want || m.destructorTraceRunning {
		t.Fatalf("source imp destructor trace: got %q, want %q, active=%v", output.String(), want, m.destructorTraceRunning)
	}
	serviceMust(t, m.core.CleanupValue(&leaked))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, mm.DecManagerRef())
}

func TestPolynomialImpDestructorDirectoryAssertionSwitch(t *testing.T) {
	polynomialAssertionActionsFixture(t, "T")
	d := diagnostic.BorrowProcessDebugContext()
	w := &polynomialImpDisableAssertionsWriter{debug: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	m, mm := polynomialImpTraceFixture(t, tags)
	e := m.SourceFinalize()
	if polynomialSourceDebug && tags.ShouldTrace("polynomial") {
		serviceMust(t, e)
		if !w.disabled || !m.sourceFinalized || len(m.core.polynomials) != 0 {
			t.Fatal("disabled reached destructor assertion switch after directory reset", w.disabled, m.sourceFinalized, len(m.core.polynomials))
		}
		serviceMust(t, d.EnableAssertions(true))
	} else {
		serviceMust(t, e)
	}
	serviceMust(t, m.CloseChecked())
	serviceMust(t, mm.DecManagerRef())
}

func TestPolynomialImpDestructorTraceReentryGuard(t *testing.T) {
	w := &polynomialImpReentryWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	m, mm := polynomialImpTraceFixture(t, tags)
	w.manager = m
	serviceMust(t, m.SourceFinalize())
	if polynomialSourceDebug && tags.ShouldTrace("polynomial") {
		if !w.called || w.cause != ErrPolynomialPending {
			t.Fatal("destructor trace reentry guard", w.called, w.cause)
		}
	}
	serviceMust(t, m.CloseChecked())
	serviceMust(t, mm.DecManagerRef())
}

type polynomialImpDisableAssertionsWriter struct {
	debug    *diagnostic.DebugContext
	disabled bool
}

func (w *polynomialImpDisableAssertionsWriter) Write(p []byte) (int, error) {
	if !w.disabled && bytes.Contains(p, []byte("leaked polynomials")) {
		if e := w.debug.EnableAssertions(false); e != nil {
			return 0, e
		}
		w.disabled = true
	}
	return len(p), nil
}

type polynomialImpReentryWriter struct {
	manager *MultivariatePolynomialManager
	called  bool
	cause   error
}

func (w *polynomialImpReentryWriter) Write(p []byte) (int, error) {
	if !w.called && bytes.Contains(p, []byte("leaked polynomials")) {
		w.called = true
		w.cause = w.manager.SourceFinalize()
	}
	return len(p), nil
}
