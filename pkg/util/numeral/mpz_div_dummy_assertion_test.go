package numeral

import (
	"errors"
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
		{"remainder", false, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := bindMPNDebugActions(t, "")
			m, _ := integerManager(t)
			a := NewInt(7)
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
