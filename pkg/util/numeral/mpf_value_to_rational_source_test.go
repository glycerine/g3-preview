package numeral

// Source: src/util/mpf.cpp:1695-1711,1922-1945;
// src/ast/rewriter/fpa_rewriter.cpp:793-809.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// src/test/mpf.cpp contains no direct to_rational assertion. These are literal
// reachable rewriter-input cases; they earn no original driver assertion credit.
func TestArbitraryFloatToRationalLiteralFiniteInputs(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	qm, e := NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	for _, v := range []struct {
		sign        bool
		exponent    int64
		significand int32
		want        string
	}{
		{false, 3, 4, "12"}, {true, 3, 4, "-12"}, {false, 0, 0, "1"},
		{false, -127, 4, "1/170141183460469231731687303715884105728"},
		{false, -127, 0, "0"}, {true, -127, 0, "0"},
	} {
		var a ArbitraryFloat
		n := NewInt(v.significand)
		serviceMust(t, m.SetComponents(&a, 8, 4, v.sign, v.exponent, &n))
		out := NewRational()
		serviceMust(t, m.ToRational(&a, qm, &out))
		if got := rationalString(t, qm, &out); got != v.want {
			t.Fatal(v, got)
		}
		serviceMust(t, qm.Del(&out))
		serviceMust(t, m.Del(&a))
	}
	serviceMust(t, qm.CloseChecked())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
