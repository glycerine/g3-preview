// Licensed under the MIT license.
package memory

import "testing"

// These are tests of the source counter transitions, not an execution of
// malloc/realloc or the Windows API/context/BV-sort memory campaign.
func TestMemoryAccountingTLSStrictThresholds(t *testing.T) {
	var a utilMemoryAccounting
	var lane utilMemoryLaneAccounting
	if got := a.accountAllocation(&lane, 100000, true); got != memoryAccountingOK {
		t.Fatal(got)
	}
	if a.allocationSize() != 0 || lane.size != 100000 || a.allocationCount() != 0 {
		t.Fatal("getter must not flush local")
	}
	if got := a.accountAllocation(&lane, 1, true); got != memoryAccountingOK {
		t.Fatal(got)
	}
	if a.allocationSize() != 100001 || a.maximumUsed() != 100001 || a.allocationCount() != 2 || lane.size != 0 || lane.count != 2 {
		t.Fatal("strict > threshold synchronization", a, lane)
	}
	memoryAccountingTestDeallocate(&a, &lane, 100000, true)
	if a.allocationSize() != 100001 || lane.size != -100000 {
		t.Fatal("strict negative threshold")
	}
	memoryAccountingTestDeallocate(&a, &lane, 1, true)
	if a.allocationSize() != 0 || lane.size != 0 || a.allocationCount() != 4 || lane.count != 2 {
		t.Fatal("deallocation sync still adds retained local count", a, lane)
	}
}

func TestMemoryAccountingTLSRepeatedCountAndLimitOrder(t *testing.T) {
	var a utilMemoryAccounting
	var lane utilMemoryLaneAccounting
	a.maxCount = 2
	if a.accountAllocation(&lane, 100001, true) != memoryAccountingOK {
		t.Fatal("first contribution")
	}
	if a.accountAllocation(&lane, 100001, true) != memoryAccountingCountLimit {
		t.Fatal("source retained count 1+2 exceeds2")
	}
	if a.allocationCount() != 3 || lane.count != 2 || lane.size != 0 || a.outOfMemory {
		t.Fatal("completed count-limit prefix", a, lane)
	}
	a.maxSize = 200002
	if a.accountAllocation(&lane, 100001, true) != memoryAccountingOOM {
		t.Fatal("memory limit takes precedence over count")
	}
	if !a.outOfMemory || a.allocationCount() != 6 || a.allocationSize() != 300003 {
		t.Fatal("OOM synchronization prefix")
	}
}

func TestMemoryAccountingGlobalCounterPrefixes(t *testing.T) {
	var a utilMemoryAccounting
	a.maxSize = 12
	if a.accountAllocation(nil, 12, false) != memoryAccountingOK {
		t.Fatal("equality must fit")
	}
	if a.accountAllocation(nil, 1, false) != memoryAccountingOOM {
		t.Fatal("strict memory maximum")
	}
	// This is the before-malloc counter route: refusal does NOT roll back size.
	if a.allocationSize() != 13 || a.maximumUsed() != 13 || a.allocationCount() != 2 || !a.outOfMemory {
		t.Fatal("counter prefix before backend", a)
	}
	a.maxSize = 0
	a.maxCount = 2
	a.outOfMemory = false
	if a.accountAllocation(nil, 1, false) != memoryAccountingCountLimit {
		t.Fatal("count maximum")
	}
	if a.allocationSize() != 14 || a.maximumUsed() != 14 || a.allocationCount() != 3 || a.outOfMemory {
		t.Fatal("count-limit state")
	}
}

func TestMemoryAccountingDeallocationClampAndWatermark(t *testing.T) {
	var a utilMemoryAccounting
	var lane utilMemoryLaneAccounting
	a.watermark = 5
	if a.accountAllocation(nil, 5, false) != memoryAccountingOK || a.aboveWatermark() {
		t.Fatal("watermark equality")
	}
	a.accountAllocation(nil, 1, false)
	if !a.aboveWatermark() {
		t.Fatal("strict watermark")
	}
	a.watermark = 0
	if a.aboveWatermark() {
		t.Fatal("zero disables watermark")
	}
	memoryAccountingTestDeallocate(&a, nil, 20, false)
	if a.size != -14 || a.allocationSize() != 0 || a.maximumUsed() != 6 || a.allocationCount() != 2 {
		t.Fatal("negative clamp/retained peak", a)
	}
	// synchronize_counters(false) publishes over-limit counters without transport.
	a.size = 0
	a.maxSize = 1
	a.maxCount = 1
	lane.size = 200002
	lane.count = 7
	memoryAccountingTestDeallocate(&a, &lane, 100001, true)
	if a.accountingSynchronize(&lane, false) != memoryAccountingOK || a.size != 100001 || a.count != 9 || a.outOfMemory {
		t.Fatal("deallocating sync must not throw", a, lane)
	}
}

func TestMemoryAccountingBoundedIndependentLaneModel(t *testing.T) {
	var a utilMemoryAccounting
	var lanes [3]utilMemoryLaneAccounting
	var sizes, counts [3]int64
	var size, count, peak int64
	for i := 0; i < 300; i++ {
		lane := i % 3
		// Independent event stream, including shrink-realloc negative deltas.
		delta := int64((i*7919)%90001 - 30000)
		sizes[lane] += delta
		counts[lane]++
		if sizes[lane] > 100000 {
			size += sizes[lane]
			count += counts[lane]
			sizes[lane] = 0
			if size > peak {
				peak = size
			}
		}
		if a.accountAllocation(&lanes[lane], delta, true) != memoryAccountingOK {
			t.Fatal("unlimited accounting")
		}
		if a.size != size || a.count != count || a.peak != peak || lanes[lane].size != sizes[lane] || lanes[lane].count != counts[lane] {
			t.Fatalf("event%d counter model differs", i)
		}
	}
}

func TestMemoryAccountingSignedBoundaryPrefixes(t *testing.T) {
	const maximum = int64(1<<63 - 1)
	var a utilMemoryAccounting
	a.count = maximum
	if a.accountAllocation(nil, 1, false) != memoryAccountingDomain {
		t.Fatal("signed source overflow needs explicit boundary")
	}
	if a.size != 1 || a.count != maximum || a.peak != 0 {
		t.Fatal("size precedes undefined count increment", a)
	}
	var lane utilMemoryLaneAccounting
	lane.count = maximum
	if a.accountAllocation(&lane, 1, true) != memoryAccountingDomain || lane.size != 1 || lane.count != maximum {
		t.Fatal("local size precedes count domain guard")
	}
	a = utilMemoryAccounting{}
	if a.accountAllocation(nil, 1, true) != memoryAccountingDomain {
		t.Fatal("missing required explicit TLS lane")
	}
}

func TestMemoryAccountingDeallocationPhysicalBoundary(t *testing.T) {
	var a utilMemoryAccounting
	var lane utilMemoryLaneAccounting
	if a.accountAllocation(&lane, 100001, true) != memoryAccountingOK {
		t.Fatal("initial synchronization")
	}
	if a.beginDeallocation(&lane, 100001, true) != memoryAccountingOK {
		t.Fatal("before free")
	}
	if lane.size != -100001 || a.allocationSize() != 100001 || a.allocationCount() != 1 {
		t.Fatal("free must occur before global synchronization")
	}
	// The actual allocation adapter performs free here; this fixture verifies
	// only the source counter state on each side of that required physical action.
	if a.finishDeallocation(&lane, true) != memoryAccountingOK {
		t.Fatal("after free")
	}
	if a.allocationSize() != 0 || a.allocationCount() != 2 || lane.size != 0 {
		t.Fatal("post-free synchronization")
	}
}

// Compose the counter-only phases for transition tests. Physical free belongs
// between these calls in the real runtime; it is not implemented by this helper.
func memoryAccountingTestDeallocate(a *utilMemoryAccounting, lane *utilMemoryLaneAccounting, extent int64, tls bool) memoryAccountingResult {
	if r := a.beginDeallocation(lane, extent, tls); r != memoryAccountingOK {
		return r
	}
	return a.finishDeallocation(lane, tls)
}

func TestMemoryAccountingSourceSizeTPromotions(t *testing.T) {
	const maximum = int64(1<<63 - 1)
	var a utilMemoryAccounting
	a.size = maximum - 1
	// C++ long long += uint64_t is unsigned addition then the selected
	// two's-complement signed conversion; this is not signed-addition UB.
	if a.accountExtentAllocation(nil, 8, 64, false) != memoryAccountingOK {
		t.Fatal("LP64 extent addition")
	}
	if a.size != -maximum+5 || a.count != 1 || a.peak != 0 {
		t.Fatal("source LP64 unsigned promotion", a)
	}
	a = utilMemoryAccounting{}
	a.size = 5
	// On a32-bit size_t profile, subtraction happens as uint32 first. LL64
	// can represent that uint32 delta, so the promoted addition is positive.
	before, after := uint32(16), uint32(8)
	if a.accountExtentAllocation(nil, uint64(after-before), 32, false) != memoryAccountingOK {
		t.Fatal("ILP32 shrink accounting")
	}
	if a.size != 4294967293 {
		t.Fatal("source uint32 delta must not become signed -8", a.size)
	}
	a = utilMemoryAccounting{}
	a.size = -maximum - 1
	if a.beginExtentDeallocation(nil, 1, 64, false) != memoryAccountingOK || a.size != maximum {
		t.Fatal("LP64 unsigned subtraction")
	}
	a = utilMemoryAccounting{}
	a.size = maximum
	if a.accountExtentAllocation(nil, 1, 32, false) != memoryAccountingDomain || a.size != maximum {
		t.Fatal("ILP32 promoted LL signed overflow boundary")
	}
}
