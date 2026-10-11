// Source: src/test/lp.cpp:58-63,515-569.
package lp

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestLPLPZeroDriverActualSourceProfile(t *testing.T) {
	w := &lpDriverFlushWriter{}
	d := NewLPZeroDriver(w)
	err := d.Run()
	wantErr := ErrBounds
	if sourceDebugEnabled {
		wantErr = ErrAssertion
	}
	if err != wantErr || w.String() != " test_lp_0 \n" || w.flushes != 1 || d.closed || d.Core == nil || d.Core.Factorization != nil || d.Core.Status != StatusFeasible || d.Iterations != 0 {
		t.Fatal("default source profile", err, w.String(), d)
	}
	if d.Settings.SimplexStrategy != StrategyTableauRows || d.Settings.MaxTotalIterations != 20000000 || !d.Settings.BackupCosts || d.Core.EpsilonOfReducedCost != 1e-7 {
		t.Fatal("source settings and reached InitRun", d.Settings, d.Core.EpsilonOfReducedCost)
	}
	if !reflect.DeepEqual(d.B, []float64{1329, 762, 2511}) || !reflect.DeepEqual(d.X, []float64{225, 117, 420, 0, 0, 0, 0}) || !reflect.DeepEqual(d.Basis, []uint32{0, 1, 2}) || !reflect.DeepEqual(d.NBasis, []uint32{6, 5, 4, 3}) || !reflect.DeepEqual(d.Heading, []int32{0, 1, 2, -4, -3, -2, -1}) {
		t.Fatal("literal model and heading publications", d.B, d.X, d.Basis, d.NBasis, d.Heading)
	}
	for j, cost := range []float64{19, 13, 12, 17, 0, 0, 0} {
		if d.Costs[j] != cost/19 || d.Core.CostsBackup[j] != cost/19 || d.Types[j] != ColumnLowBound || (*d.Core.LowBounds)[j] != 0 {
			t.Fatal("normalization and zero-low constructor", j, d.Costs, d.Core.CostsBackup)
		}
	}
	if len(d.Upper) != 0 || d.Core.LowBounds != &d.Core.lowBoundsDummy || d.Core.X != &d.X || d.Core.B != &d.B || d.Core.Basis != &d.Basis || d.Core.Heading != &d.Heading {
		t.Fatal("borrowed source vector headers")
	}
	for i, row := range [][]float64{{3, 2, 1, 2, 1, 0, 0}, {1, 1, 1, 1, 0, 1, 0}, {4, 3, 3, 4, 0, 0, 1}} {
		for j, value := range row {
			got, e := d.Matrix.Element(uint32(i), uint32(j))
			if e != nil || got != value {
				t.Fatal("literal matrix", i, j, got, e)
			}
		}
	}
	if d.Core.Names.ColumnName(0) != "x0" || d.Core.Names.ColumnName(4294967295) != "x4294967295" {
		t.Fatal("simple source names")
	}
	if d.Run() != ErrExactVectorPending {
		t.Fatal("repeated run")
	}
	if err = d.Close(); err != nil {
		t.Fatal(err)
	}
	if err = d.Close(); err != nil || d.Core != nil || !d.closed {
		t.Fatal("cleanup", err)
	}
	if d.Run() != ErrExactVectorPending {
		t.Fatal("closed run")
	}
}

func TestLPLPZeroDriverEveryReportPrefix(t *testing.T) {
	want := " test_lp_0 \n"
	for cut := 0; cut < len(want); cut++ {
		out := &larConstraintPrefixWriter{remaining: cut, err: io.ErrClosedPipe}
		d := NewLPZeroDriver(out)
		err := d.Run()
		if !errors.Is(err, io.ErrClosedPipe) || out.output.String() != want[:cut] || d.Matrix != nil || d.Core != nil || d.Settings != nil || d.B != nil || d.X != nil {
			t.Fatal("report before model", cut, err, out.output.String(), d)
		}
		if err = d.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for _, short := range []bool{false, true} {
		var out io.Writer
		if short {
			out = &larConstraintPrefixWriter{remaining: 2}
		} else {
			out = &lpDriverFlushWriter{failAt: 1}
		}
		d := NewLPZeroDriver(out)
		err := d.Run()
		wantErr := lpErrResidualWrite
		if short {
			wantErr = io.ErrShortWrite
		}
		if !errors.Is(err, wantErr) || d.Matrix != nil || d.Core != nil {
			t.Fatal("endl/short-write prefix", err, d)
		}
		if err = d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLPLPZeroDriverIndependentStorage(t *testing.T) {
	a, b := NewLPZeroDriver(io.Discard), NewLPZeroDriver(io.Discard)
	_ = a.Run()
	_ = b.Run()
	if a.Matrix == nil || b.Matrix == nil || a.Core == nil || b.Core == nil {
		t.Fatal("missing actual children")
	}
	a.X[0] = 99
	a.B[0] = 77
	a.Basis[0] = 2
	a.Heading[0] = -9
	a.Costs[0] = 123
	if err := a.Matrix.Set(0, 0, 456); err != nil {
		t.Fatal(err)
	}
	if b.X[0] != 225 || b.B[0] != 1329 || b.Basis[0] != 0 || b.Heading[0] != 0 || b.Costs[0] != 1 {
		t.Fatal("shared source model headers")
	}
	if got, err := b.Matrix.Element(0, 0); err != nil || got != 3 {
		t.Fatal("shared matrix", got, err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if b.Core == nil {
		t.Fatal("sibling child closed")
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}
