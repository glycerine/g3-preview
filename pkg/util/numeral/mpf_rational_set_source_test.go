package numeral

// Source: src/util/mpf.cpp:233-300; src/util/mpf.h:82.

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatExponentRationalSetLiteralNormalization(t *testing.T) {
	for _, tc := range []struct {
		n, d, exp int64
		want      float64
	}{{0, 1, 0, 0}, {1, 2, 0, 0.5}, {3, 4, 0, 0.75}, {3, 2, 0, 1.5}, {7, 2, 0, 3.5}, {-7, 2, 1, -7}, {3, 2, -2, 0.375}} {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		q := NewRational()
		serviceMust(t, m.rational.SetFraction(&q, tc.n, uint64(tc.d)))
		ex := NewInt(int32(tc.exp))
		var out ArbitraryFloat
		serviceMust(t, m.SetExponentRational(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &q))
		got, e := m.ToDouble(&out)
		serviceMust(t, e)
		if math.Float64bits(got) != math.Float64bits(tc.want) {
			t.Fatal("literal normalization", tc, got)
		}
		serviceMust(t, m.Del(&out))
		serviceMust(t, m.rational.Del(&q))
		serviceMust(t, m.SourceFinalize())
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}

func TestArbitraryFloatExponentRationalSetSourceAliasAndBIGExponent(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	q := NewRational()
	serviceMust(t, m.rational.SetFraction(&q, 3, 2))
	var out ArbitraryFloat
	three := NewInt(3)
	serviceMust(t, m.SetComponents(&out, 8, 24, false, 0, &three))
	serviceMust(t, m.SetExponentRational(&out, 8, 24, ArbitraryFloatNearestEven, &out.significand, &q))
	got, e := m.ToDouble(&out)
	serviceMust(t, e)
	if got != 12 {
		t.Fatal("literal exp copied before caller floor publication", got)
	}
	var ex Int
	for _, v := range []int64{1 << 40, -1 << 40} {
		serviceMust(t, m.integer.SetInt64(&ex, v))
		serviceMust(t, m.SetExponentRational(&out, 8, 24, ArbitraryFloatNearestEven, &ex, &q))
		inf, e := m.IsInf(&out)
		serviceMust(t, e)
		if !inf || out.Sign() {
			t.Fatal("literal non-small exponent always infinity", v, out)
		}
	}
	serviceMust(t, m.integer.Del(&ex))
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
