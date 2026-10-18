// Source: upolynomial.cpp:532,535,551; reached by get_root_id's Sturm loop.
package polynomial

import (
	"bytes"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"strconv"
)

type upolyRootIDRemainderTrace struct {
	displays   [2]upolyDisplayInvocationReceipt
	insertions [2]unumeral.NumeralInsertionReceipt
}

// Stream insertion failures set the diagnostic ostream failbit. Numerical
// display/to_string computations still execute at their source positions.
type upolyRemainderTraceStream struct{ io.Writer }

func (s upolyRemainderTraceStream) AppendCString(v string) error {
	_, _ = io.WriteString(s.Writer, v)
	return nil
}
func (s upolyRemainderTraceStream) AppendNumeralText(v string) error { return s.AppendCString(v) }
func (s upolyRemainderTraceStream) AppendInt32(v int32) error {
	return s.AppendCString(strconv.FormatInt(int64(v), 10))
}

func (q *upolyRootIDScope) remainderTracePending() bool {
	for i := range q.remainderTrace.displays {
		if q.owner.displayReceiptPending(&q.remainderTrace.displays[i]) {
			return true
		}
	}
	for i := range q.remainderTrace.insertions {
		if q.remainderTrace.insertions[i].Active() {
			return true
		}
	}
	return false
}
func (q *upolyRootIDScope) remainderTraceSourceCause() error {
	for i := range q.remainderTrace.displays {
		if e := q.owner.displayReceiptSourceCause(&q.remainderTrace.displays[i]); e != nil {
			return e
		}
	}
	for i := range q.remainderTrace.insertions {
		if e := q.remainderTrace.insertions[i].SourceCause(); e != nil {
			return e
		}
	}
	return nil
}
func (q *upolyRootIDScope) cleanupRemainderTrace(abort bool) error {
	z := q.owner.config.Integer
	for i := len(q.remainderTrace.insertions) - 1; i >= 0; i-- {
		r := &q.remainderTrace.insertions[i]
		if r.Active() {
			if e := r.Complete(z.Naturals(), abort); e != nil {
				return e
			}
		}
	}
	if !abort {
		for i := len(q.remainderTrace.displays) - 1; i >= 0; i-- {
			if e := q.owner.cleanupDisplayReceipt(&q.remainderTrace.displays[i]); e != nil {
				return e
			}
		}
	}
	return nil
}

func (q *upolyRootIDScope) traceRemainder(phase uint8, b UPolynomialView, r *upolyVectorState, bn *unumeral.Int) error {
	m := q.owner
	tags := m.config.Integer.BorrowTraceTags()
	if !tags.ShouldTrace("rem_bug") {
		return nil
	}
	if q.context().rootIDFrame(m.config.Integer) != q || !q.scalar.constructed {
		return ErrUPolynomialOwner
	}
	if q.remainderTracePending() {
		return ErrUPolynomialCleanupPending
	}
	line := 532
	if phase == 1 {
		line = 535
	} else if phase == 2 {
		line = 551
	}
	var computation error
	m.callbackDepth++
	defer func() { m.callbackDepth-- }()
	_ = tags.EmitConditional("rem_bug", "rem", "src/math/polynomial/upolynomial.cpp", line, func() bool { return true }, func(w io.Writer) error {
		out := upolyRemainderTraceStream{w}
		var e error
		switch phase {
		case 0:
			_ = out.AppendCString("rem loop, p2:\n")
			e = m.displayIntoWithReceipt(b, out, "x", false, &q.remainderTrace.displays[0])
			if e == nil {
				_ = out.AppendCString("\nbuffer:\n")
				e = m.displayIntoWithReceipt(m.stateView(r), out, "x", false, &q.remainderTrace.displays[1])
			}
			if e == nil {
				_ = out.AppendCString("\n")
			}
		case 1:
			_ = out.AppendCString("finished\n")
		case 2:
			for i, a := range [2]*unumeral.Int{&q.scalar.value, bn} {
				if i == 0 {
					_ = out.AppendCString("a_m: ")
				} else {
					_ = out.AppendCString(", b_n: ")
				}
				var text bytes.Buffer
				e = m.config.Integer.InsertSourceIntegerTracked(upolyNormalizeStringStream{&text}, a, &q.remainderTrace.insertions[i], nil)
				if e != nil {
					break
				}
				_ = out.AppendCString(text.String())
			}
			if e == nil {
				_ = out.AppendCString("\n")
			}
		}
		computation = e
		return e
	})
	return computation
}
