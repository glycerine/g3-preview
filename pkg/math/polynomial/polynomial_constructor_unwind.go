// Source: src/math/polynomial/polynomial.cpp:1821-1850,2324-2344;
// src/util/mpzzp.h:97-105; src/util/vector.h.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

// Only completed selected members unwind. No imp destructor body executes:
// body-created cells and the body monomial-manager retain remain named roots
// for explicit owning abort. The missing m_upm is not synthesized here.
func (m *MultivariatePolynomialCore) unwindConstructorMembers() error {
	m.constructorUnwinding = true
	if m.tmp1.completed && !m.tmp1SourceConsumed {
		if m.tmp1.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&m.tmp1.raw); e != nil {
				return e
			}
		}
		m.tmp1SourceConsumed = true
	}
	if m.constructorUnwindStage == 0 {
		if m.idsInitialized {
			m.idsSourceDestroyed = true
			if e := m.ids.SourceFinalize(); e != nil {
				return e
			}
			m.idsInitialized = false
		}
		m.constructorUnwindStage++
	}
	if m.constructorUnwindStage == 1 {
		m.directorySourceDestroyed = m.directoryInitialized
		// The published pointer-vector backing belongs to the completed
		// member. A failed initial lexical allocation remains pending and
		// belongs only to explicit owning cleanup, not this destructor.
		if m.directoryInitialized && m.polynomials.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&m.polynomials.raw); e != nil {
				return e
			}
		}
		m.constructorUnwindStage++
	}
	if m.constructorUnwindStage == 2 {
		if m.modular != nil {
			m.modularSourceDestroyed = true
			if e := m.modular.CleanupConstructedMembers(&m.constructorModularIndex); e != nil {
				return e
			}
		}
		m.constructorUnwindStage++
	}
	m.constructorUnwinding = false
	return nil
}

// After source member destruction, owning abort still has genuine numeral
// fields. It uses the borrowed Integer owner directly, never a destroyed
// modular source object or a repaired ID/directory member.
func (m *MultivariatePolynomialCore) deleteNumeral(a *util.Int) error {
	if m.modularSourceDestroyed {
		return m.integer.Del(a)
	}
	return m.modular.Del(a)
}
