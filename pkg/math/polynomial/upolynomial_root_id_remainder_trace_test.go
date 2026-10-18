package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestUPolynomialRootIDRemainderTraceTextAndFailbit(t *testing.T) {
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
			endpoint := unumeral.NewBinaryInt(1)
			id, e := m.GetRootID(v, &endpoint)
			serviceMust(t, m.config.Integer.BindTrace(nil))
			serviceMust(t, e)
			if id != 1 {
				t.Fatal("independent root count below 1", id)
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
			if m.rootID.active || m.rootID.remainderTracePending() {
				t.Fatal("successful trace retained child")
			}
		})
	}
}

func TestUPolynomialRootIDRemainderTraceCallbackRejectsOwnership(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	v, e := m.wholeView(p)
	serviceMust(t, e)
	z := m.config.Integer
	output := &linearOperationProbeWriter{probe: func() {
		used := z.Budget().Used()
		for _, e := range []error{m.Close(), m.CleanupRootID(), m.DelPolynomial(&p)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("trace callback acquired root owner", e)
			}
		}
		context := m.rootID.context()
		probe := unumeral.NewInt(0)
		if context.rootIDFrame(z) != nil || z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 7) == nil || probe.Val != 0 || z.Budget().Used() != used {
			t.Fatal("trace callback acquired numeric span")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("rem_bug")
	serviceMust(t, z.BindTrace(tags))
	endpoint := unumeral.NewBinaryInt(1)
	id, e := m.GetRootID(v, &endpoint)
	serviceMust(t, z.BindTrace(nil))
	serviceMust(t, e)
	if id != 1 || output.called != tags.ShouldTrace("rem_bug") {
		t.Fatal("trace callback/erasure", id, output.called)
	}
}

func TestUPolynomialRootIDRemainderDisabledTraceSkipsOperands(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	serviceMust(t, m.config.Integer.BindTrace(tags))
	defer func() { serviceMust(t, m.config.Integer.BindTrace(nil)) }()
	var receipt upolyRootIDInvocationReceipt
	q, e := m.beginRootID(UPolynomialView{}, &receipt)
	serviceMust(t, e)
	before, used := q.remainderTrace, budget.Used()
	budget.FailAfter(0)
	for phase := uint8(0); phase < 3; phase++ {
		serviceMust(t, q.traceRemainder(phase, UPolynomialView{}, nil, nil))
	}
	budget.FailAfter(-1)
	if before != q.remainderTrace || used != budget.Used() || text.Len() != 0 {
		t.Fatal("disabled remainder trace evaluated operands")
	}
	serviceMust(t, q.finish(nil))
}

type rootIDRemainderDisplayStopWriter struct {
	t         *testing.T
	m         *UPolynomialManager
	reached   bool
	ref       unumeral.NumericCellRef
	requested uint64
}

func (w *rootIDRemainderDisplayStopWriter) Write(p []byte) (int, error) {
	if !w.reached && string(p) == "55340232221128654849" {
		q := w.m.currentDisplay
		if q == nil || !q.constructed {
			w.t.Fatal("missing actual nested display NUM")
		}
		z := w.m.config.Integer
		w.ref = q.numeral.Cell
		if !unumeral.ValidNumericRef(z.Cells, w.ref) || z.Cell(w.ref).Released {
			w.t.Fatal("missing actual nested display backing")
		}
		cell := z.Cell(w.ref)
		w.requested = cell.Backing.Requested
		cell.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(w.ref, cell)
		w.reached = true
	}
	return len(p), nil
}

func TestUPolynomialRootIDRemainderTraceDisplayStopProtectsOlderLocals(t *testing.T) {
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
	var receipt upolyRootIDInvocationReceipt
	_, body := m.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	serviceMust(t, m.config.Integer.BindTrace(nil))
	q, z := &m.rootID, m.config.Integer
	if tags.ShouldTrace("rem_bug") {
		child := &q.remainderTrace.displays[0]
		if !writer.reached || !errors.Is(body, arena.ErrBounds) || !m.displayReceiptPending(child) || m.displayReceiptSourceCause(child) == nil || q.scalar.childStop == nil || !q.scalar.constructed || q.scalar.sourceCursor != 0 || q.sourceCursor != ([3]uint32{}) || !q.live[0] || !q.live[1] || !q.live[2] || z.Cell(writer.ref).Released {
			t.Fatal("display source stop lost exact child/older locals", body, q.sourceCursor)
		}
		for _, state := range []*upolyVectorState{&q.remainder, &q.sequence.coefficients} {
			for _, value := range m.values(state)[:state.size] {
				if value.Cell != 0 && z.Cell(value.Cell).Released {
					t.Fatal("older source coefficient released")
				}
			}
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
		serviceMust(t, m.cleanupRootIDWithReceipt(&receipt))
		if q.active || q.scalar.constructed || q.scalar.sourceCursor != 0 || q.scalar.ownCursor != 1 || q.sourceCursor != ([3]uint32{}) || m.displayReceiptPending(child) {
			t.Fatal("owning cleanup changed source history")
		}
	} else {
		serviceMust(t, body)
		if writer.reached {
			t.Fatal("erased trace reached writer")
		}
	}
	id, e := m.GetRootID(v, r.interval.BorrowLower())
	serviceMust(t, e)
	if id != 0 {
		t.Fatal("independent root count below lower1", id)
	}
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialRootIDRemainderTracePreservesOlderDisplay(t *testing.T) {
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
	var receipt upolyRootIDInvocationReceipt
	id, e := m.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
	serviceMust(t, z.BindTrace(nil))
	serviceMust(t, e)
	serviceMust(t, m.cleanupRootIDWithReceipt(&receipt))
	if id != 0 || !reflect.DeepEqual(before, *oldReceipt.frame) || !m.displayReceiptPending(&oldReceipt) || z.Cell(ref).Released {
		t.Fatal("root-ID adopted older display", id)
	}
	if tags.ShouldTrace("rem_bug") && text.Len() == 0 {
		t.Fatal("trace did not enter fresh display")
	}
	cell := z.Cell(ref)
	cell.Backing.Requested = requested
	z.Store(ref, cell)
	serviceMust(t, m.cleanupDisplayReceipt(&oldReceipt))
	serviceMust(t, c.Del(&root))
}
