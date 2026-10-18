package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func TestUPolynomialQuadraticPolynomialTraceRecords(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			p, out := upolySourceNew(t, m, []int32{-1, 0, 1}), upolySourceNew(t, m, nil)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
			var text strings.Builder
			bad := factorDegreeRefusedTrace{cause: errors.New("quadratic trace refused")}
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			if refused {
				tags = diagnostic.NewTraceTagsWithWriter(&bad)
			}
			tags.Enable("factor")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			_, e = s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
			serviceMust(t, e)
			serviceMust(t, fs.Multiply(out))
			upolyGCDExpect(t, m, out, []int64{-1, 0, 1})
			if !tags.ShouldTrace("factor") {
				if text.Len() != 0 || bad.calls != 0 {
					t.Fatal("erased quadratic trace evaluated")
				}
				return
			}
			if refused {
				if bad.calls != 1 || bad.flushes != 4 {
					t.Fatal("sticky quadratic records", bad.calls, bad.flushes)
				}
				return
			}
			want := "-------- [factor] factor_2_sqf_pp src/math/polynomial/upolynomial.cpp:2969 ---------\nfactor square free (degree == 2):\nx^2 - 1\n------------------------------------------------\n-------- [factor] factor_2_sqf_pp src/math/polynomial/upolynomial.cpp:2974 ---------\na: 1\nb: 0\nc: -1\n------------------------------------------------\n-------- [factor] factor_2_sqf_pp src/math/polynomial/upolynomial.cpp:2990 ---------\ndisc_sqrt: 2\n------------------------------------------------\n-------- [factor] factor_2_sqf_pp src/math/polynomial/upolynomial.cpp:3003 ---------\nf1: x - 1\nf2: x + 1\n------------------------------------------------\n"
			if text.String() != want {
				t.Fatal("quadratic polynomial records", text.String(), want)
			}
		})
	}
}

// Select the actual source record by accumulating its segmented header.
type quadraticPolynomialQuota struct {
	factorDegreeTraceFailure
	service  *UPolynomialFactorizationService
	header   strings.Builder
	line     int
	caption  string
	captured bool
}

func (w *quadraticPolynomialQuota) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "-------- [") {
		w.header.Reset()
	}
	w.header.Write(p)
	selected := strings.Contains(w.header.String(), fmt.Sprintf("] factor_2_sqf_pp src/math/polynomial/upolynomial.cpp:%d ", w.line))
	if selected && !w.captured && string(p) == w.caption {
		f := &w.service.frontend
		if !f.active || !f.executing {
			w.t.Fatal("quadratic trace lost actual frontend")
		}
		want := uint8(0)
		if w.line == 3003 {
			want = 6
		}
		if f.quadratic.constructed != want {
			w.t.Fatal("wrong quadratic declaration frontier", f.quadratic.constructed, want)
		}
		w.captured = true
		w.trigger = w.caption
	}
	return w.factorDegreeTraceFailure.Write(p)
}
func TestUPolynomialQuadraticPolynomialTraceStops(t *testing.T) {
	for _, site := range []struct {
		line    int
		caption string
	}{{2969, "factor square free (degree == 2):\n"}, {3003, "f1: "}, {3003, "\nf2: "}} {
		t.Run(fmt.Sprint(site.line, site.caption), func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 512)
			m, z := s.manager, q.Integers()
			// (x+3000000001)*(x+3000000003), independently expanded.
			var ns [3]unumeral.Int
			for i, v := range []string{"9000000012000000003", "6000000004", "1"} {
				serviceMust(t, z.SetString(&ns[i], v))
			}
			p := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(p, ns[:]))
			for i := range ns {
				serviceMust(t, z.Del(&ns[i]))
			}
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			w := quadraticPolynomialQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 512, held: make([]unumeral.Int, 512), trigger: "not reached"}, service: s, line: site.line, caption: site.caption}
			t.Cleanup(func() {
				serviceMust(t, s.CleanupRetained())
				for i := 0; i < w.count; i++ {
					serviceMust(t, z.Del(&w.held[i]))
				}
			})
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("factor")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			if !tags.ShouldTrace("factor") {
				t.Skip("quadratic trace erased")
			}
			_, e = s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
			f := &s.frontend
			want := uint8(0)
			if site.line == 3003 {
				want = 6
			}
			if !errors.Is(e, arena.ErrSlotLimit) || !w.captured || !w.armed || !f.active || !m.displayReceiptPending(&f.coreTrace) || f.history.coreTrace != f.coreTrace || f.quadratic.constructed != want || fs.count != 0 {
				t.Fatal("actual quadratic display stop", e, w.captured, f.quadratic.constructed, want)
			}
			source, body := f.sourceCause, f.bodyCause
			serviceMust(t, s.CleanupRetained())
			if f.active || m.pendingFactorCoreTrace != nil || m.displayReceiptPending(&f.coreTrace) || f.sourceCause != source || f.bodyCause != body {
				t.Fatal("quadratic owning cleanup changed source history")
			}
		})
	}
}
