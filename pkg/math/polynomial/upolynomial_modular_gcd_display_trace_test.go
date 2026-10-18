package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type modGCDBigDisplayQuota struct {
	factorDegreeTraceFailure
	m       *UPolynomialManager
	caption string
	result  *upolyVectorState
}

func (w *modGCDBigDisplayQuota) Write(p []byte) (int, error) {
	if !w.armed && string(p) == w.caption {
		state := &w.m.mgcd[4]
		if w.caption == "new combined:\n" {
			state = &w.m.mgcd[5]
		}
		if w.caption == "r: " {
			state = w.result
		}
		for _, n := range w.m.values(state)[:state.size] {
			if !n.IsSmall() {
				w.trigger = w.caption
				break
			}
		}
	}
	return w.factorDegreeTraceFailure.Write(p)
}
func TestUPolynomialModularGCDLaterDisplayReceipts(t *testing.T) {
	for _, caption := range []string{"new combined:\n", "candidate:\n", "r: "} {
		t.Run(caption, func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 512)
			m, z := s.manager, q.Integers()
			a, b, g := upolySourceNew(t, m, nil), upolySourceNew(t, m, nil), upolySourceNew(t, m, []int32{99})
			// Independent expansions (x+N)(x+1), (x+N)(x+2), N=2^64+1.
			for i, coeffs := range [][]string{{"18446744073709551617", "18446744073709551618", "1"}, {"36893488147419103234", "18446744073709551619", "1"}} {
				var values [3]unumeral.Int
				for j, v := range coeffs {
					serviceMust(t, z.SetString(&values[j], v))
				}
				dest := a
				if i == 1 {
					dest = b
				}
				serviceMust(t, m.SetInts(dest, values[:]))
				for j := range values {
					serviceMust(t, z.Del(&values[j]))
				}
			}
			out, e := m.descriptor(g)
			serviceMust(t, e)
			w := modGCDBigDisplayQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 512, held: make([]unumeral.Int, 512), trigger: "not reached"}, m: m, caption: caption, result: &out.state}
			t.Cleanup(func() {
				serviceMust(t, m.CleanupModularGCDTrace())
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
			e = m.GCD(a, b, g)
			if !errors.Is(e, arena.ErrSlotLimit) || !w.armed || !m.displayReceiptPending(&m.modGCDTrace) || m.modGCDScalars.constructed != 6 || m.modGCDScalars.ownRemaining != 6 {
				t.Fatal("actual later display source stop", e, m.modGCDScalars.constructed, m.modGCDScalars.ownRemaining)
			}
			if caption == "r: " {
				if out.state.size != 2 {
					t.Fatal("result display preceded source swap")
				}
			} else if out.state.size != 1 || m.values(&out.state)[0] != unumeral.NewInt(99) {
				t.Fatal("output published before source swap")
			}
			source, body := m.modGCDScalars.sourceCause, m.modGCDScalars.bodyCause
			if m.SourceFinalize() == nil {
				t.Fatal("source retirement crossed display")
			}
			serviceMust(t, m.CleanupModularGCDTrace())
			if m.modGCDScalars.constructed != 6 || m.modGCDScalars.ownRemaining != 0 || m.modGCDScalars.sourceCause != source || m.modGCDScalars.bodyCause != body {
				t.Fatal("owning cleanup rewrote source history")
			}
		})
	}
}

func TestUPolynomialModularGCDLaterDisplayRecords(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{-200000, -99998, 1})
	b := upolySourceNew(t, m, []int32{-300000, -99997, 1})
	g := upolySourceNew(t, m, nil)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("mgcd")
	serviceMust(t, m.modular.BindTrace(tags))
	defer m.modular.BindTrace(nil)
	serviceMust(t, m.GCD(a, b, g))
	upolyGCDExpect(t, m, g, []int64{-100000, 1})
	if tags.ShouldTrace("mgcd") {
		for _, part := range []string{"new q:\nx + 17309\n", "new combined:\nx - 100000\n", "candidate:\nx - 100000\n", "r: x - 100000\n"} {
			if !strings.Contains(text.String(), part) {
				t.Fatal("source display", part, text.String())
			}
		}
	}
}

func TestUPolynomialModularGCDLaterDisplayRetainsFactorFrontend(t *testing.T) {
	for _, caption := range []string{"new combined:\n", "candidate:\n"} {
		t.Run(caption, func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 512)
			m, z := s.manager, q.Integers()
			// Independently expanded (x+3000000001)^2.
			var coefficients [3]unumeral.Int
			for i, v := range []string{"9000000006000000001", "6000000002", "1"} {
				serviceMust(t, z.SetString(&coefficients[i], v))
			}
			a := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(a, coefficients[:]))
			for i := range coefficients {
				serviceMust(t, z.Del(&coefficients[i]))
			}
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			w := modGCDBigDisplayQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 512, held: make([]unumeral.Int, 512), trigger: "not reached"}, m: m, caption: caption}
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
				t.Skip("source trace erased")
			}
			_, e = s.Factor(mustUPolyView(t, m, a), fs, DefaultUPolynomialFactorParams())
			f := &s.frontend
			if !errors.Is(e, arena.ErrSlotLimit) || !w.armed || !f.active || !m.displayReceiptPending(&f.modGCDTrace) || f.modularGCD.constructed != 6 || f.modularGCD.sourceCause == nil || f.history.modGCDTrace != f.modGCDTrace {
				t.Fatal("actual modular-GCD frontend source stop", e, f.modularGCD.constructed)
			}
			source, body := f.modularGCD.sourceCause, f.modularGCD.bodyCause
			serviceMust(t, m.CleanupModularGCDTrace())
			if !m.displayReceiptPending(&f.modGCDTrace) {
				t.Fatal("default cleanup adopted frontend display")
			}
			serviceMust(t, s.CleanupRetained())
			if f.active || m.displayReceiptPending(&f.modGCDTrace) || f.modularGCD.sourceCause != source || f.modularGCD.bodyCause != body {
				t.Fatal("owning cleanup lost frontend source history")
			}
		})
	}
}
