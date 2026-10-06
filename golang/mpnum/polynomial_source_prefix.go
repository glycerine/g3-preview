// Source: src/math/polynomial/polynomial.cpp:2346-2370. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

var ErrPolynomialSourceMember = errors.New("mpnum: missing multivariate interpolator source member binding")

// SourceFinalizePrefix performs only the actual constructed selected fields'
// destructor body prefix. It stops at m_mgcd_iterpolators.flush(), not a fake
// empty implementation or a completed polynomial::manager::imp destructor.
// Owning Close is distinct and never advances this stopped source history.
func (m *MultivariatePolynomialCore) SourceFinalizePrefix() error {
	if m != nil && m.self == m && m.quasiResultant.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.collectingVariables {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.normalizationPending() {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && (m.variableLocal.active || m.pscClassic.active || m.pscClassic.executing || m.pscLocal.active || m.pscNext.active || m.lazardLocal.active) {
		return ErrPolynomialPending
	}
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if m.closed || m.closing || !m.completed || m.originalCause != nil {
		return arena.ErrClosed
	}
	m.sourceStarted = true
	for m.sourceStage < 7 {
		var e error
		switch m.sourceStage {
		case 0, 1:
			e = m.sourcePrivateDEC(uint32(m.sourceStage))
		case 2:
			e = m.som.reset()
		case 3:
			e = m.som2.reset()
		case 4:
			e = m.cheap.reset()
		case 5:
			e = m.cheap2.reset()
		case 6:
			e = m.deleteNumeral(&m.zeroNumeral)
		}
		if e != nil {
			m.sourceCause = e
			return e
		}
		m.sourceStage++
	}
	m.sourceCause = ErrPolynomialSourceMember
	return ErrPolynomialSourceMember
}
func (m *MultivariatePolynomialCore) sourcePrivateDEC(i uint32) error {
	p := [2]Polynomial{m.zero, m.unit}[i]
	if !m.sourceDecDone[i] {
		r, e := m.readWord(p, 0)
		if e != nil {
			return e
		}
		if modularSourceDebug && r == 0 {
			return ErrPolynomialAssertion
		}
		r--
		if e = m.writeWord(p, 0, r); e != nil {
			return e
		}
		m.sourceDecDone[i] = true
		m.sourceDeleteNeeded[i] = r == 0
	}
	if m.sourceDeleteNeeded[i] {
		return m.deleteCore(p)
	}
	return nil
}
