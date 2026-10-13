// Source: src/util/scoped_numeral_vector.h:23-67, instantiated as
// scoped_mpf_vector by src/math/subpaving/subpaving.cpp and
// src/ast/fpa_decl_plugin.cpp.
package numeral

import "github.com/glycerine/g3/pkg/arena"

// ScopedArbitraryFloatVector owns MPF numerals while borrowing their manager.
// It uses a native slice; callers must release borrowed slots before growth.
type ScopedArbitraryFloatVector struct {
	self    *ScopedArbitraryFloatVector
	manager *ArbitraryFloatManager
	data    []ArbitraryFloat
	closed  bool
}

func NewScopedArbitraryFloatVector(m *ArbitraryFloatManager) (*ScopedArbitraryFloatVector, error) {
	if m == nil {
		return nil, ErrScopedOwner
	}
	if err := m.valueOwnerGuard(); err != nil {
		return nil, err
	}
	v := &ScopedArbitraryFloatVector{manager: m}
	v.self = v
	return v, nil
}

func (v *ScopedArbitraryFloatVector) open() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return arena.ErrClosed
	}
	return v.manager.valueOwnerGuard()
}

func (v *ScopedArbitraryFloatVector) Size() uint32 {
	if v == nil || v.self != v || v.closed {
		return 0
	}
	return uint32(len(v.data))
}

// Reserve follows the source vector's logical-size reserve. Its new slots are
// default MPF numerals, ready for the manager's Set operation.
func (v *ScopedArbitraryFloatVector) Reserve(n uint32) error {
	if err := v.open(); err != nil {
		return err
	}
	if uint64(n) > uint64(int(^uint(0)>>1)) {
		return arena.ErrBounds
	}
	if int(n) > len(v.data) {
		v.data = append(v.data, make([]ArbitraryFloat, int(n)-len(v.data))...)
	}
	return nil
}

func (v *ScopedArbitraryFloatVector) BorrowSlot(i uint32) (*ArbitraryFloat, error) {
	if err := v.open(); err != nil {
		return nil, err
	}
	if uint64(i) >= uint64(len(v.data)) {
		return nil, arena.ErrBounds
	}
	return &v.data[i], nil
}

func (v *ScopedArbitraryFloatVector) BorrowedValues() ([]ArbitraryFloat, error) {
	if err := v.open(); err != nil {
		return nil, err
	}
	return v.data, nil
}

func (v *ScopedArbitraryFloatVector) Set(i uint32, value *ArbitraryFloat) error {
	slot, err := v.BorrowSlot(i)
	if err != nil {
		return err
	}
	return v.manager.Copy(slot, value)
}

func (v *ScopedArbitraryFloatVector) Shrink(n uint32) error {
	if err := v.open(); err != nil {
		return err
	}
	if uint64(n) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	for i := int(n); i < len(v.data); i++ {
		if err := v.manager.Del(&v.data[i]); err != nil {
			return err
		}
	}
	v.data = v.data[:n]
	return nil
}

func (v *ScopedArbitraryFloatVector) Close() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return nil
	}
	if err := v.Shrink(0); err != nil {
		return err
	}
	v.data = nil
	v.closed = true
	return nil
}
