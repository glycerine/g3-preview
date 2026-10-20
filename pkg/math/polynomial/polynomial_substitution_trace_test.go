package polynomial

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type integerSubstitutionTraceWriter struct {
	strings.Builder
	t       *testing.T
	c       *MultivariatePolynomialCore
	entry   bool
	created Monomial
}

func (w *integerSubstitutionTraceWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *integerSubstitutionTraceWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if w.c.quadraticTrace != nil && w.c.quadraticTrace.tags.OutputCallbackActive() {
		s := &w.c.substitution.scope
		if !w.entry && strings.Contains(w.String(), ":6364 ") {
			w.entry = true
			if s.posConstructed || s.newAConstructed || s.tmpConstructed {
				w.t.Fatal("entry constructed later source locals")
			}
		}
		if s.traceMonomial != (Monomial{}) {
			h, e := w.c.monomials.headerCore(s.traceMonomial)
			serviceMust(w.t, e)
			if h.RefCount != 0 || w.c.generatedActive {
				w.t.Fatal("diagnostic factory result not actual raw root", h.RefCount)
			}
			w.created = s.traceMonomial
		}
		for _, action := range []func() error{w.c.CloseChecked, w.c.CleanupSubstitution} {
			if e := action(); !errors.Is(e, ErrPolynomialPending) {
				w.t.Fatal("executing substitution owner", e)
			}
		}
		vars := w.c.monomials.nextVar
		for _, action := range []func() error{w.c.monomials.CloseChecked, w.c.monomials.SourceFinalize, w.c.monomials.CleanupCreation, w.c.monomials.CleanupRename, func() error { _, e := w.c.monomials.MkVar(); return e }} {
			if e := action(); !errors.Is(e, ErrMonomialPending) {
				w.t.Fatal("executing diagnostic monomial owner", e)
			}
		}
		if w.c.monomials.nextVar != vars {
			w.t.Fatal("callback changed monomial variable namespace")
		}
		if e := w.c.integer.CloseChecked(); e == nil {
			w.t.Fatal("diagnostic released numerical owner")
		}
	}
	return n, e
}
func TestPolynomialIntegerSubstitutionTraceLiteralAndActualFactory(t *testing.T) {
	for _, array := range []bool{false, true} {
		c := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, c, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}, {1, 2}}, {{2, 1}}, nil})
		w := &integerSubstitutionTraceWriter{t: t, c: c}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("polynomial")
		c.monomials.config.Trace = tags
		t.Cleanup(func() { serviceMust(t, c.CleanupSubstitution()) })
		var result Polynomial
		var e error
		if array {
			result, e = c.SubstituteIntegers(p, 2, variableHelperArray([]uint32{0, 2}), []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(-3)})
		} else {
			v := unumeral.NewInt(2)
			result, e = c.SubstituteInteger(p, 0, &v)
		}
		serviceMust(t, e)
		result = polynomialCaller(t, c, result)
		expected := map[[3]uint32]int64{{0, 2, 0}: 12, {0, 0, 1}: 5, {}: 7}
		entry := "substitute num_vars: 1\nx0 -> 2\n"
		if array {
			expected = map[[3]uint32]int64{{0, 2, 0}: 12, {}: -8}
			entry = "substitute num_vars: 2\nx0 -> 2\nx2 -> -3\n"
		}
		polynomialQueryExpect(t, c, result, expected)
		if tags.ShouldTrace("polynomial") {
			if !w.entry || w.created == (Monomial{}) || !strings.Contains(w.String(), entry) || !strings.Contains(w.String(), "processing 3 x0^2 x1^2\nnew_a: 12 x1^2\n") || strings.Count(w.String(), ":6394 ") != 3 {
				t.Fatal("source records/actual factory", w.String(), w.created)
			}
			found := false
			size, e := c.readWord(result, 8)
			serviceMust(t, e)
			for i := uint32(0); i < size; i++ {
				m, e := c.monomialCore(result, i)
				serviceMust(t, e)
				found = found || m == w.created
			}
			if !found {
				t.Fatal("later add did not use the actual diagnostic-created interned value")
			}
		} else if w.Len() != 0 || w.created != (Monomial{}) {
			t.Fatal("erased trace executed")
		}
		tags.Finalize()
	}
}
func TestPolynomialIntegerSubstitutionTraceZeroAddRetainsActualRawRoot(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{{{0, 2}, {1, 2}}})
	w := &integerSubstitutionTraceWriter{t: t, c: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	t.Cleanup(func() { serviceMust(t, c.CleanupSubstitution()) })
	defer tags.Finalize()
	zero := unumeral.NewInt(0)
	result, e := c.SubstituteInteger(p, 0, &zero)
	serviceMust(t, e)
	result = polynomialCaller(t, c, result)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{})
	if tags.ShouldTrace("polynomial") {
		if w.created == (Monomial{}) || len(c.substitution.orphans) != 1 || c.substitution.orphans[0] != w.created {
			t.Fatal("actual zero-add root not retained", w.created, c.substitution.orphans)
		}
		h, e := c.monomials.Info(w.created)
		serviceMust(t, e)
		if h.RefCount != 0 {
			t.Fatal("invented diagnostic reference", h.RefCount)
		}
	}
	serviceMust(t, c.CleanupUnreferencedSubstitutionMonomials())
}

type integerSubstitutionCharacterGuard struct {
	strings.Builder
	c       *MultivariatePolynomialCore
	guarded bool
}

func (w *integerSubstitutionCharacterGuard) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *integerSubstitutionCharacterGuard) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	r := &w.c.substitution.scope.traceInsertion
	if !w.guarded && strings.Contains(w.String(), "src/util/mpn.cpp:395 ") && r.Active() && r.Character != nil && r.Character.Completed {
		w.guarded = true
		r.Character.Buffer = r.Character.Buffer[:0]
	}
	return n, e
}
func TestPolynomialIntegerSubstitutionTraceEntryActualCharacterGuard(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 1}}})
	var big unumeral.Int
	serviceMust(t, c.integer.SetString(&big, "18446744073709551619"))
	t.Cleanup(func() { serviceMust(t, c.CleanupSubstitution()); serviceMust(t, c.integer.Del(&big)) })
	w := &integerSubstitutionCharacterGuard{c: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	tags.Enable("mpn_to_string")
	c.monomials.config.Trace = tags
	serviceMust(t, c.integer.BindTrace(tags))
	defer func() { serviceMust(t, c.integer.BindTrace(nil)) }()
	defer tags.Finalize()
	unexpected, body := c.SubstituteInteger(p, 0, &big)
	if body == nil {
		polynomialCaller(t, c, unexpected)
	}
	s := &c.substitution.scope
	if !w.guarded || !errors.Is(body, arena.ErrBounds) || !s.traceInsertion.Active() || !s.active || s.posConstructed || s.newAConstructed || s.tmpConstructed || s.traceInsertion.SourceCause() != nil {
		t.Fatal("entry actual body failure or source frontier", body, w.guarded, s.active)
	}
	serviceMust(t, c.CleanupSubstitution())
	if s.active || s.traceInsertion.Active() {
		t.Fatal("matched entry cleanup retained formatter")
	}
	tags.Finalize()
	result, e := c.SubstituteInteger(p, 0, &big)
	serviceMust(t, e)
	result = polynomialCaller(t, c, result)
	a, e := c.coefficientCore(result, 0)
	serviceMust(t, e)
	cmp, e := c.integer.Compare(&a, &big)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("entry guard altered source input", cmp)
	}
}

func TestPolynomialIntegerSubstitutionTraceRefusingStreamStillInvokesFactory(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{{{0, 2}, {1, 2}}})
	w := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	t.Cleanup(func() { serviceMust(t, c.CleanupSubstitution()) })
	before := c.monomials.continuationGeneration
	value := unumeral.NewInt(2)
	result, e := c.SubstituteInteger(p, 0, &value)
	serviceMust(t, e)
	result = polynomialCaller(t, c, result)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{0, 2, 0}: 12})
	if tags.ShouldTrace("polynomial") && (w.calls != 1 || c.monomials.continuationGeneration-before != 3) {
		t.Fatal("sticky refusal skipped the actual diagnostic intern call", w.calls, c.monomials.continuationGeneration-before)
	}
}

func TestPolynomialIntegerSubstitutionTraceEveryFactoryReservation(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	for _, value := range []int32{0, 2} {
		success, refused := false, false
		for prefix := int64(0); prefix < 1024 && !success; prefix++ {
			if !t.Run(strconv.Itoa(int(value))+"/"+strconv.FormatInt(prefix, 10), func(t *testing.T) {
				c := polynomialCellDirectObjectFixture(t)
				p := polynomialQueryTerms(t, c, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}, {1, 2}}, {{2, 1}}, nil})
				tags := diagnostic.NewTraceTagsWithWriter(&strings.Builder{})
				tags.Enable("polynomial")
				c.monomials.config.Trace = tags
				defer tags.Finalize()
				t.Cleanup(func() {
					c.integer.Budget().FailAfter(-1)
					serviceMust(t, c.CleanupSubstitution())
					serviceMust(t, c.CleanupUnreferencedSubstitutionMonomials())
				})
				v := unumeral.NewInt(value)
				c.integer.Budget().FailAfter(prefix)
				result, body := c.SubstituteInteger(p, 0, &v)
				c.integer.Budget().FailAfter(-1)
				polys := []Polynomial{p}
				if body == nil {
					success = true
					result = polynomialCaller(t, c, result)
					polys = append(polys, result)
				} else {
					refused = true
					if !numericPhysicalReservationRefusal(body) {
						t.Fatal("actual allocation category", body)
					}
				}
				polynomialSubstitutionPhysical(t, c, nil, polys, nil, &v)
				serviceMust(t, c.CleanupSubstitution())
				polynomialSubstitutionPhysical(t, c, nil, polys, nil, &v)
				if body == nil {
					expected := map[[3]uint32]int64{{0, 0, 1}: 5, {}: 7}
					if value != 0 {
						expected[[3]uint32{0, 2, 0}] = 12
					}
					polynomialQueryExpect(t, c, result, expected)
				}
			}) {
				return
			}
		}
		if !success || !refused {
			t.Fatal("factory reservation endpoints", value, success, refused)
		}
	}
}

type integerSubstitutionAfterFactoryGate struct {
	c       *MultivariatePolynomialCore
	created Monomial
	armed   bool
}

func (w *integerSubstitutionAfterFactoryGate) Write(p []byte) (int, error) {
	if !w.armed && w.c.substitution.scope.traceMonomial != (Monomial{}) {
		w.created = w.c.substitution.scope.traceMonomial
		w.armed = true
		w.c.integer.Budget().FailAfter(0)
	}
	return len(p), nil
}
func TestPolynomialIntegerSubstitutionTraceActualFailureAfterDiagnosticFactory(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	c := polynomialCellFixture(t)
	var big unumeral.Int
	digits := make([]uint32, 64)
	digits[0], digits[63] = 3, 1
	serviceMust(t, c.integer.SetLimbs(&big, digits))
	mono, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}, {1, 2}})
	serviceMust(t, e)
	p, e := c.MkTerms([]unumeral.Int{big}, []Monomial{mono})
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.CleanupSubstitution())
		serviceMust(t, c.integer.Del(&big))
	})
	w := &integerSubstitutionAfterFactoryGate{c: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	value := unumeral.NewInt(2)
	_, body := c.SubstituteInteger(p, 0, &value)
	c.integer.Budget().FailAfter(-1)
	if !w.armed || w.created == (Monomial{}) || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual post-diagnostic allocation site not reached", body, w.armed)
	}
	polynomialSubstitutionPhysical(t, c, nil, []Polynomial{p}, nil, &big, &value)
	serviceMust(t, c.CleanupSubstitution())
	polynomialSubstitutionPhysical(t, c, nil, []Polynomial{p}, nil, &big, &value)
	if c.substitution.scope.traceMonomial != (Monomial{}) {
		t.Fatal("matched cleanup retained diagnostic raw root")
	}
}

func TestPolynomialIntegerSubstitutionTraceEmptyInputDoesNotReadArrays(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 1}}})
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	t.Cleanup(func() { serviceMust(t, c.CleanupSubstitution()) })
	result, e := c.SubstituteIntegers(p, 0, PolynomialVariablesView{}, nil)
	serviceMust(t, e)
	result = polynomialCaller(t, c, result)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 1})
	if tags.ShouldTrace("polynomial") && !strings.Contains(text.String(), "substitute num_vars: 0\n") {
		t.Fatal("empty prelude", text.String())
	}
}
