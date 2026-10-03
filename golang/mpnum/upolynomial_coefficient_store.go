// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// UPolynomialCoefficientStore owns the actual coefficient backing directory.
// Related arithmetic cores borrow the same IntegerManager and retain this
// store; their descriptors, modular state and scratch ownership stay distinct.
// It is caller-confined, as are its source arithmetic managers.
type UPolynomialCoefficientStore struct {
	self     *UPolynomialCoefficientStore
	origin   uint64
	integer  *IntegerManager
	backings []upolyBacking
	owners   uint32
	header   uint64
	closed   bool
}
type UPolynomialCoefficientStoreStats struct {
	Owners, LiveBackings, OwnedNumeralCells              uint32
	OwnerBytes, DirectoryBytes, BackingBytes, TotalBytes uint64
}

func newUPolynomialCoefficientStore(z *IntegerManager, origin uint64) (*UPolynomialCoefficientStore, error) {
	n := uint64(reflect.TypeOf(UPolynomialCoefficientStore{}).Size())
	if e := z.budget.Reserve(n); e != nil {
		return nil, e
	}
	s := &UPolynomialCoefficientStore{integer: z, origin: origin, owners: 1, header: n}
	s.self = s
	return s, nil
}
func (s *UPolynomialCoefficientStore) check() error {
	if s == nil || s.self != s || s.integer == nil {
		return ErrUPolynomialOwner
	}
	if s.closed || s.integer.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *UPolynomialCoefficientStore) retain() error {
	if e := s.check(); e != nil {
		return e
	}
	if s.owners == math.MaxUint32 {
		return arena.ErrSlotLimit
	}
	s.owners++
	return nil
}
func (s *UPolynomialCoefficientStore) release() {
	s.owners--
	if s.owners != 0 {
		return
	}
	s.integer.budget.Release(uint64(cap(s.backings)) * uint64(reflect.TypeOf(upolyBacking{}).Size()))
	s.integer.budget.Release(s.header)
	s.backings = nil
	s.closed = true
}
func NewRelatedUPolynomialManager(parent *UPolynomialManager) (*UPolynomialManager, error) {
	if e := parent.check(); e != nil {
		return nil, e
	}
	return newUPolynomialManager(parent.config, parent.store)
}
func (m *UPolynomialManager) CoefficientStore() (*UPolynomialCoefficientStore, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	return m.store, nil
}

// Stats is the once-only physical cohort graph. Add it once to each core's
// ExclusiveBytes. Numeral payload allocation remains in the borrowed Integer;
// OwnedNumeralCells counts live coefficient roots, excluding modular caches.
func (s *UPolynomialCoefficientStore) Stats() (UPolynomialCoefficientStoreStats, error) {
	if e := s.check(); e != nil {
		return UPolynomialCoefficientStoreStats{}, e
	}
	st := UPolynomialCoefficientStoreStats{Owners: s.owners, OwnerBytes: s.header, DirectoryBytes: uint64(cap(s.backings)) * uint64(reflect.TypeOf(upolyBacking{}).Size())}
	for i := range s.backings {
		b := &s.backings[i]
		if !b.live {
			continue
		}
		st.LiveBackings++
		st.BackingBytes += uint64(cap(b.values)) * uint64(reflect.TypeOf(Int{}).Size())
		for j := uint32(0); j < b.initialized; j++ {
			if validNumericRef(s.integer.cells, b.values[j].cell) {
				st.OwnedNumeralCells++
			}
		}
	}
	st.TotalBytes = st.OwnerBytes + st.DirectoryBytes + st.BackingBytes
	return st, nil
}
