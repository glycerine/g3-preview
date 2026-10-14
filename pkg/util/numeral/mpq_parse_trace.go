// Source: src/util/mpq.cpp:185-270.
package numeral

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *RationalManager) parseTraceTags() *diagnostic.TraceTags {
	if m.Z.trace != nil {
		return m.Z.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

// set(char const*) traces before it has established a valid denominator.
// Source to_string prints the raw numerator and denominator in that state.
func (m *RationalManager) parseRawText(a *Rational) (string, error) {
	n, err := m.Z.String(&a.Num)
	if err != nil {
		return "", err
	}
	if isIntegerRational(a) {
		return n, nil
	}
	d, err := m.Z.String(&a.Den)
	if err != nil {
		return "", err
	}
	return n + "/" + d, nil
}

func (m *RationalManager) traceParseBefore(a *Rational) {
	_ = m.parseTraceTags().EmitConditional("mpq_set", "set", "src/util/mpq.cpp", 203,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeNumeralDisplayPart(w, "[before] a: "); err != nil {
				return err
			}
			v, err := m.parseRawText(a)
			if err != nil {
				return err
			}
			for _, part := range [...]string{v, "\n"} {
				if err := writeNumeralDisplayPart(w, part); err != nil {
					return err
				}
			}
			return nil
		})
}

func (m *RationalManager) traceParseExponent(exp uint64, character byte) {
	_ = m.parseTraceTags().EmitConditional("mpq_set", "set", "src/util/mpq.cpp", 241,
		func() bool { return true }, func(w io.Writer) error {
			for _, part := range [...]string{"[exp]: ", strconv.FormatUint(exp, 10), ", str[0]: ", strconv.FormatInt(int64(character)-'0', 10)} {
				if err := writeNumeralDisplayPart(w, part); err != nil {
					return err
				}
			}
			return diagnostic.WriteTraceEndl(w)
		})
}

func (m *RationalManager) traceParsePower(a *Rational, negative bool, exp uint64, power *Rational) {
	_ = m.parseTraceTags().EmitConditional("mpq_set", "set", "src/util/mpq.cpp", 260,
		func() bool { return true }, func(w io.Writer) error {
			if err := writeNumeralDisplayPart(w, "a: "); err != nil {
				return err
			}
			value, err := m.parseRawText(a)
			if err != nil {
				return err
			}
			sign := "0"
			if negative {
				sign = "1"
			}
			for _, part := range [...]string{value, ", exp_sign:", sign, ", exp: ", strconv.FormatUint(exp, 10), " "} {
				if err := writeNumeralDisplayPart(w, part); err != nil {
					return err
				}
			}
			factor, err := m.parseRawText(power)
			if err != nil {
				return err
			}
			if err := writeNumeralDisplayPart(w, factor); err != nil {
				return err
			}
			return diagnostic.WriteTraceEndl(w)
		})
}
