// Source: src/util/mpf.cpp:1922-1929.
package numeral

import (
	"io"
	"strconv"
)

func (m *ArbitraryFloatManager) traceUnpackEntry(a *ArbitraryFloat, normalize bool) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "unpack", "src/util/mpf.cpp", 1923,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "unpack "); e != nil {
				return e
			}
			value, e := m.formatFloatTraceValue(a)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, value+": ebits="+strconv.FormatUint(uint64(a.EBits()), 10)+" sbits="+strconv.FormatUint(uint64(a.SBits()), 10)+" normalize="+traceBoolDigit(normalize)); e != nil {
				return e
			}
			top, e := m.hasTopExponent(a)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, " has_top_exp="+traceBoolDigit(top)+" ("); e != nil {
				return e
			}
			topExp, e := m.topExponent(a.EBits())
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, strconv.FormatInt(topExp, 10)+")"); e != nil {
				return e
			}
			bottom, e := m.hasBottomExponent(a)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, " has_bot_exp="+traceBoolDigit(bottom)+" ("); e != nil {
				return e
			}
			bottomExp, e := m.bottomExponent(a.EBits())
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, strconv.FormatInt(bottomExp, 10)+")"); e != nil {
				return e
			}
			zero, e := m.IsZero(a)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, " is_zero="+traceBoolDigit(zero)+"\n")
		})
	return formatError
}
