// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpz.cpp power/is_power_of_two/ensure_capacity/normalize/
// machine_div2k/mul2k (_MP_INTERNAL 32-bit digits) and mpz.h copy wrappers.
package mpnum

import (
	"errors"
	"math"

	"github.com/glycerine/g3/golang/arena"
)

var ErrInvalidPower = errors.New("mpnum: integer zero to zero violates source power precondition")
var ErrNonterminatingPower = errors.New("mpnum: source power mask would not terminate")

type PowerDomainReason uint8

const (
	PowerZeroToZero PowerDomainReason = iota + 1
	PowerMaskOverflow
)

// PowerDomainError observes a source assertion or the actual mask-wrap boundary.
// Completed arithmetic/publication before that boundary remains visible.
type PowerDomainError struct {
	Exponent uint32
	Reason   PowerDomainReason
}

func (e *PowerDomainError) Error() string { return e.Unwrap().Error() }
func (e *PowerDomainError) Unwrap() error {
	if e.Reason == PowerZeroToZero {
		return ErrInvalidPower
	}
	return ErrNonterminatingPower
}

// ensureOwnedCapacity translates ensure_capacity, including requested capacity
// six and INT_MIN promotion. Old ownership is released after replacement exists.
func (m *IntegerManager) ensureOwnedCapacity(a *Int, capacity uint32) error {
	if capacity <= 1 {
		return nil
	}
	if capacity < 6 {
		capacity = 6
	}
	if !a.IsSmall() && m.cell(a.cell).capacity >= capacity {
		return nil
	}
	old := a.cell
	size := uint32(1)
	if old != 0 {
		size = m.cell(old).size
	}
	ref, e := m.allocate(capacity, size)
	if e != nil {
		return e
	}
	if modularSourceDebug && m.cell(ref).capacity != capacity {
		m.free(ref)
		return ErrIntegerAssertion
	}
	sign := a.val
	if old != 0 {
		for i := uint32(0); i < size; i++ {
			m.writeDigit(ref, i, m.digit(old, i))
		}
	} else {
		if a.val == math.MinInt32 {
			m.writeDigit(ref, 0, m.digit(m.intMin.cell, 0))
			sign = -1
		} else if a.val < 0 {
			m.writeDigit(ref, 0, uint32(-a.val))
			sign = -1
		} else {
			m.writeDigit(ref, 0, uint32(a.val))
			sign = 1
		}
	}
	if old != 0 {
		m.free(old)
	}
	a.cell = ref
	a.val = sign
	return nil
}
func (m *IntegerManager) normalizeOwned(a *Int) error {
	size := m.cell(a.cell).size
	for size > 0 && m.digit(a.cell, size-1) == 0 {
		size--
	}
	if size == 0 {
		return m.Reset(a)
	}
	if size == 1 && m.digit(a.cell, 0) <= math.MaxInt32 {
		v := int32(m.digit(a.cell, 0))
		if a.val < 0 {
			v = -v
		}
		if e := m.Del(a); e != nil {
			return e
		}
		a.val = v
		return nil
	}
	m.resizeCell(a.cell, size)
	return nil
}
func (m *IntegerManager) Power(a *Int, p uint32, out *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	return m.powerCore(a, p, out)
}
func (m *IntegerManager) powerCore(a *Int, p uint32, out *Int) error {
	if a.IsSmall() {
		switch a.val {
		case 2:
			if p < 31 {
				if e := m.Del(out); e != nil {
					return e
				}
				out.val = int32(uint32(1) << p)
				return nil
			}
			size := p/32 + 1
			shift := p % 32
			if modularSourceDebug && size == 0 {
				return ErrIntegerAssertion
			}
			if e := m.allocateIfNeeded(out, size); e != nil {
				return e
			}
			if modularSourceDebug && m.cell(out.cell).capacity < size {
				return ErrIntegerAssertion
			}
			// The physical Go span is guarded at its actual access, after
			// all source allocation and sign-publication actions.
			if size > m.cell(out.cell).capacity {
				return arena.ErrBounds
			}
			m.resizeCell(out.cell, size)
			for i := uint32(0); i < size-1; i++ {
				m.writeDigit(out.cell, i, 0)
			}
			// Explicit unsigned limb shift defines the intended bit at source signed 1<<31.
			m.writeDigit(out.cell, size-1, uint32(1)<<shift)
			out.val = 1
			return nil
		case 0:
			if modularSourceDebug && p == 0 {
				return &PowerDomainError{Exponent: p, Reason: PowerZeroToZero}
			}
			return m.Reset(out)
		case 1:
			return m.SetInt64(out, 1)
		}
	}
	var power Int
	defer m.Del(&power)
	if e := m.Copy(&power, a); e != nil {
		return e
	}
	if e := m.SetInt64(out, 1); e != nil {
		return e
	}
	for mask := uint32(1); mask <= p; mask <<= 1 {
		if mask == 0 {
			return &PowerDomainError{Exponent: p, Reason: PowerMaskOverflow}
		}
		if mask&p != 0 {
			if e := m.Mul(out, &power, out); e != nil {
				return e
			}
		}
		// The final square is deliberately retained, even when its value is unused.
		if e := m.Mul(&power, &power, &power); e != nil {
			return e
		}
	}
	return nil
}
func (m *IntegerManager) IsPowerOfTwo(a *Int) (bool, uint32, error) {
	if e := m.check(a); e != nil {
		return false, 0, e
	}
	if a.val <= 0 {
		return false, 0, nil
	}
	if a.IsSmall() {
		v := uint32(a.val)
		if v&(v-1) != 0 {
			return false, 0, nil
		}
		return true, log2Uint32(v), nil
	}
	size := m.cell(a.cell).size
	for i := uint32(0); i < size-1; i++ {
		if m.digit(a.cell, i) != 0 {
			return false, 0, nil
		}
	}
	v := m.digit(a.cell, size-1)
	if v == 0 || v&(v-1) != 0 {
		return false, 0, nil
	}
	k, e := m.Log2(a)
	return e == nil, k, e
}

func (m *IntegerManager) MachineDiv2k(a *Int, k uint32, out *Int) error {
	return m.shift(a, k, out, false)
}
func (m *IntegerManager) Mul2k(a *Int, k uint32, out *Int) error { return m.shift(a, k, out, true) }
func (m *IntegerManager) MachineDiv2kInPlace(a *Int, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.machineDiv2kCore(a, k)
}
func (m *IntegerManager) Mul2kInPlace(a *Int, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.mul2kCore(a, k)
}
func (m *IntegerManager) shift(a *Int, k uint32, out *Int, left bool) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if e := m.Copy(out, a); e != nil {
		return e
	}
	if left {
		return m.mul2kCore(out, k)
	}
	return m.machineDiv2kCore(out, k)
}
func (m *IntegerManager) machineDiv2kCore(a *Int, k uint32) error {
	if k == 0 || a.val == 0 {
		return nil
	}
	if a.IsSmall() {
		if k < 32 {
			twok := int32(uint32(1) << k)
			a.val /= twok
		} else {
			a.val = 0
		}
		return nil
	}
	digitShift := k / 32
	size := m.cell(a.cell).size
	if digitShift >= size {
		return m.Reset(a)
	}
	bitShift := k % 32
	compShift := 32 - bitShift
	newSize := size - digitShift
	if modularSourceDebug && newSize < 1 {
		return ErrIntegerAssertion
	}
	if newSize < size {
		i, j := uint32(0), digitShift
		if bitShift != 0 {
			for ; i < newSize-1; i, j = i+1, j+1 {
				v := m.digit(a.cell, j)>>bitShift | m.digit(a.cell, j+1)<<compShift
				m.writeDigit(a.cell, i, v)
			}
			m.writeDigit(a.cell, i, m.digit(a.cell, j)>>bitShift)
		} else {
			for ; i < newSize; i, j = i+1, j+1 {
				m.writeDigit(a.cell, i, m.digit(a.cell, j))
			}
		}
	} else {
		if modularSourceDebug && (newSize != size || bitShift == 0) {
			return ErrIntegerAssertion
		}
		var i uint32
		for ; i < newSize-1; i++ {
			v := m.digit(a.cell, i)>>bitShift | m.digit(a.cell, i+1)<<compShift
			m.writeDigit(a.cell, i, v)
		}
		m.writeDigit(a.cell, i, m.digit(a.cell, i)>>bitShift)
	}
	m.resizeCell(a.cell, newSize)
	return m.normalizeOwned(a)
}
func (m *IntegerManager) mul2kCore(a *Int, k uint32) error {
	if k == 0 || a.val == 0 {
		return nil
	}
	if a.IsSmall() && k < 32 {
		return m.SetInt64(a, int64(a.val)*(int64(1)<<k))
	}
	wordShift, bitShift := k/32, k%32
	oldSize := uint32(1)
	if !a.IsSmall() {
		oldSize = m.cell(a.cell).size
	}
	newSize := oldSize + wordShift + 1 // source unsigned32 addition
	if e := m.ensureOwnedCapacity(a, newSize); e != nil {
		return e
	}
	if modularSourceDebug && a.IsSmall() {
		return ErrIntegerAssertion
	}
	// Guard the actual encoded backing access after source capacity preparation.
	if a.IsSmall() {
		return arena.ErrBounds
	}
	oldSize = m.cell(a.cell).size
	cell := m.cell(a.cell)
	// Prepare only the concrete Words access span. Retain the source cell size
	// through zero filling; publish newSize at the actual source statement.
	span := newSize
	if span > cell.capacity {
		span = cell.capacity
	}
	payload, e := m.words.Resize(cell.payload, span)
	if e != nil {
		return e
	}
	cell.payload = payload
	m.store(a.cell, cell)
	for i := oldSize; i < newSize; i++ {
		if i >= cell.capacity || i >= span {
			return arena.ErrBounds
		}
		m.writeDigit(a.cell, i, 0)
	}
	cell.size = newSize
	m.store(a.cell, cell)
	if wordShift > 0 {
		j, i := oldSize, oldSize+wordShift
		for j > 0 {
			j--
			i--
			if i >= cell.capacity || i >= span || j >= cell.capacity || j >= span {
				return arena.ErrBounds
			}
			m.writeDigit(a.cell, i, m.digit(a.cell, j))
		}
		for i > 0 {
			i--
			if i >= cell.capacity || i >= span {
				return arena.ErrBounds
			}
			m.writeDigit(a.cell, i, 0)
		}
	}
	if bitShift > 0 {
		if modularSourceDebug {
			for i := uint32(0); i < wordShift; i++ {
				if i >= cell.capacity || i >= span {
					return arena.ErrBounds
				}
				if m.digit(a.cell, i) != 0 {
					return ErrIntegerAssertion
				}
			}
		}
		compShift := 32 - bitShift
		var previous uint32
		for i := wordShift; i < newSize; i++ {
			if i >= cell.capacity || i >= span {
				return arena.ErrBounds
			}
			v := m.digit(a.cell, i)
			next := v >> compShift
			m.writeDigit(a.cell, i, v<<bitShift|previous)
			previous = next
		}
	}
	return m.normalizeOwned(a)
}
