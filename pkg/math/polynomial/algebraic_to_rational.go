// Source: src/math/polynomial/algebraic_numbers.cpp:235-241,337-341,2831-2833.
package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// ToRational copies into a caller-owned raw rational of this core's rational
// manager. VERIFY always runs the rationality predicate; it may replace a
// nonminimal rational root with a basic cell before the value is read.
func (m *AlgebraicRootCore) ToRational(a *Algebraic, out *unumeral.Rational) error {
	if e := m.check(); e != nil {
		return e
	}
	q := m.config.Rationals
	if q.Integers().MatchesCopyAllocationInvocation(&m.toRationalCopy) {
		return ErrAlgebraicPending
	}
	rational, e := m.IsRational(a)
	if e != nil {
		return e
	}
	if !rational {
		if e = m.toRationalVerifyFailure(); e != nil {
			return e
		}
	}
	if e = m.predicateBasicAssert(a, 339); e != nil {
		return e
	}
	// basic_value has its own reached assertion, independently of the caller.
	if e = m.predicateBasicAssert(a, 236); e != nil {
		return e
	}
	l, e := m.location(a)
	if e != nil {
		return e
	}
	value := m.zero
	if l != nil {
		if l.kind != 0 {
			return ErrAlgebraicOwner
		}
		record, e := m.load(l)
		if e != nil {
			return e
		}
		value = record.value
	}
	m.toRationalRunning = true
	defer func() { m.toRationalRunning = false }()
	return q.CopyWithAllocationReceipt(out, &value, &m.toRationalCopy)
}
func (m *AlgebraicRootCore) toRationalVerifyFailure() error {
	m.predicateDiagnosticRunning = true
	defer func() { m.predicateDiagnosticRunning = false }()
	if e := diagnostic.BorrowProcessDebugContext().VerifyFailure("src/math/polynomial/algebraic_numbers.cpp", 338, "is_rational(a)"); e != nil {
		return errors.Join(ErrAlgebraicAssertion, e)
	}
	return nil
}

// CleanupToRational acknowledges only this raw-rational overload's actual
// numerical copy. It neither deletes caller output nor sweeps another scope.
func (m *AlgebraicRootCore) CleanupToRational() error {
	if m == nil || m.self != m || m.closed {
		return ErrAlgebraicOwner
	}
	if m.toRationalRunning || m.predicateDiagnosticRunning {
		return ErrAlgebraicPending
	}
	return m.config.Rationals.Integers().CleanupCopyAllocationInvocation(&m.toRationalCopy)
}
