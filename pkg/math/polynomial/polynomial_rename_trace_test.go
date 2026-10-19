package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialRenameSourceTrace(t *testing.T) {
	m, _, _ := polynomialLexFixture(t)
	a := unumeral.NewInt(31)
	deleted, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(deleted))
	serviceMust(t, m.core.DecRef(deleted))
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("rename")
	m.core.monomials.config.Trace = tags
	serviceMust(t, m.Rename([]uint32{1, 0}))
	if tags.ShouldTrace("rename") {
		for _, part := range []string{
			"[rename] rename src/math/polynomial/polynomial.cpp:6071",
			"1 0 \npolynomials before rename\n0\n1\n",
			"17 x1^3 + 13 x0^2 + 11 x0 x1^2 + 19 x0 x1 + 23\n",
			"[rename] rename src/math/polynomial/polynomial.cpp:6091",
			"polynomials after rename\n",
			"13 x1^2 + 17 x0^3 + 11 x0^2 x1 + 19 x0 x1 + 23\n",
		} {
			if !strings.Contains(out.String(), part) {
				t.Fatalf("missing %q in %q", part, out.String())
			}
		}
	} else if out.Len() != 0 {
		t.Fatalf("erased trace emitted %q", out.String())
	}
	tags.Disable("rename")
	previous := out.String()
	serviceMust(t, m.Rename([]uint32{1, 0}))
	if out.String() != previous {
		t.Fatal("disabled trace emitted output")
	}
}

type polynomialRenameTraceWriter struct {
	m      *MultivariatePolynomialManager
	t      *testing.T
	calls  int
	refuse bool
}

func (w *polynomialRenameTraceWriter) Write(p []byte) (int, error) {
	w.calls++
	for _, e := range []error{w.m.Rename([]uint32{0, 1}), w.m.CloseChecked(), w.m.core.CloseChecked(), w.m.core.SourceFinalizePrefix(), w.m.CleanupDisplay()} {
		if !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatalf("trace callback admitted mutation/cleanup: %v", e)
		}
	}
	if w.refuse {
		return 0, errors.New("ostream failed")
	}
	return len(p), nil
}

func TestPolynomialRenameTraceRefusalAndReentry(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m, p, _ := polynomialLexFixture(t)
		w := &polynomialRenameTraceWriter{m: m, t: t, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("rename")
		m.core.monomials.config.Trace = tags
		serviceMust(t, m.Rename([]uint32{1, 0}))
		monomial, e := m.core.monomialCore(p, 0)
		serviceMust(t, e)
		power, e := m.core.monomials.ReadPower(monomial, 0)
		serviceMust(t, e)
		if power.Var != 1 || power.Degree != 2 {
			t.Fatal("stream refusal interrupted rename", power)
		}
		if tags.ShouldTrace("rename") && w.calls == 0 {
			t.Fatal("trace writer was never reached")
		}
		tags.Disable("rename")
	}
}

func TestPolynomialRenameTraceBeforeRejectedMutation(t *testing.T) {
	polynomialAssertionActionsFixture(t, "T")
	m, _, ms := polynomialLexFixture(t)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("rename")
	m.core.monomials.config.Trace = tags
	if e := m.Rename(nil); e == nil {
		t.Fatal("empty permutation unexpectedly accepted")
	}
	if tags.ShouldTrace("rename") {
		if !strings.Contains(out.String(), "polynomials before rename\n") || strings.Contains(out.String(), "polynomials after rename") {
			t.Fatalf("rejected mutation trace order: %q", out.String())
		}
	}
	power, e := m.core.monomials.ReadPower(ms[1], 0)
	serviceMust(t, e)
	if power.Var != 0 || power.Degree != 2 {
		t.Fatal("rejected rename changed source monomial", power)
	}
	tags.Disable("rename")
}

func TestPolynomialRenameTraceFormattingFailurePrecedesMutation(t *testing.T) {
	m, _, ms := polynomialLexFixture(t)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("rename")
	if !tags.ShouldTrace("rename") {
		t.Skip("TRACE compiled out")
	}
	m.core.monomials.config.Trace = tags
	budget := m.core.integer.Budget()
	budget.FailAfter(0)
	e := m.Rename([]uint32{1, 0})
	budget.FailAfter(-1)
	if e == nil {
		t.Fatal("formatting reservation failure was ignored")
	}
	if strings.Contains(out.String(), "polynomials after rename") {
		t.Fatalf("failed entry formatter reached exit trace: %q", out.String())
	}
	power, e := m.core.monomials.ReadPower(ms[1], 0)
	serviceMust(t, e)
	if power.Var != 0 || power.Degree != 2 {
		t.Fatal("failed entry formatter allowed rename", power)
	}
	serviceMust(t, m.CleanupDisplay())
	serviceMust(t, m.Rename([]uint32{1, 0}))
	tags.Disable("rename")
}
