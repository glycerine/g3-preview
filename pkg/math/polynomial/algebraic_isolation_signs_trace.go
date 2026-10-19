// Source: src/math/polynomial/algebraic_numbers.cpp:2578-2592,2598.
package polynomial

import (
	"errors"
	"io"
)

func (m *AlgebraicRootCore) traceIsolationSignsRoots(pm *MultivariatePolynomialManager, p Polynomial, values PolynomialAlgebraicEvaluation, roots *AlgebraicVector) error {
	tags := m.config.Rationals.Integers().BorrowTraceTags()
	if !tags.ShouldTrace("isolate_roots_bug") {
		return nil
	}
	m.isolationSigns.traceRunning = true
	pm.core.algebraicTraceRunning = true
	defer func() {
		m.isolationSigns.traceRunning = false
		pm.core.algebraicTraceRunning = false
	}()
	var computation error
	_ = tags.EmitConditional("isolate_roots_bug", "isolate_roots", "src/math/polynomial/algebraic_numbers.cpp", 2578,
		func() bool { return true }, func(w io.Writer) error {
			computation = m.traceIsolationSignsRootsBody(w, pm, p, values, roots)
			return computation
		})
	return computation
}

func (m *AlgebraicRootCore) traceIsolationSignsRootsBody(w io.Writer, pm *MultivariatePolynomialManager, p Polynomial, values PolynomialAlgebraicEvaluation, roots *AlgebraicVector) (cause error) {
	sink := &polynomialTraceSink{writer: w}
	out := algebraicPredicateTraceStream{w}
	_ = out.AppendCString("p: ")
	if e := pm.core.displayIntoCore(p, sink, false); e != nil {
		return e
	}
	_ = out.AppendCString("\n")
	var xs PolynomialVariableDegrees
	variableEmbedded(&xs, pm.core.memory, true)
	defer func() { cause = errors.Join(cause, xs.CloseChecked()) }()
	if e := pm.core.varsCore(p, &xs); e != nil {
		return e
	}
	for i := 0; i < len(xs.values); i++ {
		x := xs.values[i]
		has, e := values.Contains(x)
		if e != nil {
			return e
		}
		if !has {
			continue
		}
		_ = out.AppendCString("x")
		_ = sink.WriteUnsigned(x)
		_ = out.AppendCString(" -> ")
		a, e := values.Algebraic(x)
		if e != nil {
			return e
		}
		location, e := m.locationCleanup(a)
		if e != nil {
			return e
		}
		if e = m.displayRootAtLocation(out, location, false); e != nil {
			return e
		}
		_ = out.AppendCString(" ")
		// Source calls x2v(xs[i]) again for the interval insertion.
		a, e = values.Algebraic(x)
		if e != nil {
			return e
		}
		location, e = m.locationCleanup(a)
		if e != nil {
			return e
		}
		if e = m.displayIntervalAtLocation(algebraicComparisonTraceWriter{w}, location); e != nil {
			return e
		}
		_ = out.AppendCString("\n")
	}
	for i := 0; i < len(roots.values); i++ {
		_ = out.AppendCString("root[i]: ")
		location, e := m.locationCleanup(&roots.values[i])
		if e != nil {
			return e
		}
		if e = m.displayRootAtLocation(out, location, false); e != nil {
			return e
		}
		_ = out.AppendCString("\n")
	}
	return nil
}

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
