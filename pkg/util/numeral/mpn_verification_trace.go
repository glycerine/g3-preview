// Source: src/util/mpn.cpp:215, after the debug quotient/remainder check.
package numeral

import "io"

func (m *NaturalManager) traceDivisionVerification(ok bool, sum limbView) {
	_ = m.sourceTraceTags().EmitConditional("mpn_dbg", "div", "src/util/mpn.cpp", 215,
		func() bool { return !ok }, func(w io.Writer) error {
			if err := writeNumeralDisplayPart(w, "DIV BUG: quot * denom + rem = "); err != nil {
				return err
			}
			if err := m.DisplayRaw(w, sum.external[:sum.size]); err != nil {
				return err
			}
			return writeNumeralDisplayPart(w, "\n")
		})
}
