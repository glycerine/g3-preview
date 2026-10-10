// Source: src/util/mpf.cpp:335-374; src/util/mpz.h:156-164.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedSetEveryDirectReservationPrefix(t *testing.T) {
	success := false
	captured := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		one := NewInt(1)
		var x, out ArbitraryFloat
		serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
		var receipt ArbitraryFloatFormattedSetReceipt
		b.FailAfter(ordinal)
		body := m.SetFormatWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, &x, &receipt)
		s := &m.RoundScope
		integerPhysicalAudit(t, m.integer, &x.significand, &out.significand, &s.sigma, &s.cap, &s.sticky)
		if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("before-lift complete named cold graph", ordinal, b.Used(), m.memory.Stats().TotalBytes, m.OwnedStorageBytes())
		}
		if h := m.formattedSet.history; h != nil {
			if receipt != h.Issued || h.bodyCause != m.formattedSet.bodyCause {
				t.Fatal("actual leaf record publication")
			}
			if h.Numeric.Cell.Object != (umemory.ObjectRef{}) || h.Transferred.Block() != (umemory.Block{}) {
				captured = true
			}
		}
		b.FailAfter(-1)
		if body == nil {
			got, e := m.integer.Int64(&out.significand)
			serviceMust(t, e)
			if got != 0 || out.exponent != 0 {
				t.Fatal("preserved success arithmetic", ordinal, got, out.exponent)
			}
			success = true
		}
		serviceMust(t, m.CleanupFormattedSetWithReceipt(&receipt, false))
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.CleanupFormattedSetWithReceipt(&receipt, true))
		serviceMust(t, m.CleanupFormattedSet())
		serviceMust(t, m.Del(&x))
		if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("owner leak", ordinal, b.Used())
		}
		if success {
			break
		}
	}
	if !success || !captured {
		t.Fatal("missing source success/actual direct acquisition", success, captured)
	}
}
func TestArbitraryFloatFormattedSetErasedBranchesFailAfterZero(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	var nan, inf, zero, same, out ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.MkNaN(8, 24, &nan, nil))
	serviceMust(t, m.MkInf(8, 24, true, &inf))
	serviceMust(t, m.MkZero(8, 24, true, &zero))
	serviceMust(t, m.SetComponents(&same, 8, 24, false, 0, &one))
	for _, x := range []*ArbitraryFloat{&nan, &inf, &zero, &same} {
		serviceMust(t, m.SetFormat(&out, 8, 24, ArbitraryFloatNearestEven, x))
	}
	used := b.Used()
	var r ArbitraryFloatFormattedSetReceipt
	b.FailAfter(0)
	for repeat := 0; repeat < 4; repeat++ {
		for _, x := range []*ArbitraryFloat{&nan, &inf, &zero, &same} {
			serviceMust(t, m.SetFormatWithReceipt(&out, 8, 24, ArbitraryFloatNearestEven, x, &r))
			if r != (ArbitraryFloatFormattedSetReceipt{}) || b.Used() != used {
				t.Fatal("source erased branch allocated/adopted", repeat)
			}
		}
	}
	b.FailAfter(-1)
	for _, x := range []*ArbitraryFloat{&nan, &inf, &zero, &same, &out} {
		serviceMust(t, m.Del(x))
	}
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
