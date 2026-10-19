// Source: src/math/polynomial/algebraic_numbers.cpp:945,951,984.
package polynomial

import "io"

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
