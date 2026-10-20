package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type basicLookupTraceWriter struct {
	strings.Builder
	t             *testing.T
	core          *AlgebraicRootCore
	cell          *algebraicLocation
	before, after bool
}

func (w *basicLookupTraceWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *basicLookupTraceWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	for _, action := range []func() error{w.core.Close, w.core.CleanupRetained} {
		if err := action(); !errors.Is(err, ErrAlgebraicPending) {
			w.t.Fatal("lookup executing borrow", err)
		}
	}
	if !w.before && strings.Contains(w.String(), "getting value of x0 -> ") {
		w.before = true
		// A qualified callback changes the actual valid inline NUM field. The
		// to_string operand must read it after the preceding literal insertion.
		serviceMust(w.t, w.core.config.Allocator.WriteUint32(w.cell.raw, 0, 5))
	}
	if !w.after && strings.Contains(w.String(), "5/2\n------------------------------------------------") {
		w.after = true
		// The return expression reaches basic_value again after the TRACE.
		serviceMust(w.t, w.core.config.Allocator.WriteUint32(w.cell.raw, 0, 7))
	}
	return n, e
}
func TestAlgebraicBasicLookupTraceLiteralAndSecondGetter(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	l, e := c.location(&values.values[0])
	serviceMust(t, e)
	w := &basicLookupTraceWriter{t: t, core: c, cell: l}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("var2basic")
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&values.values[0]))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	mapping := algebraicBasicRationals{algebraicOptimisticRationals: algebraicOptimisticRationals{owner: c, values: &values}}
	r, e := mapping.Rational(0)
	serviceMust(t, e)
	text, e := c.config.Rationals.String(r)
	serviceMust(t, e)
	want := "3/2"
	if tags.ShouldTrace("var2basic") {
		want = "7/2"
		expected := "-------- [var2basic] operator() src/math/polynomial/algebraic_numbers.cpp:1957 ---------\ngetting value of x0 -> 5/2\n------------------------------------------------\n"
		if !w.before || !w.after || w.String() != expected {
			t.Fatal("source record or getter order", w.String())
		}
	} else if w.Len() != 0 {
		t.Fatal("erased trace executed")
	}
	if text != want || len(values.reads) != 1 {
		t.Fatal("return getter or repeated virtual map lookup", text, values.reads)
	}
}
func TestAlgebraicBasicLookupTraceRefusingStreamStillFormats(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619/7"))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&values.values[0]))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	w := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("var2basic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	mapping := algebraicBasicRationals{algebraicOptimisticRationals: algebraicOptimisticRationals{owner: c, values: &values}}
	r, e := mapping.Rational(0)
	serviceMust(t, e)
	if tags.ShouldTrace("var2basic") && (w.calls != 1 || c.rootTrace.insertions[0].Character == nil) {
		t.Fatal("refusing stream suppressed numeric formatter", w.calls)
	}
	text, e := c.config.Rationals.String(r)
	serviceMust(t, e)
	if text != "18446744073709551619/7" {
		t.Fatal(text)
	}
}

func TestAlgebraicBasicLookupTraceActualSubstitutionCharacterGuard(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}})
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619/7"))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	values.values[1] = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&values.values[1]))
		serviceMust(t, c.Del(&values.values[0]))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	// Guard the actual completed formatter's reached C-string read, using the
	// existing qualified span fixture. No numeral/allocation extent is changed.
	w := &comparisonCharacterReadGuard{core: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("var2basic")
	tags.Enable("mpn_to_string")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	_, _, _, body := c.prepareSignPolynomial(pm, p, &values)
	if !w.guarded || !errors.Is(body, arena.ErrBounds) || !c.rootTracePending() {
		t.Fatal("nested actual formatter not retained", body, w.guarded)
	}
	receipt := &c.rootTrace.insertions[0]
	if !receipt.Active() || receipt.SourceCause() != nil {
		t.Fatal("body read failure fabricated source destructor stop")
	}
	child := &pm.core.substitution.scope
	if child.active || child.stage != 4 || child.sourceCause != nil || !c.signSubstitution.active || c.signSubstitution.core != pm.core || c.signSubstitution.prime.root.value != (Polynomial{}) {
		t.Fatal("source locals or retained callee acknowledgment", child.active, child.stage, child.sourceCause)
	}
	serviceMust(t, c.CleanupRetained())
	if c.rootTracePending() {
		t.Fatal("accepted formatter survived owning acknowledgment")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	// The original assignment remains valid after the failed diagnostic.
	l, e := c.location(&values.values[0])
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	cmp, e := c.config.Rationals.Compare(&r.value, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("formatter changed input", cmp)
	}
}

func TestAlgebraicBasicLookupTraceRetainedIsolationConsumer(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}})
	values := algebraicIsolationAssignments{assigned: [3]bool{true, true, false}}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 2))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	values.values[1] = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	roots, e := c.NewVector()
	serviceMust(t, e)
	var difference Algebraic
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&difference))
		serviceMust(t, roots.Close())
		serviceMust(t, c.Del(&values.values[1]))
		serviceMust(t, c.Del(&values.values[0]))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("var2basic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.IsolateMultivariateRoots(pm, p, &values, roots))
	if len(roots.values) != 1 {
		t.Fatal("linear mixed root count", len(roots.values))
	}
	if tags.ShouldTrace("var2basic") && (!strings.Contains(output.String(), "getting value of x0 -> 1/2\n") || strings.Contains(output.String(), "getting value of x1 -> ")) {
		t.Fatal("rational fragment getter selection", output.String())
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.Sub(&roots.values[0], &values.values[1], &difference))
	cmp, e := c.CompareRational(&difference, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("retained isolation changed root", cmp)
	}
}
