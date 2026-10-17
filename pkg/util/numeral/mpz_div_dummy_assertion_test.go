package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPZDivisionDummyAssertionReadsAfterPublication(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name   string
		divide bool
		want   int32
	}{
		{"quotient", true, 0},
		{"remainder", false, math.MinInt32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := bindMPNDebugActions(t, "")
			m, _ := integerManager(t)
			// INT_MIN uses the source's prebuilt cell and skips the earlier
			// get_sign_cell argument-size assertion.
			a := NewInt(math.MinInt32)
			var b Int
			serviceMust(t, m.SetLimbs(&b, []uint32{0, 1}))
			c := NewInt(99)
			serviceMust(t, d.Close())
			var err error
			if tc.divide {
				err = m.MachineDiv(&a, &b, &c)
			} else {
				err = m.Rem(&a, &b, &c)
			}
			if !errors.Is(err, arena.ErrClosed) || c.Val != tc.want || c.Cell != 0 {
				t.Fatalf("post-publication source assertion = (%v, %+v), want closed and value %d", err, c, tc.want)
			}
		})
	}
}
