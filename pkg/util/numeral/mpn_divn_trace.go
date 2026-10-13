// Source: src/util/mpn.cpp:350-354.
package numeral

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
)

// Source display_raw requests m+n+1 limbs while numer owns only m+n.
// Preserve the defined output prefix and stop before the uninitialized read.
func (m *NaturalManager) traceDivisionManyStep(quotient, remainder uint64, n, numerSize, borrow uint32) {
	_ = m.sourceTraceTags().EmitConditional("mpn_div", "div_n", "src/util/mpn.cpp", 350,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeNumeralDisplayPart(w, "q_hat="+strconv.FormatUint(quotient, 10)+
				" r_hat="+strconv.FormatUint(remainder, 10)+" ms="); err != nil {
				return err
			}
			if err := m.DisplayRaw(w, m.Division[2][:n]); err != nil {
				return err
			}
			if err := writeNumeralDisplayPart(w, " new numer="); err != nil {
				return err
			}
			if uint64(numerSize)+1 > uint64(len(m.Division[0])) {
				return arena.ErrBounds
			}
			if err := m.DisplayRaw(w, m.Division[0][:numerSize+1]); err != nil {
				return err
			}
			return writeNumeralDisplayPart(w, " borrow="+strconv.FormatUint(uint64(borrow), 10)+"\n")
		})
}
