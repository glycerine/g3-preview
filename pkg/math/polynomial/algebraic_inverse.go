// Source: src/math/polynomial/algebraic_numbers.cpp:1579-1647,2937-2939.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

var ErrAlgebraicDivisionByZero = errors.New("division by zero")

// The two source scoped rationals survive a refused destruction in the owner.
type algebraicInverseScope struct {
	polynomialCaller *polynomialAlgebraicEvaluationScope
	sourceStop       error
	values           [2]unumeral.Rational // inv_lower, inv_upper
	remaining        uint8
	executing        bool
}

func (m *AlgebraicRootCore) cleanupInverse() error {
	q := &m.inversion
	if q.executing {
		return ErrAlgebraicPending
	}
	q.executing = true
	defer func() { q.executing = false }()
	for q.remaining != 0 {
		if e := m.config.Rationals.Del(&q.values[q.remaining-1]); e != nil {
			return e
		}
		q.remaining--
	}
	return nil
}

// refineNonzeroBounds follows the source halving search. General refinement
// would change the nonzero endpoint as well and would not preserve this path.
func (m *AlgebraicRootCore) refineNonzeroBounds(a *Algebraic) error {
	l, e := m.location(a)
	if e != nil || l == nil || l.kind == 0 {
		return e
	}
	r, e := m.load(l)
	if e != nil {
		return e
	}
	b := m.config.Binary
	lower, upper := r.interval.BorrowLower(), r.interval.BorrowUpper()
	lz, e := b.IsZero(lower)
	if e != nil {
		return e
	}
	uz, e := b.IsZero(upper)
	if e != nil || !lz && !uz {
		return e
	}
	sign := 1
	if r.flags&2 != 0 {
		sign = -1
	}
	bound, other := lower, upper
	if !lz {
		bound, other = upper, lower
		sign = -sign
	}
	e = b.Copy(bound, other)
	if e = m.save(l, &r, e); e != nil {
		return e
	}
	v, e := m.sourcePolynomial(&r)
	if e != nil {
		return e
	}
	for {
		e = b.Div2(bound)
		if e = m.save(l, &r, e); e != nil {
			return e
		}
		next, e := m.config.Polynomials.EvalSignBinary(v, bound)
		if e != nil {
			return e
		}
		if next == 0 {
			if e = m.beginQ(); e != nil {
				return e
			}
			e = b.ToRational(m.config.Rationals, bound, &m.localQ)
			if e == nil {
				e = m.SetTake(a, &m.localQ)
			}
			return m.finishQ(e)
		}
		if next == sign {
			return nil
		}
	}
}

// Inv replaces a nonzero value in place. An algebraic root keeps its cell and
// minimality flag while reversing its polynomial and reciprocating its bounds.
func (m *AlgebraicRootCore) Inv(a *Algebraic) (cause error) {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	if l == nil {
		if e := diagnostic.BorrowProcessDebugContext().UnreachableSource("src/math/polynomial/algebraic_numbers.cpp", 1622); e != nil {
			return e
		}
		return ErrAlgebraicDivisionByZero
	}
	if m.inversion.remaining != 0 || m.localActive || m.negation.active {
		return ErrAlgebraicPending
	}
	if e = m.refineNonzeroBounds(a); e != nil {
		return e
	}
	l, e = m.location(a)
	if e != nil {
		return e
	}
	r, e := m.load(l)
	if e != nil {
		return e
	}
	q := &m.inversion
	q.polynomialCaller = m.polynomialCaller
	q.sourceStop = nil
	q.executing = true
	defer func() {
		q.executing = false
		cleanup := m.cleanupInverse()
		if cleanup != nil {
			q.sourceStop = cleanup
		}
		cause = errors.Join(cause, cleanup)
	}()
	if l.kind == 0 {
		e = m.config.Rationals.InvInPlace(&r.value)
		return m.save(l, &r, e)
	}
	p := m.config.Polynomials
	v, e := m.sourcePolynomial(&r)
	if e != nil {
		return e
	}
	if e = p.ReciprocalViewInPlace(v); e != nil {
		return e
	}
	q.values = [2]unumeral.Rational{unumeral.NewRational(), unumeral.NewRational()}
	q.remaining = 2
	qm, b := m.config.Rationals, m.config.Binary
	if e = b.ToRational(qm, r.interval.BorrowLower(), &q.values[0]); e != nil {
		return e
	}
	if e = b.ToRational(qm, r.interval.BorrowUpper(), &q.values[1]); e != nil {
		return e
	}
	if e = qm.InvInPlace(&q.values[0]); e != nil {
		return e
	}
	if e = qm.InvInPlace(&q.values[1]); e != nil {
		return e
	}
	if e = qm.Swap(&q.values[0], &q.values[1]); e != nil {
		return e
	}
	_, e = p.ConvertRationalToBinaryInterval(v, qm, &q.values[0], &q.values[1], b, r.interval.BorrowLower(), r.interval.BorrowUpper())
	return m.save(l, &r, e)
}
