// Source: src/util/small_object_allocator.cpp; src/util/memory_manager.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package memory

// PendingRoots exposes only the actual allocation and two lexical scratch
// roots retained by this checked owner. It is an identity-only diagnostic,
// including during stopped cleanup, with no source action or ownership sweep.
func (a *SmallAllocator) PendingRoots() ([3]Block, error) {
	if a == nil || a.self != a {
		return [3]Block{}, ErrAllocatorOwner
	}
	return [3]Block{a.pending, a.pendingScratch[0], a.pendingScratch[1]}, nil
}
