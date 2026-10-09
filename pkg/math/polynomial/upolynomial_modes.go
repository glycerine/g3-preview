// Source: src/math/polynomial/upolynomial.h:151-156;
// src/math/polynomial/upolynomial_factorization.cpp:1213-1240.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

func (m *UPolynomialManager) SetZ() error {
	if e := m.check(); e != nil {
		return e
	}
	if m.trialCallPending() || m.recombinationCallPending() || m.preparationCallPending() || m.primePreparationCallPending() {
		return ErrUPolynomialCleanupPending
	}
	return m.modular.SetZ()
}
func (m *UPolynomialManager) SetZP(p *util.Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if m.trialCallPending() || m.recombinationCallPending() || m.preparationCallPending() || m.primePreparationCallPending() {
		return ErrUPolynomialCleanupPending
	}
	return m.modular.SetZP(p)
}
func (m *UPolynomialManager) SetZPUint64(p uint64) error {
	if e := m.check(); e != nil {
		return e
	}
	if m.trialCallPending() || m.recombinationCallPending() || m.preparationCallPending() || m.primePreparationCallPending() {
		return ErrUPolynomialCleanupPending
	}
	return m.modular.SetZPUint64(p)
}
func (m *UPolynomialManager) Modular() (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return m.modular.Modular()
}
func (m *UPolynomialManager) Field() (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return m.modular.Field()
}

// NumeralManager borrows the actual embedded source coefficient manager.
// Closing it independently violates its owning parent's lifetime contract.
func (m *UPolynomialManager) NumeralManager() (*util.IntegerModularManager, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return m.modular, nil
}
func (m *UPolynomialManager) borrowedModulus() (*util.Int, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return m.modular.BorrowedModulus()
}

type UPolynomialStats struct {
	StoreOwnerBytes                                                                                uint64
	ExclusiveBytes, SharedStoreBytes                                                               uint64
	OwnerBytes, DirectoryBytes, PageBytes, BackingBytes, ModularBytes, TotalBytes                  uint64
	IntegerCellSlots, IntegerLiveCells, IntegerRetiredCells, IntegerCachedCells, OwnedNumeralCells uint32
	LivePolynomials, LiveBackings                                                                  uint32
}

// Stats accounts the concrete physical storage graph; numeral cell payloads
// remain in the borrowed IntegerManager and are not charged twice here.
func (m *UPolynomialManager) Stats() UPolynomialStats {
	if m == nil || m.self != m || m.closed {
		return UPolynomialStats{}
	}
	s := UPolynomialStats{OwnerBytes: m.header, LivePolynomials: m.live, ModularBytes: m.modular.OwnerBytes()}
	for f := m.displayFrames; f != nil; f = f.next {
		s.OwnerBytes += f.header
	}
	s.DirectoryBytes = uint64(cap(m.pages))*uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()) + uint64(cap(m.bindings))*uint64(reflect.TypeOf(upolyStoreBinding{}).Size())
	s.PageBytes = uint64(len(m.pages)) * uint64(reflect.TypeOf(upolyDescriptorPage{}).Size())
	m.addStoreStats(&s, m.store)
	for _, binding := range m.bindings {
		m.addStoreStats(&s, binding.store)
	}
	s.TotalBytes = s.OwnerBytes + s.StoreOwnerBytes + s.DirectoryBytes + s.PageBytes + s.BackingBytes + s.ModularBytes
	s.ExclusiveBytes = s.TotalBytes - s.SharedStoreBytes
	z := m.config.Integer
	numeric := z.StorageStats()
	s.IntegerLiveCells, s.IntegerCellSlots, s.IntegerRetiredCells = numeric.LiveCells, numeric.CellSlots, numeric.RetiredCells
	s.IntegerCachedCells = numeric.CachedCells
	if z.IsLiveCellToken(m.display.numeral.NativeCellToken()) {
		s.OwnedNumeralCells++
	}
	for f := m.displayFrames; f != nil; f = f.next {
		if z.IsLiveCellToken(f.numeral.NativeCellToken()) {
			s.OwnedNumeralCells++
		}
	}
	s.OwnedNumeralCells += m.modular.OwnedNumeralCells()

	for _, page := range m.pages {
		for i := range page.rows {
			d := &page.rows[i]
			if d.live {
				for j := uint32(0); j < d.state.size; j++ {
					if z.IsLiveCellToken(m.values(&d.state)[j].NativeCellToken()) {
						s.OwnedNumeralCells++
					}
				}
			}
		}
	}
	states := [21]*upolyVectorState{&m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.mgcd[0], &m.mgcd[1], &m.mgcd[2], &m.mgcd[3], &m.mgcd[4], &m.mgcd[5], &m.sqf1, &m.sqf2, &m.pw, &m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push}
	for _, state := range states {
		for j := uint32(0); j < state.size; j++ {
			if z.IsLiveCellToken(m.values(state)[j].NativeCellToken()) {
				s.OwnedNumeralCells++
			}
		}
	}

	return s
}

func (m *UPolynomialManager) Checkpoint() error {
	if e := m.check(); e != nil {
		return e
	}
	return m.checkpoint()
}

// Modulus returns an owned copy of source p(); borrowedModulus is the actual
// private source-reference route for future coefficient/backend integration.
func (m *UPolynomialManager) Modulus(out *util.Int) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.modular.Modulus(out)
}

func (m *UPolynomialManager) addStoreStats(s *UPolynomialStats, store *UPolynomialCoefficientStore) {
	dir := uint64(cap(store.backings)) * uint64(reflect.TypeOf(upolyBacking{}).Size())
	s.StoreOwnerBytes += store.header
	s.DirectoryBytes += dir
	s.SharedStoreBytes += store.header + dir
	for i := range store.backings {
		b := &store.backings[i]
		if !b.live {
			continue
		}
		s.LiveBackings++
		bytes := uint64(cap(b.values)) * uint64(reflect.TypeOf(util.Int{}).Size())
		s.BackingBytes += bytes
		s.SharedStoreBytes += bytes
	}
}
