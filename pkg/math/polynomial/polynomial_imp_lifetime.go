// Source: src/math/polynomial/polynomial.cpp:1821-1850,2346-2370;
// src/util/vector.h; src/util/mpzzp.h:97-105. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The actual lex-bucket member is a vector of unsigned-vector objects. Its
// directory records the concrete child objects, not numeral ownership inferred
// from a registry. The outer Runtime vector contains their scalar bindings.
type polynomialImpBuckets struct {
	owner    *MultivariatePolynomialCore
	data     polynomialVector
	children []polynomialVector
	cursor   uint32
}

func (b *polynomialImpBuckets) sourceFinalize() error {
	n, e := b.data.size(b.owner)
	if e != nil {
		return e
	}
	for b.cursor < n {
		child, e := b.at(b.cursor)
		if e != nil {
			return e
		}
		if e := polynomialImpRawSource(b.owner, child); e != nil {
			return e
		}
		b.cursor++
	}
	return polynomialImpRawSource(b.owner, &b.data)
}
func (b *polynomialImpBuckets) close() error {
	for i := range b.children {
		if e := b.children[i].close(b.owner); e != nil {
			return e
		}
	}
	if b.data.old != (util.Block{}) {
		if e := b.owner.memory.Deallocate(&b.data.old); e != nil {
			return e
		}
	}
	if e := b.data.close(b.owner); e != nil {
		return e
	}
	if cap(b.children) > 0 {
		b.owner.integer.Budget().Release(uint64(cap(b.children)) * uint64(reflect.TypeOf(polynomialVector{}).Size()))
	}
	b.children = nil
	return nil
}
func polynomialImpRawSource(m *MultivariatePolynomialCore, v *polynomialVector) error {
	if v.raw != (util.Block{}) {
		return m.memory.Deallocate(&v.raw)
	}
	return nil
}

// This records only actual completed member construction. Failed constructors
// never execute the imp destructor body or release the body MM retain.
func (m *MultivariatePolynomialManager) unwindMembers() error {
	if m.core == nil {
		return nil
	}
	return m.destroyMembers()
}
func (m *MultivariatePolynomialManager) destroyMembers() error {
	// All late declarations are default constructed before the body starts.
	if m.constructed >= 5 {
		for m.memberStage < 15 {
			var e error
			switch m.memberStage {
			case 0:
				e = m.core.translateBuffers.SourceFinalize()
			case 1:
				e = m.core.substitution.positions.SourceFinalize()
			case 2:
				e = m.core.substitution.max.SourceFinalize()
			case 3:
				e = m.skeletons.SourceFinalize()
			case 4:
				e = m.newton.sourceFinalizeMember()
			case 5:
				e = m.iccp.SourceFinalize()
			case 6:
				e = polynomialImpRawSource(m.core, &m.varDegrees)
			case 7:
				e = m.core.substitution.found.sourceFinalize()
			case 8:
				e = polynomialImpRawSource(m.core, &m.lexPermutation)
			case 9:
				e = m.lexBuckets.sourceFinalize()
			case 10:
				e = m.sourceBufferMembers(&m.core.cheap2, false)
			case 11:
				e = m.sourceBufferMembers(&m.core.cheap, false)
			case 12:
				e = m.sourceBufferMembers(&m.core.som2, true)
			case 13:
				e = m.sourceBufferMembers(&m.core.som, true)
			case 14:
				e = polynomialImpRawSource(m.core, &m.degree2pos)
			}
			if e != nil {
				return e
			}
			m.memberStage++
		}
	}
	if m.constructed >= 4 && m.memberStage < 19 {
		for m.memberStage < 19 {
			var e error
			switch m.memberStage {
			case 15:
				e = polynomialImpRawSource(m.core, &m.core.linearMonomials)
			case 16:
				e = m.sourceNumeralVector(&m.core.linearTerms)
			case 17:
				e = m.sourceNumeralVector(&m.core.rationalTerms)
			case 18:
				if m.core.tmp1.raw == (util.Block{}) && !m.core.tmp1.destroyConsumed {
					e = ErrMonomialPending
				} else if m.core.tmp1.raw != (util.Block{}) {
					e = m.core.memory.Deallocate(&m.core.tmp1.raw)
					if m.core.tmp1.raw == (util.Block{}) {
						m.core.tmp1.destroyConsumed = true
					}
				}
			}
			if e != nil {
				return e
			}
			m.memberStage++
		}
	}
	if m.constructed >= 3 && m.memberStage < 22 {
		for m.memberStage < 22 {
			var e error
			switch m.memberStage {
			case 19:
				if m.core.queryPositions.raw != (util.Block{}) {
					e = m.core.memory.Deallocate(&m.core.queryPositions.raw)
				}
			case 20:
				m.core.idsSourceDestroyed = true
				e = m.core.ids.SourceFinalize()
				if e == nil {
					m.core.idsInitialized = false
				}
			case 21:
				m.core.directorySourceDestroyed = true
				e = polynomialImpRawSource(m.core, &m.core.polynomials)
			}
			if e != nil {
				return e
			}
			m.memberStage++
		}
	}
	if m.constructed >= 2 && m.memberStage < 23 {
		if e := m.upm.SourceFinalize(); e != nil {
			return e
		}
		m.memberStage = 23
	}
	if m.constructed >= 1 {
		m.core.modularSourceDestroyed = true
		if e := m.core.modular.FinalizeSourceMembers(&m.modularIndex); e != nil {
			return e
		}
	}
	return nil
}

// Literal vector<mpz> destruction has no mpz destructor/Del. Actual discarded
// initialized fields need separately charged Go owning retention before the raw
// member disappears; this is an owning adaptation, not a source free event.
func (m *MultivariatePolynomialManager) sourceNumeralVector(v *polynomialVector) error {
	return m.retireSourceNumeralVector(v)
}
func (m *MultivariatePolynomialManager) sourceBufferMembers(b *polynomialSOM, positions bool) error {
	if e := polynomialImpRawSource(m.core, &b.ms); e != nil {
		return e
	}
	if e := m.sourceNumeralVector(&b.as); e != nil {
		return e
	}
	if positions && b.positions.raw != (util.Block{}) {
		return m.core.memory.Deallocate(&b.positions.raw)
	}
	return nil
}

func (v *PolynomialNewtonVector) sourceFinalizeMember() error {
	if v.sourceCause != nil {
		return v.sourceCause
	}
	v.sourceFinalizing = true
	if v.sourceStage == 0 {
		if e := v.flushMode(false); e != nil {
			v.sourceCause = e
			return e
		}
		v.sourceStage = 1
	}
	if v.sourceStage == 1 {
		if e := polynomialImpRawSource(v.storage, &v.data); e != nil {
			v.sourceCause = e
			return e
		}
		v.sourceStage = 2
	}
	v.sourceFinalized = true
	return nil
}

func (m *MultivariatePolynomialManager) SourceFinalize() error {
	if m != nil && m.self == m && m.factorInline.manager != nil {
		if cause := m.factorInline.manager.factorConstantInvocationSourceCause(&m.factorInline); cause != nil {
			return cause
		}
	}
	if m != nil && m.matchesFactorCore(&m.factorRoot) && !m.factorRoot.frame.executing && !m.factorRoot.frame.finishing && m.factorRoot.frame.sourceCause != nil {
		return m.factorRoot.frame.sourceCause
	}
	if m != nil && m.self == m && m.factorUnivariate.invocation != 0 && !m.factorUnivariate.executing && !m.factorUnivariate.finishing && m.factorUnivariate.sourceCause != nil {
		return m.factorUnivariate.sourceCause
	}
	if m != nil && m.self == m && m.factorNumeralVector.invocation != 0 && !m.factorNumeralVector.executing && !m.factorNumeralVector.finishing && m.factorNumeralVector.sourceCause != nil {
		return m.factorNumeralVector.sourceCause
	}
	if m != nil && m.self == m && m.toPolynomial.active && !m.toPolynomial.executing && !m.toPolynomial.finishing && m.toPolynomial.sourceCause != nil {
		return m.toPolynomial.sourceCause
	}
	if m != nil && m.self == m && (m.core.factorConstantExecuting() || m.core.toPolynomialCallerExecuting() || numericIntegerManager_factorNumeralVectorCallerExecuting(m.core.integer) || numericIntegerManager_factorUnivariateCallerExecuting(m.core.integer) || numericIntegerManager_factorCoreCallerExecuting(m.core.integer)) {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.core != nil && m.core.self == m.core && (m.core.quasiResultant.active || m.core.variableNumeral.active) {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.core != nil && m.core.self == m.core && m.core.collectingVariables {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.core != nil && m.core.self == m.core && (m.normalization.active || m.core.normalizationPending()) {
		return ErrPolynomialPending
	}
	if e := m.identity(); e != nil {
		return e
	}
	if m.discriminant.active || m.subresultant.active || m.pscChain1.active || m.pscChain2.active || m.core.pscClassic.active || m.core.pscClassic.executing {
		return ErrPolynomialPending
	}
	if m.closing || !m.completed || m.originalCause != nil {
		return arena.ErrClosed
	}
	if m.sourceCause != nil {
		return m.sourceCause
	}
	if m.sourceFinalized {
		return nil
	}
	m.sourceStarted = true
	m.core.sourceStarted = true
	for m.bodyStage < 11 {
		var e error
		switch m.bodyStage {
		case 0, 1:
			e = m.core.sourcePrivateDEC(uint32(m.bodyStage))
		case 2:
			e = m.core.som.reset()
		case 3:
			e = m.core.som2.reset()
		case 4:
			e = m.core.cheap.reset()
		case 5:
			e = m.core.cheap2.reset()
		case 6:
			_, e = upolynomialSourceNumeralDel(m.core.integer, &m.core.zeroNumeral)
		case 7:
			e = m.newton.flushMode(false)
		case 8:
			e = m.skeletons.resetSource()
		case 9:
			if polynomialSourceDebug {
				e = m.core.polynomials.reset(m.core)
			}
			if e == nil && polynomialSourceDebug {
				var n uint32
				n, e = m.core.polynomials.size(m.core)
				if e == nil && n != 0 {
					e = ErrPolynomialAssertion
				}
			}
		case 10:
			before := m.core.monomials.managerRefs
			e = m.core.monomials.DecManagerRef()
			// This actual source operand decrements before the child destructor.
			// The owning abort must never repeat that decrement after a child stop.
			m.mmDropDone = m.core.monomials.managerRefs != before
			if m.mmDropDone {
				m.core.monomialRetained = false
			}
		}
		if e != nil {
			m.sourceCause = e
			return e
		}
		m.bodyStage++
		if m.bodyStage <= 7 {
			m.core.sourceStage = m.bodyStage
		}
	}
	if e := m.destroyMembers(); e != nil {
		m.sourceCause = e
		return e
	}
	m.sourceFinalized = true
	return nil
}
