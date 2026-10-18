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

func TestUPolynomialFactorCoreTraceRecordsAndStickyStream(t *testing.T) {
	flushes := 0
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			p, out := upolySourceNew(t, m, []int32{-1, 3, -3, 1}), upolySourceNew(t, m, nil)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
			var text strings.Builder
			bad := factorDegreeRefusedTrace{cause: errors.New("core trace refused")}
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			if refused {
				tags = diagnostic.NewTraceTagsWithWriter(&bad)
			}
			tags.Enable("factor")
			tags.Enable("factor_bug")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			_, e = s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
			serviceMust(t, e)
			serviceMust(t, fs.Multiply(out))
			upolyGCDExpect(t, m, out, []int64{-1, 3, -3, 1})
			if !tags.ShouldTrace("factor") {
				if text.Len() != 0 || bad.calls != 0 {
					t.Fatal("erased core trace executed")
				}
				return
			}
			if refused {
				if bad.calls != 1 || bad.flushes != flushes {
					t.Fatal("sticky physical writes/footer count", bad.calls, bad.flushes, flushes)
				}
				return
			}
			flushes = strings.Count(text.String(), "-------- [")
			// Only factor_core records; factor_bug also reaches exact_div pointer records.
			var records strings.Builder
			for _, record := range strings.SplitAfter(text.String(), "------------------------------------------------\n") {
				if strings.Contains(record, "] factor_core src/math/polynomial/upolynomial.cpp:") {
					records.WriteString(record)
				}
			}
			var want strings.Builder
			add := func(line int, body string) {
				tag := "factor_bug"
				if line == 3080 || line == 3093 {
					tag = "factor"
				}
				fmt.Fprintf(&want, "-------- [%s] factor_core src/math/polynomial/upolynomial.cpp:%d ---------\n%s------------------------------------------------\n", tag, line, body)
			}
			add(3075, "C: x^3 - 3 x^2 + 3 x - 1\nB: x^2 - 2 x + 1\nA: x - 1\n")
			for j := 1; j <= 3; j++ {
				b := "x^2 - 2 x + 1"
				if j == 2 {
					b = "x - 1"
				}
				if j == 3 {
					b = "1"
				}
				add(3080, fmt.Sprintf("factor_core main loop j: %d\nA: x - 1\nB: %s\n", j, b))
				d := "x - 1"
				if j < 3 {
					add(3093, "const C: 1\n")
				} else {
					d = "1"
				}
				add(3099, "B: "+b+"\nD: "+d+"\n")
			}
			add(3106, "A: 1\n")
			if records.String() != want.String() {
				t.Fatal("core source records", records.String(), "want", want.String())
			}
		})
	}
}

type factorCoreTraceQuota struct {
	header strings.Builder
	factorDegreeTraceFailure
	service     *UPolynomialFactorizationService
	target      int
	selected    bool
	captured    bool
	constructed uint8
	sourceIndex uint32
	vectors     [5]upolyVectorState
}

func (w *factorCoreTraceQuota) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "-------- [") {
		w.header.Reset()
	}
	w.header.Write(p)
	w.selected = strings.Contains(w.header.String(), fmt.Sprintf("] factor_core src/math/polynomial/upolynomial.cpp:%d ", w.target))

	caption := "C: "
	if w.target == 3080 {
		caption = "A: "
	}
	if w.target == 3099 {
		caption = "B: "
	}
	if w.selected && !w.captured && string(p) == caption {
		q := &w.service.frontend
		if !q.active || !q.executing {
			w.t.Fatal("trace lost actual frontend")
		}
		w.constructed, w.sourceIndex = q.constructed, q.sourceIndex
		w.vectors = [5]upolyVectorState{q.pp, q.prime, q.A, q.B, q.D}
		w.captured = true
		w.trigger = caption
	}
	return w.factorDegreeTraceFailure.Write(p)
}
func TestUPolynomialFactorCoreTraceRetainsActualFrontend(t *testing.T) {
	for _, line := range []int{3075, 3080, 3099} {
		t.Run(fmt.Sprint(line), func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 512)
			m, z := s.manager, q.Integers()
			// Independently expanded (x+3000000001)^3.
			var coefficients [4]unumeral.Int
			for i, v := range []string{"27000000027000000009000000001", "27000000018000000003", "9000000003", "1"} {
				serviceMust(t, z.SetString(&coefficients[i], v))
			}
			p := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(p, coefficients[:]))
			for i := range coefficients {
				serviceMust(t, z.Del(&coefficients[i]))
			}
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			w := factorCoreTraceQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 512, held: make([]unumeral.Int, 512), trigger: "not reached"}, service: s, target: line}
			t.Cleanup(func() {
				serviceMust(t, s.CleanupRetained())
				for i := 0; i < w.count; i++ {
					serviceMust(t, z.Del(&w.held[i]))
				}
			})
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("factor")
			tags.Enable("factor_bug")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			if !tags.ShouldTrace("factor") {
				t.Skip("core trace erased")
			}
			_, e = s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
			f := &s.frontend
			if !errors.Is(e, arena.ErrSlotLimit) || !w.captured || !w.armed || !f.active || !m.displayReceiptPending(&f.coreTrace) || f.history.coreTrace != f.coreTrace || f.constructed != w.constructed || f.sourceIndex != w.sourceIndex || [5]upolyVectorState{f.pp, f.prime, f.A, f.B, f.D} != w.vectors {
				t.Fatal("actual core display source stop", e, w.captured, f.constructed, w.constructed)
			}
			body, source := f.bodyCause, f.sourceCause
			if m.SourceFinalize() == nil {
				t.Fatal("source retirement crossed accepted core display")
			}
			serviceMust(t, s.CleanupRetained())
			if f.active || m.displayReceiptPending(&f.coreTrace) || m.pendingFactorCoreTrace != nil || f.bodyCause != body || f.sourceCause != source {
				t.Fatal("owning core cleanup lost source history")
			}
		})
	}
}
