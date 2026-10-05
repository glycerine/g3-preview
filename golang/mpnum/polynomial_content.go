// Source: src/math/polynomial/polynomial.cpp:3344-3359,3384-3430.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

type polynomialContentScope struct {
	buffers                   [4]naturalPNBufferReceipt
	verification              naturalPNVerificationReceipt
	invocation                uint64
	executing                 bool
	receipt                   polynomialContentInvocationReceipt
	normalize                 integerPNormalizeInvocationReceipt
	division, extended        integerRawInvocationReceipt
	gcd                       integerRawInvocationReceipt
	aiConstructed             bool
	entered                   bool
	ai                        Int
	active, aiEnded, topOwner bool
	generation                uint64
	append                    polynomialCheapAppendInvocationReceipt
	factory                   polynomialFactoryInvocationReceipt
	numeric                   polynomialNumericRoots
	bodyCause, cleanupCause   error
}

func (m *MultivariatePolynomialCore) finishIntegerContent(owning bool) error {
	return m.finishIntegerContentPhase(owning, owning)
}
func (m *MultivariatePolynomialCore) finishIntegerContentPhase(owning, abort bool) error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.contentLocal
	if !owning && s.cleanupCause != nil {
		return s.cleanupCause
	}
	if !s.active {
		return nil
	}
	if owning && s.executing {
		return ErrPolynomialPending
	}

	if !owning {
		for _, e := range []error{m.integer.rawInvocationSourceCause(&s.gcd), m.integer.rawInvocationSourceCause(&s.division), m.integer.rawInvocationSourceCause(&s.extended), m.modular.pNormalizeInvocationSourceCause(&s.normalize), m.integer.natural.pnVerificationSourceCause(&s.verification)} {
			if e != nil {
				if s.cleanupCause == nil {
					s.cleanupCause = e
				}
				return s.cleanupCause
			}
		}
	}
	if owning {
		if e := m.integer.cleanupRawInvocation(&s.gcd, false); e != nil {
			return e
		}
		if e := m.modular.cleanupPNormalizeInvocation(&s.normalize, false); e != nil {
			return e
		}
		if e := m.integer.natural.cleanupPNVerification(&s.verification, false); e != nil {
			return e
		}
		if e := m.integer.cleanupRawInvocation(&s.division, false); e != nil {
			return e
		}
		if e := m.integer.cleanupRawInvocation(&s.extended, false); e != nil {
			return e
		}
	}
	if !owning && s.aiEnded {
		return nil
	}
	var e error
	if s.aiConstructed {
		if owning {
			e = m.deleteNumeral(&s.ai)
		} else {
			_, e = upolynomialSourceNumeralDel(m.integer, &s.ai)
		}
	}
	if e != nil {
		if !owning && s.cleanupCause == nil {
			s.cleanupCause = e
		}
		return e
	}
	if s.aiConstructed {
		s.aiEnded = true
	}
	if owning && abort {
		if e = m.integer.cleanupRawInvocation(&s.gcd, true); e != nil {
			return e
		}
		if e = m.modular.cleanupPNormalizeInvocation(&s.normalize, true); e != nil {
			return e
		}
		if e = m.integer.natural.cleanupPNVerification(&s.verification, true); e != nil {
			return e
		}
		if e = m.integer.cleanupRawInvocation(&s.division, true); e != nil {
			return e
		}
		if e = m.integer.cleanupRawInvocation(&s.extended, true); e != nil {
			return e
		}
		for j := len(s.buffers) - 1; j >= 0; j-- {
			if e = s.buffers[j].cleanup(); e != nil {
				return e
			}
		}
		if e = m.cleanupFactoryInvocation(&s.factory, true); e != nil {
			return e
		}
		if e = m.cheap.cleanupAppendInvocation(&s.append, true); e != nil {
			return e
		}
		if e = s.numeric.cleanup(m.integer); e != nil {
			return e
		}
		*s = polynomialContentScope{invocation: s.invocation, bodyCause: s.bodyCause, cleanupCause: s.cleanupCause, gcd: s.gcd, division: s.division, extended: s.extended, normalize: s.normalize, verification: s.verification}
		return nil
	}
	pending := m.integer.matchesRawInvocation(&s.gcd) || m.integer.matchesRawInvocation(&s.division) || m.integer.matchesRawInvocation(&s.extended) || m.integer.natural.matchesPNVerification(&s.verification) || polynomialNegNumericPending(&s.numeric)
	for j := range s.buffers {
		pending = pending || s.buffers[j].pending != (naturalPNBufferReceipt{}).pending
	}
	pending = pending || m.cheap.matchesAppendInvocation(&s.append) || m.matchesFactoryInvocation(&s.factory) || s.bodyCause != nil
	pending = pending || s.numeric.cell.object.OwnerID() != 0
	for _, p := range s.numeric.allocator {
		pending = pending || p.OwnerID() != 0
	}
	pending = pending || (s.topOwner && m.pscPending())
	if !pending {
		*s = polynomialContentScope{invocation: s.invocation, bodyCause: s.bodyCause, cleanupCause: s.cleanupCause, gcd: s.gcd, division: s.division, extended: s.extended, normalize: s.normalize, verification: s.verification}
	}

	return nil
}
func (m *MultivariatePolynomialCore) IntegerContent(p Polynomial, i *Int, c *PolynomialRef) error {
	if e := m.check(); e != nil {
		return e
	}
	if i == nil || c == nil || c.owner != m {
		return ErrPolynomialOwner
	}
	if e := c.open(); e != nil {
		return e
	}
	return m.integerContent(p, i, &c.root)
}
func (m *MultivariatePolynomialCore) integerContent(p Polynomial, i *Int, c *polynomialReference) error {
	return m.integerContentWithReceipt(p, i, c, nil)
}
func (m *MultivariatePolynomialCore) integerContentWithReceipt(p Polynomial, i *Int, c *polynomialReference, r *polynomialContentInvocationReceipt) error {
	return m.integerContentWithReceipts(p, i, c, r, nil, nil)
}
func (m *MultivariatePolynomialCore) integerContentForICCPShortcut(p Polynomial, i *Int, c *polynomialReference, parent *MultivariatePolynomialManager, r *polynomialICCPInvocationReceipt) error {
	return m.integerContentWithReceipts(p, i, c, nil, parent, r)
}
func (m *MultivariatePolynomialCore) beginContentInvocation(i *Int, receipt *polynomialContentInvocationReceipt, shortcut *MultivariatePolynomialManager, outer *polynomialICCPInvocationReceipt) (*polynomialContentScope, error) {
	if shortcut != nil && (shortcut.self != shortcut || shortcut.core != m) {
		return nil, ErrPolynomialOwner
	}
	s := &m.contentLocal
	if s.active || s.invocation == ^uint64(0) {
		return nil, ErrPolynomialPending
	}
	if e := m.integer.check(i); e != nil {
		return nil, e
	}
	*s = polynomialContentScope{invocation: s.invocation + 1, active: true, executing: true}
	s.receipt = polynomialContentInvocationReceipt{m, s, s.invocation}
	if shortcut != nil {
		shortcut.iccpShortcutContent = s.receipt
		if outer != nil {
			*outer = polynomialICCPInvocationReceipt{manager: shortcut, shortcutContent: s.receipt}
		}
	}

	if receipt != nil {
		*receipt = s.receipt
	}
	return s, nil
}

func (m *MultivariatePolynomialCore) integerContentWithReceipts(p Polynomial, i *Int, c *polynomialReference, receipt *polynomialContentInvocationReceipt, shortcut *MultivariatePolynomialManager, outer *polynomialICCPInvocationReceipt) (err error) {
	zero, e := m.IsZero(p)
	if e != nil {
		return e
	}
	if zero {
		if e = m.modular.Reset(i); e != nil {
			return e
		}
		return c.assign(m, p)
	}
	constant, e := m.IsConstant(p)
	if e != nil {
		return e
	}
	if constant {
		a, e := m.coefficientCore(p, 0)
		if e != nil {
			return e
		}
		s, e := m.beginContentInvocation(i, receipt, shortcut, outer)
		if e != nil {
			return e
		}
		defer func() {
			s.executing = false
			s.bodyCause = err
			if x := m.finishIntegerContent(false); x != nil {
				err = polynomialNewtonJoin(err, x)
			}
		}()
		if e = m.modular.copyWithContentReceipt(i, &a, &s.receipt); e != nil {
			return e
		}
		return c.assign(m, m.unit)
	}
	n, e := m.readWord(p, 8)
	if e != nil {
		return e
	}
	l, e := m.polynomialLocation(p)
	if e != nil {
		return e
	}
	view := integerArrayView{allocator: m.monomials.allocator, object: l.raw, count: n, offset: 12, stride: 12}
	s, e := m.beginContentInvocation(i, receipt, shortcut, outer)
	if e != nil {
		return e
	}
	defer func() {
		s.executing = false
		if s.entered {
			m.pscLeave()
			s.entered = false
		}
		s.bodyCause = err
		if x := m.finishIntegerContent(false); x != nil {
			err = polynomialNewtonJoin(err, x)
		}
	}()
	e = m.modular.gcdManyIndexedWithContentReceipt(view, i, &s.receipt)
	if e != nil {
		return e
	}
	one, e := m.modular.IsOne(i)
	if e != nil {
		return e
	}
	if one {
		return c.assign(m, p)
	}
	if e = m.cheap.reset(); e != nil {
		return e
	}
	// This is the literal reached scoped_numeral ai constructor, after reset.
	s.ai = Int{}
	s.aiConstructed = true
	top := m.pscReceipts.depth == 0
	if e = m.pscEnter(); e != nil {
		return e
	}
	s.entered = true
	s.topOwner = top
	s.generation = m.pscExecutingOwner()
	for j := uint32(0); j < n; j++ {
		term, e := m.monomialCore(p, j)
		if e != nil {
			return e
		}
		a, e := m.coefficientCore(p, j)
		if e != nil {
			return e
		}
		e = m.modular.divWithAllocationContext(&a, i, &s.ai, m.modular.contentAllocationContext(&s.receipt))
		if e != nil {
			return e
		}
		e = m.cheap.addConsumedReceipts(&s.ai, term, true, nil, nil, &s.append)
		if e != nil {
			return e
		}
	}
	r, e := m.cheap.mkWithAppendReceipt(false, &s.append, &s.factory)
	if e != nil {
		return e
	}
	return c.assignWithFactoryReceipt(m, r, &s.factory)
}

// IntegerScalarContent is the source numeral-only overload. Unlike ic(p,a,c),
// its first SET preserves a lone negative coefficient and its early-one test
// occurs before reading the next coefficient.
func (full *MultivariatePolynomialManager) IntegerScalarContent(p Polynomial, out *Int) error {
	if e := full.ready(); e != nil {
		return e
	}
	if out == nil {
		return ErrPolynomialOwner
	}
	return full.core.integerScalarContent(p, out)
}
func (m *MultivariatePolynomialCore) integerScalarContent(p Polynomial, out *Int) error {
	zero, e := m.IsZero(p)
	if e != nil {
		return e
	}
	if zero {
		return m.modular.Reset(out)
	}
	constant, e := m.IsConstant(p)
	if e != nil {
		return e
	}
	if constant {
		a, e := m.coefficientCore(p, 0)
		if e != nil {
			return e
		}
		return m.modular.Copy(out, &a)
	}
	a, e := m.coefficientCore(p, 0)
	if e != nil {
		return e
	}
	if e = m.modular.Copy(out, &a); e != nil {
		return e
	}
	size, e := m.readWord(p, 8)
	if e != nil {
		return e
	}
	for i := uint32(1); i < size; i++ {
		one, e := m.modular.IsOne(out)
		if e != nil {
			return e
		}
		if one {
			return nil
		}
		a, e := m.coefficientCore(p, i)
		if e != nil {
			return e
		}
		if e = m.modular.GCD(out, &a, out); e != nil {
			return e
		}
	}
	return nil
}

// CleanupIntegerContent is explicit owning continuation of this actual local.
// Direct pending leaves and actual child frames belong to this continuation.
// Published numerals and returned references remain owned by their callers.
func (m *MultivariatePolynomialCore) CleanupIntegerContent() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	s := &m.contentLocal
	top, g := s.topOwner, s.generation
	if e := m.finishIntegerContent(true); e != nil {
		return e
	}
	if top && g != 0 && m.pscReceipts.generation == g {
		return m.CleanupPSC()
	}
	return nil
}
