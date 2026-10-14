// Source: src/util/mpf.cpp:76,105.
package numeral

import "io"

func (m *ArbitraryFloatManager) traceSetInt32Result(out *ArbitraryFloat) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "set", "src/util/mpf.cpp", 105,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "set: res = "); e != nil {
				return e
			}
			value, e := m.formatFloatTraceValue(out)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, value+"\n")
		})
	return formatError
}
