// Source: src/util/mpff.cpp:782-824.
package numeral

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *FastFloatManager) mulTraceTags() *diagnostic.TraceTags {
	if m.trace != nil {
		return m.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (m *FastFloatManager) traceMulOuter(a, b, c *FastFloat, before bool) {
	tags := m.mulTraceTags()
	if !tags.ShouldTrace("mpff_trace") {
		return
	}
	_ = tags.EmitShort("mpff_trace", func(w io.Writer) error {
		if !before {
			if err := m.DisplayTo(w, c); err != nil {
				return err
			}
			return writeFastFloatPart(w, "\n")
		}
		if err := writeFastFloatPart(w, "[mpff] ("); err != nil {
			return err
		}
		if err := m.DisplayTo(w, a); err != nil {
			return err
		}
		if err := writeFastFloatPart(w, ") * ("); err != nil {
			return err
		}
		if err := m.DisplayTo(w, b); err != nil {
			return err
		}
		bound := ") >= "
		if m.toPlus {
			bound = ") <= "
		}
		return writeFastFloatPart(w, bound)
	})
}

func (m *FastFloatManager) traceMulEntry(a, b *FastFloat) {
	tags := m.mulTraceTags()
	if !tags.ShouldTrace("mpff") {
		return
	}
	_ = tags.EmitConditional("mpff", "mul", "src/util/mpff.cpp", 788,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeFastFloatPart(w, "mul("); err != nil {
				return err
			}
			if err := m.DisplayTo(w, a); err != nil {
				return err
			}
			if err := writeFastFloatPart(w, ", "); err != nil {
				return err
			}
			if err := m.DisplayTo(w, b); err != nil {
				return err
			}
			return writeFastFloatPart(w, ")\n")
		})
}

func (m *FastFloatManager) traceMulLeading(zeros uint32) {
	tags := m.mulTraceTags()
	if !tags.ShouldTrace("mpff") {
		return
	}
	_ = tags.EmitConditional("mpff", "mul", "src/util/mpff.cpp", 800,
		func() bool { return true }, func(w io.Writer) error {
			return writeFastFloatPart(w, "num_leading_zeros: "+strconv.FormatUint(uint64(zeros), 10)+"\n")
		})
}

func traceBoolDigit(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (m *FastFloatManager) traceMulRounding(c *FastFloat, product ffLimbView, shift uint32, inc bool) {
	tags := m.mulTraceTags()
	if !tags.ShouldTrace("mpff") {
		return
	}
	_ = tags.EmitConditional("mpff", "mul", "src/util/mpff.cpp", 811,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeFastFloatPart(w, "c.m_sign: "+traceBoolDigit(c.negative())+", m_to_plus_inf: "+traceBoolDigit(m.toPlus)+"\n"); err != nil {
				return err
			}
			if err := writeFastFloatPart(w, "has_one_at_first_k_bits: "+traceBoolDigit(ffHasLow(product, shift))+"\n"); err != nil {
				return err
			}
			return writeFastFloatPart(w, "_inc_significand: "+traceBoolDigit(inc)+"\n")
		})
}

func (m *FastFloatManager) traceMulResult(c *FastFloat) {
	tags := m.mulTraceTags()
	if !tags.ShouldTrace("mpff") {
		return
	}
	_ = tags.EmitConditional("mpff", "mul", "src/util/mpff.cpp", 821,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeFastFloatPart(w, "result: "); err != nil {
				return err
			}
			if err := m.DisplayTo(w, c); err != nil {
				return err
			}
			return writeFastFloatPart(w, "\n")
		})
}
