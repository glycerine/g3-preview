package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestUPolynomialSequenceSturmRemainderTraceTextAndFailbit(t *testing.T) {
	for _, closed := range []bool{false, true} {
		name := "text"
		if closed {
			name = "closed"
		}
		t.Run(name, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{-2, 0, 1})
			v, e := m.wholeView(p)
			serviceMust(t, e)
			var text strings.Builder
			refused := &linearTraceFailWriter{}
			var out io.Writer = &text
			if closed {
				out = refused
			}
			tags := diagnostic.NewTraceTagsWithWriter(out)
			tags.Enable("rem_bug")
			serviceMust(t, m.config.Integer.BindTrace(tags))
			sequence, e := m.NewSequence()
			serviceMust(t, e)
			e = m.SturmSequence(v, sequence)
			serviceMust(t, m.config.Integer.BindTrace(nil))
			serviceMust(t, e)
			if len(sequence.state.sizes) != 3 {
				t.Fatal("independent Sturm row count", sequence.state.sizes)
			}
			if tags.ShouldTrace("rem_bug") {
				if closed {
					if refused.calls != 1 {
						t.Fatal("sticky stream retried", refused.calls)
					}
				} else {
					pos := 0
					for _, body := range []string{"rem loop, p2:\n2 x\nbuffer:\nx^2 - 2\n", "a_m: 1, b_n: 2\n", "rem loop, p2:\n2 x\nbuffer:\n-4\n", "finished\n"} {
						i := strings.Index(text.String()[pos:], body)
						if i < 0 {
							t.Fatal("missing/out-of-order remainder trace", body, text.String())
						}
						pos += i + len(body)
					}
				}
			} else if text.Len() != 0 || refused.calls != 0 {
				t.Fatal("erased trace emitted output")
			}
			if sequence.sturm.active || sequence.sturm.remainderTracePending() {
				t.Fatal("successful trace retained child")
			}
			serviceMust(t, sequence.Close())
		})
	}
}

func TestUPolynomialSequenceSturmRemainderTraceDisplayStopProtectsOlderLocals(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	writer := &rootIDRemainderDisplayStopWriter{t: t, m: m}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("rem_bug")
	serviceMust(t, m.config.Integer.BindTrace(tags))
	sequence, e := m.NewSequence()
	serviceMust(t, e)
	body := m.SturmSequence(v, sequence)
	serviceMust(t, m.config.Integer.BindTrace(nil))
	q, z := &sequence.sturm, m.config.Integer
	if tags.ShouldTrace("rem_bug") {
		child := &q.remainderTrace.displays[0]
		if !writer.reached || !errors.Is(body, arena.ErrBounds) || !m.displayReceiptPending(child) || m.displayReceiptSourceCause(child) == nil || q.scalar.childStop == nil || !q.scalar.constructed || q.scalar.sourceCursor != 0 || q.sourceCursor != ([2]uint32{}) || !q.live[0] || !q.live[1] || z.Cell(writer.ref).Released {
			t.Fatal("display source stop lost exact child/older locals", body, q.sourceCursor)
		}
		for _, state := range []*upolyVectorState{&q.remainder, &sequence.state.coefficients} {
			for _, value := range m.values(state)[:state.size] {
				if value.Cell != 0 && z.Cell(value.Cell).Released {
					t.Fatal("older source coefficient released")
				}
			}
		}
		if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("manager crossed stopped diagnostic", e)
		}
		before, used, stats := q.scalar, z.Budget().Used(), z.RuntimeOwner().Stats()
		if e := q.finishRemainderScalar(nil); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		if before != q.scalar || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Fatal("source scalar finalizer replayed stopped display")
		}
		cell := z.Cell(writer.ref)
		cell.Backing.Requested = writer.requested
		z.Store(writer.ref, cell)
		serviceMust(t, sequence.CleanupSturm())
		if q.active || q.scalar.constructed || q.scalar.sourceCursor != 0 || q.scalar.ownCursor != 1 || q.sourceCursor != ([2]uint32{}) || m.displayReceiptPending(child) {
			t.Fatal("owning cleanup changed source history")
		}
	} else {
		serviceMust(t, body)
		if writer.reached {
			t.Fatal("erased trace reached writer")
		}
	}
	serviceMust(t, m.SturmSequence(v, sequence))
	serviceMust(t, sequence.Close())
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSequenceSturmRemainderTraceReachedReservations(t *testing.T) {
	success, failed, diagnosticFailure := false, false, false
	traceEnabled := false
	for after := int64(0); after < 2048 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			z := m.config.Integer
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("rem_bug")
			serviceMust(t, z.BindTrace(tags))
			traceEnabled = tags.ShouldTrace("rem_bug")
			sequence, e := m.NewSequence()
			serviceMust(t, e)
			z.Budget().FailAfter(after)
			body := m.SturmSequence(v, sequence)
			z.Budget().FailAfter(-1)
			serviceMust(t, z.BindTrace(nil))
			q := &sequence.sturm
			if body == nil {
				success = true
				minus, e := m.SignVariationsMinusInfinity(sequence)
				serviceMust(t, e)
				plus, e := m.SignVariationsPlusInfinity(sequence)
				serviceMust(t, e)
				if minus-plus != 2 {
					t.Fatal("independent two real roots", minus, plus)
				}
			} else {
				failed = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected refusal", body)
				}
				for i := range q.remainderTrace.displays {
					child := &q.remainderTrace.displays[i]
					if m.matchesDisplayReceipt(child) && child.frame.bodyCause != nil {
						diagnosticFailure = true
					}
				}
				for i := range q.remainderTrace.insertions {
					if q.remainderTrace.insertions[i].Active() {
						diagnosticFailure = true
					}
				}
			}
			source, scalarSource := q.sourceCursor, q.scalar.sourceCursor
			serviceMust(t, sequence.CleanupSturm())
			if q.active || q.remainderTracePending() || source != q.sourceCursor || scalarSource != q.scalar.sourceCursor {
				t.Fatal("exact parent cleanup lost source history")
			}
			if m.display.active {
				t.Fatal("parent lost embedded diagnostic display")
			}
			for f := m.displayFrames; f != nil; f = f.next {
				if f.active {
					t.Fatal("parent lost allocating diagnostic display frame")
				}
			}
			serviceMust(t, sequence.Close())
			algebraicComparisonAudit(t, c, &root)
			serviceMust(t, c.Del(&root))
		})
	}
	if !success || !failed || traceEnabled && !diagnosticFailure {
		t.Fatal("public Sturm trace allocation campaign incomplete", success, failed, traceEnabled, diagnosticFailure)
	}
}

func TestUPolynomialSequenceSturmRemainderTraceCallbackRejectsOwnership(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	v, e := m.wholeView(p)
	serviceMust(t, e)
	sequence, e := m.NewSequence()
	serviceMust(t, e)
	z := m.config.Integer
	output := &linearOperationProbeWriter{probe: func() {
		used := z.Budget().Used()
		for _, e := range []error{m.Close(), sequence.Close(), sequence.CleanupSturm(), sequence.Reset()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("trace callback acquired public scope", e)
			}
		}
		context := sequence.sturm.context()
		probe := unumeral.NewInt(0)
		if context.sequenceSturmFrame(z) != nil || z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 7) == nil || probe.Val != 0 || z.Budget().Used() != used {
			t.Fatal("trace callback acquired numeric span")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("rem_bug")
	serviceMust(t, z.BindTrace(tags))
	body := m.SturmSequence(v, sequence)
	serviceMust(t, z.BindTrace(nil))
	serviceMust(t, body)
	if output.called != tags.ShouldTrace("rem_bug") {
		t.Fatal("trace callback/erasure", output.called)
	}
	serviceMust(t, sequence.Close())
}

func TestUPolynomialSequenceSturmRemainderDisabledTraceSkipsOperands(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	serviceMust(t, m.config.Integer.BindTrace(tags))
	defer func() { serviceMust(t, m.config.Integer.BindTrace(nil)) }()
	sequence, e := m.NewSequence()
	serviceMust(t, e)
	q, e := sequence.beginSturm()
	serviceMust(t, e)
	before, used := q.remainderTrace, budget.Used()
	budget.FailAfter(0)
	for phase := uint8(0); phase < 3; phase++ {
		serviceMust(t, q.traceRemainder(phase, UPolynomialView{}, nil, nil))
	}
	budget.FailAfter(-1)
	if before != q.remainderTrace || used != budget.Used() || text.Len() != 0 {
		t.Fatal("disabled trace evaluated operands")
	}
	serviceMust(t, q.finish(nil))
	serviceMust(t, sequence.Close())
}

func TestUPolynomialSequenceSturmRemainderTracePreservesOlderDisplay(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	z := m.config.Integer
	older := upolySourceNew(t, m, nil)
	constant, e := m.read(v, 0)
	serviceMust(t, e)
	serviceMust(t, m.SetInts(older, []unumeral.Int{*constant}))
	ov, e := m.wholeView(older)
	serviceMust(t, e)
	var ref unumeral.NumericCellRef
	var requested uint64
	sink := &upolyDisplaySink{numeralHook: func(string) error {
		ref = m.currentDisplay.numeral.Cell
		cell := z.Cell(ref)
		requested = cell.Backing.Requested
		cell.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(ref, cell)
		return nil
	}}
	var oldReceipt upolyDisplayInvocationReceipt
	if e := m.displayIntoWithReceipt(ov, sink, "x", false, &oldReceipt); !errors.Is(e, arena.ErrBounds) || !m.displayReceiptPending(&oldReceipt) {
		t.Fatal("older display did not stop", e)
	}
	before := *oldReceipt.frame
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("rem_bug")
	serviceMust(t, z.BindTrace(tags))
	sequence, e := m.NewSequence()
	serviceMust(t, e)
	e = m.SturmSequence(v, sequence)
	serviceMust(t, z.BindTrace(nil))
	serviceMust(t, e)
	serviceMust(t, sequence.CleanupSturm())
	if !reflect.DeepEqual(before, *oldReceipt.frame) || !m.displayReceiptPending(&oldReceipt) || z.Cell(ref).Released {
		t.Fatal("public Sturm adopted older display")
	}
	if tags.ShouldTrace("rem_bug") && text.Len() == 0 {
		t.Fatal("trace did not enter fresh display")
	}
	serviceMust(t, sequence.Close())
	if !reflect.DeepEqual(before, *oldReceipt.frame) || !m.displayReceiptPending(&oldReceipt) {
		t.Fatal("sequence close adopted older display")
	}
	cell := z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, m.cleanupDisplayReceipt(&oldReceipt))
	serviceMust(t, c.Del(&root))
}
