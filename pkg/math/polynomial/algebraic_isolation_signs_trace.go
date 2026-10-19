// Source: src/math/polynomial/algebraic_numbers.cpp:2598.
package polynomial

import "io"

func (m *AlgebraicRootCore) traceIsolationSignsSample() error {
	tags := m.config.Rationals.Integers().BorrowTraceTags()
	if !tags.ShouldTrace("isolate_roots_bug") {
		return nil
	}
	m.isolationSigns.traceRunning = true
	defer func() { m.isolationSigns.traceRunning = false }()
	var computation error
	_ = tags.EmitConditional("isolate_roots_bug", "isolate_roots", "src/math/polynomial/algebraic_numbers.cpp", 2598,
		func() bool { return true }, func(w io.Writer) error {
			out := algebraicPredicateTraceStream{w}
			_ = out.AppendCString("w: ")
			location, e := m.locationCleanup(&m.isolationSigns.sample)
			computation = e
			if computation == nil {
				computation = m.displayRootAtLocation(out, location, false)
			}
			if computation != nil {
				return computation
			}
			_ = out.AppendCString("\n")
			return nil
		})
	return computation
}
