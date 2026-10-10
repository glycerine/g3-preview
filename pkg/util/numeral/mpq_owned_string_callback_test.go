// Source: src/util/mpq.cpp:123-127; actual denominator-display transport.
package numeral

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestRationalOwnedStringNativeDenominatorScratch(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	a := Rational{Num: NewInt(3), Den: NewInt(1)}
	var out utext.OwnedString
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			t.Fatal("native denominator formatting entered Runtime transport")
		}

		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, m.integer.Mul2kInPlace(&a.Den, 8192))
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.rational.ToOwnedString(&a, &out)
	armed = false
	if body != nil || calls != 0 || m.rational.stringScope.active || !out.Matches("3/"+new(big.Int).Lsh(big.NewInt(1), 8192).String()) {
		t.Fatal("native denominator formatting", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &a.Num, &a.Den)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("immediate denominator graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(1<<28))
	serviceMust(t, m.rational.CleanupOwnedString())
	serviceMust(t, m.rational.Del(&a))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("denominator transport refunds", b.Used())
	}
}
