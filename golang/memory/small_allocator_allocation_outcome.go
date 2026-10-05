// Source: src/util/small_object_allocator.cpp:allocate.
// Licensed under the MIT license.
package memory

type smallAllocatorAllocationScope struct {
	invocation uint64
	executing  bool
}

// Outcomes are stamped only at the real guarded allocation and acquisition
// leaves. FreshRaw distinguishes direct/new-chunk acquisition from a free-list
// or existing-chunk object; successful allocator-owned chunks are not pending.
type SmallAllocatorAllocationOutcome struct {
	owner      *SmallAllocator
	frame      *smallAllocatorAllocationScope
	invocation uint64
	Object     ObjectRef
	Pending    Block
	FreshRaw   bool
}

func (a *SmallAllocator) MatchesAllocationOutcome(o *SmallAllocatorAllocationOutcome) bool {
	return o != nil && o.owner == a && o.frame == &a.allocationInvocation && o.invocation != 0 && o.invocation == o.frame.invocation
}
func (a *SmallAllocator) rememberPendingWithOutcome(h Block, o *SmallAllocatorAllocationOutcome) {
	a.rememberPending(h)
	if h != (Block{}) && a.MatchesAllocationOutcome(o) {
		o.Pending = h
		o.FreshRaw = true
	}
}
func (a *SmallAllocator) recordAllocationObject(o *SmallAllocatorAllocationOutcome, r ObjectRef, fresh bool) ObjectRef {
	if a.MatchesAllocationOutcome(o) {
		o.Object = r
		o.FreshRaw = fresh
	}
	return r
}
