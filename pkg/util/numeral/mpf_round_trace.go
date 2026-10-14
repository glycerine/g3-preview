// Source: src/util/mpf.cpp:2000-2009.
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
			prefix := "e_min_norm = " + strconv.FormatInt(emin, 10) + "\ne_max_norm = " + strconv.FormatInt(emax, 10) + "\nbeta = " + strconv.FormatInt(beta, 10) + ", (beta < e_min) = " + traceBoolDigit(belowMin) + "\nLZ = " + strconv.FormatInt(lz, 10) + "\nsigma = "
			if e := writeNumeralDisplayPart(w, prefix); e != nil {
				return e
			}
			s, e := m.integer.String(sigma)
			if e != nil {
				formatError = e
				return e
			}
			if e := writeNumeralDisplayPart(w, s+"\nsigma_cap = "); e != nil {
				return e
			}
			s, e = m.integer.String(cap)
			if e != nil {
				formatError = e
				return e
			}
			return writeNumeralDisplayPart(w, s+"\n")
		})
	return formatError
}

func (m *ArbitraryFloatManager) traceRoundShiftDistance(sigma int64) {
	direction := "(RIGHT)"
	if sigma >= 0 {
		direction = "(LEFT)"
	}
	m.traceFloatDebugText("round", "Shift distance: "+strconv.FormatInt(sigma, 10)+" "+direction, 2009)
}
