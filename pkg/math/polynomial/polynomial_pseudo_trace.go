// Source: src/math/polynomial/polynomial.cpp:4909,4955-4962,4973.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
)

type polynomialPseudoTraceBorrow struct{ executing bool }

func (b *polynomialPseudoTraceBorrow) NumericExecuting() bool { return b.executing }
func (m *MultivariatePolynomialCore) emitPseudoTrace(line int, body func(*polynomialTraceSink) error) error {
	tags := m.monomials.monomialTraceTags()
	if !tags.ShouldTrace("polynomial") {
		return nil
	}
	borrow := polynomialPseudoTraceBorrow{executing: true}
	var integer, natural unumeral.NumericCallerBinding
	if e := m.integer.BindCaller(&integer, &borrow, 0); e != nil {
		return e
	}
	defer func() {
		if e := integer.Unbind(); e != nil {
			panic(e)
		}
	}()
	if e := m.integer.Natural.BindCaller(&natural, &borrow, 0); e != nil {
		return e
	}
	defer func() {
		if e := natural.Unbind(); e != nil {
			panic(e)
		}
	}()
	previous := m.quadraticTrace
	m.quadraticTrace = &polynomialQuadraticTraceBorrow{tags: tags}
	defer func() { m.quadraticTrace = previous }()
	var computation error
	_ = tags.EmitConditional("polynomial", "pseudo_division_core", "src/math/polynomial/polynomial.cpp", line, func() bool { return true }, func(w io.Writer) error { computation = body(&polynomialTraceSink{writer: w}); return computation })
	return computation
}
func (m *MultivariatePolynomialCore) pseudoTracePolynomial(out *polynomialTraceSink, label string, p Polynomial, direct bool) error {
	_ = out.WriteLiteral(label)
	r := &m.pseudoDivision.traceDisplay
	if m.displayReceiptPending(r) {
		return ErrPolynomialPending
	}
	*r = polynomialDisplayInvocationReceipt{}
	if direct {
		return m.displayPolynomialDirectWithReceipt(p, out, false, r)
	}
	return m.displayIntoCoreWithReceipt(p, out, false, r)
}
func (m *MultivariatePolynomialCore) tracePseudoEntry(p, q Polynomial, x uint32) error {
	return m.emitPseudoTrace(4909, func(out *polynomialTraceSink) error {
		if e := m.pseudoTracePolynomial(out, "pseudo_division\np: ", p, true); e != nil {
			return e
		}
		if e := m.pseudoTracePolynomial(out, "\nq: ", q, true); e != nil {
			return e
		}
		_ = out.WriteLiteral("\nx: ")
		_ = out.WriteUnsigned(x)
		return out.WriteLiteral("\n")
	})
}
func (m *MultivariatePolynomialCore) tracePseudoIteration(p, q Polynomial, Q, R *polynomialReference, d *uint32) error {
	return m.emitPseudoTrace(4955, func(out *polynomialTraceSink) error {
		s := &m.pseudoDivision
		if e := m.pseudoTracePolynomial(out, "A: ", p, true); e != nil {
			return e
		}
		if e := m.pseudoTracePolynomial(out, "\nB: ", q, true); e != nil {
			return e
		}
		if e := m.pseudoTraceReference(out, "\nl_B: ", &s.refs[pseudoLeadB], true); e != nil {
			return e
		}
		if e := m.pseudoTraceReference(out, "\nr_B: ", &s.refs[pseudoReductB], true); e != nil {
			return e
		}
		if e := m.pseudoTraceReference(out, "\nQ: ", Q, true); e != nil {
			return e
		}
		if e := m.pseudoTraceReference(out, "\nR: ", R, true); e != nil {
			return e
		}
		_ = out.WriteLiteral("\nd: ")
		_ = out.WriteUnsigned(*d)
		return out.WriteLiteral("\n")
	})
}
func (m *MultivariatePolynomialCore) tracePseudoAdjusted() error {
	return m.emitPseudoTrace(4973, func(out *polynomialTraceSink) error {
		if e := m.pseudoTraceReference(out, "l_B_e: ", &m.pseudoDivision.refs[pseudoAdjusted], false); e != nil {
			return e
		}
		return out.WriteLiteral("\n")
	})
}

func (m *MultivariatePolynomialCore) pseudoTraceReference(out *polynomialTraceSink, label string, r *polynomialReference, direct bool) error {
	_ = out.WriteLiteral(label)
	return m.pseudoTracePolynomial(out, "", r.value, direct)
}
