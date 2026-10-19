// Source: src/math/polynomial/algebraic_numbers.cpp:945,951,984,1046-1050.
package polynomial

import (
	"io"
	"strconv"
)

func (m *AlgebraicRootCore) traceSetCorePolynomial(p UPolynomialView) error {
	return m.emitRootTrace("set_core", 945, func(w io.Writer) error {
		out := upolyRemainderTraceStream{w}
		_ = out.AppendCString("set_core p: ")
		if e := m.config.Polynomials.displayIntoWithReceipt(p, out, "x", false, &m.rootTrace.univariate); e != nil {
			return e
		}
		return out.AppendCString("\n")
	})
}

func (m *AlgebraicRootCore) traceSetCoreReset(a *Algebraic) error {
	return m.emitRootTrace("set_core", 951, func(w io.Writer) error {
		out := algebraicPredicateTraceStream{w}
		_ = out.AppendCString("reseting\nresult: ")
		l, e := m.locationCleanup(a)
		if e != nil {
			return e
		}
		if e = m.displayRootAtLocation(out, l, false); e != nil {
			return e
		}
		return out.AppendCString("\n")
	})
}

func (m *AlgebraicRootCore) traceSetCoreAlgebraic() error {
	return m.emitRootTrace("set_core", 984, func(w io.Writer) error {
		return (algebraicPredicateTraceStream{w}).AppendCString("set_core...\n")
	})
}

func (m *AlgebraicRootCore) traceBinaryIntervals(a, b *Algebraic, lv, uv uint32) error {
	return m.emitRootTrace("mk_binary", 1046, func(w io.Writer) error {
		out := algebraicComparisonTraceWriter{w}
		_ = writeAlgebraicIntervalPart(out, "r_i: ")
		if e := m.config.Intervals.Display(out, &m.binary.interval); e != nil {
			return e
		}
		_ = writeAlgebraicIntervalPart(out, "\nlV: "+strconv.FormatUint(uint64(lv), 10)+", uV: "+strconv.FormatUint(uint64(uv), 10)+"\na.m_interval: ")
		l, e := m.locationCleanup(a)
		if e != nil {
			return e
		}
		r, e := m.load(l)
		if e != nil {
			return e
		}
		if e = m.config.Intervals.Display(out, &r.interval); e != nil {
			return e
		}
		_ = writeAlgebraicIntervalPart(out, "\nb.m_interval: ")
		l, e = m.locationCleanup(b)
		if e != nil {
			return e
		}
		r, e = m.load(l)
		if e != nil {
			return e
		}
		if e = m.config.Intervals.Display(out, &r.interval); e != nil {
			return e
		}
		return writeAlgebraicIntervalPart(out, "\n")
	})
}
