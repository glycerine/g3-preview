// Source: src/util/mpfx.h:267-268; src/util/mpfx.cpp:221-234,447-517.
package numeral

import (
	"io"
	"strconv"
)

func (m *FixedPointManager) traceFixedIntegerDivision(num int64, den uint64, out *FixedPoint, after bool) {
	tags := m.fixedTraceTags()
	if !tags.ShouldTrace("mpfx_trace") {
		return
	}
	_ = tags.EmitShort("mpfx_trace", func(w io.Writer) error {
		if after {
			if err := m.Display(w, out); err != nil {
				return err
			}
			return writeFixedPointPart(w, "\n")
		}
		bound := ") >= "
		if m.roundUp {
			bound = ") <= "
		}
		return writeFixedPointPart(w, "[mpfx] ("+strconv.FormatInt(num, 10)+") / ("+strconv.FormatUint(den, 10)+bound)
	})
}

func (m *FixedPointManager) traceFixedInverse(a *FixedPoint, after bool) {
	tags := m.fixedTraceTags()
	if !tags.ShouldTrace("mpfx_trace") {
		return
	}
	_ = tags.EmitShort("mpfx_trace", func(w io.Writer) error {
		if after {
			if err := m.Display(w, a); err != nil {
				return err
			}
			return writeFixedPointPart(w, "\n")
		}
		if err := writeFixedPointPart(w, "[mpfx] (1) / ("); err != nil {
			return err
		}
		if err := m.Display(w, a); err != nil {
			return err
		}
		bound := ") >= "
		if m.roundUp {
			bound = ") <= "
		}
		return writeFixedPointPart(w, bound)
	})
}
