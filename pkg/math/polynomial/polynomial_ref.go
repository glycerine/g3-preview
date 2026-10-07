// Source: src/util/obj_ref.h; src/util/ref_vector.h; src/math/polynomial/polynomial.h:108-109.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type polynomialReference struct {
	cleanupCause                                error
	value, incoming                             Polynomial
	decremented, incomingDecremented, assigning bool
}

func polynomialReferenceDrop(m *MultivariatePolynomialCore, p *Polynomial, started *bool) error {
	if *p == (Polynomial{}) {
		*started = false
		return nil
	}
	if *started {
		if e := m.CleanupValue(p); e != nil {
			return e
		}
		*started = false
		return nil
	}
	e := m.decRefCore(*p)
	if e != nil {
		l, x := m.polynomialLocation(*p)
		if x == nil {
			if l.deleteStage != 0 || l.callbackStarted {
				*started = true
			}
		}
		return e
	}
	*p = Polynomial{}
	return nil
}
func (r *polynomialReference) assign(m *MultivariatePolynomialCore, p Polynomial) error {
	return r.assignWithFactoryReceipt(m, p, nil)
}
func (r *polynomialReference) assignWithFactoryReceipt(m *MultivariatePolynomialCore, p Polynomial, receipt *polynomialFactoryInvocationReceipt) error {
	if r.assigning {
		return ErrPolynomialPending
	}
	if m.matchesFactoryInvocation(receipt) && m.factoryInvocation.result != p {
		return ErrPolynomialOwner
	}
	if p != (Polynomial{}) {
		if e := m.incRefCore(p); e != nil {
			return e
		}
	}
	if f := m.factorsOwner; f != nil && f.executing && m.callbackDepth == 0 && f.freshResult == p {
		f.freshResult = Polynomial{}
	}
	r.incoming = p
	r.assigning = true
	// Actual INC has completed and this real caller reference now owns incoming.
	if e := m.transferFactoryResult(receipt, p); e != nil {
		return e
	}
	if owner := m.squareRootOwner; owner != nil && owner.executing && m.callbackDepth == 0 && owner.freshResult == p {
		owner.freshResult = Polynomial{}
	}
	if e := polynomialReferenceDrop(m, &r.value, &r.decremented); e != nil {
		r.cleanupCause = e
		return e
	}
	r.value = r.incoming
	r.incoming = Polynomial{}
	r.assigning = false
	return nil
}
func (r *polynomialReference) closeValue(m *MultivariatePolynomialCore, owning bool) error {
	if r.decremented && !owning {
		return r.cleanupCause
	}
	if e := polynomialReferenceDrop(m, &r.value, &r.decremented); e != nil {
		r.cleanupCause = e
		return e
	}
	r.cleanupCause = nil
	return nil
}
func (r *polynomialReference) abort(m *MultivariatePolynomialCore) error {
	if !r.assigning {
		return nil
	}
	if e := polynomialReferenceDrop(m, &r.incoming, &r.incomingDecremented); e != nil {
		return e
	}
	r.assigning = false
	return nil
}
func (r *polynomialReference) close(m *MultivariatePolynomialCore) error {
	if e := r.closeValue(m, true); e != nil {
		return e
	}
	return r.abort(m)
}

// PolynomialRef is one actual charged cold owning object. Shallow copies are invalid.
type PolynomialRef struct {
	quasiFrame      *polynomialQuasiResultantScope
	quasiInvocation uint64
	self            *PolynomialRef
	owner           *MultivariatePolynomialCore
	root            polynomialReference
	header          uint64
	closed, closing bool
}

func NewPolynomialRef(m *MultivariatePolynomialCore) (*PolynomialRef, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialRef{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	r := &PolynomialRef{owner: m, header: h}
	r.self = r
	return r, nil
}
func (r *PolynomialRef) open() error {
	if r == nil || r.self != r {
		return ErrPolynomialOwner
	}
	if r.closed || r.closing {
		return arena.ErrClosed
	}
	return r.owner.check()
}
func (r *PolynomialRef) Assign(p Polynomial) error {
	if e := r.open(); e != nil {
		return e
	}
	return r.root.assign(r.owner, p)
}
func (r *PolynomialRef) Get() (Polynomial, error) {
	if e := r.open(); e != nil {
		return Polynomial{}, e
	}
	return r.root.value, nil
}
func (r *PolynomialRef) Reset() error { return r.Assign(Polynomial{}) }
func (r *PolynomialRef) Close() error {
	if r.quasiResultantPending() {
		return ErrPolynomialPending
	}
	if r == nil || r.self != r {
		return ErrPolynomialOwner
	}
	if r.closed {
		return nil
	}
	r.closing = true
	if e := r.root.close(r.owner); e != nil {
		return e
	}
	r.owner.integer.Budget().Release(r.header)
	r.header = 0
	r.closed = true
	return nil
}

type polynomialReferenceVector struct {
	pushInvocation                          uint64
	pushActive, pushExecuting               bool
	pushBodyCause, pushCleanupCause         error
	setInvocation                           uint64
	setIndex                                uint32
	setActive, setExecuting, setOldConsumed bool
	setOldRead                              bool
	resetExecuting                          bool
	setBodyCause, setCleanupCause           error
	values                                  polynomialVector
	cleanupCause                            error
	incoming                                Polynomial
	incomingDec                             bool
	drainDec                                bool
	drain                                   uint32
	draining                                bool
}

func (v *polynomialReferenceVector) get(m *MultivariatePolynomialCore, i uint32) (Polynomial, error) {
	n, e := v.values.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if polynomialSourceDebug && i >= n {
		return Polynomial{}, ErrPolynomialAssertion
	}
	if i >= v.values.initialized {
		return Polynomial{}, arena.ErrBounds
	}
	a, e := v.values.monomial(m, i)
	return Polynomial{owner: a.owner, slot: a.slot, generation: a.generation}, e
}
func (v *polynomialReferenceVector) push(m *MultivariatePolynomialCore, p Polynomial) error {
	return v.pushSource(m, p, nil, nil)
}
func (v *polynomialReferenceVector) pushSource(m *MultivariatePolynomialCore, p Polynomial, owner *PolynomialRefVector, r *polynomialReferenceVectorPushReceipt) (err error) {
	if v.pushActive || v.setActive || v.draining || v.incoming != (Polynomial{}) {
		return ErrPolynomialPending
	}
	if r != nil {
		if owner == nil || owner.self != owner || owner.owner != m || v != &owner.state {
			return ErrPolynomialOwner
		}
		if v.pushInvocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		v.pushInvocation++
		v.pushActive = true
		v.pushExecuting = true
		v.pushBodyCause = nil
		v.pushCleanupCause = nil
		*r = polynomialReferenceVectorPushReceipt{m, owner, v, v.pushInvocation}
		defer func() {
			v.pushExecuting = false
			if err != nil && v.pushBodyCause == nil {
				v.pushBodyCause = err
			}
		}()
	}
	if p != (Polynomial{}) {
		if e := m.incRefCore(p); e != nil {
			v.pushActive = false
			return e
		}
	}
	v.incoming = p
	i, e := v.values.appendSlot(m)
	if e != nil {
		return e
	}
	if e = v.values.putMonomial(m, i, Monomial{owner: p.owner, slot: p.slot, generation: p.generation}); e != nil {
		return e
	}
	if i >= v.values.initialized {
		v.values.initialized = i + 1
	}
	if e = v.values.publishSize(m, i+1); e != nil {
		return e
	}
	v.incoming = Polynomial{}
	v.pushActive = false
	return nil
}
func (v *polynomialReferenceVector) reset(m *MultivariatePolynomialCore) error {
	return v.resetMode(m, false)
}
func (v *polynomialReferenceVector) resetMode(m *MultivariatePolynomialCore, owning bool) (err error) {
	if v.resetExecuting || v.pushExecuting {
		return ErrPolynomialPending
	}
	if v.setActive || (!owning && v.pushActive) {
		return ErrPolynomialPending
	}
	if !owning && v.cleanupCause != nil {
		return v.cleanupCause
	}
	v.resetExecuting = true
	defer func() { v.resetExecuting = false }()
	defer func() {
		if err != nil {
			v.cleanupCause = err
		} else {
			v.cleanupCause = nil
		}
	}()
	v.draining = true
	n, e := v.values.size(m)
	if e != nil {
		return e
	}
	for v.drain < n {
		p, e := v.get(m, v.drain)
		if e != nil {
			return e
		}
		if e = polynomialReferenceDrop(m, &p, &v.drainDec); e != nil {
			return e
		}
		v.drain++
	}
	if v.values.raw == (util.Block{}) {
		v.drain = 0
		v.draining = false
		return nil
	}
	if e = v.values.reset(m); e != nil {
		return e
	}
	v.drain = 0
	v.draining = false
	return nil
}
func (v *polynomialReferenceVector) close(m *MultivariatePolynomialCore) error {
	if v.pushExecuting {
		return ErrPolynomialPending
	}
	if e := v.abortSet(m); e != nil {
		return e
	}
	if e := v.resetMode(m, true); e != nil {
		return e
	}
	if e := v.values.close(m); e != nil {
		return e
	}
	if e := v.abortIncoming(m); e != nil {
		return e
	}
	v.pushActive = false
	return nil
}

func (v *polynomialReferenceVector) abortIncoming(m *MultivariatePolynomialCore) error {
	return polynomialReferenceDrop(m, &v.incoming, &v.incomingDec)
}

// PolynomialRefVector owns exactly the logical source pointer range; retained
// tail bits are borrowed and do not create a second reference.
type PolynomialRefVector struct {
	self            *PolynomialRefVector
	owner           *MultivariatePolynomialCore
	state           polynomialReferenceVector
	header          uint64
	closed, closing bool
}

func NewPolynomialRefVector(m *MultivariatePolynomialCore) (*PolynomialRefVector, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(PolynomialRefVector{}).Size())
	if e := m.integer.Budget().Reserve(h); e != nil {
		return nil, e
	}
	v := &PolynomialRefVector{owner: m, header: h}
	v.self = v
	v.state.values.width = 16
	return v, nil
}
func (v *PolynomialRefVector) open() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || v.closing {
		return arena.ErrClosed
	}
	return v.owner.check()
}
func (v *PolynomialRefVector) Push(p Polynomial) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.state.push(v.owner, p)
}
func (v *PolynomialRefVector) Get(i uint32) (Polynomial, error) {
	if e := v.open(); e != nil {
		return Polynomial{}, e
	}
	return v.state.get(v.owner, i)
}
func (v *PolynomialRefVector) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.state.values.size(v.owner)
}
func (v *PolynomialRefVector) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	return v.state.reset(v.owner)
}
func (v *PolynomialRefVector) Close() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed {
		return nil
	}
	if v.state.pushExecuting || v.state.setExecuting || v.state.resetExecuting {
		return ErrPolynomialPending
	}
	v.closing = true
	if e := v.state.close(v.owner); e != nil {
		return e
	}
	v.owner.integer.Budget().Release(v.header)
	v.header = 0
	v.closed = true
	return nil
}
