// Source: src/math/polynomial/polynomial.cpp:3541-3672,4352-4506;
// src/util/obj_ref.h; src/util/scoped_numeral.h; src/util/buffer.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// A reusable charged cold record describes one concrete source invocation.
// No source local is constructed by reserving this adapter. The scalar stack
// records the actual declaration sites; destruction traverses only that stack.
// PRS has nine references and three numerals. The dispatcher instead begins
// with its two actual sbuffer<power,32> objects.
type polynomialGCDFunctionScope struct {
	resultantPseudo      polynomialPseudoDivisionInvocationReceipt
	resultantPower       polynomialPowerInvocationReceipt
	resultantArithmetic  polynomialArithmeticInvocationReceipt
	resultantCoefficient polynomialCoefficientInvocationReceipt
	resultantAppend      polynomialCheapAppendInvocationReceipt
	resultantFactory     polynomialFactoryInvocationReceipt
	resultantNeg         polynomialNegInvocationReceipt

	quadraticArithmetic   [9]polynomialArithmeticInvocationReceipt
	quadraticSqrt         polynomialSqrtInvocationReceipt
	quadraticNeg          [3]polynomialNegInvocationReceipt
	quadraticCoefficients [3]polynomialCoefficientInvocationReceipt
	quadraticAppend       [3]polynomialCheapAppendInvocationReceipt
	quadraticFactory      [3]polynomialFactoryInvocationReceipt

	invocation              uint64
	childInvocation         uint64
	returned                polynomialReference
	iccpInvocation          polynomialICCPInvocationReceipt
	contentInvocation       polynomialContentInvocationReceipt
	normalizeInvocation     polynomialNormalizeInvocationReceipt
	derivativeInvocation    polynomialDerivativeInvocationReceipt
	divisionInvocation      polynomialExactDivInvocationReceipt
	dividesInvocation       polynomialDividesInvocationReceipt
	parent, child           *polynomialGCDFunctionScope
	refs                    [14]PolynomialRef
	monomials               [2]polynomialDivisionReference
	nums                    [10]Int
	powers                  [2]polynomialGCDPowerBuffer
	stack                   [20]uint8
	depth                   uint8
	active, topOwner        bool
	generation              uint64
	numeric                 polynomialNumericRoots
	bodyCause, cleanupCause error
	zp                      polynomialScopedSetZP
	vars                    polynomialGCDVariableBuffer
	sparse                  PolynomialSparseInterpolator
	monomialBuffer          polynomialGCDMonomialBuffer
	newSkeleton             *PolynomialSkeleton
	header                  uint64
}

const (
	gcdBufferBase       uint8 = 0
	gcdNumeralBase      uint8 = 2
	gcdReferenceBase    uint8 = 12
	gcdZPId             uint8 = 26
	gcdSparseId         uint8 = 27
	gcdVariablesId      uint8 = 28
	gcdMonomialBase     uint8 = 29
	gcdMonomialBufferID uint8 = 31
)

func (s *polynomialGCDFunctionScope) constructNumeral(i uint8) {
	s.nums[i] = Int{}
	s.stack[s.depth] = gcdNumeralBase + i
	s.depth++
}
func (s *polynomialGCDFunctionScope) constructReference(c *MultivariatePolynomialCore, i uint8) {
	r := &s.refs[i]
	*r = PolynomialRef{owner: c}
	r.self = r
	s.stack[s.depth] = gcdReferenceBase + i
	s.depth++
}
func (s *polynomialGCDFunctionScope) constructMonomial(i uint8) {
	s.stack[s.depth] = gcdMonomialBase + i
	s.depth++
}
func (s *polynomialGCDFunctionScope) constructBuffer(c *MultivariatePolynomialCore, i uint8) {
	s.powers[i].construct(c.memory)
	s.stack[s.depth] = gcdBufferBase + i
	s.depth++
}
func (m *MultivariatePolynomialManager) beginGCDFunction() (*polynomialGCDFunctionScope, error) {
	var s *polynomialGCDFunctionScope
	if int(m.gcdDepth) < len(m.gcdFrames) {
		s = m.gcdFrames[m.gcdDepth]
		if s.active {
			return nil, ErrPolynomialPending
		}
	} else {
		old, n := cap(m.gcdFrames), cap(m.gcdFrames)
		if len(m.gcdFrames) == n {
			n = n*2 + 1
			stride := uint64(reflect.TypeOf((*polynomialGCDFunctionScope)(nil)).Size())
			if e := m.core.integer.budget.Reserve(uint64(n) * stride); e != nil {
				return nil, e
			}
			next := make([]*polynomialGCDFunctionScope, len(m.gcdFrames), n)
			copy(next, m.gcdFrames)
			m.gcdFrames = next
			m.core.integer.budget.Release(uint64(old) * stride)
		}
		h := uint64(reflect.TypeOf(polynomialGCDFunctionScope{}).Size())
		if e := m.core.integer.budget.Reserve(h); e != nil {
			return nil, e
		}
		s = &polynomialGCDFunctionScope{header: h}
		m.gcdFrames = append(m.gcdFrames, s)
	}
	if s.invocation == ^uint64(0) {
		return nil, arena.ErrBounds
	}
	top := m.core.pscReceipts.depth == 0
	if e := m.core.pscEnter(); e != nil {
		return nil, e
	}
	h := s.header
	*s = polynomialGCDFunctionScope{header: h, invocation: s.invocation + 1, active: true, topOwner: top, generation: m.core.pscExecutingOwner()}
	if m.gcdDepth != 0 {
		s.parent = m.gcdFrames[m.gcdDepth-1]
		s.parent.child = s
		s.parent.childInvocation = s.invocation
	}
	m.gcdDepth++
	return s, nil
}
func (m *MultivariatePolynomialManager) gcdYoungerSourceStop(s *polynomialGCDFunctionScope) error {
	if e := m.resultantConcreteChildSourceCause(s); e != nil {
		if s.cleanupCause == nil {
			s.cleanupCause = e
		}
		return s.cleanupCause
	}
	if e := m.core.arithmeticInvocationSourceCause(&s.resultantArithmetic); e != nil {
		if s.cleanupCause == nil {
			s.cleanupCause = e
		}
		return s.cleanupCause
	}
	if e := m.core.pseudoInvocationSourceCause(&s.resultantPseudo); e != nil {
		if s.cleanupCause == nil {
			s.cleanupCause = e
		}
		return s.cleanupCause
	}
	if e := m.core.powerInvocationSourceCause(&s.resultantPower); e != nil {
		if s.cleanupCause == nil {
			s.cleanupCause = e
		}
		return s.cleanupCause
	}
	for i := len(s.quadraticArithmetic) - 1; i >= 0; i-- {
		if e := m.core.arithmeticInvocationSourceCause(&s.quadraticArithmetic[i]); e != nil {
			if s.cleanupCause == nil {
				s.cleanupCause = e
			}
			return s.cleanupCause
		}
	}
	if e := m.sqrtInvocationSourceCause(&s.quadraticSqrt); e != nil {
		s.cleanupCause = e
		return e
	}
	for i := 2; i >= 0; i-- {
		if e := m.core.negInvocationSourceCause(&s.quadraticNeg[i]); e != nil {
			s.cleanupCause = e
			return e
		}
	}

	if s.cleanupCause != nil {
		return s.cleanupCause
	}
	if s.child != nil && s.child.invocation == s.childInvocation && s.child.active && s.child.generation == s.generation && s.child.cleanupCause != nil {
		s.cleanupCause = s.child.cleanupCause
		return s.cleanupCause
	}
	if e := m.core.contentInvocationSourceCause(&s.contentInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if e := m.iccpInvocationSourceCause(&s.iccpInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if e := m.normalizeInvocationSourceCause(&s.normalizeInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if e := m.derivativeInvocationSourceCause(&s.derivativeInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if e := m.core.dividesInvocationSourceCause(&s.dividesInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if e := m.core.exactDivInvocationSourceCause(&s.divisionInvocation); e != nil {
		s.cleanupCause = e
		return e
	}
	if s.sparse.self != nil && s.sparse.cleanupCause != nil {
		s.cleanupCause = s.sparse.cleanupCause
		return s.cleanupCause
	}
	if m.core.pscReceipts.generation == s.generation {
		if e := m.core.pscSourceStopped(); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	return nil
}
func (m *MultivariatePolynomialManager) finishGCDFunctionSource(s *polynomialGCDFunctionScope) error {
	return m.finishGCDLocalsTo(s, 0)
}

func (m *MultivariatePolynomialManager) finishGCDLocalsTo(s *polynomialGCDFunctionScope, frontier uint8) error {
	if e := m.gcdYoungerSourceStop(s); e != nil {
		return e
	}
	for s.depth > frontier {
		id := s.stack[s.depth-1]
		var e error
		switch {
		case id == gcdMonomialBufferID:
			e = s.monomialBuffer.close(m.core, false)
		case id >= gcdMonomialBase:
			r := &s.monomials[id-gcdMonomialBase]
			e = r.drop(m.core, &r.value, &r.decrementStarted, false)
		case id == gcdVariablesId:
			e = s.vars.close(false)
		case id == gcdSparseId:
			e = s.sparse.SourceFinalize()
		case id == gcdZPId:
			e = s.zp.sourceEnd()
		case id < gcdNumeralBase:
			e = s.powers[id].close(false)
		case id < gcdReferenceBase:
			_, e = sourceNumeralDel(m.core.integer, &s.nums[id-gcdNumeralBase])
		default:
			e = s.refs[id-gcdReferenceBase].root.closeValue(m.core, false)
		}
		if e != nil {
			s.cleanupCause = e
			return e
		}
		s.depth--
	}
	if frontier != 0 {
		return nil
	}
	pending := (s.child != nil && s.child.active && s.child.invocation == s.childInvocation) || m.matchesICCPInvocation(&s.iccpInvocation) || m.matchesICCPShortcutContent(&s.iccpInvocation) || m.core.matchesContentInvocation(&s.contentInvocation) || s.returned.value != (Polynomial{}) || s.returned.assigning || s.zp.active || s.vars.pending.OwnerID() != 0 || (s.sparse.self != nil && !s.sparse.closed && (s.sparse.inputs.pending.OwnerID() != 0 || s.sparse.outputs.pending.OwnerID() != 0)) || s.topOwner && m.core.pscPending()
	pending = pending || m.matchesNormalizeInvocation(&s.normalizeInvocation)
	pending = pending || (m.matchesDerivativeInvocation(&s.derivativeInvocation) || m.derivativeInvocationHasOrphans(&s.derivativeInvocation)) || m.core.matchesExactDivInvocation(&s.divisionInvocation) || m.core.dividesInvocationPending(&s.dividesInvocation)
	pending = pending || m.quadraticPending(s) || m.core.matchesPowerInvocation(&s.resultantPower) || m.core.matchesPseudoInvocation(&s.resultantPseudo) || m.core.matchesArithmeticInvocation(&s.resultantArithmetic) || m.resultantConcreteChildrenPending(s)
	pending = pending || s.newSkeleton != nil || s.monomialBuffer.incoming != (Monomial{}) || s.monomialBuffer.pending.OwnerID() != 0
	for i := range s.monomials {
		pending = pending || s.monomials[i].assigning
	}
	for i := range s.refs {
		pending = pending || s.refs[i].root.assigning
	}
	pending = pending || s.numeric.cell.object.OwnerID() != 0
	for _, p := range s.numeric.allocator {
		pending = pending || p.OwnerID() != 0
	}
	for i := range s.powers {
		pending = pending || s.powers[i].pending.OwnerID() != 0
	}
	if !pending {
		h := s.header
		*s = polynomialGCDFunctionScope{header: h, invocation: s.invocation, bodyCause: s.bodyCause, cleanupCause: s.cleanupCause, normalizeInvocation: s.normalizeInvocation, resultantPower: s.resultantPower, resultantPseudo: s.resultantPseudo, resultantArithmetic: s.resultantArithmetic, resultantCoefficient: s.resultantCoefficient, resultantAppend: s.resultantAppend, resultantFactory: s.resultantFactory, resultantNeg: s.resultantNeg}
	}
	return nil
}

func (m *MultivariatePolynomialManager) gcdPending() bool {
	for _, s := range m.gcdFrames {
		if s.active {
			return true
		}
	}
	return false
}
func (m *MultivariatePolynomialManager) closeGCDMetadata() error {
	if m.gcdPending() || m.iccpPending() {
		return ErrPolynomialPending
	}
	for _, s := range m.gcdFrames {
		m.originalBudget.Release(s.header)
		s.header = 0
	}
	m.originalBudget.Release(uint64(cap(m.gcdFrames)) * uint64(reflect.TypeOf((*polynomialGCDFunctionScope)(nil)).Size()))
	m.gcdFrames = nil
	for _, s := range m.iccpFrames {
		m.originalBudget.Release(s.header)
		s.header = 0
	}
	m.originalBudget.Release(uint64(cap(m.iccpFrames)) * uint64(reflect.TypeOf((*polynomialICCPScope)(nil)).Size()))
	m.iccpFrames = nil
	return nil
}
