// Source: src/util/mpz.cpp:1385-1408 and src/util/mpq.cpp:130-139.
package numeral

import (
	"io"

	"github.com/glycerine/g3/pkg/arena"
)

func writeNumeralDisplayPart(w io.Writer, s string) error {
	n, err := io.WriteString(w, s)
	if err == nil && n != len(s) {
		return io.ErrShortWrite
	}
	return err
}

// DisplayTo writes an inline integer in one insertion. A large negative
// integer writes its sign before formatting and inserting the magnitude.
func (m *IntegerManager) DisplayTo(w io.Writer, a *Int) error {
	if w == nil {
		return arena.ErrInvalidSlot
	}
	if err := m.check(a); err != nil {
		return err
	}
	largeNegative := !a.IsSmall() && a.Val < 0
	if largeNegative {
		if err := writeNumeralDisplayPart(w, "-"); err != nil {
			return err
		}
	}
	s, err := m.String(a)
	if err != nil {
		return err
	}
	if largeNegative {
		s = s[1:]
	}
	return writeNumeralDisplayPart(w, s)
}

// DisplayTo preserves the numerator, slash and denominator source insertions.
func (m *RationalManager) DisplayTo(w io.Writer, a *Rational) error {
	if w == nil {
		return arena.ErrInvalidSlot
	}
	if err := m.check(a); err != nil {
		return err
	}
	integer := isIntegerRational(a)
	if err := m.Z.DisplayTo(w, &a.Num); err != nil {
		return err
	}
	if integer {
		return nil
	}
	if err := writeNumeralDisplayPart(w, "/"); err != nil {
		return err
	}
	return m.Z.DisplayTo(w, &a.Den)
}
