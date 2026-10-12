//go:build g3debug

package lp

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strings"
	"testing"
)

func TestLPDebugMatrixPrintSourceFixtures(t *testing.T) {
	for _, world := range []struct {
		rows [][]string
		want string
	}{
		{nil, "\n"}, {[][]string{{}, {}}, "\n\n\n"},
		{[][]string{{"1", "20"}, {"300", "4"}}, "  1 20 \n300  4 \n\n"},
		{[][]string{{"é", "x"}, {"a", "long"}}, "é    x \n a long \n\n"},
		{[][]string{{"", "a"}, {"xx", ""}}, "   a \nxx   \n\n"},
	} {
		var out bytes.Buffer
		lpStaticMust(t, PrintDebugStringMatrix(world.rows, &out))
		if out.String() != world.want {
			t.Fatal(out.String(), world.want)
		}
	}
	m := lpDebugDense(t, [][]float64{{1.23456789, math.Copysign(0, -1), 1e-5, 1e6, math.Inf(1), math.NaN()}, {-12.5, 0, -1e-5, -1e6, math.Inf(-1), math.Copysign(math.NaN(), -1)}})
	var out bytes.Buffer
	lpStaticMust(t, PrintFloatDebugMatrix(m, &out))
	want := "1.23457 -0  1e-05  1e+06  inf  nan \n  -12.5  0 -1e-05 -1e+06 -inf -nan \n\n"
	if out.String() != want {
		t.Fatal(out.String(), want)
	}
}
func TestLPDebugMatrixPrintIndependentColumnWorlds(t *testing.T) {
	r := rand.New(rand.NewSource(191))
	for world := 0; world < 100; world++ {
		rows, cols := 1+r.Intn(7), 1+r.Intn(5)
		m, err := NewFloatDenseMatrix(uint32(rows), uint32(cols))
		lpStaticMust(t, err)
		var want strings.Builder
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				limit := int(math.Pow10(j + 1))
				value := limit
				if i != 0 {
					value = r.Intn(limit)
				}
				lpStaticMust(t, m.SetElement(uint32(i), uint32(j), float64(value)))
				fmt.Fprintf(&want, "%*d ", j+2, value)
			}
			want.WriteByte('\n')
		}
		want.WriteByte('\n')
		var out bytes.Buffer
		lpStaticMust(t, PrintFloatDebugMatrix(m, &out))
		if out.String() != want.String() {
			t.Fatal(world, out.String(), want.String())
		}
	}
}
func TestLPDebugMatrixPrintConcreteOverloads(t *testing.T) {
	a := NewFloatStaticMatrix(2, 2)
	lpStaticMust(t, a.Set(0, 0, 1))
	lpStaticMust(t, a.Set(1, 1, 22))
	sparse, err := NewFloatSparseBasisMatrix(a, []uint32{0, 1})
	lpStaticMust(t, err)
	for _, base := range []bool{false, true} {
		w := &lpMPSDriverActionWriter{}
		if base {
			lpStaticMust(t, PrintFloatDebugMatrix(a, w))
		} else {
			lpStaticMust(t, PrintFloatStaticDebugMatrix(a, w))
		}
		want := "1  0 \n0 22 \n"
		flushes := 2
		if base {
			want += "\n"
			flushes++
		}
		if w.buffer.String() != want || w.flushes != flushes {
			t.Fatal(base, w.buffer.String(), w.flushes)
		}
	}

	var out bytes.Buffer
	lpStaticMust(t, PrintFloatSparseDebugMatrix(sparse, &out))
	if out.String() != "1  0 \n0 22 \n" {
		t.Fatal(out.String())
	}
}
func TestLPDebugMatrixPrintEveryByteAndFlush(t *testing.T) {
	m := lpDebugDense(t, [][]float64{{1, 200}, {-33, 4}})
	for _, base := range []bool{false, true} {
		run := func(out interface{ Write([]byte) (int, error) }) error {
			if base {
				return PrintFloatDebugMatrix(m, out)
			}
			return printFloatDebugMatrixRows(m, 2, 2, out)
		}
		w := &lpMPSDriverActionWriter{}
		lpStaticMust(t, run(w))
		want, flushes := w.buffer.String(), w.flushes
		for accepted := 0; accepted <= len(want); accepted++ {
			writer := &lpResidualFailWriter{remaining: accepted}
			err := run(lpResidualWriterOnly{writer: writer})
			if writer.String() != want[:accepted] || accepted < len(want) && err != lpErrResidualWrite || accepted == len(want) && err != nil {
				t.Fatal(base, accepted, err, writer.String())
			}
			short := &lpValidationPrefixWriter{larConstraintPrefixWriter: larConstraintPrefixWriter{remaining: accepted}}
			err = run(short)
			var wantErr error
			if accepted < len(want) {
				wantErr = io.ErrShortWrite
			}
			if err != wantErr || short.output.String() != want[:accepted] || short.flushes != strings.Count(want[:accepted], "\n") {
				t.Fatal("short write", base, accepted, err, short.output.String(), short.flushes)
			}
		}
		for flush := 1; flush <= flushes; flush++ {
			writer := &lpMPSDriverActionWriter{failFlush: flush}
			err := run(writer)
			if err != lpErrResidualWrite || writer.flushes != flush || !strings.HasPrefix(want, writer.buffer.String()) {
				t.Fatal(base, flush, err)
			}
		}
	}
	// Padding streams one byte at a time; huge positive counts can stop at an
	// actual writer refusal without allocating a host string of that size.
	w := &lpResidualFailWriter{remaining: 5}
	if err := debugPrintBlanks(math.MaxInt32, lpResidualWriterOnly{writer: w}); err != lpErrResidualWrite || w.String() != "     " {
		t.Fatal(err, w.String())
	}
}
func TestLPDebugMatrixPrintMalformedAndGetterPrefixes(t *testing.T) {
	var out bytes.Buffer
	if width, err := DebugMatrixColumnWidth(0, [][]string{{"long"}, {}}); err != ErrBounds || width != 4 {
		t.Fatal(width, err)
	}
	if width, err := DebugMatrixColumnWidth(math.MaxUint32, nil); err != nil || width != 0 {
		t.Fatal(width, err)
	}
	if err := PrintDebugStringMatrix([][]string{{"a", "b"}, {"c"}}, &out); err != ErrBounds || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	if err := PrintDebugStringMatrix([][]string{{"a"}, {"b", "c"}}, &out); err != ErrBounds || out.String() != "a \nb " {
		t.Fatal(err, out.String())
	}
	out.Reset()
	for _, width := range []uint32{0, (1 << 31) + 2, math.MaxUint32} {
		if err := PrintDebugMatrixWithWidths([][]string{{"xx"}}, []uint32{width}, &out); err != ErrBounds || out.Len() != 0 {
			t.Fatal(width, err, out.String())
		}
	}
	if debugPrintBlanks(-1, &out) != ErrBounds || debugPrintBlanks(0, nil) != ErrBounds {
		t.Fatal("invalid blanks")
	}
	if PrintDebugMatrixWithWidths(nil, nil, nil) != ErrBounds {
		t.Fatal("nil output")
	}
	broken := lpDebugDense(t, [][]float64{{1, 2}, {3, 4}})
	broken.values = broken.values[:3]
	if err := PrintFloatDebugMatrix(broken, &out); err != ErrBounds || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	if err := printFloatDebugMatrixRows(broken, 2, 2, &out); err != ErrBounds || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	if PrintFloatDebugMatrix(nil, &out) != ErrBounds || PrintFloatStaticDebugMatrix(nil, &out) != ErrBounds || PrintFloatSparseDebugMatrix(nil, &out) != ErrBounds {
		t.Fatal("nil matrix")
	}
	if _, err := floatDebugMatrixStrings(nil, 0, 0); err != ErrBounds {
		t.Fatal(err)
	}
}
func TestLPDebugMatrixPrintLiveStringsAndNumericSnapshots(t *testing.T) {
	rows := [][]string{{"x"}}
	w := &lpMPSDriverActionWriter{}
	changed := false
	w.action = func(text string) error {
		if text == " " && !changed {
			rows[0][0] = "changed"
			changed = true
		}
		return nil
	}
	lpStaticMust(t, PrintDebugMatrixWithWidths(rows, []uint32{3}, w))
	if w.buffer.String() != "  changed \n" {
		t.Fatal(w.buffer.String())
	}
	rows = [][]string{{"bbb"}, {"x"}}
	w = &lpMPSDriverActionWriter{}
	w.action = func(text string) error {
		if text == "\n" {
			rows[1][0] = "yy"
		}
		return nil
	}
	lpStaticMust(t, PrintDebugStringMatrix(rows, w))
	if w.buffer.String() != "bbb \n yy \n\n" {
		t.Fatal(w.buffer.String())
	}
	m := lpDebugDense(t, [][]float64{{1, 22}})
	w = &lpMPSDriverActionWriter{}
	w.action = func(text string) error { return m.SetElement(0, 1, 999) }
	lpStaticMust(t, PrintFloatDebugMatrix(m, w))
	if w.buffer.String() != "1 22 \n\n" {
		t.Fatal(w.buffer.String())
	}
	// A live cell's row can disappear while preceding blanks are written.
	rows = [][]string{{"x"}}
	w = &lpMPSDriverActionWriter{}
	w.action = func(text string) error { rows[0] = nil; return nil }
	if err := PrintDebugMatrixWithWidths(rows, []uint32{2}, w); err != ErrBounds || w.buffer.String() != " " {
		t.Fatal(err, w.buffer.String())
	}
}
