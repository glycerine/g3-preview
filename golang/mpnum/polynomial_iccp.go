// Source: src/math/polynomial/polynomial.cpp:3432-3534;
// src/math/polynomial/polynomial.h:polynomial manager content overloads.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// One concrete lexical invocation, including the two actual sbuffer<unsigned,
// 128> objects and the ci reference constructed only after the integer content.
// The cold Go directory is an owning adapter, not a source allocation ticket.
type polynomialICCPScope struct {
	invocation                      uint64
	childInvocation                 uint64
	gcdInvocation                   polynomialGCDInvocationReceipt
	contentInvocation               polynomialContentInvocationReceipt
	parent, child                   *polynomialICCPScope
	filter, powers                  polynomialContentUnsignedBuffer
	ci                              polynomialReference
	active, ciConstructed, topOwner bool
	generation                      uint64
	cursor                          uint8
	bodyCause, cleanupCause         error
	header                          uint64
}

func (m *MultivariatePolynomialManager) iccpFrame() (*polynomialICCPScope, error) {
	if int(m.iccpDepth) < len(m.iccpFrames) {
		s := m.iccpFrames[m.iccpDepth]
		if s.active {
			return nil, ErrPolynomialPending
		}
		return s, nil
	}
	old, capacity := cap(m.iccpFrames), cap(m.iccpFrames)
	if len(m.iccpFrames) == capacity {
		capacity = capacity*2 + 1
		bytes := uint64(capacity) * uint64(reflect.TypeOf((*polynomialICCPScope)(nil)).Size())
		if e := m.core.integer.budget.Reserve(bytes); e != nil {
			return nil, e
		}
		frames := make([]*polynomialICCPScope, len(m.iccpFrames), capacity)
		copy(frames, m.iccpFrames)
		m.iccpFrames = frames
		m.core.integer.budget.Release(uint64(old) * uint64(reflect.TypeOf((*polynomialICCPScope)(nil)).Size()))
	}
	h := uint64(reflect.TypeOf(polynomialICCPScope{}).Size())
	if e := m.core.integer.budget.Reserve(h); e != nil {
		return nil, e
	}
	s := &polynomialICCPScope{header: h}
	m.iccpFrames = append(m.iccpFrames, s)
	return s, nil
}
func (m *MultivariatePolynomialManager) iccpSourceEnd(s *polynomialICCPScope) error {
	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	c := m.core
	if e := c.contentInvocationSourceCause(&s.contentInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if e := m.gcdInvocationSourceCause(&s.gcdInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if s.child != nil && s.child.invocation == s.childInvocation && s.child.active && s.child.cleanupCause != nil {
		s.cleanupCause = s.child.cleanupCause
		return s.cleanupCause
	}
	if e := c.pscSourceStopped(); e != nil {
		s.cleanupCause = e
		return e
	}
	if s.cursor == 0 {
		if s.ciConstructed {
			if e := s.ci.closeValue(c, false); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.cursor = 1
	}
	if s.cursor == 1 {
		if e := s.powers.close(false); e != nil {
			s.cleanupCause = e
			return e
		}
		s.cursor = 2
	}
	if s.cursor == 2 {
		if e := s.filter.close(false); e != nil {
			s.cleanupCause = e
			return e
		}
		s.cursor = 3
	}
	if s.filter.pending == (memory.Block{}) && s.powers.pending == (memory.Block{}) && !s.ci.assigning && !(s.topOwner && c.pscPending()) && !c.matchesContentInvocation(&s.contentInvocation) {
		h := s.header
		*s = polynomialICCPScope{header: h, invocation: s.invocation}
	}
	return nil
}
func (m *MultivariatePolynomialManager) flipLeadingSign(p *polynomialReference) error {
	c := m.core
	zero, e := c.IsZero(p.value)
	if e != nil || zero {
		return e
	}
	pos, e := c.gradedLexMaxPosition(p.value)
	if e != nil {
		return e
	}
	if modularSourceDebug && pos == ^uint32(0) {
		return ErrPolynomialAssertion
	}
	a, e := c.coefficientCore(p.value, pos)
	if e != nil {
		return e
	}
	negative, e := c.modular.IsNeg(&a)
	if e != nil || !negative {
		return e
	}
	q, e := c.Neg(p.value)
	if e != nil {
		return e
	}
	return p.assign(c, q)
}
func (m *MultivariatePolynomialManager) IntegerContentPrimitiveParts(p Polynomial, x uint32, i *Int, c, pp *PolynomialRef) error {
	if e := m.ready(); e != nil {
		return e
	}
	if i == nil || c == nil || pp == nil || c.owner != m.core || pp.owner != m.core {
		return ErrPolynomialOwner
	}
	if e := c.open(); e != nil {
		return e
	}
	if e := pp.open(); e != nil {
		return e
	}
	return m.iccpParts(p, x, i, &c.root, &pp.root)
}
func (m *MultivariatePolynomialManager) iccpParts(p Polynomial, x uint32, i *Int, c, pp *polynomialReference) error {
	return m.iccpPartsWithReceipt(p, x, i, c, pp, nil)
}
func (m *MultivariatePolynomialManager) iccpPartsWithReceipt(p Polynomial, x uint32, i *Int, c, pp *polynomialReference, receipt *polynomialICCPInvocationReceipt) (err error) {
	core := m.core
	zero, e := core.IsZero(p)
	if e != nil {
		return e
	}
	if zero {
		if e = core.modular.Reset(i); e != nil {
			return e
		}
		if e = c.assign(core, core.unit); e != nil {
			return e
		}
		return pp.assign(core, p)
	}
	constant, e := core.IsConstant(p)
	if e != nil {
		return e
	}
	if constant {
		a, e := core.coefficientCore(p, 0)
		if e != nil {
			return e
		}
		e = core.modular.Copy(i, &a)
		if e != nil {
			return e
		}
		if e = c.assign(core, core.unit); e != nil {
			return e
		}
		return pp.assign(core, core.unit)
	}
	d, e := core.Degree(p, x)
	if e != nil {
		return e
	}
	if d == 0 {
		if e = core.integerContentForICCPShortcut(p, i, c, m, receipt); e != nil {
			return e
		}
		return pp.assign(core, core.unit)
	}
	s, e := m.iccpFrame()
	if e != nil {
		return e
	}
	if s.invocation == ^uint64(0) {
		return arena.ErrBounds
	}
	top := core.pscReceipts.depth == 0
	if e = core.pscEnter(); e != nil {
		return e
	}
	h := s.header
	*s = polynomialICCPScope{invocation: s.invocation + 1, active: true, topOwner: top, generation: core.pscExecutingOwner(), header: h}
	if m.iccpDepth != 0 {
		s.parent = m.iccpFrames[m.iccpDepth-1]
		s.parent.child = s
		s.parent.childInvocation = s.invocation
	}
	if receipt != nil {
		*receipt = polynomialICCPInvocationReceipt{manager: m, frame: s, invocation: s.invocation}
	}
	m.iccpDepth++
	s.filter.construct(core.memory)
	s.powers.construct(core.memory)
	defer func() { m.iccpDepth--; core.pscLeave(); s.bodyCause = err; err = errors.Join(err, m.iccpSourceEnd(s)) }()
	if e = s.filter.resize(d+1, 0); e != nil {
		return e
	}
	s.powers.reset()
	for j := uint32(0); j <= d; j++ {
		if e = s.filter.set(j, 0); e != nil {
			return e
		}
	}
	size, e := core.readWord(p, 8)
	if e != nil {
		return e
	}
	for j := uint32(0); j < size; j++ {
		term, e := core.monomialCore(p, j)
		if e != nil {
			return e
		}
		k, e := core.monomials.DegreeOf(term, x)
		if e != nil {
			return e
		}
		count, e := s.filter.get(k)
		if e != nil {
			return e
		}
		if count == 0 {
			if e = s.powers.push(k); e != nil {
				return e
			}
		}
		sz, e := core.monomials.sizeCore(term)
		if e != nil {
			return e
		}
		expected := uint32(0)
		if k > 0 {
			expected = 1
		}
		// Source rereads the counter in ++ / += after the reached size read.
		count, e = s.filter.get(k)
		if e != nil {
			return e
		}
		if sz == expected {
			count++
		} else {
			count += 2
		}
		if e = s.filter.set(k, count); e != nil {
			return e
		}
	}
	if modularSourceDebug && s.powers.count == 0 {
		return ErrPolynomialAssertion
	}
	number := s.powers.count
	for j := uint32(0); j < number; j++ {
		k, e := s.powers.get(j)
		if e != nil {
			return e
		}
		if modularSourceDebug {
			count, e := s.filter.get(k)
			if e != nil {
				return e
			}
			if count == 0 {
				return ErrPolynomialAssertion
			}
		}
		k, e = s.powers.get(j)
		if e != nil {
			return e
		}
		count, e := s.filter.get(k)
		if e != nil {
			return e
		}
		if count == 1 {
			if e = core.integerContentWithReceipt(p, i, pp, &s.contentInvocation); e != nil {
				return e
			}
			return c.assign(core, core.unit)
		}
	}
	if e = core.integerContentWithReceipt(p, i, pp, &s.contentInvocation); e != nil {
		return e
	}
	s.ciConstructed = true
	k, e := s.powers.get(0)
	if e != nil {
		return e
	}
	q, e := core.Coeff(pp.value, x, k)
	if e != nil {
		return e
	}
	if e = c.assign(core, q); e != nil {
		return e
	}
	for j := uint32(1); j < number; j++ {
		k, e := s.powers.get(j)
		if e != nil {
			return e
		}
		q, e := core.Coeff(pp.value, x, k)
		if e != nil {
			return e
		}
		if e = s.ci.assign(core, q); e != nil {
			return e
		}
		if e = m.gcdIntoWithReceipt(c.value, s.ci.value, c, &s.gcdInvocation); e != nil {
			return e
		}
		constant, e := core.IsConstant(c.value)
		if e != nil {
			return e
		}
		if constant {
			return c.assign(core, core.unit)
		}
	}
	if modularSourceDebug {
		constant, e := core.IsConstant(c.value)
		if e != nil {
			return e
		}
		if constant {
			return ErrPolynomialAssertion
		}
	}
	if e = m.flipLeadingSign(c); e != nil {
		return e
	}
	q, e = core.ExactDiv(pp.value, c.value)
	if e != nil {
		return e
	}
	return pp.assign(core, q)
}

// CleanupContentPrimitiveParts is the separate owning continuation. Completed
// younger source objects are acknowledged before unjoined reference INCs.
func (m *MultivariatePolynomialManager) CleanupContentPrimitiveParts() error {
	if m == nil || m.self != m || m.core == nil {
		return ErrPolynomialOwner
	}
	if m.iccpDepth != 0 {
		return ErrPolynomialPending
	}
	if e := m.core.cleanupContentInvocation(&m.iccpShortcutContent, false); e != nil {
		return e
	}
	if e := m.cleanupICCPScopes(m.iccpFrames, false); e != nil {
		return e
	}
	if e := m.cleanupICCPScopes(m.iccpFrames, true); e != nil {
		return e
	}
	return m.core.cleanupContentInvocation(&m.iccpShortcutContent, true)
}
func (m *MultivariatePolynomialManager) cleanupICCPScopes(scopes []*polynomialICCPScope, abort bool) error {
	c := m.core
	if !abort {
		for j := len(scopes) - 1; j >= 0; j-- {
			s := scopes[j]
			if !s.active {
				continue
			}
			if e := m.cleanupGCDInvocation(&s.gcdInvocation, false); e != nil {
				return e
			}
			if c.matchesContentInvocation(&s.contentInvocation) {
				if e := c.cleanupContentInvocation(&s.contentInvocation, false); e != nil {
					return e
				}
			}
			if s.ciConstructed {
				if e := s.ci.closeValue(c, true); e != nil {
					return e
				}
			}
			if e := s.powers.closePhase(true, false); e != nil {
				return e
			}
			if e := s.filter.closePhase(true, false); e != nil {
				return e
			}
		}
		return nil
	}
	for j := len(scopes) - 1; j >= 0; j-- {
		s := scopes[j]
		if !s.active {
			continue
		}
		if e := m.cleanupGCDInvocation(&s.gcdInvocation, true); e != nil {
			return e
		}
		if c.matchesContentInvocation(&s.contentInvocation) {
			if e := c.cleanupContentInvocation(&s.contentInvocation, true); e != nil {
				return e
			}
		}
		if e := s.ci.abort(c); e != nil {
			return e
		}
		if e := s.powers.close(true); e != nil {
			return e
		}
		if e := s.filter.close(true); e != nil {
			return e
		}
	}
	for _, s := range scopes {
		if s.active && s.topOwner && s.generation != 0 && s.generation == c.pscReceipts.generation {
			if e := c.CleanupPSC(); e != nil {
				return e
			}
		}
	}
	for _, s := range scopes {
		if s.active {
			h := s.header
			*s = polynomialICCPScope{header: h, invocation: s.invocation}
		}
	}
	return nil
}
func (m *MultivariatePolynomialManager) iccpPending() bool {
	if m.core.matchesContentInvocation(&m.iccpShortcutContent) {
		return true
	}
	for _, s := range m.iccpFrames {
		if s.active {
			return true
		}
	}
	return false
}
