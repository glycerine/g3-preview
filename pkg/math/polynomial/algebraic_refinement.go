// algebraic_numbers.cpp:243–335,703–781,897–929.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicPredicateScope struct {
	leadingCopy                                                                                unumeral.IntegerCopyAllocationReceipt
	savedCopy                                                                                  unumeral.IntegerCopyAllocationReceipt
	refinement                                                                                 algebraicRefinementInvocation
	childStop, ownCause                                                                        error
	ownStage                                                                                   uint8
	finishing                                                                                  bool
	saved                                                                                      unumeral.BinaryInterval
	owner                                                                                      Algebraic
	lower, upper                                                                               unumeral.BinaryRational
	candidate                                                                                  unumeral.Int
	active, savedActive, savedConstructed, restored, candidateActive, lowerActive, upperActive bool
	stage                                                                                      uint8
	cause, cleanup                                                                             error
}

func (m *AlgebraicRootCore) magnitude(l, u *unumeral.BinaryRational) (int32, error) {
	b := m.config.Binary
	z := m.config.Rationals.Integers()
	if polynomialSourceDebug {
		nn, e := b.IsNonNeg(l)
		if e != nil {
			return 0, e
		}
		np, e := b.IsNonPos(u)
		if e != nil {
			return 0, e
		}
		if !nn && !np {
			return 0, ErrAlgebraicAssertion
		}
	}
	if l.K() == u.K() {
		return b.MagnitudeUpper(l)
	}
	positive, e := b.IsNonNeg(l)
	if e != nil {
		return 0, e
	}
	var x, y uint32
	if positive {
		x, e = z.Log2(&(*u.BorrowNumerator()))
		if e != nil {
			return 0, e
		}
		y, e = z.Log2(&(*l.BorrowNumerator()))
	} else {
		x, e = z.MLog2(&(*u.BorrowNumerator()))
		if e != nil {
			return 0, e
		}
		y, e = z.MLog2(&(*l.BorrowNumerator()))
	}
	if e != nil {
		return 0, e
	}
	// Source starts with unsigned log2, so each signed k participates after
	// unsigned32 promotion. Final uint32-to-int conversion uses the selected
	// int32 signed-bit profile, including representable extreme exponents.
	return int32(x - y - u.K() + l.K() - u.K()), nil
}
func (m *AlgebraicRootCore) refine(a *Algebraic) (bool, error) { return m.refineAt(a, 0, false) }
func (m *AlgebraicRootCore) refineN(a *Algebraic, k uint32) (bool, error) {
	for i := uint32(0); i < k; i++ {
		ok, e := m.refine(a)
		if e != nil || !ok {
			return ok, e
		}
	}
	return true, nil
}
func (m *AlgebraicRootCore) Refine(a *Algebraic) error { _, e := m.refine(a); return e }
func (m *AlgebraicRootCore) RefineToPrecision(a *Algebraic, k uint32) error {
	_, e := m.refineAt(a, k, true)
	return e
}
func (m *AlgebraicRootCore) refineAt(a *Algebraic, k uint32, precision bool) (bool, error) {
	return m.refineAtWithReceipt(a, k, precision, nil)
}
func (m *AlgebraicRootCore) refineAtWithReceipt(a *Algebraic, k uint32, precision bool, receipt *algebraicRefinementInvocation) (bool, error) {
	if m != nil && m.self == m && (m.refinement.pending(m) || m.refinementConversion.pending(m) || m.refinementQ.active) {
		return false, ErrAlgebraicPending
	}
	l, e := m.location(a)
	if e != nil {
		return false, e
	}
	if l == nil || l.kind == 0 {
		return precision, nil
	}
	r, e := m.load(l)
	if e != nil {
		return false, e
	}
	v, e := m.sourcePolynomial(&r)
	if e != nil {
		return false, e
	}
	s := 1
	if r.flags&2 != 0 {
		s = -1
	}
	ok, e := m.refineRecordWithReceipt(l, &r, v, s, k, precision, receipt)
	if e != nil {
		return false, e
	}
	if ok {
		return true, nil
	}
	return false, m.promoteRefinedRoot(a, &r)
}

// The exact-root branch after refineRecord has published its interval.
func (m *AlgebraicRootCore) promoteRefinedRoot(a *Algebraic, r *algebraicRecord) error {
	if e := m.beginRefinementQ(); e != nil {
		return e
	}
	if e := m.refinementConversion.run(m, r.interval.BorrowLower(), &m.localQ); e != nil {
		return m.finishRefinementQ(e)
	}
	if e := m.Del(a); e != nil {
		return m.finishRefinementQ(e)
	}
	e := m.SetTake(a, &m.localQ)
	return m.finishRefinementQ(e)
}
func (m *AlgebraicRootCore) beginPredicate() error {
	if m.predicate.active {
		return ErrAlgebraicPending
	}
	m.predicate = algebraicPredicateScope{active: true}
	return nil
}
func (m *AlgebraicRootCore) restoreSaved() error {
	p := &m.predicate
	if !p.savedActive || !p.savedConstructed || p.restored {
		return nil
	}
	p.restored = true
	l, e := m.locationCleanup(&p.owner)
	if e != nil {
		return e
	}
	if l == nil || l.kind == 0 {
		return nil
	}
	r, e := m.load(l)
	if e != nil {
		return e
	}
	mag, e := m.magnitude(&(*r.interval.BorrowLower()), &(*r.interval.BorrowUpper()))
	if e != nil {
		return e
	}
	if mag < m.minMagnitude {
		e = m.config.Intervals.Swap(&r.interval, &p.saved)
		return m.save(l, &r, e)
	}
	return nil
}
func (m *AlgebraicRootCore) finishPredicate(cause error) error {
	p := &m.predicate
	if !p.active {
		return cause
	}
	if p.finishing {
		return errors.Join(cause, ErrAlgebraicPending)
	}
	p.finishing = true
	defer func() { p.finishing = false }()
	if p.cause == nil {
		p.cause = cause
	}
	if p.childStop != nil {
		return errors.Join(cause, p.childStop)
	}
	// Actual scoped locals destruct in reverse declaration order. A failed saved
	// constructor has no source destructor; its partial fields get owning cleanup.
	for p.stage < 5 {
		var e error
		switch p.stage {
		case 0:
			if p.candidateActive {
				e = m.config.Rationals.Integers().Del(&p.candidate)
			}
		case 1:
			if p.upperActive {
				e = m.config.Binary.Del(&p.upper)
			}
		case 2:
			if p.lowerActive {
				e = m.config.Binary.Del(&p.lower)
			}
		case 3:
			e = m.restoreSaved()
		case 4:
			if p.savedActive && p.savedConstructed {
				e = m.config.Intervals.Del(&p.saved)
			}
		}
		if e != nil {
			p.cleanup = e
			if cause != nil {
				return cause
			}
			return e
		}
		p.stage++
	}
	if p.savedActive && !p.savedConstructed {
		return cause
	}
	z := m.config.Rationals.Integers()
	if z.MatchesCopyAllocationInvocation(&p.savedCopy) || z.MatchesCopyAllocationInvocation(&p.leadingCopy) {
		return cause
	}
	m.predicate = algebraicPredicateScope{}
	return cause
}
func (m *AlgebraicRootCore) IsInt(a *Algebraic) (result bool, err error) {
	l, err := m.location(a)
	if err != nil {
		return false, err
	}
	if l == nil {
		return true, nil
	}
	r, err := m.load(l)
	if err != nil {
		return false, err
	}
	if l.kind == 0 {
		return m.config.Rationals.IsInt(&r.value)
	}
	if r.flags&4 != 0 {
		return false, nil
	}
	ok, err := m.refineAt(a, 1, true)
	if err != nil {
		return false, err
	}
	if !ok {
		return m.IsInt(a)
	}
	l, err = m.location(a)
	if err != nil {
		return false, err
	}
	r, err = m.load(l)
	if err != nil {
		return false, err
	}
	if err = m.beginPredicate(); err != nil {
		return false, err
	}
	defer func() { err = m.finishPredicate(err) }()
	p := &m.predicate
	p.candidateActive = true
	if err = m.config.Binary.Floor(&(*r.interval.BorrowUpper()), &p.candidate); err != nil {
		return false, err
	}
	if polynomialSourceDebug {
		ok, e := m.config.Binary.GreaterEqualInteger(&(*r.interval.BorrowUpper()), &p.candidate)
		if e != nil {
			return false, e
		}
		if !ok {
			return false, ErrAlgebraicAssertion
		}
	}
	less, err := m.config.Binary.LessInteger(&(*r.interval.BorrowLower()), &p.candidate)
	if err != nil || !less {
		return false, err
	}
	v, err := m.sourcePolynomial(&r)
	if err != nil {
		return false, err
	}
	s, err := m.config.Polynomials.EvalSignInt(v, &p.candidate)
	if err != nil || s != 0 {
		return false, err
	}
	if err = m.beginQ(); err != nil {
		return false, err
	}
	err = m.config.Rationals.SetInteger(&m.localQ, &p.candidate)
	if err == nil {
		err = m.SetTake(a, &m.localQ)
	}
	err = m.finishQ(err)
	return err == nil, err
}
func (m *AlgebraicRootCore) IsRational(a *Algebraic) (result bool, err error) {
	l, err := m.location(a)
	if err != nil {
		return false, err
	}
	if l == nil || l.kind == 0 {
		return true, nil
	}
	r, err := m.load(l)
	if err != nil {
		return false, err
	}
	if r.flags&4 != 0 {
		return false, nil
	}
	if err = m.beginPredicate(); err != nil {
		return false, err
	}
	// cpp292 passes c, implicitly creating temporary anum (header366),
	// whose const& in save_intervals cpp897–929 dangles. Go explicitly binds
	// this actual caller for safe ownership; this is not defined C++ cleanup.
	defer func() { m.predicate.owner = *a; err = m.finishPredicate(err) }()
	p := &m.predicate
	p.savedActive = true
	p.owner = *a
	if err = m.config.Intervals.CopyWithAllocationReceipt(&p.saved, &r.interval, &p.savedCopy); err != nil {
		return false, err
	}
	p.savedConstructed = true
	v, err := m.sourcePolynomial(&r)
	if err != nil {
		return false, err
	}
	lead, err := m.config.Polynomials.read(v, r.size-1)
	if err != nil {
		return false, err
	}
	z := m.config.Rationals.Integers()
	if err = z.CopyFromOwnerWithReceipt(&m.isRationalTmp, z, lead, &p.leadingCopy); err != nil {
		return false, err
	}
	if err = z.Abs(&m.isRationalTmp); err != nil {
		return false, err
	}
	k, err := z.Log2(&m.isRationalTmp)
	if err != nil {
		return false, err
	}
	k++
	ok, err := m.refineAtWithReceipt(a, k, true, &p.refinement)
	if err != nil {
		p.childStop = m.refinementSourceStop(&p.refinement)
	}
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	l, err = m.location(a)
	if err != nil {
		return false, err
	}
	r, err = m.load(l)
	if err != nil {
		return false, err
	}
	p.lowerActive = true
	if err = m.config.Binary.MulInteger(&(*r.interval.BorrowLower()), &m.isRationalTmp, &p.lower); err != nil {
		return false, err
	}
	p.upperActive = true
	if err = m.config.Binary.MulInteger(&(*r.interval.BorrowUpper()), &m.isRationalTmp, &p.upper); err != nil {
		return false, err
	}
	p.candidateActive = true
	if err = m.config.Binary.Floor(&p.upper, &p.candidate); err != nil {
		return false, err
	}
	if err = m.beginQ(); err != nil {
		return false, err
	}
	defer func() { err = m.finishQ(err) }()
	if err = m.config.Rationals.SetParts(&m.localQ, &p.candidate, &m.isRationalTmp); err != nil {
		return false, err
	}
	if polynomialSourceDebug {
		ok, e := m.config.Binary.GreaterEqualRational(&(*r.interval.BorrowUpper()), m.config.Rationals, &m.localQ)
		if e != nil {
			return false, e
		}
		if !ok {
			return false, ErrAlgebraicAssertion
		}
	}
	less, err := m.config.Binary.LessRational(&(*r.interval.BorrowLower()), m.config.Rationals, &m.localQ)
	if err != nil {
		return false, err
	}
	if less {
		v, err = m.sourcePolynomial(&r)
		if err != nil {
			return false, err
		}
		s, e := m.config.Polynomials.EvalSignRational(v, &m.localQ)
		if e != nil {
			return false, e
		}
		if s == 0 {
			if err = m.restoreSaved(); err != nil {
				return false, err
			}
			err = m.SetTake(a, &m.localQ)
			return err == nil, err
		}
	}
	if err = m.restoreSaved(); err != nil {
		return false, err
	}
	l, err = m.location(a)
	if err != nil {
		return false, err
	}
	r, err = m.load(l)
	if err != nil {
		return false, err
	}
	r.flags |= 4
	return false, m.store(l, &r)
}

// Store every interval publication, including partial body/error results,
// before releasing the executing span. Rational-root promotion follows it.
func (m *AlgebraicRootCore) refineRecord(l *algebraicLocation, r *algebraicRecord, v UPolynomialView, sign int, k uint32, precision bool) (ok bool, cause error) {
	return m.refineRecordWithReceipt(l, r, v, sign, k, precision, nil)
}
func (m *AlgebraicRootCore) refineRecordWithReceipt(l *algebraicLocation, r *algebraicRecord, v UPolynomialView, sign int, k uint32, precision bool, receipt *algebraicRefinementInvocation) (ok bool, cause error) {
	s := &m.refinement
	if e := s.begin(m); e != nil {
		return false, e
	}
	if receipt != nil {
		*receipt = algebraicRefinementInvocation{m, s, s.invocation}
	}
	defer func() { s.executing = false }()
	p, b := m.config.Polynomials, m.config.Binary
	if precision {
		// cpp:773 recomputes the sign via refine, unlike refine_core(c):730.
		ok, cause = p.refineToPrecisionWithEvaluationReceipt(v, b, r.interval.BorrowLower(), r.interval.BorrowUpper(), k, &s.evaluation, s)
	} else {
		ok, cause = p.refineCoreWithEvaluationReceipt(v, sign, b, r.interval.BorrowLower(), r.interval.BorrowUpper(), &s.evaluation, s)
	}
	return ok, m.save(l, r, cause)
}

// Promotion's source scoped_mpq destructor is separate from owning cleanup.
func (m *AlgebraicRootCore) finishRefinementQ(cause error) error {
	return m.refinementQ.finish(m, cause)
}
