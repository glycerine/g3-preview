// Source: algebraic_numbers.cpp:2288-2443 multivariate isolation TRACE sites.
package polynomial

import (
	"io"
	"strconv"
)

func (m *AlgebraicRootCore) traceMultivariateIsolationLiteral(line int, text string) error {
	return m.emitRootTraceTag("isolate_roots", "isolate_roots", line, func(w io.Writer) error {
		return (algebraicPredicateTraceStream{w}).AppendCString(text)
	})
}
func (m *AlgebraicRootCore) traceMultivariateIsolationPolynomial(pm *MultivariatePolynomialManager, p Polynomial, line int, prefix string) error {
	if !m.config.Rationals.Integers().BorrowTraceTags().ShouldTrace("isolate_roots") {
		return nil
	}
	previous := pm.core.algebraicTraceRunning
	pm.core.algebraicTraceRunning = true
	defer func() { pm.core.algebraicTraceRunning = previous }()
	return m.emitRootTraceTag("isolate_roots", "isolate_roots", line, func(w io.Writer) error {
		sink := &polynomialTraceSink{writer: w}
		_ = sink.WriteLiteral(prefix)
		m.rootTrace.polynomial = pm.core
		if e := pm.core.displayIntoCoreWithReceipt(p, sink, false, &m.rootTrace.poly); e != nil {
			return e
		}
		return sink.WriteLiteral("\n")
	})
}
func (m *AlgebraicRootCore) traceMultivariateIsolationResultant(pm *MultivariatePolynomialManager, i, y uint32, py, q Polynomial) error {
	if !m.config.Rationals.Integers().BorrowTraceTags().ShouldTrace("isolate_roots") {
		return nil
	}
	previous := pm.core.algebraicTraceRunning
	pm.core.algebraicTraceRunning = true
	defer func() { pm.core.algebraicTraceRunning = previous }()
	return m.emitRootTraceTag("isolate_roots", "isolate_roots", 2360, func(w io.Writer) error {
		sink := &polynomialTraceSink{writer: w}
		_ = sink.WriteLiteral("resultant loop i: " + strconv.FormatUint(uint64(i), 10) + ", y: x" + strconv.FormatUint(uint64(y), 10) + "\np_y: ")
		m.rootTrace.polynomial = pm.core
		if e := pm.core.displayIntoCoreWithReceipt(py, sink, false, &m.rootTrace.poly); e != nil {
			return e
		}
		_ = sink.WriteLiteral("\nq: ")
		if e := pm.core.displayIntoCoreWithReceipt(q, sink, false, &m.rootTrace.poly); e != nil {
			return e
		}
		return sink.WriteLiteral("\n")
	})
}

func (m *AlgebraicRootCore) traceMultivariateIsolationVariableCount(count uint32) error {
	return m.emitRootTraceTag("isolate_roots", "isolate_roots", 2332, func(w io.Writer) error {
		return (algebraicPredicateTraceStream{w}).AppendCString("there are " + strconv.FormatUint(uint64(count), 10) + " variables assigned to nonbasic numbers...\n")
	})
}
