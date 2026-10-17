package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerPerfectSquareLoopSourceAssertionStop(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	input, out := NewInt(4), NewInt(17)
	serviceMust(t, d.Close())
	exact, err := m.IsPerfectSquare(&input, &out)
	s := &m.PerfectSquare
	if exact || !errors.Is(err, arena.ErrClosed) || out != NewInt(0) || !s.Active || s.EndEntered {
		t.Fatal("first loop assertion must stop after source setup", exact, err, out)
	}
	for i, want := range []Int{NewInt(1), NewInt(4), NewInt(0), NewInt(1), NewInt(16), NewInt(0)} {
		if s.Values[i] != want {
			t.Fatal("source setup before first loop assertion", i, s.Values[i], want)
		}
	}
	serviceMust(t, m.CleanupPerfectSquare())
}
