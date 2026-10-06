// Source: src/util/mpzzp.h. Licensed under the MIT license.
// Concrete source util/mpzzp.h manager; borrows one actual IntegerManager.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrModularOwner = errors.New("mpnum: invalid modular integer owner")

var ErrModularPrimeUninitialized = errors.New("mpnum: source modular constructor left prime flag uninitialized")
var ErrModularAssertion = errors.New("mpnum: source modular assertion failed")
var ErrModularUnsupported = errors.New("mpnum: source modular operation unsupported")

// IntegerModularManager owns seven source numerals and borrows its integer
// manager. It is caller-confined; Int operands obey the borrowed manager's
// implicit ownership contract. Explicit Close replaces the source destructor.
type IntegerModularManager struct {
	normalization                                      integerPNormalizeScope
	self                                               *IntegerModularManager
	manager                                            *IntegerManager
	z, prime, primeKnown, closed                       bool
	p, lower, upper, invTmp1, invTmp2, invTmp3, divTmp Int
	header                                             uint64
}

func NewIntegerModularManager(z *IntegerManager) (*IntegerModularManager, error) {
	if z == nil {
		return nil, arena.ErrConfig
	}
	if z.closed {
		return nil, arena.ErrClosed
	}
	n := uint64(reflect.TypeOf(IntegerModularManager{}).Size())
	if e := z.budget.Reserve(n); e != nil {
		return nil, e
	}
	m := &IntegerModularManager{manager: z, z: true, header: n}
	m.self = m
	return m, nil
}

// Source ignores prime and never initializes its bool. Only actual reads of
// that indeterminate field are rejected; normalization does not read it.
func NewIntegerModularManagerWithModulus(z *IntegerManager, p *Int, prime bool) (*IntegerModularManager, error) {
	m, e := NewIntegerModularManager(z)
	if e != nil {
		return nil, e
	}
	m.z = false
	if e = z.Copy(&m.p, p); e == nil {
		e = m.setupP()
	}
	if e != nil {
		m.Close()
		return nil, e
	}
	return m, nil
}
func NewIntegerModularManagerWithUint64(z *IntegerManager, p uint64, prime bool) (*IntegerModularManager, error) {
	m, e := NewIntegerModularManager(z)
	if e != nil {
		return nil, e
	}
	m.z = false
	if e = z.SetUint64(&m.p, p); e == nil {
		e = m.setupP()
	}
	if e != nil {
		m.Close()
		return nil, e
	}
	return m, nil
}
func (m *IntegerModularManager) check() error {
	if m == nil || m.self != m || m.manager == nil {
		return ErrModularOwner
	}
	if m.closed || m.manager.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *IntegerModularManager) Manager() (*IntegerManager, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return m.manager, nil
}
func (m *IntegerModularManager) Precise() bool { return true }
func (m *IntegerModularManager) OwnerBytes() uint64 {
	if m == nil || m.self != m || m.closed {
		return 0
	}
	return m.header
}
func (m *IntegerModularManager) Close() error {
	if m != nil && m.self == m && m.normalization.active {
		return ErrPolynomialPending
	}
	if m == nil || m.self != m || m.manager == nil {
		return ErrModularOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if !m.manager.closed {
		for _, v := range []*Int{&m.p, &m.lower, &m.upper, &m.invTmp1, &m.invTmp2, &m.invTmp3, &m.divTmp} {
			if e := m.manager.Del(v); e != nil {
				return e
			}
		}
	}
	m.manager.budget.Release(m.header)
	m.closed = true
	return nil
}
func (m *IntegerModularManager) Field() (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if m.z {
		return false, nil
	}
	if !m.primeKnown {
		return false, ErrModularPrimeUninitialized
	}
	return m.prime, nil
}
func (m *IntegerModularManager) Finite() (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return !m.z, nil
}
func (m *IntegerModularManager) Modular() (bool, error) { return m.Finite() }
func (m *IntegerModularManager) Modulus(out *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.manager.Copy(out, &m.p)
}
func (m *IntegerModularManager) setupP() error {
	return m.setupPWithAllocationContext(integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) setupPWithAllocationContext(context integerAllocationInvocationContext) error {
	z := m.manager
	if modularSourceDebug {
		pos, e := z.IsPos(&m.p)
		if e != nil {
			return e
		}
		if !pos {
			return ErrModularAssertion
		}
		one, e := z.IsOne(&m.p)
		if e != nil {
			return e
		}
		if one {
			return ErrModularAssertion
		}
	} // source invalid modulus precondition
	even, e := z.IsEven(&m.p)
	if e != nil {
		return e
	}
	two := NewInt(2)
	if e = z.divWithAllocationContext(&m.p, &two, &m.upper, context); e != nil {
		return e
	}
	if e = z.copyFromOwnerWithAllocationContext(&m.lower, z, &m.upper, nil, context); e != nil {
		return e
	}
	if e = z.negWithAllocationContext(&m.lower, context); e != nil {
		return e
	}
	if even {
		one := NewInt(1)
		return z.addSubWithAllocationContext(&m.lower, &one, &m.lower, false, context)
	}
	return nil
}
func (m *IntegerModularManager) SetZ() error {
	if e := m.check(); e != nil {
		return e
	}
	m.z = true
	return nil
}
func (m *IntegerModularManager) SetZP(p *Int) error {
	return m.setZPWithAllocationContext(p, integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) setZPWithAllocationContext(p *Int, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	m.z = false
	m.prime = true
	m.primeKnown = true
	if e := m.manager.copyFromOwnerWithAllocationContext(&m.p, m.manager, p, nil, context); e != nil {
		return e
	}
	return m.setupPWithAllocationContext(context)
}
func (m *IntegerModularManager) SetZPUint64(p uint64) error {
	if e := m.check(); e != nil {
		return e
	}
	m.z = false
	m.prime = true
	m.primeKnown = true
	if e := m.manager.SetUint64(&m.p, p); e != nil {
		return e
	}
	return m.setupP()
}
func (m *IntegerModularManager) SetPSquare() error {
	return m.setPSquareWithAllocationContext(integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) SetZPSwap(p *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if modularSourceDebug && m.z {
		return ErrModularAssertion
	}
	if e := m.manager.Swap(&m.p, p); e != nil {
		return e
	}
	return m.setupP()
}
func (m *IntegerModularManager) IsPNormalized(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if m.z {
		return true, nil
	}
	ge, e := m.manager.GreaterEqual(a, &m.lower)
	if e != nil || !ge {
		return ge, e
	}
	return m.manager.LessEqual(a, &m.upper)
}
func (m *IntegerModularManager) assertNormalized(a *Int) error {
	if !modularSourceDebug {
		return nil
	}
	b, e := m.IsPNormalized(a)
	if e != nil {
		return e
	}
	if !b {
		return ErrModularAssertion
	}
	return nil
}
func (m *IntegerModularManager) PNormalize(a *Int) error { return m.pNormalizeWithReceipt(a, nil) }
func (m *IntegerModularManager) pNormalizeWithReceipt(a *Int, receipt *integerPNormalizeInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	var scope *integerPNormalizeScope
	if receipt != nil && !m.z {
		// These are the reached Rem operand/divisor guards, before invocation.
		if e := m.manager.check(a); e != nil {
			return e
		}
		if e := m.manager.check(&m.p); e != nil {
			return e
		}
		if m.p.val == 0 {
			return ErrDivideByZero
		}
		scope = &m.normalization
		if scope.active || scope.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*scope = integerPNormalizeScope{owner: m, invocation: scope.invocation + 1, active: true, executing: true, stage: 1}
		*receipt = integerPNormalizeInvocationReceipt{m, scope, scope.invocation}
		defer func() {
			if scope.sourceCause == nil {
				scope.sourceCause = m.manager.natural.pnVerificationSourceCause(&scope.verification)
			}
			scope.executing = false
			if !scope.pending() {
				scope.active = false
			}
		}()
	}
	if !m.z {
		// This execution flag only protects the actual numerical owner from
		// returning-transport destruction. Allocation authority stays explicit.
		previousExecuting := m.manager.normalizationExecuting
		m.manager.normalizationExecuting = true
		defer func() { m.manager.normalizationExecuting = previousExecuting }()
		if e := m.manager.remWithPNReceipt(a, &m.p, a, scope); e != nil {
			return e
		}
		if scope != nil {
			scope.stage = 2
		}
		gt, e := m.manager.Greater(a, &m.upper)
		if e != nil {
			return e
		}
		if gt {
			if e = m.manager.addSubWithPNReceipt(a, &m.p, a, true, scope); e != nil {
				return e
			}
		} else {
			lt, e := m.manager.Less(a, &m.lower)
			if e != nil {
				return e
			}
			if lt {
				if e = m.manager.addSubWithPNReceipt(a, &m.p, a, false, scope); e != nil {
					return e
				}
			}
		}
		if e := m.assertNormalized(a); e != nil {
			return e
		} // core diagnostic
	}
	return m.assertNormalized(a)
}
func (m *IntegerModularManager) Inv(a *Int) error {
	return m.invWithAllocationContext(a, integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) invWithAllocationContext(a *Int, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	if m.z {
		return ErrModularUnsupported
	}
	if modularSourceDebug {
		zero, e := m.IsZero(a)
		if e != nil {
			return e
		}
		if zero {
			return ErrModularAssertion
		}
	}
	if e := m.normalizeWithSourceContext(a, context); e != nil {
		return e
	}
	if e := m.manager.extendedGCDWithAllocationContext(a, &m.p, &m.invTmp1, &m.invTmp2, &m.invTmp3, context); e != nil {
		return e
	}
	if e := m.normalizeWithSourceContext(&m.invTmp1, context); e != nil {
		return e
	}
	if e := m.manager.Swap(a, &m.invTmp1); e != nil {
		return e
	}
	if modularSourceDebug {
		one, e := m.manager.IsOne(&m.invTmp3)
		if e != nil {
			return e
		}
		if !one {
			return ErrModularAssertion
		}
	}
	return nil
}
func (m *IntegerModularManager) Div(a, b, c *Int) error {
	return m.divWithAllocationContext(a, b, c, integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) divWithAllocationContext(a, b, c *Int, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	if m.z {
		return m.manager.divWithAllocationContext(a, b, c, context)
	}
	if modularSourceDebug {
		f, e := m.Field()
		if e != nil {
			return e
		}
		if !f {
			return ErrModularAssertion
		}
		if e = m.assertNormalized(a); e != nil {
			return e
		}
	}
	if e := m.manager.copyFromOwnerWithAllocationContext(&m.divTmp, m.manager, b, nil, context); e != nil {
		return e
	}
	if e := m.invWithAllocationContext(&m.divTmp, context); e != nil {
		return e
	}
	if e := m.mulWithSourceContext(a, &m.divTmp, c, context); e != nil {
		return e
	}
	return m.assertNormalized(c)
}
func (m *IntegerModularManager) Divides(a, b *Int) (bool, error) {
	return m.dividesWithAllocationContext(a, b, integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) GCD(a, b, c *Int) error {
	return m.gcdWithSourceContext(a, b, c, integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) gcdWithSourceContext(a, b, c *Int, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	if e := m.manager.gcdWithAllocationContext(a, b, c, context); e != nil {
		return e
	}
	return m.assertNormalized(c)
}
func (m *IntegerModularManager) GCDMany(as []Int, g *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.GCDMany(as, g); e != nil {
		return e
	}
	return m.assertNormalized(g)
}
func (m *IntegerModularManager) ExtendedGCD(r1, r2, a, b, g *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(r1); e != nil {
		return e
	}
	if e := m.assertNormalized(r2); e != nil {
		return e
	}
	if e := m.manager.ExtendedGCD(r1, r2, a, b, g); e != nil {
		return e
	}
	if e := m.PNormalize(a); e != nil {
		return e
	}
	return m.PNormalize(b)
}
func (m *IntegerModularManager) Power(a *Int, k uint32, b *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	var power Int
	defer m.manager.Del(&power)
	if e := m.Copy(&power, a); e != nil {
		return e
	}
	if e := m.SetInt32(b, 1); e != nil {
		return e
	}
	for mask := uint32(1); mask <= k; {
		if mask&k != 0 {
			if e := m.Mul(b, &power, b); e != nil {
				return e
			}
		}
		if e := m.Mul(&power, &power, &power); e != nil {
			return e
		}
		mask <<= 1
		if mask == 0 {
			return &PowerDomainError{Exponent: k, Reason: PowerMaskOverflow}
		}
	}
	return nil
}
func (m *IntegerModularManager) IsPerfectSquare(a, out *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if !m.z {
		return false, ErrModularUnsupported
	}
	return m.manager.IsPerfectSquare(a, out)
}
func (m *IntegerModularManager) Root(a *Int, n uint32, out *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if modularSourceDebug && !m.z {
		return false, ErrModularAssertion
	}
	return m.manager.Root(a, n, out)
}
func (m *IntegerModularManager) RootInPlace(a *Int, n uint32) (bool, error) {
	return m.rootInPlaceWithReceipt(a, n, nil)
}
func (m *IntegerModularManager) rootInPlaceWithReceipt(a *Int, n uint32, receipt *integerRootInvocationReceipt) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if modularSourceDebug && !m.z {
		return false, ErrModularAssertion
	}
	return m.manager.rootInPlaceWithReceipt(a, n, receipt)
}

// borrowedModulus mirrors source p(): private non-owning actual numeral.
// The caller must hold this manager lifetime and must not mutate/delete it.
func (m *IntegerModularManager) borrowedModulus() (*Int, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return &m.p, nil
}
