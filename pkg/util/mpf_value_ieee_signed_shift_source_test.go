// Source: src/util/mpf.cpp:1712-1737; CMakeLists.txt:224-229.
package util

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// C++11 CWG1457's corrected signed-left-shift rule allows these nonnegative
// operands whose shifted value fits uint64. Conversion back to signed uses
// the explicit two's-complement target profile. These raw source fields are
// intentionally distinct from canonical mathematical floating-point values.
func TestArbitraryFloatToDoubleDefinedUnsignedRepresentableSignedShifts(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	cases := []struct {
		sb   uint32
		exp  int64
		sig  uint64
		bits uint64
	}{
		{53, 1025, 0, 0x8000000000000000},
		{53, 3072, 0, 0xfff0000000000000},
		{52, 0, 1 << 62, 0xbff0000000000000},
	}
	for i, c := range cases {
		var x ArbitraryFloat
		var sig Int
		serviceMust(t, m.integer.SetUint64(&sig, c.sig))
		serviceMust(t, m.SetComponents(&x, 11, c.sb, false, c.exp, &sig))
		serviceMust(t, m.integer.Del(&sig))
		before := x
		got, e := m.ToDouble(&x)
		if e != nil {
			t.Error("defined raw-field case rejected", i, e)
		} else if math.Float64bits(got) != c.bits || x != before {
			t.Fatal("literal unsigned-representable shift", i, math.Float64bits(got), c.bits)
		}
		serviceMust(t, m.Del(&x))
	}
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
