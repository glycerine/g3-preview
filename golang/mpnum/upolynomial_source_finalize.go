// Source: src/math/polynomial/upolynomial.cpp:1734-1764.
// Source: src/math/polynomial/upolynomial.cpp:138-157,187,1259-1265; src/util/mpzzp.h:97-105. Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// SourceFinalize runs only the source private-member destructor sequence.
// Caller-owned polynomial descriptors and all cold ownership metadata survive
// for explicit owning cleanup. No descriptor-directory scan acquires ownership.
func (m *UPolynomialManager) SourceFinalize() error { return m.sourceFinalizeWithHensel(nil) }
func (m *UPolynomialManager) sourceFinalizeWithHensel(r *upolyHenselPairInvocationReceipt) error {
	return m.sourceFinalizeWithHenselOwners(r, nil)
}
func (m *UPolynomialManager) sourceFinalizeWithHenselOwners(r *upolyHenselPairInvocationReceipt, c *upolyHenselCallerScope) error {
	return m.sourceFinalizeWithCallerOwners(r, c, nil)
}
func (m *UPolynomialManager) sourceFinalizeWithCallerOwners(r *upolyHenselPairInvocationReceipt, c *upolyHenselCallerScope, p *upolyFactorPrimePreparationInvocationReceipt) error {
	if m.integerEvaluationExecuting() || m.rationalEvaluationExecuting() || m.binaryEvaluationExecuting() {
		return ErrUPolynomialCleanupPending
	}
	if m != nil && m.self == m && m.algebraicNegCaller != nil && m.algebraicNegCaller.negationExecuting() {
		return ErrUPolynomialCleanupPending
	}
	if m == nil || m.self != m || m.config.Integer == nil {
		return ErrUPolynomialOwner
	}
	if m.integerEvaluation.active && m.integerEvaluation.sourceCause != nil {
		return m.integerEvaluation.sourceCause
	}

	if m.rationalEvaluation.active && m.rationalEvaluation.sourceCause != nil {
		return m.rationalEvaluation.sourceCause
	}

	if m.binaryEvaluation.active && m.binaryEvaluation.sourceCause != nil {
		return m.binaryEvaluation.sourceCause
	}
	if m.mignotteCallPending() && !m.henselDestructorAuthorized(r, false) && !m.henselCallerDestructorAuthorized(c, false) && !m.primePreparationDestructorAuthorized(p, false) {
		return ErrUPolynomialCleanupPending
	}
	if m.sourceFinalized {
		return nil
	}
	if m.closed || m.config.Integer.closed || (m.closing && !m.sourceFinalizing) {
		return arena.ErrClosed
	}
	// A stopped source invocation cannot acknowledge an owning release on
	// reentry. Explicit Close performs that continuation and keeps this history.
	if m.sourceCause != nil {
		return m.sourceCause
	}
	// Complete the actual display source suffix before private older members;
	// failed SMT RAW bodies remain unjoined for explicit owning cleanup.
	if q := &m.display; q.active {
		q.enterCall()
		e := q.finish(nil)
		q.leaveCall()
		if e != nil {
			return e
		}
	}
	for q := m.displayFrames; q != nil; q = q.next {
		if !q.active {
			continue
		}
		q.enterCall()
		e := q.finish(nil)
		q.leaveCall()
		if e != nil {
			return e
		}
	}
	m.sourceFinalizing = true
	derived := [5]*upolyVectorState{&m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push}
	core := [16]*upolyVectorState{&m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.mgcd[0], &m.mgcd[1], &m.mgcd[2], &m.mgcd[3], &m.mgcd[4], &m.mgcd[5], &m.sqf1, &m.sqf2, &m.pw}
	for m.sourceStage < 5 {
		switch m.sourceStage {
		case 0, 2:
			states := derived[:]
			if m.sourceStage == 2 {
				states = core[:]
			}
			for _, s := range states { // sourceIndex is per reached vector, not an inferred live prefix.
				// Earlier vectors in this body have completed logical size zero.
				if s.size == 0 {
					continue
				}
				if s.backing == 0 {
					m.sourceGuard = true
					m.sourceCause = arena.ErrBounds
					return m.sourceCause
				}
				a := m.values(s)
				for m.sourceIndex < s.size {
					if m.sourceIndex >= s.initialized || uint64(m.sourceIndex) >= uint64(len(a)) {
						m.sourceGuard = true
						m.sourceCause = arena.ErrBounds
						return m.sourceCause
					}
					guard, e := upolynomialSourceNumeralDel(m.config.Integer, &a[m.sourceIndex])
					if e != nil {
						m.sourceGuard = guard
						m.sourceCause = e
						return e
					}
					m.sourceIndex++
				}
				s.size = 0
				m.sourceIndex = 0
			}
		case 1:
			for i := len(derived) - 1; i >= 0; i-- {
				m.freeBacking(derived[i])
			}
		case 3:
			for i := len(core) - 1; i >= 0; i-- {
				m.freeBacking(core[i])
			}
		case 4:
			roots := [7]*Int{&m.modular.p, &m.modular.lower, &m.modular.upper, &m.modular.invTmp1, &m.modular.invTmp2, &m.modular.invTmp3, &m.modular.divTmp}
			for m.sourceIndex < 7 {
				guard, e := upolynomialSourceNumeralDel(m.config.Integer, roots[m.sourceIndex])
				if e != nil {
					m.sourceGuard = guard
					m.sourceCause = e
					return e
				}
				m.sourceIndex++
			}
		}
		m.sourceStage++
		m.sourceIndex = 0
	}
	m.sourceFinalized = true
	m.sourceCause = nil
	return nil
}

// CleanupPolynomial is an owning-abort continuation for the actual provided
// caller field, including after source modular destruction. It does not revive
// the owner, reconstruct source fields, or discover caller roots in a registry.
func (m *UPolynomialManager) CleanupPolynomial(p *UPolynomial) error {
	if m == nil || m.self != m || m.config.Integer == nil {
		return ErrUPolynomialOwner
	}
	if m.closed || m.config.Integer.closed {
		return arena.ErrClosed
	}
	if p == nil {
		return ErrUPolynomialOwner
	}
	if *p == (UPolynomial{}) {
		return nil
	}
	if p.origin != m.origin || p.slot == 0 {
		return ErrUPolynomialOwner
	}
	i := p.slot - 1
	if uint64(i/32) >= uint64(len(m.pages)) {
		return ErrUPolynomialOwner
	}
	d := &m.pages[i/32].rows[i%32]
	if !d.live || d.generation != p.generation {
		return ErrUPolynomialOwner
	}
	if d.state.backing != 0 {
		a := m.values(&d.state)
		for j := uint32(0); j < d.state.initialized; j++ {
			if e := m.config.Integer.Del(&a[j]); e != nil {
				return e
			}
		}
	}
	m.freeBacking(&d.state)
	d.live = false
	if d.generation != ^uint32(0) {
		d.generation++
	}
	m.live--
	*p = UPolynomial{}
	return nil
}

// This literal source leaf never acknowledges a previously consumed physical
// release. The descriptor's actual owning release cursor belongs to Close.
func upolynomialSourceNumeralDel(z *IntegerManager, a *Int) (bool, error) {
	if a.cell != 0 && validNumericRef(z.cells, a.cell) {
		c := z.cell(a.cell)
		if c.cleanupPending || c.released {
			return true, arena.ErrInvalidSlot
		}
	}
	return sourceNumeralDel(z, a)
}
