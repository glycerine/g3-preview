// Source: src/util/mpn.cpp:257-259.
package numeral

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *NaturalManager) sourceTraceTags() *diagnostic.TraceTags {
	if m.trace != nil {
		return m.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

// Both numeric division and decimal conversion reach div_normalize.
func (m *NaturalManager) traceDivisionNormalize(numer, denom []uint32) {
	_ = m.sourceTraceTags().EmitConditional("mpn_norm", "div_normalize", "src/util/mpn.cpp", 257,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeNumeralDisplayPart(w, "Normalized: n_numer="); err != nil {
				return err
			}
			if err := m.DisplayRaw(w, numer); err != nil {
				return err
			}
			if err := writeNumeralDisplayPart(w, " n_denom="); err != nil {
				return err
			}
			if err := m.DisplayRaw(w, denom); err != nil {
				return err
			}
			return writeNumeralDisplayPart(w, "\n")
		})
}
