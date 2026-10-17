// Source: src/math/polynomial/algebraic_numbers.cpp:2861-2865,2911-2915.
package polynomial

import (
	"errors"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicIntegerSetScope struct {
	polynomialCaller *polynomialAlgebraicEvaluationScope
	value            unumeral.Rational
	addition         *algebraicIntegerAddScope
	active, running  bool
	sourceStop       error
}
type algebraicIntegerAddScope struct {
	polynomialCaller *polynomialAlgebraicEvaluationScope
	value            Algebraic
	active, running  bool
	sourceStop       error
}

func (m *AlgebraicRootCore) cleanupIntegerSet() error {
	s := &m.integerSet
	if s.running {
		return ErrAlgebraicPending
	}
	if !s.active {
		return nil
	}
	if e := m.config.Rationals.Del(&s.value); e != nil {
		return e
	}
	s.active = false
	return nil
}
func (m *AlgebraicRootCore) cleanupIntegerAdd(owning bool) error {
	s := &m.integerAdd
	if s.running {
		return ErrAlgebraicPending
	}
	if !s.active {
		return nil
	}
	if owning {
		l, e := m.locationCleanup(&s.value)
		if e != nil {
			return e
		}
		if l != nil {
			if e = m.destroyCell(&s.value, l, true); e != nil {
				return e
			}
		}
	} else if e := m.Del(&s.value); e != nil {
		return e
	}
	s.active = false
	return nil
}

// SetInteger preserves the source mpz -> scoped mpq -> algebraic setter path.
func (m *AlgebraicRootCore) SetInteger(out *Algebraic, n *unumeral.Int) (cause error) {
	if e := m.check(); e != nil {
		return e
	}
	s := &m.integerSet
	if s.active {
		return ErrAlgebraicPending
	}
	*s = algebraicIntegerSetScope{polynomialCaller: m.polynomialCaller, value: unumeral.NewRational(), active: true, running: true}
	if m.integerAdd.running {
		s.addition = &m.integerAdd
	}
	defer func() {
		s.running = false
		if m.localIntegerSet == s && m.localActive && m.localCleanup != nil {
			s.sourceStop = m.localCleanup
			cause = errors.Join(cause, s.sourceStop)
			return
		}
		cleanup := m.cleanupIntegerSet()
		if cleanup != nil {
			s.sourceStop = cleanup
		}
		cause = errors.Join(cause, cleanup)
	}()
	if e := m.config.Rationals.SetInteger(&s.value, n); e != nil {
		return e
	}
	return m.SetRational(out, &s.value)
}

// AddInteger constructs a real scoped algebraic coefficient before adding it.
// The source overload is not equivalent in lifetime to adding a borrowed mpq.
func (m *AlgebraicRootCore) AddInteger(a *Algebraic, n *unumeral.Int, out *Algebraic) (cause error) {
	if e := m.check(); e != nil {
		return e
	}
	s := &m.integerAdd
	if s.active {
		return ErrAlgebraicPending
	}
	*s = algebraicIntegerAddScope{polynomialCaller: m.polynomialCaller, active: true, running: true}
	defer func() {
		s.running = false
		var stopped error
		if m.integerSet.addition == s && m.integerSet.active {
			stopped = m.integerSet.sourceStop
		}
		if m.localIntegerAdd == s && m.localActive && m.localCleanup != nil {
			stopped = m.localCleanup
		}
		if m.arithmetic.integerAddition == s && m.arithmetic.active && m.arithmetic.sourceStop != nil {
			stopped = m.arithmetic.sourceStop
		}
		if stopped != nil {
			s.sourceStop = stopped
			cause = errors.Join(cause, stopped)
			return
		}
		cleanup := m.cleanupIntegerAdd(false)
		if cleanup != nil {
			s.sourceStop = cleanup
		}
		cause = errors.Join(cause, cleanup)
	}()
	if e := m.SetInteger(&s.value, n); e != nil {
		return e
	}
	return m.Add(a, &s.value, out)
}
