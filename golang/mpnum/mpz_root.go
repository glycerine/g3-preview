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

// Root is the source copy-then-in-place root overload, using staged owned work
// so predictable failures leave out unchanged. A false result publishes the
// source upper bound (signed ceiling), and either output/input alias is allowed.
func (m *IntegerManager) Root(a *Int, n uint32, out *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(out); e != nil {
		return false, e
	}
	if e := rootDomain(a, n); e != nil {
		return false, e
	}
	var work Int
	defer m.Del(&work)
	if e := m.Copy(&work, a); e != nil {
		return false, e
	}
	exact, e := m.rootCore(&work, n)
	if e != nil {
		return false, e
	}
	if e = m.Move(out, &work); e != nil {
		return false, e
	}
	return exact, nil
}
func (m *IntegerManager) RootInPlace(a *Int, n uint32) (bool, error) { return m.Root(a, n, a) }
func (m *IntegerManager) rootCore(a *Int, n uint32) (bool, error) {
	if a.val == 0 {
		return true, nil
	}
	var lower, upper, mid, midN Int
	defer func() { m.Del(&lower); m.Del(&upper); m.Del(&mid); m.Del(&midN) }()
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
	// Keep source unsigned k+1 width and div_l/div_u bounds. Degree zero has
	// already been classified; zero input returned above before that division.
	l := k / n
	u := (k + 1) / n
	if (k+1)%n != 0 {
		u++
	}
	two := NewInt(2)
	if a.val > 0 {
		if e = m.Power(&two, l, &lower); e != nil {
			return false, e
		}
		if e = m.Power(&two, u, &upper); e != nil {
			return false, e
		}
	} else {
		if e = m.Power(&two, u, &lower); e != nil {
			return false, e
		}
		if e = m.Power(&two, l, &upper); e != nil {
			return false, e
		}
		if e = m.Neg(&lower); e != nil {
			return false, e
		}
		if e = m.Neg(&upper); e != nil {
			return false, e
		}
	}
	compare, e := m.Compare(&lower, &upper)
	if e != nil {
		return false, e
	}
	if compare == 0 {
		if e = m.Swap(a, &lower); e != nil {
			return false, e
		}
		return true, nil
	}
	for {
		if e = m.Add(&upper, &lower, &mid); e != nil {
			return false, e
		}
		if e = m.machineDiv2kCore(&mid, 1); e != nil {
			return false, e
		}
		if e = m.Power(&mid, n, &midN); e != nil {
			return false, e
		}
		compare, e = m.Compare(&midN, a)
		if e != nil {
			return false, e
		}
		if compare == 0 {
			if e = m.Swap(a, &mid); e != nil {
				return false, e
			}
			return true, nil
		}
		atLower, e := m.Compare(&mid, &lower)
		if e != nil {
			return false, e
		}
		atUpper, e := m.Compare(&mid, &upper)
		if e != nil {
			return false, e
		}
		if atLower == 0 || atUpper == 0 {
			if e = m.Swap(a, &upper); e != nil {
				return false, e
			}
			return false, nil
		}
		if compare < 0 {
			e = m.Swap(&mid, &lower)
		} else {
			e = m.Swap(&mid, &upper)
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
