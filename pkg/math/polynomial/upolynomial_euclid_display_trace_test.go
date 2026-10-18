package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialEuclidPolynomialRecords(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(map[bool]string{false: "records", true: "refused"}[refused], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a, b, g := upolySourceNew(t, m, []int32{2, 3, 1}), upolySourceNew(t, m, []int32{3, 4, 1}), upolySourceNew(t, m, nil)
			var text strings.Builder
			bad := factorDegreeRefusedTrace{cause: errors.New("Euclid stream refused")}
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			if refused {
				tags = diagnostic.NewTraceTagsWithWriter(&bad)
			}
			tags.Enable("upolynomial")
			serviceMust(t, m.modular.BindTrace(tags))
			defer m.modular.BindTrace(nil)
			serviceMust(t, m.EuclidGCDViews(mustUPolyView(t, m, a), mustUPolyView(t, m, b), g))
			upolyGCDExpect(t, m, g, []int64{1, 1})
			if !tags.ShouldTrace("upolynomial") {
				if text.Len() != 0 || bad.calls != 0 {
					t.Fatal("erased trace evaluated")
				}
				return
			}
			if refused {
				if bad.calls != 1 || bad.flushes != 5 {
					t.Fatal("sticky writes/footer flushes", bad.calls, bad.flushes)
				}
				return
			}
			for _, body := range []string{"A: x^2 + 3 x + 2\nB: x^2 + 4 x + 3\n", "A: x^2 + 4 x + 3\nB: -1 x - 1\n", "A: -1 x - 1\nB: 0\n", "GCD\nx^2 + 3 x + 2\nx^2 + 4 x + 3\n--->\nx + 1\n"} {
				if !strings.Contains(text.String(), body) {
					t.Fatal("source Euclid record", body, text.String())
				}
			}
			if strings.Count(text.String(), ":836 ") != 3 || strings.Count(text.String(), ":847 ") != 1 {
				t.Fatal("loop/terminal record counts", text.String())
			}
		})
	}
}
func TestUPolynomialEuclidDisplayReceipts(t *testing.T) {
	for _, caption := range []string{"A: ", "\nB: ", "GCD\n", "\n--->\n"} {
		t.Run(caption, func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 512)
			m, z := s.manager, q.Integers()
			// (x+N)(x+1), (x+N)(x+2), N=2^64+1.
			a, b, g := upolySourceNew(t, m, nil), upolySourceNew(t, m, nil), upolySourceNew(t, m, []int32{99})
			for i, coeffs := range [][]string{{"18446744073709551617", "18446744073709551618", "1"}, {"36893488147419103234", "18446744073709551619", "1"}} {
				var ns [3]unumeral.Int
				for j, v := range coeffs {
					serviceMust(t, z.SetString(&ns[j], v))
				}
				dst := a
				if i == 1 {
					dst = b
				}
				serviceMust(t, m.SetInts(dst, ns[:]))
				for j := range ns {
					serviceMust(t, z.Del(&ns[j]))
				}
			}
			out, e := m.descriptor(g)
			serviceMust(t, e)
			w := factorDegreeTraceFailure{t: t, z: z, limit: 512, held: make([]unumeral.Int, 512), trigger: caption}
			t.Cleanup(func() {
				serviceMust(t, m.CleanupEuclideanGCDTrace())
				for i := 0; i < w.count; i++ {
					serviceMust(t, z.Del(&w.held[i]))
				}
			})
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("upolynomial")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			if !tags.ShouldTrace("upolynomial") {
				t.Skip("source trace erased")
			}
			e = m.EuclidGCDViews(mustUPolyView(t, m, a), mustUPolyView(t, m, b), g)
			receipt := m.euclidTrace
			if !errors.Is(e, arena.ErrSlotLimit) || !w.armed || !m.displayReceiptPending(&receipt) {
				t.Fatal("actual Euclid display stop", e)
			}
			if caption == "A: " || caption == "\nB: " {
				if out.state.size != 1 || m.values(&out.state)[0] != unumeral.NewInt(99) {
					t.Fatal("loop trace published result")
				}
			} else if out.state.size != 2 {
				t.Fatal("terminal trace preceded final swap")
			}
			if m.SourceFinalize() == nil {
				t.Fatal("source retirement crossed display")
			}
			serviceMust(t, m.CleanupEuclideanGCDTrace())
			if m.pendingEuclidTrace != nil || m.displayReceiptPending(&receipt) {
				t.Fatal("accepted Euclid display cleanup")
			}
			var later upolyDisplayInvocationReceipt
			var text strings.Builder
			e = m.displayIntoWithReceipt(mustUPolyView(t, m, a), factorizationTraceStream{&text}, "x", false, &later)
			if !errors.Is(e, arena.ErrSlotLimit) || !m.displayReceiptPending(&later) {
				t.Fatal("later independent display", e)
			}
			serviceMust(t, m.CleanupEuclideanGCDTrace())
			if !m.displayReceiptPending(&later) {
				t.Fatal("stale Euclid cleanup adopted replacement")
			}
			serviceMust(t, m.cleanupDisplayReceipt(&later))
		})
	}
}

func TestUPolynomialEuclidDisplayRetainsActualFactorFallback(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 512)
	m, z := s.manager, q.Integers()
	var p, twice, square unumeral.Int
	serviceMust(t, z.SetInt64(&p, 1))
	for _, prime := range upolyBigPrimes {
		factor := unumeral.NewInt(int32(prime))
		serviceMust(t, z.Mul(&p, &factor, &p))
	}
	two := unumeral.NewInt(2)
	serviceMust(t, z.Mul(&p, &two, &twice))
	serviceMust(t, z.Mul(&p, &p, &square))
	a := upolySourceNew(t, m, nil)
	// Independent expansion (1+P*x)^2. Every retained prime kills its leading
	// coefficient, so the real modular-GCD body reaches its Euclidean fallback.
	serviceMust(t, m.SetInts(a, []unumeral.Int{unumeral.NewInt(1), twice, square}))
	serviceMust(t, z.Del(&square))
	serviceMust(t, z.Del(&twice))
	serviceMust(t, z.Del(&p))
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	w := factorDegreeTraceFailure{t: t, z: z, limit: 512, held: make([]unumeral.Int, 512), trigger: "A: "}
	t.Cleanup(func() {
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("upolynomial")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	if !tags.ShouldTrace("upolynomial") {
		t.Skip("source trace erased")
	}
	_, e = s.Factor(mustUPolyView(t, m, a), fs, DefaultUPolynomialFactorParams())
	f := &s.frontend
	if !errors.Is(e, arena.ErrSlotLimit) || !w.armed || !f.active || !m.displayReceiptPending(&f.euclidTrace) || f.modularGCD.constructed != 6 || f.modularGCD.sourceCause == nil || f.history.euclidTrace != f.euclidTrace || f.image.invocation != uint64(len(upolyBigPrimes)) {
		t.Fatal("actual fallback Euclid display stop", e, f.modularGCD.constructed, f.image.invocation)
	}
	source, body := f.modularGCD.sourceCause, f.modularGCD.bodyCause
	serviceMust(t, m.CleanupEuclideanGCDTrace())
	if !m.displayReceiptPending(&f.euclidTrace) {
		t.Fatal("default cleanup adopted actual frontend")
	}
	serviceMust(t, s.CleanupRetained())
	if f.active || m.displayReceiptPending(&f.euclidTrace) || f.modularGCD.sourceCause != source || f.modularGCD.bodyCause != body {
		t.Fatal("owning cleanup rewrote fallback source history")
	}
}
