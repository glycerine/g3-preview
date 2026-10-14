// Source: src/util/mpf.cpp:1214-1217,1225,1252.
package numeral

import "io"

func (m *ArbitraryFloatManager) traceSBVDecision(x *ArbitraryFloat, z *Int, last, round, sticky, increment bool) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg_sbv") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg_sbv", "to_sbv_mpq", "src/util/mpf.cpp", 1214,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "SBV: ("); e != nil {
				return e
			}
			value, e := m.formatFloatTraceValue(x)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, value+") == "); e != nil {
				return e
			}
			integer, e := m.integer.String(z)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, integer+"\nsign="+traceBoolDigit(x.Sign())+" last="+traceBoolDigit(last)+" round="+traceBoolDigit(round)+" sticky="+traceBoolDigit(sticky)+" inc="+traceBoolDigit(increment)+"\n")
		})
	return formatError
}

func (m *ArbitraryFloatManager) traceSBVResult(out *Rational) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "to_sbv_mpq", "src/util/mpf.cpp", 1225,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "SBV = "); e != nil {
				return e
			}
			s, e := m.rational.String(out)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, s+"\n")
		})
	return formatError
}
