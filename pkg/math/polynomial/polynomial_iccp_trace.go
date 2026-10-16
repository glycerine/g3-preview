// Source: src/math/polynomial/polynomial.cpp:3454; src/util/trace.h:40-46.
package polynomial

import "io"

func (m *MultivariatePolynomialCore) traceICCPEntry(p Polynomial, x uint32) error {
	tags := m.monomials.monomialTraceTags()
	if !tags.ShouldTrace("polynomial") {
		return nil
	}
	var formatErr error
	_ = tags.EmitConditional("polynomial", "iccp", "src/math/polynomial/polynomial.cpp", 3454,
		func() bool { return true }, func(w io.Writer) error {
			sink := &polynomialTraceSink{writer: w}
			_ = sink.WriteLiteral("iccp x")
			_ = sink.WriteUnsigned(x)
			_ = sink.WriteLiteral("\n")
			formatErr = m.displayIntoCore(p, sink, false)
			if formatErr != nil {
				return formatErr
			}
			return sink.WriteLiteral("\n")
		})
	return formatErr
}
