// Source: src/util/mpf.cpp:1967-2097 (retained mpf_dbg TRACE sites).
package numeral

import (
	"io"
	"strconv"
)

func (m *ArbitraryFloatManager) traceRoundNormalization(emin, emax, beta, lz int64, belowMin bool, sigma, cap *Int) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "round", "src/util/mpf.cpp", 2000,
		func() bool { return true }, func(w io.Writer) error {
			for _, part := range [...]string{
				"e_min_norm = ", strconv.FormatInt(emin, 10), "\n",
				"e_max_norm = ", strconv.FormatInt(emax, 10), "\n",
				"beta = ", strconv.FormatInt(beta, 10), ", (beta < e_min) = ", traceBoolDigit(belowMin), "\n",
				"LZ = ", strconv.FormatInt(lz, 10), "\n", "sigma = ",
			} {
				if e := writeNumeralDisplayPart(w, part); e != nil {
					return e
				}
			}
			s, e := m.integer.String(sigma)
			if e != nil {
				formatError = e
				return e
			}
			for _, part := range [...]string{s, "\n", "sigma_cap = "} {
				if e := writeNumeralDisplayPart(w, part); e != nil {
					return e
				}
			}
			s, e = m.integer.String(cap)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, s); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
	return formatError
}

func (m *ArbitraryFloatManager) traceRoundShiftDistance(sigma int64) {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	direction := "(RIGHT)"
	if sigma >= 0 {
		direction = "(LEFT)"
	}
	_ = tags.EmitConditional("mpf_dbg", "round", "src/util/mpf.cpp", 2009,
		func() bool { return true }, func(w io.Writer) error {
			for _, part := range [...]string{"Shift distance: ", strconv.FormatInt(sigma, 10), " ", direction, "\n"} {
				if e := writeNumeralDisplayPart(w, part); e != nil {
					return e
				}
			}
			return nil
		})
}

func (m *ArbitraryFloatManager) traceRoundSignificandBranch(significand *Int, normal bool) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	line, label := 2091, "DENORMAL: "
	if normal {
		line, label = 2087, "NORMAL: "
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "round", "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, label); e != nil {
				return e
			}
			s, e := m.integer.String(significand)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, s); e != nil {
				return e
			}
			return writeNumeralDisplayPart(w, "\n")
		})
	return formatError
}

// Source TRACE formatting borrows the partially rounded value while the
// numeric body owns its scope. Only the concrete formatter call is nested.
func (m *ArbitraryFloatManager) formatRoundTraceValue(value *ArbitraryFloat, binary bool, upper, lower uint32) (string, error) {
	saved := m.valueExecuting
	m.valueExecuting = false
	defer func() { m.valueExecuting = saved }()
	if binary {
		return m.traceFloatBinaryString(value, upper, lower)
	}
	return m.formatFloatTraceValue(value)
}

func (m *ArbitraryFloatManager) traceRoundValue(label string, value *ArbitraryFloat, line int, binary bool, upper, lower uint32, suffix string) error {
	tags := m.floatTraceTags()
	if !tags.ShouldTrace("mpf_dbg") {
		return nil
	}
	var formatError error
	_ = tags.EmitConditional("mpf_dbg", "round", "src/util/mpf.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if e := writeNumeralDisplayPart(w, label); e != nil {
				return e
			}
			decimal, e := m.formatRoundTraceValue(value, false, 0, 0)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, decimal+"\n"); e != nil {
				return e
			}
			if !binary {
				return nil
			}
			bits, e := m.formatRoundTraceValue(value, true, upper, lower)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, bits+suffix+"\n")
		})
	return formatError
}
