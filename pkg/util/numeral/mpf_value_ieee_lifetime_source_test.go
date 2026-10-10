// Source: src/util/mpf.cpp:1712-1769; src/util/mpf.h:241-290.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatIEEEEveryReachedCacheReservationPrefix(t *testing.T) {
	for _, single := range []bool{false, true} {
		completed := false
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(1 << 27)
			r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
			serviceMust(t, e)
			m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			var x ArbitraryFloat
			one := NewInt(1)
			eb, sb := uint32(11), uint32(53)
			if single {
				eb, sb = 8, 24
			}
			serviceMust(t, m.SetComponents(&x, eb, sb, false, 0, &one))
			before := x
			foreign, e := r.Allocate(23)
			serviceMust(t, e)
			b.FailAfter(ordinal)
			var body error
			if single {
				_, body = m.ToFloat(&x)
			} else {
				_, body = m.ToDouble(&x)
			}
			integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{foreign}, &x.significand)
			if b.Used() != r.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before-lift exact cache/caller cold graph", single, ordinal, b.Used())
			}
			if x != before {
				t.Fatal("read-only caller changed input", single, ordinal)
			}
			b.FailAfter(-1)
			serviceMust(t, m.Del(&x))
			if e := m.Close(); e != nil {
				serviceMust(t, m.Close())
			}
			if _, e := r.Size(foreign); e != nil {
				t.Fatal("cache cleanup consumed foreign", e)
			}
			serviceMust(t, r.Deallocate(&foreign))
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			if body == nil {
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("bounded cache prefix horizon", single)
		}
	}
}

func TestArbitraryFloatIEEEActualCacheTransportRetainsEarlierCall(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	var x ArbitraryFloat
	armed := false
	calls := 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			used := b.Used()
			oldOwn, oldSource := m.ownCause, m.sourceCause
			for _, e := range []error{m.Close(), m.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize()} {
				if e != ErrNumericPending {
					t.Fatal("actual cache callback owner entry", e)
				}
			}
			if _, e := m.ToDouble(&x); e != ErrNumericPending {
				t.Fatal("same caller reentry", e)
			}
			if b.Used() != used || m.ownCause != oldOwn || m.sourceCause != oldSource {
				t.Fatal("rejected callback mutated older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 11, 53, false, 0, &one))
	before := x
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	_, body := m.ToDouble(&x)
	armed = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 {
		t.Fatal("real wrapper allocation callback", body, calls)
	}
	old := m.PowerWrappers
	if old == nil || old.bodyCause != body {
		t.Fatal("actual failed cache authority", old)
	}
	original := old.original
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &x.significand)
	if b.Used() != r.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before-lift cache physical/cold graph")
	}
	serviceMust(t, r.SetMaxSize(0))
	_, e = m.ToDouble(&x)
	serviceMust(t, e)
	if x != before || old.bodyCause != body || old.original != original || old == m.PowerWrappers {
		t.Fatal("later caller erased earlier actual cache history")
	}
	if original != (umemory.Block{}) {
		if _, e := r.Size(original); e != nil {
			t.Fatal("later caller consumed earlier cache raw", e)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
