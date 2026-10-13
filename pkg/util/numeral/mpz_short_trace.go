// Source: src/util/mpz.h:350-389,499-519.
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

// traceMPZBinary is reached by the concrete add/sub/mul bodies, including
// their nested AddMul/SubMul callers. Source operands are read before mutation.
func (m *IntegerManager) traceMPZBinary(a, b, c *Int, op string, before bool) {
	tags := m.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("mpz") {
		return
	}
	_ = tags.EmitShort("mpz", func(w io.Writer) error {
		if !before {
			s, err := m.String(c)
			if err != nil {
				return err
			}
			if err := writeNumeralDisplayPart(w, s); err != nil {
				return err
			}
			return writeNumeralDisplayPart(w, "\n")
		}
		if err := writeNumeralDisplayPart(w, "[mpz] "); err != nil {
			return err
		}
		sa, err := m.String(a)
		if err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, sa); err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, " "+op+" "); err != nil {
			return err
		}
		sb, err := m.String(b)
		if err != nil {
			return err
		}
		if err := writeNumeralDisplayPart(w, sb); err != nil {
			return err
		}
		return writeNumeralDisplayPart(w, " == ")
	})
}
