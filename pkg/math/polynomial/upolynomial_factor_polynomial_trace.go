package polynomial

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (c *upolyFactorSquareFreeScope) tracePending() bool {
	return c.trace.pending() || c.tracePolynomial.owner.displayReceiptPending(&c.tracePolynomial)
}
func (c *upolyFactorSquareFreeScope) traceSourceBlocker() error {
	if e := c.trace.sourceBlocker(); e != nil {
		return e
	}
	r := &c.tracePolynomial
	if r.owner.displayReceiptPending(r) {
		if e := r.owner.displayReceiptSourceCause(r); e != nil {
			return e
		}
		return ErrUPolynomialCleanupPending
	}
	return nil
}
func (c *upolyFactorSquareFreeScope) cleanupTrace() error {
	if e := c.trace.cleanup(); e != nil {
		return e
	}
	r := &c.tracePolynomial
	if r.owner == nil {
		return nil
	}
	return r.owner.cleanupDisplayReceipt(r)
}

// Complete retained source records at cpp1052,1100. The exact accepted
// display receipt, rather than the trace borrow, owns any unfinished child.
func (s *UPolynomialFactorizationService) traceFactorPolynomial(m *UPolynomialManager, f UPolynomialView, line int) error {
	tags := s.manager.modular.BorrowTraceTags()
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		return nil
	}
	c := s.scope
	if c == nil || c.self != c {
		return ErrUPolynomialOwner
	}
	if e := c.traceBorrow.bind(c, tags); e != nil {
		return e
	}
	defer c.traceBorrow.unbind()
	var computation error
	_ = tags.EmitConditional("polynomial::factorization::bughunt", "factor_square_free", "src/math/polynomial/upolynomial_factorization.cpp", line,
		func() bool { return true }, func(raw io.Writer) error {
			out := factorizationTraceWriter{raw}
			label, variable := "sage: f_pp = ", "x"
			switch line {
			case 1052:
				label = "sage: f_pp = "
			case 1100:
				_ = linearWriteText(out, "sage: Rp.<x_p> = GF(")
				prime, e := s.manager.modular.String(&c.p)
				if e != nil {
					computation = e
					return e
				}
				_ = linearWriteText(out, prime)
				_ = linearWriteText(out, ")['x_p']")
				_ = diagnostic.WriteTraceEndl(out)
				label, variable = "sage: f_pp_zp = ", "x_p"
			}
			_ = linearWriteText(out, label)
			if e := m.displayIntoWithReceipt(f, factorizationTraceStream{out}, variable, false, &c.tracePolynomial); e != nil {
				computation = e
				return e
			}
			_ = diagnostic.WriteTraceEndl(out)
			switch line {
			case 1052:
				_ = linearWriteText(out, "sage: if (not (f_pp * 1 == f): print 'Error, content computation wrong'")
				_ = diagnostic.WriteTraceEndl(out)
			}
			return nil
		})
	return computation
}
