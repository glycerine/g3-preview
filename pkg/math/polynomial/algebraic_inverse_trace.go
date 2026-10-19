// Source: src/math/polynomial/algebraic_numbers.cpp:1630,1641,1644.
package polynomial

import "io"

func (m *AlgebraicRootCore) traceInverseValue(a *Algebraic, after bool) error {
	tags := m.config.Rationals.Integers().BorrowTraceTags()
	if !tags.ShouldTrace("algebraic_bug") {
		return nil
	}
	line, label := 1630, "before inv: "
	if after {
		line, label = 1644, "after inv: "
	}
	var computation error
	_ = tags.EmitConditional("algebraic_bug", "inv", "src/math/polynomial/algebraic_numbers.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			out := algebraicPredicateTraceStream{w}
			_ = out.AppendCString(label)
			location, e := m.locationCleanup(a)
			if e == nil {
				e = m.displayRootAtLocation(out, location, false)
				if m.rootDisplay.active {
					m.inversion.traceDisplayPending = true
					m.inversion.traceStop = e
				}
			}
			if e == nil {
				_ = out.AppendCString("\n")
				location, e = m.locationCleanup(a)
				if e == nil {
					e = m.displayIntervalAtLocation(algebraicComparisonTraceWriter{w}, location)
				}
			}
			computation = e
			if e == nil {
				_ = out.AppendCString("\n")
			}
			return e
		})
	return computation
}

func (m *AlgebraicRootCore) traceInverseBounds() error {
	tags := m.config.Rationals.Integers().BorrowTraceTags()
	if !tags.ShouldTrace("algebraic_bug") {
		return nil
	}
	var computation error
	_ = tags.EmitConditional("algebraic_bug", "inv", "src/math/polynomial/algebraic_numbers.cpp", 1641,
		func() bool { return true }, func(w io.Writer) error {
			out := algebraicPredicateTraceStream{w}
			_ = out.AppendCString("inv new_bounds: ")
			for i := range m.inversion.values {
				if i != 0 {
					_ = out.AppendCString(", ")
				}
				text, e := m.config.Rationals.String(&m.inversion.values[i])
				if e != nil {
					computation = e
					return e
				}
				_ = out.AppendCString(text)
			}
			_ = out.AppendCString("\n")
			return nil
		})
	return computation
}
