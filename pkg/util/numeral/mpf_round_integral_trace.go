// Source: src/util/mpf.cpp:1100,1116.
package numeral

import (
	"io"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func (m *ArbitraryFloatManager) traceRoundIntegralRawOperand(value *ArbitraryFloat) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "round_to_integral", "src/util/mpf.cpp", 1100,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "A = "); e != nil {
				return e
			}
			var raw utext.OwnedString
			if e := m.ToStringRaw(value, &raw); e != nil {
				formatError = JoinNumericErrors(e, m.CleanupRawString())
				raw.OwnAbort()
				return formatError
			}
			defer raw.SourceDestroy()
			s, e := raw.View().String()
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, s); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
	return formatError
}

func (m *ArbitraryFloatManager) traceRoundIntegralQuotient(div, rem *Int) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "round_to_integral", "src/util/mpf.cpp", 1116,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "div="); e != nil {
				return e
			}
			s, e := m.integer.String(div)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, s); e != nil {
				return e
			}
			if e := writeNumeralDisplayPart(w, " rem="); e != nil {
				return e
			}
			s, e = m.integer.String(rem)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, s); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
	return formatError
}

func (m *ArbitraryFloatManager) traceRoundIntegralTie(tie, less, more bool) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitConditional("mpf_dbg", "round_to_integral", "src/util/mpf.cpp", 1125,
		func() bool { return true }, func(w io.Writer) error {
			for _, part := range [...]string{"tie= ", traceBoolDigit(tie), "; <tie = ", traceBoolDigit(less), "; >tie = ", traceBoolDigit(more), "\n"} {
				if e := writeNumeralDisplayPart(w, part); e != nil {
					return e
				}
			}
			return nil
		})
}
