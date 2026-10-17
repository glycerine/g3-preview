package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalCompareSourceSignAssertionSites(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name string
		a, b Rational
		want int
	}{
		{"zero nonpositive", Rational{Num: NewInt(0), Den: NewInt(2)}, Rational{Num: NewInt(-1), Den: NewInt(2)}, 1},
		{"positive", Rational{Num: NewInt(1), Den: NewInt(2)}, Rational{Num: NewInt(0), Den: NewInt(2)}, 1},
		{"both negative", Rational{Num: NewInt(-1), Den: NewInt(2)}, Rational{Num: NewInt(-2), Den: NewInt(3)}, 1},
		{"both positive", Rational{Num: NewInt(1), Den: NewInt(2)}, Rational{Num: NewInt(2), Den: NewInt(3)}, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := bindMPNDebugActions(t, "")
			m, _ := rationalManager(t, true)
			if v, e := m.Compare(&tc.a, &tc.b); e != nil || v != tc.want {
				t.Fatal("sign shortcut or cross-product result", v, e)
			}
			if e := d.Close(); e != nil {
				t.Fatal(e)
			}
			if _, e := m.Compare(&tc.a, &tc.b); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("reached source assertion reads closed debug owner", e)
			}
		})
	}
}
