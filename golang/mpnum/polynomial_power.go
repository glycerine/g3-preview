// Source: src/math/polynomial/polynomial.cpp:4600-4630; src/util/obj_ref.h.
// Licensed under the MIT license.
package mpnum

import "errors"

// Embedded actual result record is charged by the core's reflected header.
// It retains stopped source destruction and unjoined INC separately; no ticket.
type polynomialPowerScope struct {
	generation              uint64
	pscOwner                uint64
	substitutionOwned       bool
	result                  polynomialReference
	active                  bool
	bodyCause, cleanupCause error
}

func (m *MultivariatePolynomialCore) Power(p Polynomial, k uint32, r *PolynomialRef) (err error) {
	if e := m.check(); e != nil {
		return e
	}
	if r == nil || r.owner != m {
		return ErrPolynomialOwner
	}
	if e := r.open(); e != nil {
		return e
	}
	return m.powerInto(p, k, &r.root)
}

// The same source engine accepts a completed default source reference.
func (m *MultivariatePolynomialCore) powerInto(p Polynomial, k uint32, r *polynomialReference) (err error) {
	if k == 0 {
		return r.assign(m, m.unit)
	}
	if k == 1 {
		return r.assign(m, p)
	}
	local := &m.powerLocal
	if local.active {
		return ErrPolynomialPending
	}
	if m.powerGeneration == ^uint64(0) {
		return ErrPolynomialPending
	}
	m.powerGeneration++
	*local = polynomialPowerScope{generation: m.powerGeneration, active: true, substitutionOwned: m.substitution.scope.executing, pscOwner: m.pscExecutingOwner()}
	if f := m.factorsOwner; f != nil && f.executing && m.callbackDepth == 0 {
		f.powerGeneration = local.generation
	}
	defer func() {
		local.bodyCause = err
		c := local.result.closeValue(m, false)
		local.cleanupCause = c
		if c != nil {
			err = errors.Join(err, c)
		}
		if c == nil && local.result.value == (Polynomial{}) && local.result.incoming == (Polynomial{}) && !local.result.assigning {
			*local = polynomialPowerScope{}
		}
	}()
	if e := local.result.assign(m, p); e != nil {
		return e
	}
	for i := uint32(1); i < k; i++ {
		product, e := m.Mul(local.result.value, p)
		if e != nil {
			return e
		}
		if e = local.result.assign(m, product); e != nil {
			return e
		}
	}
	return r.assign(m, local.result.value)
}

// Explicit owning abort follows all completed source destruction. Source
// Power never silently retries a stopped destructor or consumes unjoined INC.
func (m *MultivariatePolynomialCore) CleanupPolynomialPower() error {
	return m.cleanupPolynomialPowerPhase(true, true)
}
func (m *MultivariatePolynomialCore) cleanupPolynomialPowerPhase(owning, abort bool) error {
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if !m.powerLocal.active {
		return nil
	}
	if e := m.powerLocal.result.closeValue(m, owning); e != nil {
		m.powerLocal.cleanupCause = e
		return e
	}
	if abort {
		if e := m.powerLocal.result.abort(m); e != nil {
			m.powerLocal.cleanupCause = e
			return e
		}
	}
	if m.powerLocal.result.assigning || m.powerLocal.result.incoming != (Polynomial{}) {
		return nil
	}
	m.powerLocal = polynomialPowerScope{}
	return nil
}
