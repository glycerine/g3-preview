package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

type factorEntryTraceProbe struct {
	t       *testing.T
	s       *UPolynomialFactorizationService
	text    strings.Builder
	checked bool
}

func (w *factorEntryTraceProbe) Write(p []byte) (int, error) {
	if !w.checked && string(p) == "sage: f = " {
		w.checked = true
		if w.s.scope != nil || w.s.diagnostic.active {
			w.t.Fatal("entry TRACE followed younger source scope")
		}
		for _, e := range []error{w.s.Close(), w.s.CleanupRetained(), w.s.manager.Close(), w.s.manager.SourceFinalize(), w.s.manager.CleanupDisplay()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("entry callback acquired executing owner", e)
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialFactorEntryTraceBeforePrimitiveDebug(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	w := factorEntryTraceProbe{t: t, s: s}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	p := upolySourceNew(t, m, []int32{2, 3, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	_, e = s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		if w.checked || w.text.Len() != 0 {
			t.Fatal("erased entry trace")
		}
		return
	}
	text := w.text.String()
	entry := "sage: f = x^2 + 3 x + 2\nsage: if (not f.is_squarefree()): print 'Error, f is not square-free'\nsage: print 'Factoring :', f\nsage: print 'Expected factors: ', f.factor()\n"
	if !w.checked || !strings.Contains(text, entry) || strings.Index(text, ":1020 ") >= strings.Index(text, ":1052 ") {
		t.Fatal("whole entry trace/source order", text)
	}
}

func TestUPolynomialFactorEntryTraceExactChildBeforeAnyLocal(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	var coefficients [3]unumeral.Int
	serviceMust(t, z.SetString(&coefficients[0], "1606938044258990275541962092344965554322887681987282944917506"))
	serviceMust(t, z.SetString(&coefficients[1], "-2535301200456458802993406410755"))
	coefficients[2] = unumeral.NewInt(1)
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&p)) })
	serviceMust(t, m.SetInts(p, coefficients[:]))
	for i := range coefficients {
		serviceMust(t, z.Del(&coefficients[i]))
	}
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "sage: f = "}
	t.Cleanup(func() {
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	var receipt upolyFactorSquareFreeInvocationReceipt
	_, body := s.factorSquareFreeWithReceipt(mustUPolyView(t, m, p), fs, 1, DefaultUPolynomialFactorParams(), &receipt)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		return
	}
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || s.scope != nil || s.diagnostic.active || receipt.frame != nil || !m.displayReceiptPending(&receipt.entry) {
		t.Fatal("entry actual child/frontier", body, w.armed, s.scope, s.diagnostic.active)
	}
	if !s.matchesSquareFreeHistory(&receipt) || s.factorSquareFreeInvocationSourceCause(&receipt) == nil {
		t.Fatal("accepted exact entry receipt lost")
	}
	extras := upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}}
	for i := 0; i < w.count; i++ {
		extras.Integers = append(extras.Integers, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, extras)
	serviceMust(t, s.cleanupFactorSquareFreeInvocation(&receipt, true))
	if m.displayReceiptPending(&receipt.entry) || s.scope != nil {
		t.Fatal("exact entry owning cleanup")
	}

	// A later independently accepted display is never owned by the old entry.
	var foreign upolyDisplayInvocationReceipt
	sink := upolyDisplaySink{}
	later := m.displayIntoWithReceipt(mustUPolyView(t, m, p), &sink, "x", false, &foreign)
	if !errors.Is(later, arena.ErrSlotLimit) || !m.displayReceiptPending(&foreign) {
		t.Fatal("independent replacement refusal", later)
	}
	if e := s.cleanupFactorSquareFreeInvocation(&receipt, true); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("stale entry receipt accepted", e)
	}
	serviceMust(t, s.CleanupRetained())
	if !m.displayReceiptPending(&foreign) {
		t.Fatal("entry cleanup acquired independent display")
	}
	serviceMust(t, m.cleanupDisplayReceipt(&foreign))
}

func TestUPolynomialFactorEntryTraceDisabledSkipsView(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(nil)
	serviceMust(t, s.manager.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, s.manager.modular.BindTrace(nil)) })
	before := s.manager.config.Integer.Budget().Used()
	s.manager.config.Integer.Budget().FailAfter(0)
	t.Cleanup(func() { s.manager.config.Integer.Budget().FailAfter(-1) })
	serviceMust(t, s.traceFactorEntry(UPolynomialView{}, nil))
	if s.manager.config.Integer.Budget().Used() != before || s.entryTrace.owner != nil || s.entryTraceBorrow.executing {
		t.Fatal("disabled entry acquired owner")
	}
}
