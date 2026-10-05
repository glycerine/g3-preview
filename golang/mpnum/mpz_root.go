// Licensed under the MIT license.
// Source: src/util/mpz.cpp; src/util/mpz.h (root/is_perfect_square and copy wrapper).
// Source author: Leonardo de Moura.
package mpnum

import "errors"

var ErrRootDegreeZero = errors.New("mpnum: nonzero integer root with degree zero divides by zero in source")
var ErrRootNegativeEven = errors.New("mpnum: negative integer root with even degree violates source assertion")
var ErrRootIntegerRequired = errors.New("mpnum: integer perfect-square query requires an integer rational")

type RootDomainReason uint8

const (
	RootZeroDegree RootDomainReason = iota + 1
	RootEvenNegative
	RootRequiresInteger
)

// RootDomainError replaces source invalid division/assertion domains. Valid
// source false-result roots are ordinary successful calls, not domain errors.
type RootDomainError struct {
	Degree uint32
	Reason RootDomainReason
}

func (e *RootDomainError) Error() string { return e.Unwrap().Error() }
func (e *RootDomainError) Unwrap() error {
	switch e.Reason {
	case RootEvenNegative:
		return ErrRootNegativeEven
	case RootRequiresInteger:
		return ErrRootIntegerRequired
	default:
		return ErrRootDegreeZero
	}
}
func rootDomain(a *Int, n uint32) error {
	if a.val < 0 && n%2 == 0 {
		return &RootDomainError{Degree: n, Reason: RootEvenNegative}
	}
	if a.val != 0 && n == 0 {
		return &RootDomainError{Degree: n, Reason: RootZeroDegree}
	}
	return nil
}

// Root is literal source SET(out,a) followed by in-place root. Caller output
// publication precedes domain checks and every bisection/body failure.
func (m *IntegerManager) Root(a *Int, n uint32, out *Int) (bool, error) {
	if e := m.Copy(out, a); e != nil {
		return false, e
	}
	return m.RootInPlace(out, n)
}
func (m *IntegerManager) RootInPlace(a *Int, n uint32) (bool, error) {
	return m.rootInPlaceWithReceipt(a, n, nil)
}
func (m *IntegerManager) rootInPlaceWithReceipt(a *Int, n uint32, receipt *integerRootInvocationReceipt) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := rootDomain(a, n); e != nil {
		return false, e
	}
	return m.rootCoreWithReceipt(a, n, receipt)
}
func (m *IntegerManager) rootCore(a *Int, n uint32) (bool, error) {
	return m.rootCoreWithReceipt(a, n, nil)
}
func (m *IntegerManager) rootCoreWithReceipt(a *Int, n uint32, receipt *integerRootInvocationReceipt) (exact bool, cause error) {
	if a.val == 0 {
		return true, nil
	}
	if e := m.beginRootRawScopeWithReceipt(receipt); e != nil {
		return false, e
	}
	s := &m.rootRaw
	defer func() {
		if cause != nil && !s.endEntered && s.bodyCause == nil {
			s.bodyCause = cause
		}
	}()
	lower, upper, mid, midN := &s.values[0], &s.values[1], &s.values[2], &s.values[3]
	var k uint32
	var e error
	if a.val > 0 {
		k, e = m.Log2(a)
	} else {
		k, e = m.MLog2(a)
	}
	if e != nil {
		return false, e
	}
	l, u := k/n, (k+1)/n
	if (k+1)%n != 0 {
		u++
	}
	two := NewInt(2)
	loExp, hiExp := NewInt(int32(l)), NewInt(int32(u))
	if a.val > 0 {
		if e = m.rootRawCall(0, &two, &loExp, lower); e != nil {
			return false, e
		}
		if e = m.rootRawCall(0, &two, &hiExp, upper); e != nil {
			return false, e
		}
	} else {
		if e = m.rootRawCall(0, &two, &hiExp, lower); e != nil {
			return false, e
		}
		if e = m.rootRawCall(0, &two, &loExp, upper); e != nil {
			return false, e
		}
		if e = m.Neg(lower); e != nil {
			return false, e
		}
		if e = m.Neg(upper); e != nil {
			return false, e
		}
	}
	if modularSourceDebug {
		le, e := m.LessEqual(lower, upper)
		if e != nil {
			return false, e
		}
		if !le {
			return false, ErrIntegerAssertion
		}
	}
	equal, e := m.Equal(lower, upper)
	if e != nil {
		return false, e
	}
	if equal {
		if e = m.Swap(a, lower); e != nil {
			return false, e
		}
		return true, m.endRootRawScope()
	}
	for {
		if e = m.rootRawCall(1, upper, lower, mid); e != nil {
			return false, e
		}
		one := NewInt(1)
		if e = m.rootRawCall(2, nil, &one, mid); e != nil {
			return false, e
		}
		power := NewInt(int32(n))
		if e = m.rootRawCall(0, mid, &power, midN); e != nil {
			return false, e
		}
		equal, e = m.Equal(midN, a)
		if e != nil {
			return false, e
		}
		if equal {
			if e = m.Swap(a, mid); e != nil {
				return false, e
			}
			return true, m.endRootRawScope()
		}
		atLower, e := m.Equal(mid, lower)
		if e != nil {
			return false, e
		}
		atEndpoint := atLower
		if !atEndpoint {
			atEndpoint, e = m.Equal(mid, upper)
			if e != nil {
				return false, e
			}
		}
		if atEndpoint {
			if e = m.Swap(a, upper); e != nil {
				return false, e
			}
			return false, m.endRootRawScope()
		}
		less, e := m.Less(midN, a)
		if e != nil {
			return false, e
		}
		if less {
			e = m.Swap(mid, lower)
		} else {
			if modularSourceDebug {
				less, e := m.Less(a, midN)
				if e != nil {
					return false, e
				}
				if !less {
					return false, ErrIntegerAssertion
				}
			}
			e = m.Swap(mid, upper)
		}
		if e != nil {
			return false, e
		}
	}
}

// IsPerfectSquare preserves source false-output conventions and its alias
// defect: resetting a nonnegative aliased output makes the input zero, so the
// result is true with output zero. Negative inputs leave out unchanged.
func (m *IntegerManager) IsPerfectSquare(a, out *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(out); e != nil {
		return false, e
	}
	if a.val < 0 {
		return false, nil
	}
	return m.squareCore(a, out)
}

func (m *IntegerManager) squareCore(a, out *Int) (exact bool, cause error) {
	if e := m.Reset(out); e != nil {
		return false, e
	}
	if a.val == 0 {
		return true, nil
	}
	one := NewInt(1)
	same, e := m.IsOne(a)
	if e != nil {
		return false, e
	}
	if same {
		return true, m.SetInt64(out, 1)
	}
	if e = m.beginPerfectSquareRawScope(); e != nil {
		return false, e
	}
	s := &m.perfectSquare
	defer func() {
		if cause != nil && !s.endEntered && s.bodyCause == nil {
			s.bodyCause = cause
		}
	}()
	lo, hi, mid, sqLo, sqHi, sqMid := &s.values[0], &s.values[1], &s.values[2], &s.values[3], &s.values[4], &s.values[5]
	if e = m.SetInt64(lo, 1); e != nil {
		return false, e
	}
	if e = m.perfectSquareCall(0, a, nil, hi); e != nil {
		return false, e
	}
	if e = m.SetInt64(sqLo, 1); e != nil {
		return false, e
	}
	if e = m.perfectSquareCall(2, hi, hi, sqHi); e != nil {
		return false, e
	}
	two := NewInt(2)
	for {
		if modularSourceDebug {
			cmp, x := m.Compare(lo, hi)
			if x != nil {
				return false, x
			}
			if cmp >= 0 {
				return false, ErrIntegerAssertion
			}
			cmp, x = m.Compare(sqLo, a)
			if x != nil {
				return false, x
			}
			if cmp > 0 {
				return false, ErrIntegerAssertion
			}
			cmp, x = m.Compare(a, sqHi)
			if x != nil {
				return false, x
			}
			if cmp >= 0 {
				return false, ErrIntegerAssertion
			}
		}
		same, e = m.Equal(sqLo, a)
		if e != nil {
			return false, e
		}
		if same {
			s.outputStarted = true
			s.outputChoice = 0
			if e = m.Copy(out, lo); e != nil {
				return false, e
			}
			exact = true
			break
		}
		if e = m.perfectSquareCall(1, lo, &one, mid); e != nil {
			return false, e
		}
		same, e = m.Equal(mid, hi)
		if e != nil {
			return false, e
		}
		if same {
			s.outputStarted = true
			s.outputChoice = 1
			if e = m.Copy(out, hi); e != nil {
				return false, e
			}
			exact = false
			break
		}
		if e = m.perfectSquareCall(1, hi, lo, mid); e != nil {
			return false, e
		}
		if e = m.perfectSquareCall(3, mid, &two, mid); e != nil {
			return false, e
		}
		if modularSourceDebug {
			cmp, x := m.Compare(lo, mid)
			if x != nil {
				return false, x
			}
			if cmp >= 0 {
				return false, ErrIntegerAssertion
			}
			cmp, x = m.Compare(mid, hi)
			if x != nil {
				return false, x
			}
			if cmp >= 0 {
				return false, ErrIntegerAssertion
			}
		}
		if e = m.perfectSquareCall(2, mid, mid, sqMid); e != nil {
			return false, e
		}
		cmp, x := m.Compare(sqMid, a)
		if x != nil {
			return false, x
		}
		if cmp > 0 {
			if e = m.perfectSquareCall(0, mid, nil, hi); e != nil {
				return false, e
			}
			if e = m.perfectSquareCall(0, sqMid, nil, sqHi); e != nil {
				return false, e
			}
		} else {
			if e = m.perfectSquareCall(0, mid, nil, lo); e != nil {
				return false, e
			}
			if e = m.perfectSquareCall(0, sqMid, nil, sqLo); e != nil {
				return false, e
			}
		}
	}
	if e = m.endPerfectSquareRawScope(); e != nil {
		return false, e
	}
	return exact, nil
}
