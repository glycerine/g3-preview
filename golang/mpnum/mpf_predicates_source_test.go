// Source: src/util/mpf.cpp:392-417,1771-1792,1821-1847.
// Source author: Christoph Wintersteiger.
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatLiteralClassificationPolicies(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	cases := []struct {
		exp                                   int64
		sig                                   int32
		nan, inf, zero, denormal, normal, one bool
	}{
		{128, 1, true, false, false, false, false, false},
		{128, 0, false, true, false, false, false, false},
		{-127, 0, false, false, true, false, false, false},
		{-127, 1, false, false, false, true, false, false},
		{0, 1, false, false, false, false, true, false},
		{0, 0, false, false, false, false, true, true},
	}
	for _, c := range cases {
		a := ArbitraryFloat{format: 8 | (24 << 15), exponent: c.exp, significand: NewInt(c.sig)}
		checks := []struct {
			got  bool
			err  error
			want bool
		}{}
		got, err := m.IsNaN(&a)
		checks = append(checks, struct {
			got  bool
			err  error
			want bool
		}{got, err, c.nan})
		got, err = m.IsInf(&a)
		checks = append(checks, struct {
			got  bool
			err  error
			want bool
		}{got, err, c.inf})
		got, err = m.IsZero(&a)
		checks = append(checks, struct {
			got  bool
			err  error
			want bool
		}{got, err, c.zero})
		got, err = m.IsDenormal(&a)
		checks = append(checks, struct {
			got  bool
			err  error
			want bool
		}{got, err, c.denormal})
		got, err = m.IsNormal(&a)
		checks = append(checks, struct {
			got  bool
			err  error
			want bool
		}{got, err, c.normal})
		got, err = m.IsOne(&a)
		checks = append(checks, struct {
			got  bool
			err  error
			want bool
		}{got, err, c.one})
		for i, check := range checks {
			if check.err != nil || check.got != check.want {
				t.Fatal("literal independent classification", c, i, check)
			}
		}
	}
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual cache lifecycle", b.Used())
	}
}

// Source: src/util/mpf.cpp:1840-1848. The independent bound oracle includes
// ebits=1, which is valid here while top/bottom require ebits>=2 in DEBUG.
func TestArbitraryFloatLiteralMinimumMaximumExponent(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	for _, bits := range []uint32{1, 2, 8, 15} {
		lo, e := m.minExponent(bits)
		serviceMust(t, e)
		hi, e := m.maxExponent(bits)
		serviceMust(t, e)
		want := int64(uint64(1)<<(bits-1)) - 1
		if lo != 1-want || hi != want {
			t.Fatal("literal exponent bounds", bits, lo, hi, want)
		}
		before := b.Used()
		b.FailAfter(0)
		again, e := m.minExponent(bits)
		serviceMust(t, e)
		maxAgain, e := m.maxExponent(bits)
		serviceMust(t, e)
		if again != lo || maxAgain != hi || b.Used() != before {
			t.Fatal("warm bounds changed actual caches")
		}
		if b.Used() != r.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("immediate actual cold graph")
		}
		b.FailAfter(-1)
	}
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("real owner end", b.Used())
	}
}
