package polynomial

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Raw source placement keeps the supplied repeated monomial pointers; this
// does not pass through a normalizing SOM or repair any descriptor after a stop.
func polynomialLexDuplicate(t *testing.T, m *MultivariatePolynomialManager, as []unumeral.Int) Polynomial {
	t.Helper()
	c := m.core
	x, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
	serviceMust(t, e)
	ms := make([]Monomial, len(as))
	for i := range ms {
		ms[i] = x
		serviceMust(t, c.monomials.IncRef(x))
	}
	p, e := c.MkCoreTake(as, ms)
	serviceMust(t, e)
	return polynomialCaller(t, c, p)
}

func TestPolynomialLexDebugMacrosEraseActualUnplacedOperands(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("debug suffix erased")
	}
	polynomialAssertionActionsFixture(t, "T")
	debug := diagnostic.BorrowProcessDebugContext()
	serviceMust(t, debug.EnableAssertions(false))
	m := polynomialPSCFullFixture(t)
	c := m.core
	// Actual failed source placement writes SIZE and one numeral, then stops at
	// the absent input monomial. Its unplaced pointer fields remain untouched.
	_, body := c.MkCoreTake([]unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(3)}, nil)
	if !errors.Is(body, arena.ErrBounds) || c.pending.value == (Polynomial{}) {
		t.Fatal("placement frontier", body)
	}
	p := c.pending.value
	t.Cleanup(func() { serviceMust(t, c.CleanupConstruction()) })
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, m.lexDebugCheck(p))
	tags.Enable("poly_bug")
	if tags.ShouldTrace("poly_bug") {
		if e := m.lexDebugCheck(p); e == nil {
			t.Fatal("conditional skipped its reached pointer operand")
		}
	}
	tags.Disable("poly_bug")
	serviceMust(t, debug.EnableAssertions(true))
	if e := m.lexDebugCheck(p); e == nil {
		t.Fatal("assertion skipped its reached pointer operand")
	}
	serviceMust(t, debug.EnableAssertions(false))
	serviceMust(t, m.lexDebugCheck(p))
	if text.Len() != 0 {
		t.Fatal("failed/erased predicates emitted output")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	serviceMust(t, c.CleanupConstruction())
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

type lexTraceWriter struct {
	strings.Builder
	t        *testing.T
	m        *MultivariatePolynomialManager
	p        Polynomial
	header   bool
	onHeader func()
}

func (w *lexTraceWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *lexTraceWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if !w.header && strings.Contains(w.String(), ":1451 ") {
		w.header = true
		if w.onHeader != nil {
			w.onHeader()
		}
		flags, e := w.m.core.readWord(w.p, 4)
		serviceMust(w.t, e)
		if flags>>31 != 1 {
			w.t.Fatal("diagnostic preceded actual sorted-flag publication")
		}
	}
	for _, action := range []func() error{w.m.CloseChecked, w.m.SourceFinalize, w.m.core.CloseChecked, w.m.core.SourceFinalizePrefix, w.m.CleanupLexSort, w.m.CleanupCRA, w.m.core.CleanupDisplay} {
		if e := action(); !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatal("executing lex trace owner", e)
		}
	}
	if e := w.m.core.integer.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		w.t.Fatal("executing numeric owner", e)
	}
	for _, action := range []func() error{w.m.core.monomials.CloseChecked, w.m.core.monomials.SourceFinalize, w.m.core.monomials.CleanupCreation, w.m.core.monomials.CleanupRename} {
		if e := action(); !errors.Is(e, ErrMonomialPending) {
			w.t.Fatal("executing monomial owner", e)
		}
	}
	return n, e
}
func TestPolynomialLexConditionalLiteralAndAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			m := polynomialPSCFullFixture(t)
			p := polynomialLexDuplicate(t, m, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(3)})
			t.Cleanup(func() { serviceMust(t, m.CleanupLexSort()) })
			w := &lexTraceWriter{t: t, m: m, p: p}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("poly_bug")
			m.core.monomials.config.Trace = tags
			defer tags.Finalize()
			body := m.LexSort(p)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrPolynomialAssertion) {
					t.Fatal("source assertion did not throw", body)
				}
			} else {
				serviceMust(t, body)
			}
			if polynomialSourceDebug && tags.ShouldTrace("poly_bug") {
				want := "-------- [poly_bug] lex_sort src/math/polynomial/polynomial.cpp:1451 ---------\ni: 0\npoly: 2 x0 + 3 x0\n------------------------------------------------\n"
				if w.String() != want || !w.header {
					t.Fatal("literal conditional record", w.String())
				}
			} else if w.Len() != 0 {
				t.Fatal("erased record emitted")
			}
			flags, e := m.core.readWord(p, 4)
			serviceMust(t, e)
			if flags>>31 != 1 {
				t.Fatal("source sorted publication lost")
			}
			serviceMust(t, m.LexSort(p)) // Source sorted shortcut does not repeat a record/assertion.
			if m.lexPending {
				t.Fatal("completed assertion invented an owning child")
			}
			polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
		})
	}
}

func TestPolynomialLexTraceActualTLSCopyAndCallerCleanup(t *testing.T) {
	if !polynomialSourceDebug || !polynomialTraceEnabledForTest() {
		t.Skip("conditional debug trace erased")
	}
	for _, mode := range []string{"sort", "hash", "evaluation", "cache", "cra"} {
		t.Run(mode, func(t *testing.T) {
			polynomialAssertionActionsFixture(t, "T")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			m, qm, rt, budget := polynomialDisplayTLSOwner(t)
			c := m.core
			z := qm.Integers()
			t.Cleanup(func() {
				serviceMust(t, m.CloseChecked())
				serviceMust(t, qm.CloseChecked())
				serviceMust(t, rt.Close())
				if budget.Used() != 0 {
					t.Fatal("lex graph refund", budget.Used())
				}
			})
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
			var big unumeral.Int
			limbs := make([]uint32, 64)
			limbs[0], limbs[63] = 3, 1
			serviceMust(t, z.SetLimbs(&big, limbs))
			t.Cleanup(func() { serviceMust(t, z.Del(&big)) })
			as := []unumeral.Int{{}, unumeral.NewInt(3)}
			serviceMust(t, z.Copy(&as[0], &big))
			p := polynomialLexDuplicate(t, m, as)
			var cache *PolynomialCache
			var outRef *PolynomialRef
			if mode == "cache" {
				cache, e = NewPolynomialCache(m)
				serviceMust(t, e)
			}
			if mode == "cra" {
				outRef, e = NewPolynomialRef(c)
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, outRef.Close()) })
			}
			var external []umemory.Block
			// Fixture recovery is distinct from the exact caller cleanup proved below.
			t.Cleanup(func() {
				serviceMust(t, rt.SetMaxSize(0))
				serviceMust(t, c.CleanupDisplay())
				serviceMust(t, m.CleanupLexSort())
				serviceMust(t, m.CleanupEvaluation())
				serviceMust(t, m.CleanupCRA())
				if cache != nil {
					serviceMust(t, cache.Close())
				}
				for i := range external {
					serviceMust(t, rt.Deallocate(&external[i]))
				}
			})
			armed := false
			w := &lexTraceWriter{t: t, m: m, p: p, onHeader: func() {
				armed = true
				remaining := int64(100000) - rt.State().LocalSize - 8
				if remaining < 0 {
					block, e := rt.Allocate(0)
					serviceMust(t, e)
					external = append(external, block)
					remaining = int64(100000) - rt.State().LocalSize - 8
				}
				if remaining < 0 {
					t.Fatal("actual TLS prefix", rt.State())
				}
				block, e := rt.Allocate(uint64(remaining))
				serviceMust(t, e)
				external = append(external, block)
				if rt.State().LocalSize != 100000 {
					t.Fatal("actual TLS boundary", rt.State())
				}
				serviceMust(t, rt.SetMaxSize(1))
			}}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("poly_bug")
			c.monomials.config.Trace = tags
			defer tags.Finalize()
			var receipt polynomialLexSortInvocationReceipt
			value, out := unumeral.NewInt(2), unumeral.NewInt(0)
			var body error
			switch mode {
			case "sort":
				body = m.lexSortWithReceipt(p, nil, &receipt)
			case "hash":
				_, body = m.Hash(p)
			case "evaluation":
				body = m.UnivariateEvaluate(p, 0, &value, &out)
			case "cache":
				_, body = cache.MkUnique(p)
			case "cra":
				body = m.gcdCRA(p, nil, Polynomial{}, nil, outRef)
			}
			serviceMust(t, rt.SetMaxSize(0))
			if !armed || !numericPhysicalReservationRefusal(body) || !m.lexTracePending() || !m.lexPending || c.pendingLexSort != m {
				t.Fatal("accepted lex copy refusal", mode, body, armed)
			}
			display := m.lexTraceDisplay
			objects, blocks := display.frame.numeric.PendingObject(), display.frame.numeric.PendingAllocatorBlocks()
			if objects == (umemory.ObjectRef{}) && blocks == ([3]umemory.Block{}) {
				t.Fatal("actual formatter has no allocated root")
			}
			for _, action := range []func() error{m.CloseChecked, m.SourceFinalize, c.CloseChecked, c.SourceFinalizePrefix} {
				if e := action(); !errors.Is(e, ErrPolynomialPending) {
					t.Fatal("pending borrowed owner retired", e)
				}
			}
			if m.sourceStarted || c.sourceStarted {
				t.Fatal("refused ancestor advanced source destruction")
			}
			if e := m.LexSort(p); !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("sorted shortcut adopted pending formatter", e)
			}
			extra := uint64(0)
			if cache != nil {
				extra += cache.header + cache.imp.bytes
				if !cache.pendingBound || !m.matchesLexSortInvocation(&cache.hashLex) {
					t.Fatal("cache did not bind accepted hash child")
				}
			}
			if outRef != nil {
				extra += outRef.header
				if m.cra.active || m.cra.constructed != 0 || !m.matchesLexSortInvocation(&m.craLex) {
					t.Fatal("CRA prelude constructed later NUMs")
				}
			}
			if mode == "evaluation" && (!m.evaluation.active || m.evaluation.lexGeneration != m.lexGeneration) {
				t.Fatal("evaluator lost exact sort child")
			}
			polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, []*unumeral.Int{&big, &out}, extra, external)
			if mode == "sort" {
				stale := receipt
				stale.generation++
				serviceMust(t, m.cleanupLexSortInvocation(&stale))
				copied := receipt
				copyOwner := *m
				copied.manager = &copyOwner
				serviceMust(t, m.cleanupLexSortInvocation(&copied))
				if !m.lexTracePending() || display.frame.numeric.PendingObject() != objects || display.frame.numeric.PendingAllocatorBlocks() != blocks {
					t.Fatal("stale/copied sort acquired formatter")
				}
			}
			switch mode {
			case "sort":
				serviceMust(t, m.cleanupLexSortInvocation(&receipt))
			case "hash":
				serviceMust(t, m.CleanupLexSort())
			case "evaluation":
				serviceMust(t, m.CleanupEvaluation())
			case "cache":
				serviceMust(t, cache.Close())
				extra = 0
			case "cra":
				serviceMust(t, m.CleanupCRA())
			}
			if m.lexPending || m.lexTracePending() || c.pendingLexSort != nil || c.displayReceiptPending(&display) {
				t.Fatal("actual caller cleanup omitted its formatter", mode)
			}
			polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, []*unumeral.Int{&big, &out}, extra, external)
			next := c.display.next
			serviceMust(t, m.LexSort(p))
			if c.display.next != next {
				t.Fatal("owning cleanup replayed already sorted source suffix")
			}
		})
	}
}

func TestPolynomialLexTraceFalseConditionKeepsFormatterUnconstructed(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("poly_bug")
	m.core.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, m.LexSort(p))
	if text.Len() != 0 || m.core.display.next != 0 || m.lexTraceGeneration != 0 || m.core.pendingLexSort != nil {
		t.Fatal("ordered polynomial entered diagnostic body")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}
func TestPolynomialLexTraceFailedStreamStillUsesDirectDisplay(t *testing.T) {
	polynomialAssertionActionsFixture(t, "C")
	m := polynomialPSCFullFixture(t)
	p := polynomialLexDuplicate(t, m, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(3)})
	writer := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("poly_bug")
	m.core.monomials.config.Trace = tags
	defer tags.Finalize()
	before := m.core.display.next
	serviceMust(t, m.LexSort(p))
	if polynomialSourceDebug && tags.ShouldTrace("poly_bug") {
		if writer.calls != 1 || m.core.display.next-before != 2 {
			t.Fatal("stream refusal skipped direct source term formatters", writer.calls, m.core.display.next-before)
		}
	} else if writer.calls != 0 {
		t.Fatal("erased conditional reached output")
	}
	if m.lexPending || m.lexTracePending() {
		t.Fatal("stream refusal invented a pending computation")
	}
}

func TestPolynomialLexTraceEveryReservation(t *testing.T) {
	if !polynomialSourceDebug || !polynomialTraceEnabledForTest() {
		t.Skip("conditional debug trace erased")
	}
	success, refused := false, false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			polynomialAssertionActionsFixture(t, "T")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			m := polynomialPSCFullFixture(t)
			c := m.core
			var big unumeral.Int
			limbs := make([]uint32, 64)
			limbs[0], limbs[63] = 3, 1
			serviceMust(t, c.integer.SetLimbs(&big, limbs))
			t.Cleanup(func() { serviceMust(t, c.integer.Del(&big)) })
			as := []unumeral.Int{{}, unumeral.NewInt(3), unumeral.NewInt(5)}
			serviceMust(t, c.integer.Copy(&as[0], &big))
			p := polynomialLexDuplicate(t, m, as)
			t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupLexSort()) })
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("poly_bug")
			c.monomials.config.Trace = tags
			defer tags.Finalize()
			c.integer.Budget().FailAfter(prefix)
			body := m.LexSort(p)
			c.integer.Budget().FailAfter(-1)
			if body == nil {
				success = true
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual reservation category", body)
				}
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*unumeral.Int{&big}, 0)
			serviceMust(t, m.CleanupLexSort())
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*unumeral.Int{&big}, 0)
			if m.lexTracePending() {
				t.Fatal("exact owning cleanup retained a formatter")
			}
		}) {
			return
		}
	}
	if !success || !refused {
		t.Fatal("conditional record reservation campaign incomplete", success, refused)
	}
}
