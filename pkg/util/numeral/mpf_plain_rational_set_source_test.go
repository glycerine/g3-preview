package numeral

// Source: src/util/mpf.cpp:186-192; src/util/mpf.h:80.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPlainRationalSetLiteralCaller(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	q := NewRational()
	serviceMust(t, m.rational.SetFraction(&q, 3, 4))
	var out ArbitraryFloat
	serviceMust(t, m.SetRational(&out, 8, 24, ArbitraryFloatNearestEven, &q))
	got, e := m.ToDouble(&out)
	serviceMust(t, e)
	if got != 0.75 {
		t.Fatal(got)
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
