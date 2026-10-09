// Source: src/util/uint_map.h:25-62.
package util

import "reflect"

// UintPtrMap maps uint32 indices to non-nil pointer-like Go values using a
// dense slice. The distinct name records that values are pointers rather than
// integer IDs.
type UintPtrMap struct {
	values []uintPtrMapSlot
}

type uintPtrMapSlot struct {
	value   any
	present bool
}

func NewUintPtrMap() *UintPtrMap { return &UintPtrMap{} }

func validUintPtrMapValue(value any) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Pointer && !v.IsNil()
}

func (m *UintPtrMap) Contains(key uint32) bool {
	return uint64(key) < uint64(len(m.values)) && m.values[key].present
}

func (m *UintPtrMap) Find(key uint32) (any, bool) {
	if !m.Contains(key) {
		return nil, false
	}
	return m.values[key].value, true
}

// Get returns a present value and panics when key is absent, matching the
// source asserted find overload.
func (m *UintPtrMap) Get(key uint32) any {
	value, found := m.Find(key)
	if !found {
		panic("g3: missing uint pointer-map key")
	}
	return value
}

func (m *UintPtrMap) Insert(key uint32, value any) {
	if !validUintPtrMapValue(value) {
		panic("g3: uint pointer-map value is not a non-nil pointer")
	}
	if uint64(key) >= uint64(len(m.values)) {
		next := make([]uintPtrMapSlot, uint64(key)+1)
		copy(next, m.values)
		m.values = next
	}
	m.values[key] = uintPtrMapSlot{value: value, present: true}
}

func (m *UintPtrMap) Erase(key uint32) {
	if uint64(key) < uint64(len(m.values)) {
		m.values[key] = uintPtrMapSlot{}
	}
}

func (m *UintPtrMap) Reset() {
	clear(m.values)
	m.values = nil
}
