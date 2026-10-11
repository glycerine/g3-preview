package lp

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestLPDriverPermutationIndependentResults(t *testing.T) {
	var d PermutationTestDriver
	w := &lpDriverFlushWriter{}
	if err := d.Run(w); err != nil || w.String() != "test permutations\n" || w.flushes != 1 {
		t.Fatal(err, w)
	}
	want := [][]uint32{{3, 1, 4, 2, 0}, {3, 0, 1, 4, 2}}
	if len(d.Products) != 2 {
		t.Fatal(len(d.Products))
	}
	for i, p := range d.Products {
		if !reflect.DeepEqual(p.Values(), want[i]) {
			t.Fatal(i, p.Values(), want[i])
		}
		// Independently audit both directions and the actual 0/1 matrix.
		for row, col := range want[i] {
			reverse, err := p.Reverse(col)
			if err != nil || reverse != uint32(row) {
				t.Fatal(i, row, col, reverse, err)
			}
			for j := uint32(0); j < 5; j++ {
				v, err := p.Element(uint32(row), j)
				expected := 0.0
				if j == col {
					expected = 1
				}
				if err != nil || v != expected {
					t.Fatal(i, row, j, v, expected, err)
				}
			}
		}
	}
	if !reflect.DeepEqual(d.Dense, []float64{0, 0, 0, 3, 1}) || !reflect.DeepEqual(d.Indexed.Data, d.Dense) || !reflect.DeepEqual(d.Indexed.Index, []uint32{4, 3}) {
		t.Fatal(d.Dense, d.Indexed)
	}
	var out bytes.Buffer
	if err := d.Run(&out); err != ErrExactVectorPending || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	var other PermutationTestDriver
	if err := other.Run(io.Discard); err != nil {
		t.Fatal(err)
	}
	d.Products[0].values[0] = 99
	d.Dense[0] = 99
	d.Indexed.Data[0] = 88
	if other.Products[0].values[0] != 3 || other.Dense[0] != 0 || other.Indexed.Data[0] != 0 {
		t.Fatal("independent owned scenario storage")
	}
}

func TestLPDriverPermutationReportFailures(t *testing.T) {
	const message = "test permutations\n"
	for at := 0; at < len(message); at++ {
		var d PermutationTestDriver
		w := &lpResidualFailWriter{remaining: at}
		if err := d.Run(lpResidualWriterOnly{w}); err != lpErrResidualWrite || w.String() != message[:at] || len(d.Products) != 0 || d.Dense != nil || d.Indexed != nil {
			t.Fatal(at, err, w.String(), d)
		}
		if err := d.Run(io.Discard); err != ErrExactVectorPending {
			t.Fatal(at, err)
		}
	}
	var d PermutationTestDriver
	w := &lpDriverFlushWriter{failAt: 1}
	if err := d.Run(w); err != lpErrResidualWrite || w.String() != message || len(d.Products) != 0 || d.Indexed != nil {
		t.Fatal(err, w, d)
	}
}
