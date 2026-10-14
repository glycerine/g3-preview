// Source: src/util/mpf.cpp:1282-1285,1322,1365,1394,1398,1421.
package numeral

import (
	"io"
	"strconv"
)

func (m *ArbitraryFloatManager) tracePartialRemainderEntry(x, y *ArbitraryFloat, d int64, partial bool) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg_rem") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg_rem", "partial_remainder", "src/util/mpf.cpp", 1282,
		func() bool { return true }, func(w io.Writer) error {
			for _, item := range [...]struct {
				label string
				value *ArbitraryFloat
			}{{"x=", x}, {"y=", y}} {
				if e := writeNumeralDisplayPart(w, item.label); e != nil {
					return e
				}
				s, e := m.formatFloatTraceValue(item.value)
				if e != nil {
					formatError = e
					return e
				}
				if e := writeNumeralDisplayPart(w, s+"\n"); e != nil {
					return e
				}
			}
			return writeNumeralDisplayPart(w, "d="+strconv.FormatInt(d, 10)+"\npartial="+traceBoolDigit(partial)+"\n")
		})
	return formatError
}

func (m *ArbitraryFloatManager) tracePartialRemainderText(line int, value string) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg_rem") {
		return
	}
	_ = tags.EmitConditional("mpf_dbg_rem", "partial_remainder", "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			return writeNumeralDisplayPart(w, value+"\n")
		})
}

func (m *ArbitraryFloatManager) tracePartialRemainderInt(line int, label string, value *Int) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg_rem") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg_rem", "partial_remainder", "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, label); e != nil {
				return e
			}
			s, e := m.integer.String(value)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, s+"\n")
		})
	return formatError
}
