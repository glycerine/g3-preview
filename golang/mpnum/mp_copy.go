// Copyright (c) 2006 Microsoft Corporation
// Source set/copy semantics, adapted to the explicit numerical owner boundary
// required by GO_PORT_PLAN.md B12. C++ rational uses one global manager instead.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// CopyFrom deep-copies an integer from an explicitly supplied source manager.
// Raw Int has no owner provenance: the caller must supply the actual owner of
// each operand, just as for other low-level IntegerManager methods. The public
// rational service validates provenance before calling this bridge. A single
// raw address cannot simultaneously own a value in two different managers.
// Source storage is only read; destination allocation preserves source live
// size/capacity and sign instead of formatting or normalizing the value.
func (m *IntegerManager) CopyFrom(dst *Int, source *IntegerManager, src *Int) error {
	if m == nil || source == nil {
		return arena.ErrConfig
	}
	if m == source {
		return m.Copy(dst, src)
	}
	if dst == src {
		return arena.ErrConfig
	}
	if e := m.check(dst); e != nil {
		return e
	}
	if e := source.check(src); e != nil {
		return e
	}
	if src.IsSmall() {
		value := src.val
		if e := m.Del(dst); e != nil {
			return e
		}
		dst.val = value
		return nil
	}
	cell := source.cell(src.cell)
	ref := dst.cell
	if ref == 0 || m.cell(ref).capacity < cell.size {
		var e error
		ref, e = m.allocate(cell.capacity, cell.size)
		if e != nil {
			return e
		}
	}
	m.resizeCell(ref, cell.size)
	// All fallible preparation is complete before changing an existing payload.
	for i := uint32(0); i < cell.size; i++ {
		m.writeDigit(ref, i, source.digit(src.cell, i))
	}
	old := dst.cell
	dst.cell, dst.val = ref, src.val
	m.publishCell(ref)
	if old != 0 && old != ref {
		if e := m.free(old); e != nil {
			return e
		}
	}
	return nil
}

// CopyFrom transfers numerical ownership by independent component copies.
// It preserves unreduced fractions, negative denominators and small/big tags.
// The pair is staged in fully charged destination cells so failure copying the
// denominator cannot publish only the numerator. Source limbs never escape.
func (m *RationalManager) CopyFrom(dst *Rational, source *RationalManager, src *Rational) error {
	if m == nil || source == nil {
		return arena.ErrConfig
	}
	if m == source {
		return m.Copy(dst, src)
	}
	if dst == src {
		return arena.ErrConfig
	}
	if e := m.check(dst); e != nil {
		return e
	}
	if e := source.check(src); e != nil {
		return e
	}
	work := NewRational()
	defer m.drop(&work)
	if e := m.z.CopyFrom(&work.num, source.z, &src.num); e != nil {
		return e
	}
	if e := m.z.CopyFrom(&work.den, source.z, &src.den); e != nil {
		return e
	}
	return m.Move(dst, &work)
}
