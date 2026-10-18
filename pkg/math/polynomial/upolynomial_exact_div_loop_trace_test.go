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

func TestUPolynomialExactDivLoopRecordsAndStickyStream(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "refused"}[refused], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a, b := upolySourceNew(t, m, []int32{-1, 0, 1}), upolySourceNew(t, m, []int32{-1, 1})
			out := upolySourceNew(t, m, []int32{99})
			var text strings.Builder
			failure := factorDegreeRefusedTrace{cause: errors.New("loop stream refusal")}
			var tags *diagnostic.TraceTags
			if refused {
				tags = diagnostic.NewTraceTagsWithWriter(&failure)
			} else {
				tags = diagnostic.NewTraceTagsWithWriter(&text)
			}
			tags.Enable("upolynomial")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			ok, e := m.ExactDiv(a, b, out)
			serviceMust(t, e)
			if !ok {
				t.Fatal("actual exact quotient rejected")
			}
			got := upolyDivisionValues(t, m, out)
			if len(got) != 2 || got[0] != 1 || got[1] != 1 {
				t.Fatal("loop trace changed quotient", got)
			}
			if tags.ShouldTrace("upolynomial") {
				if refused {
					if failure.calls != 1 || failure.flushes != 3 {
						t.Fatal("source record footer count/sticky physical writes", failure.calls, failure.flushes)
					}
				} else {
					for _, part := range []string{"exact_div loop...\nx^2 - 1\n0\n", "exact_div loop...\nx - 1\nx\n", "exact_div loop...\n0\nx + 1\n"} {
						if !strings.Contains(text.String(), part) {
							t.Fatal("actual exact loop state", part, text.String())
						}
					}
					if strings.Count(text.String(), ":628 ") != 3 {
						t.Fatal("missing terminal empty-remainder record", text.String())
					}
				}
			} else if text.Len() != 0 || failure.calls != 0 {
				t.Fatal("erased exact loop record executed")
			}
		})
	}
}

type exactDivLoopQuota struct {
	factorDegreeTraceFailure
	target, headers int
}

func (w *exactDivLoopQuota) Write(p []byte) (int, error) {
	if string(p) == "exact_div loop...\n" {
		w.headers++
		if w.headers == w.target {
			w.trigger = string(p)
		}
	}
	return w.factorDegreeTraceFailure.Write(p)
}
func TestUPolynomialExactDivLoopExactDisplayCleanup(t *testing.T) {
	for _, target := range []int{1, 2} {
		t.Run(map[int]string{1: "remainder", 2: "partial-quotient"}[target], func(t *testing.T) {
			q, s := preparationPhysicalFixtureWithSlots(t, 256)
			m, z := s.manager, q.Integers()
			var big, plus unumeral.Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
			serviceMust(t, z.Copy(&plus, &big))
			serviceMust(t, z.Inc(&plus))
			t.Cleanup(func() { serviceMust(t, z.Del(&plus)); serviceMust(t, z.Del(&big)) })
			a := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(a, []unumeral.Int{unumeral.NewInt(1), plus, big}))
			b := upolySourceNew(t, m, []int32{1, 1})
			out := upolySourceNew(t, m, []int32{99})
			destination, e := m.descriptor(out)
			serviceMust(t, e)
			w := exactDivLoopQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "not-a-source-insertion"}, target: target}
			t.Cleanup(func() {
				serviceMust(t, m.CleanupExactDivisionTrace())
				for i := 0; i < w.count; i++ {
					serviceMust(t, z.Del(&w.held[i]))
				}
			})
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("upolynomial")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			if !tags.ShouldTrace("upolynomial") {
				t.Skip("source trace compiled out")
			}
			ok, body := m.ExactDiv(a, b, out)
			receipt := m.exactDivTrace
			if ok || !errors.Is(body, arena.ErrSlotLimit) || !w.armed || w.headers != target || !m.displayReceiptPending(&receipt) || destination.state.size != 1 || m.values(&destination.state)[0] != unumeral.NewInt(99) {
				t.Fatal("exact loop display/output frontier", ok, body, w.headers)
			}
			if m.exactDiv.size != 2 || m.div1.size != uint32(4-target) {
				t.Fatal("private kernel state was discarded", m.exactDiv, m.div1)
			}
			if _, e := m.NewPolynomial(); !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("new work crossed exact display child", e)
			}
			serviceMust(t, m.CleanupExactDivisionTrace())
			if m.displayReceiptPending(&receipt) || m.pendingExactDivTrace != nil {
				t.Fatal("exact accepted display cleanup failed")
			}
			// A later independent display may reuse the embedded frame. The old exact
			// receipt grants no cleanup authority over the new accepted invocation.
			var independent upolyDisplayInvocationReceipt
			var sink strings.Builder
			e = m.displayIntoWithReceipt(mustUPolyView(t, m, a), factorizationTraceStream{&sink}, "x", false, &independent)
			if !errors.Is(e, arena.ErrSlotLimit) || !m.displayReceiptPending(&independent) {
				t.Fatal("independent display refusal", e)
			}
			serviceMust(t, m.CleanupExactDivisionTrace())
			if !m.displayReceiptPending(&independent) {
				t.Fatal("stale exact receipt cleaned replacement display")
			}
			serviceMust(t, m.cleanupDisplayReceipt(&independent))
		})
	}
}

type exactDivFrontendQuota struct {
	factorDegreeTraceFailure
	service     *UPolynomialFactorizationService
	constructed uint8
	sourceIndex uint32
	vectors     [5]upolyVectorState
	captured    bool
}

func (w *exactDivFrontendQuota) Write(p []byte) (int, error) {
	if !w.captured && string(p) == "exact_div loop...\n" {
		q := &w.service.frontend
		if !q.active || !q.executing {
			w.t.Fatal("public exact division trace lost actual frontend invocation")
		}
		w.constructed, w.sourceIndex = q.constructed, q.sourceIndex
		w.vectors = [5]upolyVectorState{q.pp, q.prime, q.A, q.B, q.D}
		w.captured = true
	}
	return w.factorDegreeTraceFailure.Write(p)
}
func TestUPolynomialExactDivLoopRetainsActualFactorFrontend(t *testing.T) {
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
	w := exactDivFrontendQuota{factorDegreeTraceFailure: factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "exact_div loop...\n"}, service: s}
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
		t.Skip("source trace compiled out")
	}
	_, body := s.Factor(mustUPolyView(t, m, a), fs, DefaultUPolynomialFactorParams())
	frame := &s.frontend
	if !errors.Is(body, arena.ErrSlotLimit) || !w.captured || !w.armed || !frame.active || !m.displayReceiptPending(&frame.exactTrace) || frame.constructed != w.constructed || frame.sourceIndex != w.sourceIndex || [5]upolyVectorState{frame.pp, frame.prime, frame.A, frame.B, frame.D} != w.vectors {
		t.Fatal("real frontend source suffix crossed exact trace child", body, frame.constructed, w.constructed)
	}
	if frame.history.exactTrace != frame.exactTrace || frame.sourceCause == nil {
		t.Fatal("accepted frontend trace/history lost")
	}
	originalSource, originalBody := frame.sourceCause, frame.bodyCause
	// The public default kernel cleanup has no authority over this caller slot.
	serviceMust(t, m.CleanupExactDivisionTrace())
	if !m.displayReceiptPending(&frame.exactTrace) {
		t.Fatal("default exact cleanup adopted frontend trace")
	}
	serviceMust(t, s.CleanupRetained())
	if frame.active || m.displayReceiptPending(&frame.exactTrace) || frame.sourceCause != originalSource || frame.bodyCause != originalBody {
		t.Fatal("exact owning frontend cleanup lost source history")
	}
}

type exactDivRecombinationBudget struct {
	budget  *arena.Budget
	ordinal int64
	armed   bool
}

func (w *exactDivRecombinationBudget) Write(p []byte) (int, error) {
	if !w.armed && string(p) == "exact_div loop...\n" {
		w.armed = true
		w.budget.FailAfter(w.ordinal)
	}
	return len(p), nil
}
func TestUPolynomialExactDivLoopRetainsRecombinationCaller(t *testing.T) {
	probe := diagnostic.NewTraceTags()
	probe.Enable("upolynomial")
	if !probe.ShouldTrace("upolynomial") {
		t.Skip("source trace compiled out")
	}
	reached := false
	for ordinal := int64(0); ordinal < 128 && !reached; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m, z := s.manager, q.Integers()
			tagsProbe := diagnostic.NewTraceTags()
			tagsProbe.Enable("upolynomial")
			if !tagsProbe.ShouldTrace("upolynomial") {
				t.Skip("source trace compiled out")
			}
			c, e := s.newSquareFreeScope()
			serviceMust(t, e)
			// Construct every concrete enclosing owner through the source trial
			// declaration, rather than assigning a construction count to absent owners.
			c.zp, e = NewRelatedUPolynomialManager(m)
			serviceMust(t, e)
			c.best, e = c.zp.NewFactors()
			serviceMust(t, e)
			serviceMust(t, c.ds.init(c.best))
			c.zpe, e = NewRelatedUPolynomialManager(m)
			serviceMust(t, e)
			seven := unumeral.NewInt(7)
			serviceMust(t, c.zpe.SetZP(&seven))
			c.lifted, e = c.zpe.NewFactors()
			serviceMust(t, e)
			var lifted upolyVectorState
			for _, constant := range []int32{1, 2} {
				serviceMust(t, c.zpe.set(&lifted, []unumeral.Int{unumeral.NewInt(constant), unumeral.NewInt(1)}))
				serviceMust(t, c.lifted.PushCopy(c.zpe.stateView(&lifted), 1))
			}
			serviceMust(t, c.zpe.reset(&lifted))
			c.zpe.freeBacking(&lifted)
			serviceMust(t, c.it.init(c.lifted, &c.ds))
			c.leading = unumeral.NewInt(2)
			serviceMust(t, m.set(&c.trial, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}))
			var big, fourBig, fourNext unumeral.Int
			words := make([]uint32, 80)
			words[0] = 1
			words[79] = 1 << 31
			serviceMust(t, z.SetLimbs(&big, words))
			four := unumeral.NewInt(4)
			serviceMust(t, z.Mul(&big, &four, &fourBig))
			serviceMust(t, z.Add(&fourBig, &four, &fourNext))
			// B=2^2559+1 is 2 modulo 7, consistent with the actual lifted factors.
			serviceMust(t, m.set(&c.fpp, []unumeral.Int{fourBig, fourNext, four}))
			c.constructed = 14 // All preceding actual fields/default numerals now exist.
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() {
				q.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, z.Del(&fourNext))
				serviceMust(t, z.Del(&fourBig))
				serviceMust(t, z.Del(&big))
			})
			w := exactDivRecombinationBudget{budget: q.Budget(), ordinal: ordinal}
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("upolynomial")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			_, body := s.publishRecombinationTrial(fs, 1, true)
			q.Budget().FailAfter(-1)
			frame := &c.recombinationStep
			if !frame.exactTrace.owner.displayReceiptPending(&frame.exactTrace) {
				return
			}
			reached = true
			if !w.armed || !numericPhysicalReservationRefusal(body) || !frame.active || frame.sourceCause == nil || c.constructed != 14 || fs.count != 0 {
				t.Fatal("recombination trace source frontier", body, frame.active, c.constructed)
			}
			source, bodyCause, receipt := frame.sourceCause, frame.bodyCause, frame.exactTrace
			serviceMust(t, m.CleanupExactDivisionTrace())
			if !receipt.owner.displayReceiptPending(&receipt) {
				t.Fatal("default exact cleanup adopted recombination child")
			}
			serviceMust(t, s.CleanupRetained())
			if frame.active || receipt.owner.displayReceiptPending(&receipt) || frame.sourceCause != source || frame.bodyCause != bodyCause {
				t.Fatal("exact recombination owning cleanup lost source history")
			}
		})
	}
	if !reached {
		t.Fatal("actual recombination retained display prefix not reached")
	}
}
