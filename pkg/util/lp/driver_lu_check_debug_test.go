//go:build g3debug

package lp

import (
	"io"
	"math"
	"testing"
)

func TestLPLUCheckDebugResidualSourceTolerance(t *testing.T) {
	for _, tc := range []struct {
		want, solved float64
		fail         bool
	}{
		{0, 1e-6, false}, {0, math.Nextafter(1e-6, math.Inf(1)), true},
		{1e100, 1e100 * (1 + 5e-7), false}, {1e100, 1e100 * (1 + 2e-6), true},
		{0, math.NaN(), false}, {math.Inf(1), math.Inf(1), false},
	} {
		d := NewLUCheckFileDriver("independent prepared residual fixture", io.Discard)
		d.A = NewFloatStaticMatrix(1, 27)
		if err := d.A.Set(0, 0, 1); err != nil {
			t.Fatal(err)
		}
		if err := d.A.Set(0, 26, tc.want); err != nil {
			t.Fatal(err)
		}
		d.Basis = []uint32{0}
		d.Settings, _ = NewSettings()
		var err error
		d.LU, err = NewFloatLU(d.A, d.Basis, d.Settings)
		if err != nil {
			t.Fatal(err)
		}
		d.Column = NewIndexedFloatVector(1)
		if err = d.Column.SetValue(tc.solved, 0); err != nil {
			t.Fatal(err)
		}
		if err = d.checkResidual(); (err == ErrAssertion) != tc.fail {
			t.Fatal(tc, err)
		}
		if err = d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLPLUCheckDebugResidualFailurePrefixes(t *testing.T) {
	d := NewLUCheckFileDriver("independent residual metadata fixture", io.Discard)
	d.A = NewFloatStaticMatrix(1, 27)
	d.A.Set(0, 0, 1)
	d.Basis = []uint32{0}
	d.Settings, _ = NewSettings()
	var err error
	d.LU, err = NewFloatLU(d.A, d.Basis, d.Settings)
	if err != nil {
		t.Fatal(err)
	}
	d.Column = NewIndexedFloatVector(1)
	d.Column.Index = []uint32{1}
	if err = d.checkResidual(); err != ErrBounds {
		t.Fatal(err)
	}
	d.Column.Index = nil
	// Sparse Element returns zero for an absent column; it does not bounds-check.
	d.Basis[0] = 27
	if err = d.checkResidual(); err != nil {
		t.Fatal(err)
	}
	d.Basis = nil
	if err = d.checkResidual(); err != ErrAssertion {
		t.Fatal(err)
	}
	if err = d.Close(); err != nil {
		t.Fatal(err)
	}
}
