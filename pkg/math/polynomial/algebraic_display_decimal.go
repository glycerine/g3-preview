// Source: src/math/polynomial/algebraic_numbers.cpp:2721-2747.
package polynomial

import (
	"errors"
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicDecimalWriter struct{ out io.Writer }

func (w algebraicDecimalWriter) AppendInt32(n int32) error {
	return writeAlgebraicIntervalPart(w.out, strconv.FormatInt(int64(n), 10))
}
func (w algebraicDecimalWriter) AppendCString(s string) error {
	return writeAlgebraicIntervalPart(w.out, s)
}
func (w algebraicDecimalWriter) AppendNumeralText(s string) error {
	return writeAlgebraicIntervalPart(w.out, s)
}

// DisplayDecimal refines local copies of a root cell's bounds before selecting
// the source's upper approximation or exact lower root. The stored cell is not
// refined by this display operation. A stopped numeral insertion retains its
// manager's source locals for CleanupDecimalStream or CleanupDisplay.
func (m *AlgebraicRootCore) DisplayDecimal(w io.Writer, a *Algebraic, precision uint32) (err error) {
	if w == nil {
		return arena.ErrInvalidSlot
	}
	l, err := m.location(a)
	if err != nil {
		return err
	}
	m.decimalDisplayRunning = true
	defer func() { m.decimalDisplayRunning = false }()
	if l == nil || l.kind == 0 {
		q := unumeral.NewRational()
		if l != nil {
			r, loadErr := m.load(l)
			if loadErr != nil {
				return loadErr
			}
			q = r.value
		}
		return m.config.Rationals.DisplayDecimalInto(algebraicDecimalWriter{w}, &q, precision, false)
	}
	r, err := m.load(l)
	if err != nil {
		return err
	}
	b := m.config.Binary
	lower, upper := unumeral.NewBinaryRational(), unumeral.NewBinaryRational()
	defer func() { err = errors.Join(err, b.Del(&lower)) }()
	defer func() { err = errors.Join(err, b.Del(&upper)) }()
	interval, err := m.intervalAtPrecision(&r, &lower, &upper, precision)
	if err != nil {
		return err
	}
	selected := &lower
	if interval {
		selected = &upper
	}
	return b.DisplayDecimalTo(w, selected, precision)
}
