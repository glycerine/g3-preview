// Source: src/util/mpf.cpp:813-818,830-831,844-845.
package numeral

import (
	"io"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func (m *ArbitraryFloatManager) traceFMABinaryString(value *ArbitraryFloat, upper, lower uint32) (string, error) {
	var out utext.OwnedString
	if e := m.ToStringBinary(value, upper, lower, &out); e != nil {
		cleanup := m.CleanupBinaryString()
		out.OwnAbort()
		return "", JoinNumericErrors(e, cleanup)
	}
	defer out.SourceDestroy()
	return out.View().String()
}

func (m *ArbitraryFloatManager) traceFMAUnpacked(a, b, c *ArbitraryFloat) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	values := [...]*ArbitraryFloat{a, b, c}
	labels := [...]string{"A = ", "B = ", "C = "}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "fma", "src/util/mpf.cpp", 813,
		func() bool { return true }, func(w io.Writer) error {
			for i, value := range values {
				if e := writeNumeralDisplayPart(w, labels[i]); e != nil {
					return e
				}
				formatted, e := m.formatFloatTraceValue(value)
				if e != nil {
					formatError = e
					return e
				}
				if e := writeNumeralDisplayPart(w, formatted); e != nil {
					return e
				}
				if e := writeNumeralDisplayPart(w, "\n"); e != nil {
					return e
				}
			}
			for i, value := range values {
				if e := writeNumeralDisplayPart(w, labels[i]); e != nil {
					return e
				}
				binary, e := m.traceFMABinaryString(value, 1, 0)
				if e != nil {
					formatError = e
					return e
				}
				if e := writeNumeralDisplayPart(w, binary); e != nil {
					return e
				}
				if e := writeNumeralDisplayPart(w, "\n"); e != nil {
					return e
				}
			}
			return nil
		})
	return formatError
}

func (m *ArbitraryFloatManager) traceFMADecimalBinary(label string, value *ArbitraryFloat, line int) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "fma", "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, label); e != nil {
				return e
			}
			decimal, e := m.formatFloatTraceValue(value)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, decimal+"\n"+label); e != nil {
				return e
			}
			binary, e := m.traceFMABinaryString(value, 1, 0)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, binary+"\n")
		})
	return formatError
}
