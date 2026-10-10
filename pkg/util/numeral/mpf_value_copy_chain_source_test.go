// Source: src/util/scoped_numeral_vector.h:25-65; src/util/mpf.cpp:316-333.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatCopyChainTwoActualPendingChildren(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	sig := NewInt(0)
	two := NewInt(2)
	serviceMust(t, m.integer.Power(&two, 100, &sig))
	var a ArbitraryFloat
	serviceMust(t, m.SetComponents(&a, 8, 120, false, 3, &sig))
	var rows [2]ArbitraryFloat
	var chain ArbitraryFloatValueCopyChain
	serviceMust(t, chain.Bind(m))
	rt.SetMaxSize(1)
	first := chain.CopyWithReceipt(&rows[0], &a, nil)
	second := chain.CopyWithReceipt(&rows[1], &a, nil)
	if first == nil || second == nil {
		t.Fatal("actual postphysical refusal not reached", first, second)
	}
	if chain.head == nil || chain.head.callerNext == nil || chain.head == chain.head.callerNext {
		t.Fatal("actual two issued children not retained")
	}
	firstCall, secondCall := chain.head.callerNext, chain.head
	firstTuple, secondTuple := firstCall.Issued, secondCall.Issued
	firstCause, secondCause := firstCall.bodyCause, secondCall.bodyCause
	used, state := b.Used(), rt.Stats()
	copied := chain
	if copied.Cleanup(true) == nil || copied.End() == nil || chain.End() == nil {
		t.Fatal("copied/unacknowledged chain guard")
	}
	if b.Used() != used || rt.Stats() != state || firstCall.Issued != firstTuple || secondCall.Issued != secondTuple {
		t.Fatal("rejected owner action changed retained graph")
	}
	integerPhysicalAuditWithRaw(t, m.integer, nil, &sig, &a.significand, &rows[0].significand, &rows[1].significand)
	rt.SetMaxSize(0)
	serviceMust(t, chain.Cleanup(false))
	serviceMust(t, m.CleanupValue(&rows[0]))
	serviceMust(t, m.CleanupValue(&rows[1]))
	serviceMust(t, chain.Cleanup(true))
	serviceMust(t, chain.End())
	if firstCall.Issued != firstTuple || secondCall.Issued != secondTuple || firstCall.bodyCause != firstCause || secondCall.bodyCause != secondCause || firstCall.header == 0 || secondCall.header == 0 {
		t.Fatal("actual context retains immutable issued diagnostic history")
	}
	serviceMust(t, chain.End())
	if chain.CopyWithReceipt(&rows[0], &a, nil) == nil || chain.Cleanup(true) == nil {
		t.Fatal("ended chain must not acquire or replay")
	}

	serviceMust(t, m.Del(&a))
	serviceMust(t, m.integer.Del(&sig))
	serviceMust(t, m.Close())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("exact owner end", b.Used())
	}
}
