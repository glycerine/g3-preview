// Source: src/math/polynomial/polynomial.cpp:1821-1850,2303-2370,2460-2461,
// 3185,3214,3875,4070-4071,6234,6344,6865. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Core supplies the actual shared numeral and control owners. Allocator selects
// the second source constructor; a supplied Monomials selects the first. The
// generated global allocator binding is represented by the real borrowed
// Integer Runtime when neither explicit source pointer is supplied.
type MultivariatePolynomialManagerConfig struct {
	Core      MultivariatePolynomialConfig
	Allocator *memory.SmallAllocator
}

// The core is the actual cell/arithmetic implementation object. These are real
// remaining imp members, not separately initialized arithmetic shadows. Embedded
// owners are charged by this class header and have no second standalone charge.
type MultivariatePolynomialManager struct {
	toPolynomial                                               polynomialToPolynomialScope
	monic                                                      polynomialMonicScope
	self                                                       *MultivariatePolynomialManager
	core                                                       *MultivariatePolynomialCore
	upm                                                        *UPolynomialManager
	degree2pos, lexPermutation, varDegrees                     polynomialVector
	lexBuckets                                                 polynomialImpBuckets
	iccp                                                       PolynomialSOMVector
	newton                                                     PolynomialNewtonVector
	skeletons                                                  PolynomialSkeletonVector
	originalBudget                                             *arena.Budget
	retirements                                                [6]polynomialImpRetirement
	header                                                     uint64
	constructed                                                uint8
	memberStage, bodyStage, cleanupStage                       uint8
	useSparseGCD, usePRSGCD                                    bool
	completed, sourceStarted, sourceFinalized, closing, closed bool
	originalCause, sourceCause, cleanupCause, unwindCause      error
	squareRoot                                                 polynomialSquareRootScope
	derivative                                                 polynomialDerivativeScope
	evaluation                                                 polynomialEvaluationScope
	composition                                                polynomialCompositionScope
	lexPending                                                 bool
	lexGeneration                                              uint64
	lexCause                                                   error
	modularIndex                                               uint8
	mmConstructed, mmDropDone                                  bool
}

func NewMultivariatePolynomialManager(c MultivariatePolynomialManagerConfig) (*MultivariatePolynomialManager, error) {
	if c.Allocator != nil && c.Core.Monomials != nil {
		return nil, arena.ErrConfig
	}
	if c.Core.Integer == nil {
		return nil, arena.ErrConfig
	}
	if e := polynomialIntegerReady(c.Core.Integer); e != nil {
		return nil, e
	}
	if c.Allocator != nil {
		if c.Allocator.RuntimeOwner() != c.Core.Integer.memory || c.Allocator.RuntimeOwner().Budget() != c.Core.Integer.budget {
			return nil, arena.ErrConfig
		}
	}
	h := uint64(reflect.TypeOf(MultivariatePolynomialManager{}).Size())
	if e := c.Core.Integer.budget.Reserve(h); e != nil {
		return nil, e
	}
	m := &MultivariatePolynomialManager{header: h, originalBudget: c.Core.Integer.budget}
	m.self = m
	fail := func(e error) (*MultivariatePolynomialManager, error) {
		if m.core != nil && e == m.core {
			e = m.core.originalCause
		}
		m.originalCause = e
		if m.core != nil {
			m.core.originalCause = e
		}
		m.unwindCause = m.unwindMembers()
		m.cleanupCause = m.unwindCause
		return m, m
	}
	var e error
	m.core, e = preparePolynomialCore(c.Core, false)
	if e != nil {
		return fail(e)
	}
	m.constructed = 1
	m.upm, e = NewUPolynomialManager(UPolynomialConfig{Integer: c.Core.Integer, Checkpoint: c.Core.Checkpoint, Cooperation: c.Core.Cooperation})
	if e != nil {
		return fail(e)
	}
	m.constructed = 2
	if e = m.core.constructPolynomialCoreDefaults(); e != nil {
		return fail(e)
	}
	m.constructed = 3
	// Source declaration order: directory, ID, m2pos, then actual imp tmp1.
	m.core.rationalTerms.width = 12
	m.core.linearTerms.width = 12
	m.core.linearMonomials.width = 16
	if e = monomialTempConstruct(m.core.memory, &m.core.tmp1); e != nil {
		return fail(e)
	}
	m.constructed = 4
	m.degree2pos.width = 4
	m.lexPermutation.width = 4
	m.varDegrees.width = 4
	m.lexBuckets.owner = m.core
	m.lexBuckets.data.width = 8
	for _, v := range [2]*PolynomialSOMVector{&m.iccp, &m.core.translateBuffers} {
		v.self = v
		v.storage = m.core
		v.data.width = 8
	}
	m.newton.self = &m.newton
	m.newton.storage = m.core
	m.newton.data.width = 8
	m.skeletons.self = &m.skeletons
	m.skeletons.owner = m.core
	m.skeletons.data.width = 8
	d := &m.core.substitution
	d.max.self = &d.max
	d.max.core = m.core
	d.max.runtime = m.core.memory
	d.max.constructed = true
	variableEmbedded(&d.max.maxDegree, m.core.memory, true)
	variableEmbedded(&d.max.variables, m.core.memory, true)
	d.positions.self = &d.positions
	d.positions.runtime = m.core.memory
	d.positions.constructed = true
	variableEmbedded(&d.positions.positions, m.core.memory, true)
	d.found.runtime = m.core.memory
	m.constructed = 5
	if m.core.monomials == nil {
		m.core.monomials, e = NewMonomialManager(MonomialManagerConfig{Budget: c.Core.Integer.budget, Memory: c.Core.Integer.memory, Allocator: c.Allocator})
		m.mmConstructed = m.core.monomials != nil
		if e != nil {
			return fail(e)
		}
	}
	m.core.queryPositions.self = &m.core.queryPositions
	m.core.queryPositions.manager = m.core.monomials
	if e = m.core.initializePolynomialCoreBody(); e != nil {
		return fail(e)
	}
	// Source init publishes flags only after zero/unit and their INC operations.
	m.useSparseGCD = true
	m.usePRSGCD = false
	m.completed = true
	return m, nil
}
func (m *MultivariatePolynomialManager) Error() string {
	if m.originalCause != nil {
		return m.originalCause.Error()
	}
	if m.cleanupCause != nil {
		return m.cleanupCause.Error()
	}
	if m.sourceCause != nil {
		return m.sourceCause.Error()
	}
	return ErrPolynomialPending.Error()
}
func (m *MultivariatePolynomialManager) Unwrap() error { return m.originalCause }
func (m *MultivariatePolynomialManager) identity() error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *MultivariatePolynomialManager) ready() error {
	if e := m.identity(); e != nil {
		return e
	}
	if !m.completed || m.closing || m.sourceStarted {
		return arena.ErrClosed
	}
	return m.core.check()
}
