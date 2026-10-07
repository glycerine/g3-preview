// Source: src/math/polynomial/polynomial.cpp:1613-1683,6577-6590;
// src/math/polynomial/polynomial.h:120-190; src/util/vector.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The class owns the actual constant and two Runtime vectors. The pointer
// payload uses the existing 16-byte checked polynomial codec, not a shadow
// handle slice. Charged byte phases record INC and destructive progress only.
type PolynomialFactors struct {
	memberSet                                                          polynomialFactorMemberSetScope
	memberSetHistory                                                   []polynomialFactorMemberSetHistory
	constantInvocation                                                 polynomialFactorConstantScope
	constantHistory                                                    []polynomialFactorConstantHistory
	displayReceipt                                                     *polynomialDisplayFrame
	displayGeneration                                                  uint64
	self                                                               *PolynomialFactors
	manager                                                            *MultivariatePolynomialManager
	factors, degrees                                                   polynomialVector
	constant                                                           util.Int
	total                                                              uint32
	phases                                                             []uint8
	header                                                             uint64
	constructed, closing, closed, sourceStarted, resetting             bool
	sourceStage                                                        uint8
	resetIndex                                                         uint32
	sourceCause, bodyCause, cleanupCause                               error
	current                                                            polynomialReference
	currentActive                                                      bool
	local                                                              util.Int
	localActive                                                        bool
	numeric                                                            polynomialNumericRoots
	multiplyActive, executing                                          bool
	powerGeneration, factoryGeneration, constantGeneration, generation uint64
	singleton                                                          Monomial
	singletonGeneration                                                uint64
	singletonLast                                                      bool
	generated                                                          Monomial
	calleeGeneration                                                   uint64
	somOwned, cheapOwned                                               bool
	monomialRaw                                                        [3]util.Block
	stopped                                                            bool
	freshResult                                                        Polynomial
}

const (
	factorPublished    uint8 = 1
	factorReferenced   uint8 = 2
	factorDropConsumed uint8 = 3
	factorLastDrop     uint8 = 4
	factorDrained      uint8 = 5
)

func NewPolynomialFactors(m *MultivariatePolynomialManager) (*PolynomialFactors, error) {
	if e := m.ready(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialFactors{}).Size())
	if e := m.core.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	f := &PolynomialFactors{manager: m, header: h, factors: polynomialVector{width: 16}, degrees: polynomialVector{width: 4}}
	f.self = f
	// Empty vector members are complete before the constructor body SET.
	if e := f.setMemberConstantOne(); e != nil {
		f.bodyCause = e
		return f, e
	}
	f.constructed = true
	return f, nil
}
func (f *PolynomialFactors) identity() error {
	if f == nil || f.self != f {
		return ErrPolynomialOwner
	}
	if f.closed {
		return arena.ErrClosed
	}
	return nil
}
func (f *PolynomialFactors) ready() error {
	if e := f.identity(); e != nil {
		return e
	}
	if f.closing || f.sourceStarted {
		return arena.ErrClosed
	}
	if !f.constructed {
		return ErrPolynomialPending
	}
	return f.manager.ready()
}
func (f *PolynomialFactors) open() error {
	if e := f.ready(); e != nil {
		return e
	}
	if f.memberSet.active || f.constantInvocation.active || f.sourceCause != nil || f.resetting || f.multiplyActive || f.localActive || f.numeric.PendingObject() != (util.ObjectRef{}) || f.numeric.PendingAllocatorBlocks() != ([3]util.Block{}) {
		return ErrPolynomialPending
	}
	if frame := f.displayReceipt; frame != nil && frame.generation == f.displayGeneration && frame.active && frame.sink == nil {
		return ErrPolynomialPending
	}
	return nil
}
func (f *PolynomialFactors) Manager() (*MultivariatePolynomialManager, error) {
	if e := f.ready(); e != nil {
		return nil, e
	}
	return f.manager, nil
}
func (f *PolynomialFactors) Distinct() (uint32, error) {
	if e := f.ready(); e != nil {
		return 0, e
	}
	return f.factors.size(f.manager.core)
}
func (f *PolynomialFactors) Total() (uint32, error) {
	if e := f.ready(); e != nil {
		return 0, e
	}
	return f.total, nil
}

// Constant is a source const-reference borrow; callers must not mutate it.
func (f *PolynomialFactors) Constant() (*util.Int, error) {
	if e := f.ready(); e != nil {
		return nil, e
	}
	return &f.constant, nil
}
func (f *PolynomialFactors) Degree(i uint32) (uint32, error) {
	if e := f.ready(); e != nil {
		return 0, e
	}
	return polynomialUnsignedRead(f.manager.core, &f.degrees, i)
}
func (f *PolynomialFactors) sourceSize() (uint32, error) {
	if e := f.identity(); e != nil {
		return 0, e
	}
	return f.factors.size(f.manager.core)
}
func (f *PolynomialFactors) factor(i uint32) (Polynomial, error) {
	if e := f.identity(); e != nil {
		return Polynomial{}, e
	}
	m := f.manager.core
	if polynomialSourceDebug {
		n, e := f.factors.size(m)
		if e != nil {
			return Polynomial{}, e
		}
		if i >= n {
			return Polynomial{}, ErrPolynomialAssertion
		}
	}
	a, e := f.factors.monomial(m, i)
	return Polynomial{owner: a.owner, slot: a.slot, generation: a.generation}, e
}
func (f *PolynomialFactors) reservePhase(n uint32) error {
	if uint64(n) > uint64(int(^uint(0)>>1)) {
		return arena.ErrBounds
	}
	if int(n) <= cap(f.phases) {
		if int(n) > len(f.phases) {
			f.phases = f.phases[:n]
		}
		return nil
	}
	c := cap(f.phases) * 2
	if c < int(n) {
		c = int(n)
	}
	if e := f.manager.core.integer.Budget().Reserve(uint64(c)); e != nil {
		return e
	}
	p := make([]uint8, n, c)
	copy(p, f.phases)
	f.manager.core.integer.Budget().Release(uint64(cap(f.phases)))
	f.phases = p
	return nil
}
func (f *PolynomialFactors) Push(p Polynomial, d uint32) (err error) {
	if e := f.open(); e != nil {
		return e
	}
	if polynomialSourceDebug && (p == (Polynomial{}) || d == 0) {
		return ErrPolynomialAssertion
	}
	m := f.manager.core
	n, e := f.factors.size(m)
	if e != nil {
		return e
	}
	// Cold progress storage precedes the source vector push. It does not emit
	// a source Runtime allocation, and its distinct failure is an adapter prefix.
	if e = f.reservePhase(n + 1); e != nil {
		return e
	}
	defer func() {
		if err != nil {
			f.bodyCause = err
			f.resetting = true
		}
	}()
	if e = f.factors.appendMonomial(m, Monomial{owner: p.owner, slot: p.slot, generation: p.generation}); e != nil {
		return e
	}
	f.phases[n] = factorPublished
	if e = polynomialUnsignedAppend(m, &f.degrees, d); e != nil {
		return e
	}
	f.total += d
	if e = m.incRefCore(p); e != nil {
		return e
	}
	f.phases[n] = factorReferenced
	return nil
}
func (f *PolynomialFactors) SetDegree(i, d uint32) error {
	if e := f.open(); e != nil {
		return e
	}
	if polynomialSourceDebug && i == 0 {
		return ErrPolynomialAssertion
	}
	old, e := polynomialUnsignedRead(f.manager.core, &f.degrees, i)
	if e != nil {
		return e
	}
	f.total -= old
	if e = f.manager.core.memory.WriteUint32(f.degrees.raw, 8+uint64(i)*4, d); e != nil {
		return e
	}
	f.total += d
	return nil
}
func (f *PolynomialFactors) numericCall(body func() error) error {
	before, e := polynomialNumericBefore(f.manager.core.integer)
	if e != nil {
		return e
	}
	e = body()
	x := f.numeric.after(f.manager.core.integer, before)
	return errors.Join(e, x)
}
func (f *PolynomialFactors) SetConstant(c *util.Int) error { return f.setConstantWithReceipt(c, nil) }
func (f *PolynomialFactors) At(i uint32) (*PolynomialRef, error) {
	if e := f.ready(); e != nil {
		return nil, e
	}
	p, e := f.factor(i)
	if e != nil {
		return nil, e
	}
	r, e := NewPolynomialRef(f.manager.core)
	if e != nil {
		return nil, e
	}
	if e = r.Assign(p); e != nil {
		return r, e
	}
	return r, nil
}
func (f *PolynomialFactors) sourceReset() error {
	if f.factorCallerExecuting() {
		return ErrPolynomialPending
	}
	if f.sourceCause != nil {
		return f.sourceCause
	}
	m := f.manager.core
	f.resetting = true
	for {
		n, e := f.factors.size(m)
		if e != nil {
			f.sourceCause = e
			return e
		}
		if f.resetIndex >= n {
			break
		}
		p, e := f.factor(f.resetIndex)
		if e != nil {
			f.sourceCause = e
			return e
		}
		out := polynomialDecrementOutcome{}
		e = m.decRefCoreOutcome(p, &out)
		if out.consumed {
			f.phases[f.resetIndex] = factorDropConsumed
			if out.last {
				f.phases[f.resetIndex] = factorLastDrop
			}
		}
		if e != nil {
			f.sourceCause = e
			return e
		}
		f.phases[f.resetIndex] = factorDrained
		f.resetIndex++
	}
	if e := f.factors.reset(m); e != nil {
		f.sourceCause = e
		return e
	}
	if e := f.degrees.reset(m); e != nil {
		f.sourceCause = e
		return e
	}
	f.total = 0
	// Literal nm.SET(int) first executes Del. Public owning SET accepts dead
	// cleanup carriers, so the strict source leaf supplies this reached action.
	// The exact NULL sentinel loses only its pointer token and keeps val until SET.
	if e := m.modular.Check(); e != nil {
		f.sourceCause = e
		return e
	}
	if _, e := upolynomialSourceNumeralDel(m.integer, &f.constant); e != nil {
		f.sourceCause = e
		return e
	}
	if e := f.setMemberConstantOne(); e != nil {
		f.sourceCause = e
		return e
	}
	f.resetting = false
	f.resetIndex = 0
	f.phases = f.phases[:0]
	return nil
}
func (f *PolynomialFactors) Reset() error {
	if e := f.open(); e != nil {
		return e
	}
	return f.sourceReset()
}
func (f *PolynomialFactors) SourceFinalize() error {
	if e := f.identity(); e != nil {
		return e
	}
	if f.factorCallerExecuting() || f.memberSet.executing || f.memberSet.finishing || f.constantInvocation.executing || f.constantInvocation.finishing {
		return ErrPolynomialPending
	}
	// Finish the younger invoked helper's actual source suffix before entering
	// the class Reset/member destructor. Unjoined allocation roots alone do
	// not block the older source suffix once its scoped NUM DT has completed.
	if f.memberSet.active {
		if e := f.finishFactorMemberSetSource(); e != nil {
			return e
		}
	}
	if f.constantInvocation.active {
		if e := f.finishFactorConstantSource(); e != nil {
			return e
		}
	}

	if f.sourceCause != nil {
		return f.sourceCause
	}
	if f.closing || !f.constructed {
		return ErrPolynomialPending
	}
	f.sourceStarted = true
	if f.sourceStage == 0 {
		if e := f.sourceReset(); e != nil {
			return e
		}
		f.sourceStage = 1
	}
	if f.sourceStage == 1 {
		if _, e := upolynomialSourceNumeralDel(f.manager.core.integer, &f.constant); e != nil {
			f.sourceCause = e
			return e
		}
		f.sourceStage = 2
	}
	// Reverse completed members: degree svector, then pointer vector. Raw source
	// dtors do not consume unpublished allocation prefixes.
	for f.sourceStage < 4 {
		v := &f.degrees
		if f.sourceStage == 3 {
			v = &f.factors
		}
		if e := displayDeallocatePublished(f.manager.core.memory, &v.raw); e != nil {
			f.sourceCause = e
			return e
		}
		f.sourceStage++
	}
	return nil
}

// Cleanup is explicit owning class abort. It drains only this class and its
// exact operation receipts, leaves source history intact, and does not restore
// the source constant to one or report a successful source Reset.
func (f *PolynomialFactors) Cleanup() error {
	if e := f.identity(); e != nil {
		return e
	}
	if f.factorCallerExecuting() || f.memberSet.executing || f.memberSet.finishing || f.constantInvocation.executing || f.constantInvocation.finishing {
		return ErrPolynomialPending
	}
	if e := f.cleanupFactorMemberSet(false); e != nil {
		return e
	}
	if e := f.cleanupFactorConstantInvocation(&f.constantInvocation.direct, false); e != nil {
		return e
	}
	m := f.manager.core
	if e := f.cleanupMultiply(); e != nil {
		return e
	}
	for i := uint32(0); i < uint32(len(f.phases)); i++ {
		phase := f.phases[i]
		if phase == 0 || phase == factorPublished || phase == factorDrained || phase == factorDropConsumed {
			continue
		}
		p, e := f.factor(i)
		if e != nil {
			return e
		}
		if phase == factorReferenced {
			out := polynomialDecrementOutcome{}
			e = m.decRefCoreOutcome(p, &out)
			if out.consumed {
				f.phases[i] = factorDropConsumed
				if out.last {
					f.phases[i] = factorLastDrop
				}
			}
			if e != nil {
				return e
			}
			f.phases[i] = factorDrained
		} else if phase == factorLastDrop {
			if e = m.CleanupValue(&p); e != nil {
				return e
			}
			f.phases[i] = factorDrained
		}
	}
	if e := m.integer.Del(&f.constant); e != nil {
		return e
	}
	if e := f.cleanupFactorMemberSet(true); e != nil {
		return e
	}
	if e := f.cleanupFactorConstantInvocation(&f.constantInvocation.direct, true); e != nil {
		return e
	}
	if e := f.numeric.cleanup(m.integer); e != nil {
		return e
	}
	if e := f.factors.reset(m); e != nil {
		return e
	}
	if e := f.degrees.reset(m); e != nil {
		return e
	}
	f.phases = f.phases[:0]
	f.total = 0
	f.resetIndex = 0
	f.resetting = false
	return nil
}
func (f *PolynomialFactors) CloseChecked() error {
	if f == nil || f.self != f {
		return ErrPolynomialOwner
	}
	if f.closed {
		return nil
	}
	if f.factorCallerExecuting() || f.memberSet.executing || f.memberSet.finishing || f.constantInvocation.executing || f.constantInvocation.finishing {
		return ErrPolynomialPending
	}
	f.closing = true
	if e := f.Cleanup(); e != nil {
		f.cleanupCause = e
		return e
	}
	if e := f.degrees.close(f.manager.core); e != nil {
		return e
	}
	if e := f.factors.close(f.manager.core); e != nil {
		return e
	}
	f.manager.core.integer.Budget().Release(f.header + uint64(cap(f.phases)) + uint64(cap(f.memberSetHistory))*uint64(reflect.TypeOf(polynomialFactorMemberSetHistory{}).Size()) + uint64(cap(f.constantHistory))*uint64(reflect.TypeOf(polynomialFactorConstantHistory{}).Size()))
	f.constantHistory = nil
	f.memberSetHistory = nil
	f.header = 0
	f.phases = nil
	f.closed = true
	return nil
}

func (f *PolynomialFactors) PM() (*MultivariatePolynomialManager, error) { return f.Manager() }
