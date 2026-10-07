// Source: src/util/mpf.cpp:1712-1769,1771-1792.
package util

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Independent IEEE bit checks are distinct from the original ENSURE adapter.
// Narrow denormal ToDouble deliberately preserves the literal source bottom
// exponent mapping, rather than substituting a general format conversion.
func TestArbitraryFloatIEEEIndependentBranchBitsAndWarmNoAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	cases := []struct {
		eb, sb uint32
		sign   bool
		exp    int64
		sig    uint64
		double uint64
		single uint32
		float  bool
	}{
		{11, 53, false, 1024, 0, 0x7ff0000000000000, 0, false},
		{11, 53, true, 1024, 0, 0xfff0000000000000, 0, false},
		{11, 53, false, 1024, 1 << 51, 0x7ff8000000000000, 0, false},
		{11, 53, true, -1023, 0, 0x8000000000000000, 0, false},
		{11, 53, false, -1023, 1, 1, 0, false},
		{11, 53, false, -1, 0, 0x3fe0000000000000, 0, false},
		{8, 24, false, 128, 0, 0x7ff0000000000000, 0x7f800000, true},
		{8, 24, true, 128, 0, 0xfff0000000000000, 0xff800000, true},
		{8, 24, false, 128, 1 << 22, 0x7ff8000000000000, 0x7fc00000, true},
		{8, 24, true, -127, 0, 0x8000000000000000, 0x80000000, true},
		{8, 24, false, -127, 1, 0x20000000, 1, true},
		{8, 24, false, -1, 0, 0x3fe0000000000000, 0x3f000000, true},
		{8, 24, false, 127, (1 << 23) - 1, 0x47efffffe0000000, 0x7f7fffff, true},
	}
	for i, c := range cases {
		var x ArbitraryFloat
		var sig Int
		serviceMust(t, m.integer.SetUint64(&sig, c.sig))
		serviceMust(t, m.SetComponents(&x, c.eb, c.sb, c.sign, c.exp, &sig))
		serviceMust(t, m.integer.Del(&sig))
		before := x
		got, e := m.ToDouble(&x)
		serviceMust(t, e)
		if math.Float64bits(got) != c.double {
			t.Fatal(i, math.Float64bits(got), c.double)
		}
		if c.float {
			got, e := m.ToFloat(&x)
			serviceMust(t, e)
			if math.Float32bits(got) != c.single {
				t.Fatal(i, math.Float32bits(got), c.single)
			}
		}
		used := b.Used()
		b.FailAfter(0)
		_, e = m.ToDouble(&x)
		serviceMust(t, e)
		if c.float {
			_, e = m.ToFloat(&x)
			serviceMust(t, e)
		}
		b.FailAfter(-1)
		if x != before || b.Used() != used {
			t.Fatal("read-only warm branch changed input/storage", i)
		}
		serviceMust(t, m.Del(&x))
	}
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatIEEERepresentedShiftGuardDomains(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var x ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 54, false, 0, &one))
	_, e = m.ToDouble(&x)
	if modularSourceDebug {
		if e != ErrIntegerAssertion {
			t.Fatal(e)
		}
	} else if e != arena.ErrBounds {
		t.Fatal("undefined source shift count must refuse", e)
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.SetComponents(&x, 8, 4, false, -1024, &one))
	if _, e = m.ToDouble(&x); e != arena.ErrBounds {
		t.Fatal("negative signed exponent shift domain", e)
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// The compiled source width assertion precedes the significand accessor. A
// stale borrowed NUM here is an invalid-Go-borrow ordering fixture only.
func TestArbitraryFloatIEEECompiledWidthAssertionLeavesSignificandUnread(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var x ArbitraryFloat
	var sig Int
	serviceMust(t, m.integer.SetUint64(&sig, 1<<40))
	serviceMust(t, m.SetComponents(&x, 8, 54, false, 0, &sig))
	serviceMust(t, m.integer.Del(&sig))
	stale := x
	serviceMust(t, m.Del(&x))
	used := b.Used()
	if _, e := m.ToDouble(&stale); e != ErrIntegerAssertion {
		t.Fatal("width assertion read stale significand", e)
	}
	stale.format = arbitraryFloatFormat(8, 25, false)
	if _, e := m.ToFloat(&stale); e != ErrIntegerAssertion {
		t.Fatal("width assertion read stale significand", e)
	}
	copied := *m
	if _, e := copied.ToDouble(&stale); e != ErrRuntimeOwner {
		t.Fatal("copied owner accepted", e)
	}
	if b.Used() != used {
		t.Fatal("rejected prefix allocated")
	}
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	if _, e := m.ToDouble(&stale); e != arena.ErrClosed {
		t.Fatal("closed owner accepted", e)
	}
}
