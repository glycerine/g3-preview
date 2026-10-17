// Source: algebraic_numbers.cpp:1861.
package polynomial

import "io"

// Suppress insertion failures only. The diagnostic stream keeps its failbit,
// while interval/numeral computations continue at the source operand positions.
type algebraicComparisonTraceWriter struct{ io.Writer }

func (w algebraicComparisonTraceWriter) Write(p []byte) (int, error) {
	_, _ = w.Writer.Write(p)
	return len(p), nil
}

func (m *AlgebraicRootCore) traceComparison(a, b *Algebraic) error {
	if m == nil || m.self != m {
		return nil
	}
	tags := m.config.Rationals.Integers().BorrowTraceTags()
	if !tags.ShouldTrace("algebraic") {
		return nil
	}
	if e := m.check(); e != nil {
		return e
	}
	m.comparisonTraceRunning = true
	defer func() { m.comparisonTraceRunning = false }()
	var computation error
	_ = tags.EmitConditional("algebraic", "compare", "src/math/polynomial/algebraic_numbers.cpp", 1861,
		func() bool { return true }, func(w io.Writer) error {
			out := algebraicComparisonTraceWriter{w}
			_ = writeAlgebraicIntervalPart(out, "comparing: ")
			l, e := m.locationCleanup(a)
			if e == nil {
				e = m.displayIntervalAtLocation(out, l)
			}
			if e != nil {
				computation = e
				return e
			}
			_ = writeAlgebraicIntervalPart(out, " ")
			l, e = m.locationCleanup(b)
			if e == nil {
				e = m.displayIntervalAtLocation(out, l)
			}
			if e != nil {
				computation = e
				return e
			}
			_ = writeAlgebraicIntervalPart(out, "\n")
			return nil
		})
	return computation
}
