// Source: src/math/polynomial/algebraic_numbers.cpp:2708-2719.
package polynomial

import (
	"io"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func writeAlgebraicIntervalPart(w io.Writer, text string) error {
	n, err := io.WriteString(w, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}

// DisplayInterval writes the source's basic [q, q] or root-cell MPBQ/open
// interval. A basic value obtains each rational spelling at its insertion
// point, preserving the already written prefix if formatting fails.
func (m *AlgebraicRootCore) DisplayInterval(w io.Writer, a *Algebraic) error {
	if w == nil {
		return arena.ErrInvalidSlot
	}
	l, err := m.location(a)
	if err != nil {
		return err
	}
	if l != nil && l.kind == 1 {
		r, err := m.load(l)
		if err != nil {
			return err
		}
		return m.config.Intervals.Display(w, &r.interval)
	}
	if err := writeAlgebraicIntervalPart(w, "["); err != nil {
		return err
	}
	q := unumeral.NewRational()
	if l != nil {
		r, err := m.load(l)
		if err != nil {
			return err
		}
		q = r.value
	}
	first, err := m.config.Rationals.String(&q)
	if err != nil {
		return err
	}
	if err := writeAlgebraicIntervalPart(w, first); err != nil {
		return err
	}
	if err := writeAlgebraicIntervalPart(w, ", "); err != nil {
		return err
	}
	second, err := m.config.Rationals.String(&q)
	if err != nil {
		return err
	}
	if err := writeAlgebraicIntervalPart(w, second); err != nil {
		return err
	}
	return writeAlgebraicIntervalPart(w, "]")
}
