// Source: src/math/polynomial/polynomial.cpp:4475-4493; src/util/debug.h;
// src/util/trace.h.
package polynomial

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func polynomialMGCDCheckEnabled() (bool, error) {
	if !polynomialSourceDebug {
		return false, nil
	}
	return diagnostic.BorrowProcessDebugContext().IsDebugEnabled("mgcd_check")
}

func (m *MultivariatePolynomialManager) traceMGCDCheckMismatch(u, v, first, second Polynomial) error {
	c := m.core
	tags := c.monomials.monomialTraceTags()
	if !tags.ShouldTrace("mgcd_bug") {
		return nil
	}
	equal, e := c.Equal(first, second)
	if e != nil || equal {
		return e
	}
	var formatErr error
	_ = tags.EmitConditional("mgcd_bug", "gcd", "src/math/polynomial/polynomial.cpp", 4491,
		func() bool { return true }, func(w io.Writer) error {
			sink := &polynomialTraceSink{writer: w}
			labels := []string{"u: ", "\nv: ", "\nr1: ", "\nr2: "}
			values := []Polynomial{u, v, first, second}
			for i := range values {
				_ = sink.WriteLiteral(labels[i])
				formatErr = c.displayIntoCore(values[i], sink, false)
				if formatErr != nil {
					return formatErr
				}
			}
			return sink.WriteLiteral("\n")
		})
	return formatErr
}
