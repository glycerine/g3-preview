// Source: src/math/polynomial/algebraic_numbers.cpp:2626-2656,2676-2706.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicRootDisplayScope struct {
	insertions      [2]unumeral.NumeralInsertionReceipt
	negative        unumeral.Int
	coefficients    UPolynomial
	negativeOwned   bool
	active, running bool
}

func (m *AlgebraicRootCore) cleanupRootDisplay() error { return m.finishRootDisplay(true) }
func (m *AlgebraicRootCore) finishRootDisplay(owning bool) error {
	s := &m.rootDisplay
	if s.running {
		return ErrAlgebraicPending
	}
	if !s.active {
		return nil
	}
	for i := len(s.insertions) - 1; i >= 0; i-- {
		if !s.insertions[i].Active() {
			continue
		}
		if !owning {
			if e := s.insertions[i].SourceCause(); e != nil {
				return e
			}
		}
		if e := s.insertions[i].Complete(m.config.Rationals.Integers().Natural, false); e != nil {
			return e
		}
		if owning {
			if e := s.insertions[i].Complete(m.config.Rationals.Integers().Natural, true); e != nil {
				return e
			}
		}
	}

	if s.coefficients.slot != 0 {
		if e := m.config.Polynomials.DelPolynomial(&s.coefficients); e != nil {
			return e
		}
	}
	if s.negativeOwned {
		if e := m.config.Rationals.Integers().Del(&s.negative); e != nil {
			return e
		}
		s.negativeOwned = false
	}
	s.active = false
	for i := range s.insertions {
		if s.insertions[i].Active() {
			s.active = true
		}
	}
	return nil
}

// DisplayRootInto writes (#-polynomial, index); DisplayRootSMT2Into writes
// (root-obj polynomial index). Each stream call is one reached insertion.
func (m *AlgebraicRootCore) DisplayRootInto(out unumeral.RationalSourceStream, a *Algebraic) error {
	return m.displayRootInto(out, a, false)
}
func (m *AlgebraicRootCore) DisplayRootSMT2Into(out unumeral.RationalSourceStream, a *Algebraic) error {
	return m.displayRootInto(out, a, true)
}
func (m *AlgebraicRootCore) displayRootInto(out unumeral.RationalSourceStream, a *Algebraic, smt bool) (cause error) {
	if out == nil {
		return arena.ErrConfig
	}
	l, e := m.location(a)
	if e != nil {
		return e
	}
	if m.rootDisplay.active {
		return ErrAlgebraicPending
	}
	if l == nil {
		if smt {
			return out.AppendCString("(root-obj x 1)")
		}
		return out.AppendCString("(#, 1)")
	}
	r, e := m.load(l)
	if e != nil {
		return e
	}
	p := m.config.Polynomials
	var view UPolynomialView
	s := &m.rootDisplay
	*s = algebraicRootDisplayScope{active: true, running: true}
	defer func() { s.running = false; cause = errors.Join(cause, m.finishRootDisplay(false)) }()
	if l.kind == 0 {
		s.negativeOwned = true
		z := m.config.Rationals.Integers()
		if e = z.Copy(&s.negative, &r.value.Num); e != nil {
			return e
		}
		if e = z.Neg(&s.negative); e != nil {
			return e
		}
		// The two source dup coefficients have actual independent numeral fields.
		// Native UP storage supplies their checked view without container emulation.
		s.coefficients, e = p.NewPolynomial()
		if e != nil {
			return e
		}
		if e = p.PushCoefficient(s.coefficients, &s.negative); e != nil {
			return e
		}
		if e = p.PushCoefficient(s.coefficients, &r.value.Den); e != nil {
			return e
		}
		view, e = p.wholeView(s.coefficients)
		if e != nil {
			return e
		}
	} else {
		view, e = m.sourcePolynomial(&r)
		if e != nil {
			return e
		}
	}
	prefix, variable, separator, suffix := "(", "#", ", ", ")"
	if smt {
		prefix, variable, separator = "(root-obj ", "x", " "
	}
	if e = out.AppendCString(prefix); e != nil {
		return e
	}
	if smt {
		e = p.DisplaySMT2Into(view, out, variable)
	} else {
		e = p.DisplayInto(view, out, variable, false)
	}
	if e != nil {
		return e
	}
	if l.kind == 0 {
		if smt {
			return out.AppendCString(" 1)")
		}
		return out.AppendCString(", 1)")
	}

	line := int32(2652)
	if smt {
		line = 2702
	}
	index, e := m.displayRootIndex(l, view, line)
	if e != nil {
		return e
	}

	if e = out.AppendCString(separator); e != nil {
		return e
	}
	if e = displayUnsigned(out, index); e != nil {
		return e
	}
	return out.AppendCString(suffix)
}

func (m *AlgebraicRootCore) displayRootIndex(l *algebraicLocation, view UPolynomialView, line int32) (uint32, error) {
	// m_i occupies the upper 29 bits of the source packed cell word.
	// Publish only after get_root_id returns, and before the following output.
	r, e := m.load(l)
	if e != nil {
		return 0, e
	}
	index := r.flags >> 3
	if index == 0 {
		index, e = m.config.Polynomials.GetRootID(view, r.interval.BorrowLower())
		if e != nil {
			return 0, e
		}
		index = (index + 1) & 0x1fffffff
		r.flags = r.flags&7 | index<<3
		if e = m.store(l, &r); e != nil {
			return 0, e
		}
	}

	d, enabled, e := algebraicAssertionSwitch()
	if e != nil {
		return 0, e
	}
	if enabled && index == 0 {
		if e = diagnostic.DebugAssertionFailure(d, "src/math/polynomial/algebraic_numbers.cpp", line, "c->m_i > 0"); e != nil {
			return 0, e
		}
	}
	return index, nil
}
