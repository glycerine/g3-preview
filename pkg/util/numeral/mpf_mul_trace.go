// Source: src/util/mpf.cpp:541-579,590-650,664,670-742,752-754.
package numeral

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *ArbitraryFloatManager) BindTrace(tags *diagnostic.TraceTags) error {
	if m == nil || m.self != m {
		return ErrNumericOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	m.trace = tags
	return nil
}

func (m *ArbitraryFloatManager) floatTraceTags() *diagnostic.TraceTags {
	if m.trace != nil {
		return m.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (m *ArbitraryFloatManager) traceFloatString(w io.Writer, value *ArbitraryFloat) (cause error) {
	text, err := NewArbitraryFloatFormatText(m.budget)
	if err != nil {
		return err
	}
	defer func() { cause = JoinNumericErrors(cause, text.Close()) }()
	view, err := m.ToStringInto(value, text)
	if err != nil {
		return err
	}
	n, err := view.Length()
	if err != nil {
		return err
	}
	if n > uint64(int(^uint(0)>>1)) {
		return arena.ErrBounds
	}
	data := make([]byte, int(n))
	for i := range data {
		b, e := view.Byte(uint64(i))
		if e != nil {
			return e
		}
		data[i] = b
	}
	return writeNumeralDisplayPart(w, string(data))
}

func (m *ArbitraryFloatManager) traceMulEntry(mode ArbitraryFloatRoundingMode, x, y *ArbitraryFloat) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_mul_bug") {
		return
	}
	_ = tags.EmitConditional("mpf_mul_bug", "mul", "src/util/mpf.cpp", 590,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "rm: "+strconv.FormatUint(uint64(mode), 10)+"\nX: "); e != nil {
				return e
			}
			if e := m.traceFloatString(w, x); e != nil {
				return e
			}
			if e := writeNumeralDisplayPart(w, "\nY: "); e != nil {
				return e
			}
			if e := m.traceFloatString(w, y); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
}

func (m *ArbitraryFloatManager) traceMulResult(out *ArbitraryFloat) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_mul_bug") {
		return
	}
	_ = tags.EmitConditional("mpf_mul_bug", "mul", "src/util/mpf.cpp", 664,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, "result: "); e != nil {
				return e
			}
			if e := m.traceFloatString(w, out); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
}

func (m *ArbitraryFloatManager) traceFloatDebugValue(function, label string, value *ArbitraryFloat, line int) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitConditional("mpf_dbg", function, "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, label); e != nil {
				return e
			}
			if e := m.traceFloatString(w, value); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
}

func (m *ArbitraryFloatManager) traceFloatDebugInt(function, label string, value *Int, line int) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitConditional("mpf_dbg", function, "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, label); e != nil {
				return e
			}
			s, e := m.integer.String(value)
			if e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, s+"\n")
		})
}

func (m *ArbitraryFloatManager) traceFloatDebugText(function, value string, line int) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitConditional("mpf_dbg", function, "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			return writeNumeralDisplayPart(w, value+"\n")
		})
}

func (m *ArbitraryFloatManager) traceFloatDebugSigned(function, label string, value int64, line int) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitConditional("mpf_dbg", function, "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			return writeNumeralDisplayPart(w, label+strconv.FormatInt(value, 10)+"\n")
		})
}
