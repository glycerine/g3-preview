package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"strings"
	"testing"
)

type polynomialFactorBoundaryWriter struct{ polynomialQuadraticTraceWriter }

func (w *polynomialFactorBoundaryWriter) Write(p []byte) (int, error) {
	if e := w.m.CleanupFactor(); !errors.Is(e, ErrPolynomialPending) {
		w.t.Fatal("executing factor diagnostic acquired owning cleanup", e)
	}
	boundary := w.m.factorPrelude.executing || w.m.factorEnd.executing
	for _, s := range w.m.factorFrames {
		boundary = boundary || s.childPrelude.executing
	}
	if boundary {
		if e := w.m.core.integer.CloseChecked(); !errors.Is(e, unumeral.ErrNumericPending) {
			w.t.Fatal("factor diagnostic closed integer manager", e)
		}
		if e := w.m.core.integer.Natural.CloseChecked(); !errors.Is(e, arena.ErrInvalidSlot) {
			w.t.Fatal("factor diagnostic closed natural manager", e)
		}
	}
	return w.polynomialQuadraticTraceWriter.Write(p)
}
func TestPolynomialFactorBoundaryTraceRecursiveOrderAndRefusingWriter(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		p := polynomialQueryTerms(t, m.core, []int32{1, 3, 3, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, {}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		var out bytes.Buffer
		writer := &polynomialFactorBoundaryWriter{polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: refuse}}
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		if refuse {
			tags = diagnostic.NewTraceTagsWithWriter(writer)
		}
		tags.Enable("factor")
		tags.Enable("factor_bug")
		m.core.monomials.config.Trace = tags
		defer func() { tags.Finalize(); serviceMust(t, m.CleanupFactor()); serviceMust(t, f.CloseChecked()) }()
		serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
		polynomialFactorAssertReconstruction(t, m, p, f, 1)
		if tags.ShouldTrace("factor") {
			if refuse {
				if writer.calls == 0 {
					t.Fatal("boundary writer not reached")
				}
			} else {
				text := out.String()
				if strings.Count(text, ":6755 ") != 2 || strings.Count(text, ":6756 ") != 2 || strings.Count(text, ":6829 ") != 1 {
					t.Fatal("actual recursive entry/end records", text)
				}
				first := strings.Index(text, ":6755 ")
				second := strings.Index(text, ":6756 ")
				last := strings.Index(text, ":6829 ")
				if first < 0 || second <= first || last <= second || !strings.Contains(text, "[factor] end, r.get_constant(): 1\n") {
					t.Fatal("entry/end order", text)
				}
			}
		} else if out.Len() != 0 || writer.calls != 0 {
			t.Fatal("disabled TRACE emitted")
		}
	}
}
func TestPolynomialFactorBoundaryTraceConstantBIGAndZeroShortcut(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	z := m.core.integer
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	var value unumeral.Int
	serviceMust(t, z.SetString(&value, "18446744073709551616"))
	unit, e := m.core.monomials.Unit()
	serviceMust(t, e)
	p, e := m.core.MkTerms([]unumeral.Int{value}, []Monomial{unit})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("factor_bug")
	m.core.monomials.config.Trace = tags
	defer func() {
		tags.Finalize()
		serviceMust(t, m.CleanupFactor())
		serviceMust(t, f.CloseChecked())
		serviceMust(t, m.core.DecRef(p))
		serviceMust(t, z.Del(&value))
	}()
	serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
	equal, e := z.Equal(&value, &f.constant)
	serviceMust(t, e)
	if !equal {
		t.Fatal("constant Factor result")
	}
	if len(m.factorFrames) != 0 {
		t.Fatal("diagnostics constructed nonconstant factor-core frame")
	}
	if tags.ShouldTrace("factor_bug") && (!strings.Contains(out.String(), "[factor] end, r.get_constant(): 18446744073709551616\n") || m.factorEnd.number.Character == nil) {
		t.Fatal("BIG source string receipt/output", out.String())
	}
	before := out.Len()
	entry, end := m.factorPrelude.invocation, m.factorEnd.invocation
	serviceMust(t, m.Factor(m.core.zero, f, DefaultUPolynomialFactorParams()))
	if out.Len() != before || m.factorPrelude.invocation != entry || m.factorEnd.invocation != end {
		t.Fatal("source zero shortcut entered boundary records")
	}
	serviceMust(t, f.Reset())
	writer := &polynomialFactorBoundaryWriter{polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: true}}
	refused := diagnostic.NewTraceTagsWithWriter(writer)
	refused.Enable("factor_bug")
	defer refused.Finalize()
	m.core.monomials.config.Trace = refused
	serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
	equal, e = z.Equal(&value, &f.constant)
	serviceMust(t, e)
	if !equal {
		t.Fatal("refused stream changed constant result")
	}
	if refused.ShouldTrace("factor_bug") && (writer.calls == 0 || m.factorEnd.number.Character == nil) {
		t.Fatal("refused stream skipped BIG final source string")
	}
}

func TestPolynomialFactorBoundaryTraceEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 512 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			p := polynomialQueryTerms(t, m.core, []int32{1, 3, 3, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, {}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("factor")
			tags.Enable("factor_bug")
			m.core.monomials.config.Trace = tags
			defer func() {
				tags.Finalize()
				m.originalBudget.FailAfter(-1)
				serviceMust(t, m.CleanupFactor())
				serviceMust(t, f.CloseChecked())
			}()
			m.originalBudget.FailAfter(ordinal)
			cause := m.Factor(p, f, DefaultUPolynomialFactorParams())
			m.originalBudget.FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual factor boundary reservation", cause)
			}
			if cause == nil {
				success = true
				polynomialFactorAssertReconstruction(t, m, p, f, 1)
				if tags.ShouldTrace("factor_bug") && !strings.Contains(out.String(), "[factor] end, r.get_constant(): 1\n") {
					t.Fatal("completed factor omitted final record")
				}
			}
		})
	}
	if !success {
		t.Fatal("actual factor boundary never reached success")
	}
}

func TestPolynomialFactorBoundaryTraceConstantPreservesOlderBody(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{1, 2, 1}, [][]MonomialPower{{}, {{0, 1}}, {{0, 2}}})
	c := polynomialQueryTerms(t, m.core, []int32{2}, [][]MonomialPower{{}})
	older, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	current, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	defer func() {
		tags.Finalize()
		serviceMust(t, m.CleanupFactor())
		serviceMust(t, current.CloseChecked())
		serviceMust(t, older.CloseChecked())
	}()
	m.core.config.Checkpoint = func() bool { return false }
	cause := m.Factor(p, older, DefaultUPolynomialFactorParams())
	if !errors.Is(cause, ErrUPolynomialCanceled) || !m.matchesFactorCore(&m.factorRoot) || !m.factorRoot.frame.active {
		t.Fatal("actual retained older body missing", cause)
	}
	receipt := m.factorRoot
	body, source, own := receipt.frame.bodyCause, receipt.frame.sourceCause, receipt.frame.cleanupCause
	sourceRef, ownRef := receipt.frame.sourceRef, receipt.frame.ownRef
	tags.Enable("factor")
	tags.Enable("factor_bug")
	m.core.monomials.config.Trace = tags
	serviceMust(t, m.Factor(c, current, DefaultUPolynomialFactorParams()))
	if current.constant != unumeral.NewInt(2) || m.factorRoot != receipt || receipt.frame.bodyCause != body || receipt.frame.sourceCause != source || receipt.frame.cleanupCause != own || receipt.frame.sourceRef != sourceRef || receipt.frame.ownRef != ownRef {
		t.Fatal("constant diagnostics acquired older body or its history")
	}
	if tags.ShouldTrace("factor_bug") && !strings.Contains(out.String(), "[factor] end, r.get_constant(): 2\n") {
		t.Fatal("actual later constant record missing", out.String())
	}
}
