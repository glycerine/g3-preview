// Source: src/util/mpf.cpp:968-994.
package numeral

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *IntegerManager) traceMPFSqrtBounds(input, lower, upper *Int) {
	tags := m.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitShort("mpf_dbg", func(w io.Writer) error {
		for _, part := range []struct {
			prefix string
			value  *Int
		}{{"SIG = ", input}, {" LOWER = ", lower}, {" UPPER = ", upper}} {
			if err := writeNumeralDisplayPart(w, part.prefix); err != nil {
				return err
			}
			s, err := m.String(part.value)
			if err != nil {
				return err
			}
			if err := writeNumeralDisplayPart(w, s); err != nil {
				return err
			}
		}
		return writeNumeralDisplayPart(w, "\n")
	})
}

func (m *IntegerManager) traceMPFSqrtMid(mid, product *Int) {
	tags := m.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	_ = tags.EmitShort("mpf_dbg", func(w io.Writer) error {
		if err := writeNumeralDisplayPart(w, "MID = "); err != nil {
			return err
		}
		s, err := m.String(mid)
		if err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, s); err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, " PROD = "); err != nil {
			return err
		}
		s, err = m.String(product)
		if err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, s); err != nil {
			return err
		}
		return writeNumeralDisplayPart(w, "\n")
	})
}

func (m *IntegerManager) traceMPFSqrtChoice(lower bool) {
	tags := m.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	choice := "choosing upper\n"
	if lower {
		choice = "choosing lower\n"
	}
	_ = tags.EmitShort("mpf_dbg", func(w io.Writer) error { return writeNumeralDisplayPart(w, choice) })
}
