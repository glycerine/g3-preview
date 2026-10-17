package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerRootInitialBoundsSourceAssertionStop(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	a := NewInt(4)
	serviceMust(t, d.Close())
	exact, err := m.RootInPlace(&a, 2)
	s := &m.RootRaw
	if exact || !errors.Is(err, arena.ErrClosed) || a != NewInt(4) || !s.active || s.EndEntered || s.Values[0] != NewInt(2) || s.Values[1] != NewInt(4) {
		t.Fatal("source bound assertion must stop after power approximations", exact, err, a, s.Values)
	}
	serviceMust(t, m.CleanupRoot())
}
