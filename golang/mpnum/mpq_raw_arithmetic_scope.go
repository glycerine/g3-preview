// Source: src/util/mpq.h; src/util/mpq.cpp. Licensed under the MIT license.
package mpnum

import "errors"

var ErrRationalRawArithmeticPending = errors.New("mpnum: raw arithmetic local cleanup pending")

// Actual cold owner-local storage for source RAW locals: Power's rational,
// SYNCH add's two mpz, normalize's mpz, and SYNCH comparison's two mpq.
// The nine stored Int fields are charged once in the manager header. Comparison
// denominators are normally inline1; stopped independent source calls may retain
// separate cohorts. No source exception implicitly destroys these raw objects.
type rationalRawArithmeticScope struct {
	power                                   Rational
	compareOne, compareTwo                  Rational
	compareActive, compareEnd, compareGuard bool
	compareCursor                           uint8
	compareCause                            error
	addOne, addTwo, norm                    Int
	powerActive, addActive, normActive      bool
	powerEnd, addEnd, normEnd               bool
	powerGuard, addGuard, normGuard         bool
	powerCursor, addCursor                  uint8
	powerCause, addCause, normCause         error
}

type RationalRawArithmeticState struct {
	Power, Addition, Normalization                bool
	Comparison, ComparisonEnd, ComparisonGuard    bool
	ComparisonCursor                              uint8
	ComparisonCause                               error
	PowerEnd, AdditionEnd, NormalizationEnd       bool
	PowerGuard, AdditionGuard, NormalizationGuard bool
	PowerCursor, AdditionCursor                   uint8
	PowerCause, AdditionCause, NormalizationCause error
}

func (r *rationalRawArithmeticScope) pending() bool {
	return r.powerActive || r.addActive || r.normActive || r.compareActive
}
func (m *RationalManager) RawArithmeticState() (RationalRawArithmeticState, error) {
	if e := m.checkOwnerReady(); e != nil {
		return RationalRawArithmeticState{}, e
	}
	r := &m.rawArithmetic
	return RationalRawArithmeticState{Comparison: r.compareActive, ComparisonEnd: r.compareEnd, ComparisonGuard: r.compareGuard, ComparisonCursor: r.compareCursor, ComparisonCause: r.compareCause, Power: r.powerActive, Addition: r.addActive, Normalization: r.normActive, PowerEnd: r.powerEnd, AdditionEnd: r.addEnd, NormalizationEnd: r.normEnd, PowerGuard: r.powerGuard, AdditionGuard: r.addGuard, NormalizationGuard: r.normGuard, PowerCursor: r.powerCursor, AdditionCursor: r.addCursor, PowerCause: r.powerCause, AdditionCause: r.addCause, NormalizationCause: r.normCause}, nil
}

// Comparison owns actual RAW mpq tmp1/tmp2. Body exceptions skip source del;
// normal completion destroys tmp1 NUM/DEN before tmp2 NUM/DEN.
func (m *RationalManager) finishRawCompare(e *error) {
	r := &m.rawArithmetic
	if *e != nil {
		r.compareCause = *e
		if r.compareOne.num.cell == 0 && r.compareOne.den.cell == 0 && r.compareTwo.num.cell == 0 && r.compareTwo.den.cell == 0 {
			r.compareActive = false
			r.compareOne = Rational{}
			r.compareTwo = Rational{}
		}
	}
}
func (r *rationalRawArithmeticScope) compareField() *Int {
	switch r.compareCursor {
	case 0:
		return &r.compareOne.num
	case 1:
		return &r.compareOne.den
	case 2:
		return &r.compareTwo.num
	default:
		return &r.compareTwo.den
	}
}
func (m *RationalManager) rawCompareEnd() error {
	r := &m.rawArithmetic
	r.compareEnd = true
	for r.compareCursor < 4 {
		guard, e := sourceNumeralDel(m.z, r.compareField())
		r.compareGuard = guard
		if e != nil {
			return e
		}
		r.compareCursor++
	}
	r.compareOne = Rational{}
	r.compareTwo = Rational{}
	r.compareActive = false
	r.compareCause = nil
	return nil
}
func (m *RationalManager) finishRawPower(e *error) {
	r := &m.rawArithmetic
	if *e != nil {
		r.powerCause = *e
		if r.power.num.cell == 0 && r.power.den.cell == 0 {
			r.powerActive = false
			r.power = Rational{}
		}
	}
}
func (m *RationalManager) finishRawAdd(e *error) {
	r := &m.rawArithmetic
	if *e != nil {
		r.addCause = *e
		if r.addOne.cell == 0 && r.addTwo.cell == 0 {
			r.addActive = false
			r.addOne = Int{}
			r.addTwo = Int{}
		}
	}
}
func (m *RationalManager) finishRawNormalize(e *error) {
	r := &m.rawArithmetic
	if *e != nil {
		r.normCause = *e
		if r.norm.cell == 0 {
			r.normActive = false
			r.norm = Int{}
		}
	}
}
func (m *RationalManager) rawPowerEnd() error {
	r := &m.rawArithmetic
	r.powerEnd = true
	for r.powerCursor < 2 {
		p := &r.power.num
		if r.powerCursor == 1 {
			p = &r.power.den
		}
		guard, e := sourceNumeralDel(m.z, p)
		r.powerGuard = guard
		if e != nil {
			return e
		}
		r.powerCursor++
	}
	r.power = Rational{}
	r.powerActive = false
	r.powerCause = nil
	return nil
}
func (m *RationalManager) rawAddEnd() error {
	r := &m.rawArithmetic
	r.addEnd = true
	for r.addCursor < 2 {
		p := &r.addOne
		if r.addCursor == 1 {
			p = &r.addTwo
		}
		guard, e := sourceNumeralDel(m.z, p)
		r.addGuard = guard
		if e != nil {
			return e
		}
		r.addCursor++
	}
	r.addOne = Int{}
	r.addTwo = Int{}
	r.addActive = false
	r.addCause = nil
	return nil
}
func (m *RationalManager) rawNormalizeEnd() error {
	r := &m.rawArithmetic
	r.normEnd = true
	guard, e := sourceNumeralDel(m.z, &r.norm)
	r.normGuard = guard
	if e != nil {
		return e
	}
	r.norm = Int{}
	r.normActive = false
	r.normCause = nil
	return nil
}

// Explicit owning cleanup acknowledges reached numeric release progress. This
// is not source exception unwinding: RAW mpq/mpz have no destructors.
func (m *RationalManager) cleanupRawArithmetic() error {
	r := &m.rawArithmetic
	if r.normActive {
		if e := m.z.Del(&r.norm); e != nil {
			r.normCause = e
			return e
		}
		r.normActive = false
	}
	if r.compareActive {
		for r.compareCursor < 4 {
			if e := m.z.Del(r.compareField()); e != nil {
				r.compareCause = e
				return e
			}
			r.compareCursor++
		}
		r.compareActive = false
	}
	if r.addActive {
		for r.addCursor < 2 {
			p := &r.addOne
			if r.addCursor == 1 {
				p = &r.addTwo
			}
			if e := m.z.Del(p); e != nil {
				r.addCause = e
				return e
			}
			r.addCursor++
		}
		r.addActive = false
	}
	if r.powerActive {
		for r.powerCursor < 2 {
			p := &r.power.num
			if r.powerCursor == 1 {
				p = &r.power.den
			}
			if e := m.z.Del(p); e != nil {
				r.powerCause = e
				return e
			}
			r.powerCursor++
		}
		r.powerActive = false
	}
	*r = rationalRawArithmeticScope{}
	return nil
}
func (m *RationalManager) CleanupRawArithmetic() error {
	if e := m.checkOwnerReady(); e != nil {
		return e
	}
	return m.cleanupRawArithmetic()
}
