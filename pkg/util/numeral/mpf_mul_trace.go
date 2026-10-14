// Source: src/util/mpf.cpp:590-592,664.
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
