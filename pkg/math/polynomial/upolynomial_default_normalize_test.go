package polynomial

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialDefaultNormalizeTraceSourceActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		for _, refused := range []bool{false, true} {
			t.Run(action+fmt.Sprint(refused), func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				p := upolySourceNew(t, m, []int32{3, 4})
				divisor := unumeral.NewInt(2)
				var text strings.Builder
				failed := &linearTraceFailWriter{}
				var out io.Writer = &text
				if refused {
					out = failed
				}
				tags := diagnostic.NewTraceTagsWithWriter(out)
				tags.Enable("div_bug")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Independently supplied divisor2 to coefficient3, not a rewritten GCD.
				body := m.normalizeCoefficient(mustUPolyView(t, m, p), 0, &divisor)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
						t.Fatal("source identity action", body)
					}
				} else {
					serviceMust(t, body)
				}
				upolyGCDExpect(t, m, p, []int64{1, 4})
				if polynomialSourceDebug && tags.ShouldTrace("div_bug") {
					if refused {
						if failed.calls != 1 {
							t.Fatal("sticky output", failed.calls)
						}
					} else if !strings.Contains(text.String(), "old(p[i]): 3, g: 2, p[i]: 1, tmp: 2\n") {
						t.Fatal(text.String())
					}
				} else if text.Len() != 0 || failed.calls != 0 {
					t.Fatal("erased DEBUG trace")
				}
				if m.defaultNormalization.active || m.defaultNormalization.gConstructed || m.defaultNormalization.coefficientConstructed != 0 {
					t.Fatal("completed direct coefficient retained fictitious GCD/local")
				}
			})
		}
	}
}

func TestUPolynomialDefaultNormalizeActualReservationPrefixes(t *testing.T) {
	success, failed := false, false
	for after := int64(0); after < 512 && !success; after++ {
		t.Run(fmt.Sprint(after), func(t *testing.T) {
			_, m, _ := algebraicNegTLSFixture(t, nil)
			z, budget := m.config.Integer, m.config.Integer.Budget()
			p := upolySourceNew(t, m, nil)
			var ns [3]unumeral.Int
			for i, text := range []string{"18446744073709551616", "55340232221128654848", "36893488147419103232"} {
				serviceMust(t, z.SetString(&ns[i], text))
			}
			serviceMust(t, m.SetInts(p, ns[:]))
			for i := range ns {
				serviceMust(t, z.Del(&ns[i]))
			}
			budget.FailAfter(after)
			body := m.Normalize(p)
			budget.FailAfter(-1)
			q := &m.defaultNormalization
			if body == nil {
				success = true
				upolyGCDExpect(t, m, p, []int64{1, 3, 2})
			} else {
				failed = true
				var failure umemory.Failure
				if !errors.Is(body, arena.ErrBudget) && !errors.As(body, &failure) {
					t.Fatal("unexpected refusal", body)
				}
			}
			// Audit real immediate owners before acquiring owning cleanup.
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil)
			originalBody, originalSource := q.bodyCause, q.sourceCause
			coefficient, g := q.coefficientConstructed, q.gConstructed
			serviceMust(t, m.CleanupNormalization())
			for i := range q.raw {
				if z.MatchesRawInvocation(&q.raw[i]) {
					t.Fatal("accepted failed raw child survived owning cleanup", i)
				}
			}
			if z.MatchesCopyAllocationInvocation(&q.copy) || m.modular.MatchesPNormalizeInvocation(&q.normalize) || unumeral.NumericAllocationPending(&q.numeric) {
				t.Fatal("accepted copy/normalization/allocation child survived cleanup")
			}
			for i := range q.insertions {
				if q.insertions[i].Active() {
					t.Fatal("accepted formatting child survived cleanup", i)
				}
			}
			if q.active || q.coefficientConstructed != coefficient || q.gConstructed != g || q.bodyCause != originalBody || q.sourceCause != originalSource {
				t.Fatal("owning cleanup rewrote source history")
			}
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil)
		})
	}
	if !success || !failed {
		t.Fatal("reservation campaign did not reach failure and success", success, failed)
	}
}

type defaultNormalizeReadGuard struct {
	craTraceReadGuard
	value   *unumeral.Int
	caption string
	refused bool
}

func (w *defaultNormalizeReadGuard) Write(p []byte) (int, error) {
	if string(p) == w.caption && w.ref == 0 {
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupNormalization()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("normalization output callback acquired owner", e)
			}
		}
		z := w.m.config.Integer
		context := w.m.defaultNormalization.context()
		var probe unumeral.Int
		if context.defaultNormalizationFrame(z) != nil || z.WithAllocationContext(context.utility(z)).SetInt64(&probe, 7) == nil || probe != (unumeral.Int{}) {
			w.t.Fatal("output callback acquired numerical context")
		}
		if w.value.IsSmall() {
			w.t.Fatal("selected real operand not BIG")
		}
		w.ref = w.value.Cell
		w.original = z.Cell(w.ref)
		size, e := w.original.Backing.ReadWord(z.Allocator, 0)
		serviceMust(w.t, e)
		serviceMust(w.t, z.RuntimeOwner().WriteUint32(w.raw, w.offset, size))
		projected := w.original
		projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = w.query, w.raw, w.offset
		z.Store(w.ref, projected)
		if w.refused {
			return 0, errors.New("default normalization outer trace refused")
		}
	}
	return len(p), nil
}

func TestUPolynomialDefaultNormalizeActualTraceReadStops(t *testing.T) {
	if !polynomialSourceDebug || !polynomialTraceEnabledForTest() {
		t.Skip("DEBUG trace erased")
	}
	for index, caption := range []string{"old(p[i]): ", ", g: ", ", p[i]: ", ", tmp: "} {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(index, refused), func(t *testing.T) {
				_, m, _ := algebraicNegTLSFixture(t, nil)
				z := m.config.Integer
				divisor := unumeral.NewInt(2)
				var coefficient unumeral.Int
				if index == 1 {
					coefficient = unumeral.NewInt(3)
					serviceMust(t, z.SetString(&divisor, "1606938044258990275541962092341162602522202993782792835301377"))
				} else {
					serviceMust(t, z.SetString(&coefficient, "1606938044258990275541962092341162602522202993782792835301379"))
				}
				t.Cleanup(func() { serviceMust(t, z.Del(&divisor)) })
				p := upolySourceNew(t, m, nil)
				serviceMust(t, m.SetInts(p, []unumeral.Int{coefficient, unumeral.NewInt(1)}))
				serviceMust(t, z.Del(&coefficient))
				d, e := m.descriptor(p)
				serviceMust(t, e)
				v := m.stateView(&d.state)
				scope := &m.defaultNormalization
				values := [4]*unumeral.Int{&scope.values[1], &divisor, &m.values(&d.state)[0], &scope.values[2]}
				allocator := z.SourceAllocator()
				query, e := allocator.Allocate(4096)
				serviceMust(t, e)
				raw, offset, e := allocator.RawLocation(query)
				serviceMust(t, e)
				w := defaultNormalizeReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, value: values[index], caption: caption, refused: refused}
				t.Cleanup(func() {
					w.restore()
					serviceMust(t, m.CleanupNormalization())
					serviceMust(t, allocator.Deallocate(4096, query))
				})
				tags := diagnostic.NewTraceTagsWithWriter(&w)
				tags.Enable("div_bug")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				// Actual direct source coefficient helper with an independent divisor.
				// No computed GCD, numerical scratch, or construction counters are changed.
				body := m.normalizeCoefficient(v, 0, &divisor)
				if !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || !scope.active || !scope.insertions[index].Active() {
					t.Fatal("actual trace child read", body, w.ref, scope.active)
				}
				want := uint8(0)
				if index == 0 {
					want = 1
				}
				if index == 3 {
					want = 2
				}
				if scope.coefficientConstructed != want || scope.gConstructed {
					t.Fatal("literal source NUM suffix", scope.coefficientConstructed, want)
				}
				if want > 0 && !errors.Is(scope.sourceCause, umemory.ErrRuntimeUninitialized) {
					t.Fatal("actual source destructor stop missing", scope.sourceCause)
				}
				if want == 0 && scope.sourceCause != nil {
					t.Fatal("unjoined body incorrectly stopped source destruction", scope.sourceCause)
				}
				if m.SourceFinalize() == nil {
					t.Fatal("source finalize crossed retained default normalization")
				}
				originalBody, originalSource := scope.bodyCause, scope.sourceCause
				w.restore()
				// The later independent formatter uses another genuine BIG caller numeral.
				var laterValue unumeral.Int
				serviceMust(t, z.SetString(&laterValue, "1606938044258990275541962092341162602522202993782792835301381"))
				t.Cleanup(func() { serviceMust(t, z.Del(&laterValue)) })
				original := z.Cell(laterValue.Cell)
				size, e := original.Backing.ReadWord(z.Allocator, 0)
				serviceMust(t, e)
				serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, size))
				projected := original
				projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
				z.Store(laterValue.Cell, projected)
				var later unumeral.NumeralInsertionReceipt
				_, e = m.modular.SourceStringTracked(&laterValue, &later)
				z.Store(laterValue.Cell, original)
				t.Cleanup(func() { serviceMust(t, m.modular.CleanupSourceString(&later)) })
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) || !later.Active() {
					t.Fatal("later independent accepted formatter", e)
				}
				character := later.Character
				serviceMust(t, m.CleanupNormalization())
				if scope.active || scope.coefficientConstructed != want || scope.bodyCause != originalBody || scope.sourceCause != originalSource {
					t.Fatal("owning cleanup changed source history")
				}
				if !later.Active() || later.Character != character {
					t.Fatal("cleanup adopted later formatting generation")
				}
				serviceMust(t, m.modular.CleanupSourceString(&later))
			})
		}
	}
}

func TestUPolynomialDefaultNormalizeContextGenerationAndEarlyReturns(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	zero := upolySourceNew(t, m, nil)
	serviceMust(t, m.Normalize(zero))
	if m.defaultNormalization.invocation != 0 {
		t.Fatal("empty input constructed GCD")
	}
	constant := upolySourceNew(t, m, []int32{-7})
	serviceMust(t, m.Normalize(constant))
	upolyGCDExpect(t, m, constant, []int64{-1})
	if m.defaultNormalization.invocation != 0 {
		t.Fatal("singleton input constructed GCD")
	}
	p := upolySourceNew(t, m, []int32{6, 12})
	serviceMust(t, m.Normalize(p))
	upolyGCDExpect(t, m, p, []int64{1, 2})
	old := m.defaultNormalization.context()
	q, e := m.beginDefaultNormalization(false)
	serviceMust(t, e)
	defer func() { serviceMust(t, q.finish(nil)) }()
	var probe unumeral.Int
	used := budget.Used()
	budget.FailAfter(0)
	e = m.config.Integer.WithAllocationContext(old.utility(m.config.Integer)).SetInt64(&probe, 17)
	budget.FailAfter(-1)
	if e == nil || probe != (unumeral.Int{}) || budget.Used() != used || old.defaultNormalizationFrame(m.config.Integer) != nil {
		t.Fatal("stale generation acquired later invocation", e)
	}
}

func TestUPolynomialDefaultNormalizeEndedReceiptDoesNotOwnLater(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	var old upolyDefaultNormalizeReceipt
	first, e := m.beginDefaultNormalizationWithReceipt(false, &old)
	serviceMust(t, e)
	serviceMust(t, first.finish(nil))
	later, e := m.beginDefaultNormalization(false)
	serviceMust(t, e)
	generation := later.invocation
	serviceMust(t, m.cleanupDefaultNormalizationReceipt(&old))
	if !later.active || !later.executing || later.invocation != generation {
		t.Fatal("ended receipt acquired later default invocation")
	}
	serviceMust(t, later.finish(nil))
}
