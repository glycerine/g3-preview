// Source: src/util/mpf.cpp:1712-1769; src/test/mpf.cpp:23-82.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Original ENSURE values are preserved using explicit normalized source fields.
// The original int/double/float SET driver dependencies remain separate; this
// adapted test does not claim that complete registered driver is translated.
func TestArbitraryFloatIEEEOriginalAssertionsAdapted(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	cases := []struct {
		eb, sb         uint32
		sign           bool
		exp            int64
		sig            uint64
		d              float64
		f              float32
		checkD, checkF bool
	}{
		{11, 53, false, 1, 1 << 51, 3, 0, true, false},
		{11, 53, false, -1023, 0, 0, 0, true, false},
		{11, 53, true, 0, 0, -1, 0, true, false},
		{11, 53, false, 30, uint64(1073741823) << 22, 2147483647, 0, true, false},
		{11, 53, true, 31, 0, -2147483648, 0, true, false},
		{8, 24, false, 1, 1 << 22, 3, 3, true, true},
		{8, 24, false, -127, 0, 0, 0, true, true},
		{8, 24, true, 0, 0, -1, -1, true, true},
		{8, 24, true, 31, 0, 0, -2147483648, false, true},
		{11, 53, false, 1, 1 << 50, 2.5, 0, true, false},
		{11, 53, true, 5, 41 << 45, -42.25, 0, true, false},
		{8, 24, false, 1, 1 << 21, 2.5, 0, true, false},
		{8, 24, true, 5, 41 << 16, -42.25, 0, true, false},
		{8, 24, false, 1, 1 << 21, 0, 2.5, false, true},
		{8, 24, true, 5, 41 << 16, 0, -42.25, false, true},
	}
	assertions := 0
	for i, c := range cases {
		var x ArbitraryFloat
		var sig Int
		serviceMust(t, m.integer.SetUint64(&sig, c.sig))
		serviceMust(t, m.SetComponents(&x, c.eb, c.sb, c.sign, c.exp, &sig))
		serviceMust(t, m.integer.Del(&sig))
		if c.checkD {
			got, e := m.ToDouble(&x)
			serviceMust(t, e)
			if got != c.d {
				t.Fatal("original double assertion", i, got, c.d)
			}
			assertions++
		}
		if c.checkF {
			got, e := m.ToFloat(&x)
			serviceMust(t, e)
			if got != c.f {
				t.Fatal("original float assertion", i, got, c.f)
			}
			assertions++
		}
		serviceMust(t, m.Del(&x))
	}
	if assertions != 18 {
		t.Fatal("lost original assertions", assertions)
	}
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
