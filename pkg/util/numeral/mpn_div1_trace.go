// Source: src/util/mpn.cpp:296-302.
package numeral

import (
	"io"
	"strconv"
)

func (m *NaturalManager) traceDivisionOneStep(j uint32, quotient, temp, denom, ms uint64, numer limbView, borrow bool) {
	_ = m.sourceTraceTags().EmitConditional("mpn_div1", "div_1", "src/util/mpn.cpp", 296,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeNumeralDisplayPart(w, "j="+strconv.FormatUint(uint64(j), 10)+
				" q_hat="+strconv.FormatUint(quotient, 10)+
				" r_hat="+strconv.FormatUint(temp%denom, 10)+
				" ms="+strconv.FormatUint(ms, 10)+" new numer="); err != nil {
				return err
			}
			if err := m.DisplayRaw(w, numer.external[:numer.size]); err != nil {
				return err
			}
			bit := "0"
			if borrow {
				bit = "1"
			}
			return writeNumeralDisplayPart(w, " borrow="+bit+"\n")
		})
}
