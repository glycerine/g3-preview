// Source: src/util/fixed_bit_vector.cpp:78-80; src/util/small_object_allocator.cpp:72-96.
package util

// DeallocateView is the raw source-pointer overload. The receiver's byte count
// and allocator, not the donor's count or ObjectRef owner, select source actions.
// It returns actual release observations; the caller pointer remains unchanged.
func (m *RuntimeFixedBitVectorManager) DeallocateView(v RuntimeFixedBitVectorView) (ReleaseProgress, error) {
	if e := m.open(); e != nil {
		return ReleaseProgress{}, e
	}
	if m.numBytes == 0 {
		return ReleaseProgress{}, nil
	}
	p := v.address()
	return m.allocator.DeallocateRawTracked(uint64(m.numBytes), RawObjectPointer{Runtime: p.runtime, Block: p.block, Offset: p.offset})
}
