// Source: src/math/polynomial/algebraic_numbers.cpp:1394-1396.
package polynomial

import (
	"io"
	"strconv"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicTranslationTraceStream struct{ algebraicPredicateTraceStream }

func (s algebraicTranslationTraceStream) AppendUint32(v uint32) error {
	return s.AppendCString(strconv.FormatUint(uint64(v), 10))
}

func (m *AlgebraicRootCore) traceRationalTranslationInput(r *algebraicRecord, q *unumeral.Rational) error {
	return m.emitRootTrace("add", 1394, func(w io.Writer) error {
		out := algebraicTranslationTraceStream{algebraicPredicateTraceStream{w}}
		_ = out.AppendCString("adding algebraic and basic cells:\na: ")
		p, e := m.sourcePolynomial(r)
		if e != nil {
			return e
		}
		if e = m.config.Polynomials.displayIntoWithReceipt(p, out, "x", false, &m.rootTrace.univariate); e != nil {
			return e
		}
		_ = out.AppendCString(" (")
		if e = m.config.Binary.DisplaySourceTracked(out, r.interval.BorrowLower(), &m.rootTrace.insertions[0]); e != nil {
			return e
		}
		_ = out.AppendCString(", ")
		if e = m.config.Binary.DisplaySourceTracked(out, r.interval.BorrowUpper(), &m.rootTrace.insertions[0]); e != nil {
			return e
		}
		_ = out.AppendCString(")\nb: ")
		if e = m.config.Rationals.DisplayIntoTracked(out, q, &m.rootTrace.insertions[0]); e != nil {
			return e
		}
		return out.AppendCString("\n")
	})
}
