package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialModularGCDEntryDisplay(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a, b, g := upolySourceNew(t, m, []int32{2, 3, 1}), upolySourceNew(t, m, []int32{3, 4, 1}), upolySourceNew(t, m, nil)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("mgcd")
	serviceMust(t, m.modular.BindTrace(tags))
	defer m.modular.BindTrace(nil)
	serviceMust(t, m.GCD(a, b, g))
	upolyGCDExpect(t, m, g, []int64{1, 1})
	want := "-------- [mgcd] mod_gcd src/math/polynomial/upolynomial.cpp:726 ---------\nu: x^2 + 3*x + 2\nv: x^2 + 4*x + 3\n------------------------------------------------\n"
	if tags.ShouldTrace("mgcd") {
		if !strings.HasPrefix(text.String(), want) {
			t.Fatal("entry star display", text.String())
		}
	} else if text.Len() != 0 {
		t.Fatal("erased trace evaluated")
	}
}

func TestUPolynomialModularGCDEntryDisplayReceipt(t *testing.T) {
	for _, caption := range []string{"u: ", "\nv: "} {
		t.Run(caption, func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 256)
			m, z := s.manager, q.Integers()
			var big unumeral.Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
			t.Cleanup(func() { serviceMust(t, z.Del(&big)) })
			a := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(a, []unumeral.Int{big, unumeral.NewInt(1)}))
			b, g := upolySourceNew(t, m, nil), upolySourceNew(t, m, []int32{99})
			serviceMust(t, m.SetInts(b, []unumeral.Int{big, unumeral.NewInt(2)}))
			w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: caption}
			t.Cleanup(func() {
				serviceMust(t, m.CleanupModularGCDEntryTrace())
				for i := 0; i < w.count; i++ {
					serviceMust(t, z.Del(&w.held[i]))
				}
			})
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("mgcd")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			if !tags.ShouldTrace("mgcd") {
				t.Skip("source trace erased")
			}
			e := m.GCD(a, b, g)
			receipt := m.modGCDTrace
			if !errors.Is(e, arena.ErrSlotLimit) || !w.armed || !m.displayReceiptPending(&receipt) {
				t.Fatal("actual entry display stop", e)
			}
			// This record precedes every source NUM and primitive-part mutation.
			if m.mgcd[0].size != 0 || m.mgcd[1].size != 0 {
				t.Fatal("entry failure crossed primitive construction")
			}
			out, e := m.descriptor(g)
			// New guarded descriptor access must refuse while the younger display waits.
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("new work crossed display", out, e)
			}
			if m.SourceFinalize() == nil {
				t.Fatal("source finalization crossed display")
			}
			serviceMust(t, m.CleanupModularGCDEntryTrace())
			if m.pendingModGCDTrace != nil || m.displayReceiptPending(&receipt) {
				t.Fatal("accepted entry receipt survived owning cleanup")
			}
			upolyGCDExpect(t, m, g, []int64{99})
			// Stale cleanup must leave a later independently accepted display intact.
			var later upolyDisplayInvocationReceipt
			var text strings.Builder
			e = m.displayIntoWithReceipt(mustUPolyView(t, m, a), factorizationTraceStream{&text}, "x", true, &later)
			if !errors.Is(e, arena.ErrSlotLimit) || !m.displayReceiptPending(&later) {
				t.Fatal("later display receipt", e)
			}
			serviceMust(t, m.CleanupModularGCDEntryTrace())
			if !m.displayReceiptPending(&later) {
				t.Fatal("stale entry cleanup adopted later display")
			}
			serviceMust(t, m.cleanupDisplayReceipt(&later))
		})
	}
}

type modGCDEntryFrontendQuota struct {
	factorDegreeTraceFailure
	service     *UPolynomialFactorizationService
	constructed uint8
	sourceIndex uint32
	vectors     [5]upolyVectorState
	captured    bool
}

func (w *modGCDEntryFrontendQuota) Write(p []byte) (int, error) {
	if !w.captured && string(p) == "u: " {
		q := &w.service.frontend
		if !q.active || !q.executing {
			w.t.Fatal("public modular-GCD entry trace lost actual frontend invocation")
		}
		w.constructed, w.sourceIndex = q.constructed, q.sourceIndex
		w.vectors = [5]upolyVectorState{q.pp, q.prime, q.A, q.B, q.D}
		w.captured = true
	}
	return w.factorDegreeTraceFailure.Write(p)
}
func TestUPolynomialModularGCDEntryRetainsActualFactorFrontend(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	var coefficients [4]unumeral.Int
	for i, value := range []string{"340282366920938463500268095579187314689", "340282366920938463537161583726606417923", "36893488147419103235", "1"} {
		serviceMust(t, z.SetString(&coefficients[i], value))
	}
	// Independently expanded (x+(2^64+1))^2*(x+1), retained by the real Factor API.
	a := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(a, coefficients[:]))
	for i := range coefficients {
		serviceMust(t, z.Del(&coefficients[i]))
	}
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	w := modGCDEntryFrontendQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "u: "}, service: s}
	t.Cleanup(func() {
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("mgcd")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	if !tags.ShouldTrace("mgcd") {
		t.Skip("source trace compiled out")
	}
	_, body := s.Factor(mustUPolyView(t, m, a), fs, DefaultUPolynomialFactorParams())
	frame := &s.frontend
	if !errors.Is(body, arena.ErrSlotLimit) || !w.captured || !w.armed || !frame.active || !m.displayReceiptPending(&frame.modGCDTrace) || frame.constructed != w.constructed || frame.sourceIndex != w.sourceIndex || [5]upolyVectorState{frame.pp, frame.prime, frame.A, frame.B, frame.D} != w.vectors {
		t.Fatal("real frontend source suffix crossed entry trace child", body, frame.constructed, w.constructed)
	}
	if frame.history.modGCDTrace != frame.modGCDTrace || frame.sourceCause == nil {
		t.Fatal("accepted frontend trace/history lost")
	}
	originalSource, originalBody := frame.sourceCause, frame.bodyCause
	// The public default kernel cleanup has no authority over this caller slot.
	serviceMust(t, m.CleanupModularGCDEntryTrace())
	if !m.displayReceiptPending(&frame.modGCDTrace) {
		t.Fatal("default entry cleanup adopted frontend trace")
	}
	serviceMust(t, s.CleanupRetained())
	if frame.active || m.displayReceiptPending(&frame.modGCDTrace) || frame.sourceCause != originalSource || frame.bodyCause != originalBody {
		t.Fatal("entry owning frontend cleanup lost source history")
	}
}
