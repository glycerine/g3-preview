// Source: src/util/mpz.h:499-519.
package numeral

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// traceMPZNeg emits each STRACE independently. The source INT_MIN branch
// returns before its second record, leaving the first record unterminated.
func (m *IntegerManager) traceMPZNeg(a *Int, before bool) {
	tags := m.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("mpz") {
		return
	}
	_ = tags.EmitShort("mpz", func(w io.Writer) error {
		if before {
			if err := writeNumeralDisplayPart(w, "[mpz] 0 - "); err != nil {
				return err
			}
		}
		s, err := m.String(a)
		if err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, s); err != nil {
			return err
		}
		if !before {
			return writeNumeralDisplayPart(w, "\n")
		}
		return writeNumeralDisplayPart(w, " == ")
	})
}
