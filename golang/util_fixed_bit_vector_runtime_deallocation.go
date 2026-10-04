// Source: src/util/fixed_bit_vector.cpp:78-80; src/util/small_object_allocator.cpp:72-96.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package g3

import "github.com/glycerine/g3/golang/memory"

// DeallocateView is the raw source-pointer overload. The receiver's byte count
// and allocator, not the donor's count or ObjectRef owner, select source actions.
// It returns actual release observations; the caller pointer remains unchanged.
func (m *RuntimeFixedBitVectorManager) DeallocateView(v RuntimeFixedBitVectorView) (memory.ReleaseProgress, error) {
	if e := m.open(); e != nil {
		return memory.ReleaseProgress{}, e
	}
	if m.numBytes == 0 {
		return memory.ReleaseProgress{}, nil
	}
	p := v.address()
	return m.allocator.DeallocateRawTracked(uint64(m.numBytes), memory.RawObjectPointer{Runtime: p.runtime, Block: p.block, Offset: p.offset})
}
