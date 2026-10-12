//go:build g3debug

package lp

import (
	"bytes"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var lpHolesLULiteral = [8][9]float64{
	{1, 2, 3, 4, 5, 0, 0, 0, 99}, {0, -6, 7, 8, 9, 0, 0, 0, 0},
	{0, 0, 10, 0, 0, 0, 0, 0, 0}, {0, 0, 11, -12, 0, 0, 0, 0, 0},
	{0, 0, 13, 14, 15, 0, 0, 0, 0}, {0, 0, 0, 0, 28, -18, 19, 25, 0},
	{0, 0, 0, 0, 0, 20, -21, 0, 0}, {0, 0, 0, 0, 0, 22, 23, 24, 88},
}

// Expected cells come from literal input or independent tail applications,
// rather than calling the diagnostic Element/printing implementation.
func lpHolesLUFormat(rows [][]float64, blank bool) string {
	texts := make([][]string, len(rows))
	widths := make([]int, len(rows[0]))
	for i, row := range rows {
		for j, v := range row {
			text := strconv.FormatFloat(v, 'g', 6, 64)
			texts[i] = append(texts[i], text)
			if len(text) > widths[j] {
				widths[j] = len(text)
			}
		}
	}
	var out strings.Builder
	for _, row := range texts {
		for j, text := range row {
			out.WriteString(strings.Repeat(" ", widths[j]-len(text)))
			out.WriteString(text + " ")
		}
		out.WriteByte('\n')
	}
	if blank {
		out.WriteByte('\n')
	}
	return out.String()
}

func lpHolesLUInitialReport() string {
	rows := make([][]float64, 8)
	for i := range rows {
		rows[i] = lpHolesLULiteral[i][:]
	}
	return " test_larger_lu_with_holes\n" + lpHolesLUFormat(rows, false)
}

func TestLPHolesLUDriverIndependentTailReportOracle(t *testing.T) {
	for _, density := range []float64{0, .7, math.Inf(1)} {
		s, err := NewSettings()
		lpStaticMust(t, err)
		s.DensityThreshold = density
		expected := lpHolesLUInitialReport() + "printing factorization\n"
		var d *HolesLUTestDriver
		count := 0
		w := &lpMPSDriverActionWriter{action: func(text string) error {
			if text != "printing factorization\n" {
				return nil
			}
			count = len(d.LU.Tail)
			for k := count; k > 0; {
				k--
				cells := make([][]float64, 8)
				for i := range cells {
					cells[i] = make([]float64, 8)
				}
				for j := 0; j < 8; j++ {
					column := make([]float64, 8)
					column[j] = 1
					if err := d.LU.Tail[k].ApplyLeft(column); err != nil {
						return err
					}
					for i, v := range column {
						cells[i][j] = v
						if v == 0 {
							// Sparse diagnostic views return literal +0 for
							// implicit entries, even if applying a negative
							// diagonal produced -0 in this unit vector.
							cells[i][j] = 0
						}
						// Dense dot products add to +0 and can erase a stored
						// negative zero. Source diagnostics print the raw cell.
						if dense, ok := d.LU.Tail[k].(*FloatSquareDenseSubmatrix); ok && v == 0 {
							ri, err := dense.AdjustRow(uint32(i))
							lpStaticMust(t, err)
							rj, err := dense.AdjustColumn(uint32(j))
							lpStaticMust(t, err)
							if ri >= dense.IndexStart && rj >= dense.IndexStart {
								raw := dense.Data[(ri-dense.IndexStart)*dense.Dim+rj-dense.IndexStart]
								if raw == 0 {
									cells[i][j] = raw
								}
							}
						}
					}
				}
				expected += lpHolesLUFormat(cells, true)
			}
			return nil
		}}
		d = NewHolesLUTestDriver(s, w)
		t.Cleanup(func() { lpStaticMust(t, d.Close()) })
		if err := d.Run(); err != nil {
			t.Fatalf("holes source density=%v: %v pivots=%d report=%q", density, err, d.CompletedPivots, w.buffer.String())
		}
		if !d.closed || d.InitialMismatch || d.CompletedPivots != 1 || d.TailCount != uint32(count) || d.PrintedTails != uint32(count) || w.buffer.String() != expected || w.flushes != 10+count*9 {
			t.Fatal("reverse tail reports/source completion", density, count, w.flushes, w.buffer.String(), expected)
		}
		if !reflect.DeepEqual(d.Basis, []uint32{8, 1, 2, 3, 4, 5, 6, 7}) || !reflect.DeepEqual(d.NBasis, []uint32{0}) || !reflect.DeepEqual(d.Heading, []int32{-1, 1, 2, 3, 4, 5, 6, 7, 0}) {
			t.Fatal("source pivot/headings", d.Basis, d.NBasis, d.Heading)
		}
		for i, row := range lpHolesLULiteral {
			for j, want := range row {
				got, err := d.Matrix.Element(uint32(i), uint32(j))
				lpStaticMust(t, err)
				if got != want {
					t.Fatal("literal matrix", i, j, got, want)
				}
			}
		}
		if d.Run() != ErrExactVectorPending {
			t.Fatal("repeat scenario")
		}
	}
}

func TestLPHolesLUDriverDropProfiles(t *testing.T) {
	for _, tolerance := range []float64{.1, 1, 10} {
		s, err := NewSettings()
		lpStaticMust(t, err)
		s.DropTolerance = tolerance
		var out bytes.Buffer
		d := NewHolesLUTestDriver(s, &out)
		err = d.Run()
		if tolerance == .1 {
			if err != nil || !d.closed || d.InitialMismatch || d.CompletedPivots != 1 {
				t.Fatal("small source drop profile", err, d.InitialMismatch, d.CompletedPivots)
			}
		} else if err != ErrAssertion || !d.InitialMismatch || d.CompletedPivots != 1 || d.Workspace == nil || d.closed || !strings.HasSuffix(out.String(), "different sides\n") {
			t.Fatal("mismatch must continue through source update before final assertion", tolerance, err, d.InitialMismatch, d.CompletedPivots)
		}
		lpStaticMust(t, d.Close())
	}
}

func TestLPHolesLUDriverReportFailurePrefixes(t *testing.T) {
	for _, tolerance := range []float64{1e-14, 1} {
		s, err := NewSettings()
		lpStaticMust(t, err)
		s.DropTolerance = tolerance
		full := &lpDriverFlushWriter{}
		d := NewHolesLUTestDriver(s, full)
		err = d.Run()
		if tolerance == 1e-14 && err != nil || tolerance == 1 && err != ErrAssertion {
			t.Fatal("source report fixture", tolerance, err)
		}
		report, flushes := full.String(), full.flushes
		lpStaticMust(t, d.Close())
		tailStart := strings.Index(report, "printing factorization\n") + len("printing factorization\n")
		for at := 0; at < len(report); at++ {
			w := &larConstraintPrefixWriter{remaining: at, err: lpErrResidualWrite}
			d := NewHolesLUTestDriver(s, w)
			if err := d.Run(); err != lpErrResidualWrite || w.output.String() != report[:at] || d.CompletedPivots != 0 || d.closed {
				t.Fatal("report prefix before update", tolerance, at, err, d.CompletedPivots)
			}
			if at >= tailStart && d.PrintedTails != uint32(strings.Count(report[:at], "\n\n")) {
				t.Fatal("completed reverse tail report prefix", tolerance, at, d.PrintedTails)
			}
			lpStaticMust(t, d.Close())
			lpStaticMust(t, d.Close())
		}
		for at := 1; at <= flushes; at++ {
			w := &lpDriverFlushWriter{failAt: at}
			d := NewHolesLUTestDriver(s, w)
			if err := d.Run(); err != lpErrResidualWrite || w.flushes != at || !strings.HasPrefix(report, w.String()) || d.CompletedPivots != 0 {
				t.Fatal("endl prefix before update", tolerance, at, err, w.flushes)
			}
			lpStaticMust(t, d.Close())
		}
	}
	w := &larConstraintPrefixWriter{remaining: 3}
	d := NewHolesLUTestDriver(nil, w)
	if err := d.Run(); err != io.ErrShortWrite || d.Matrix != nil {
		t.Fatal("short initial report", err)
	}
	lpStaticMust(t, d.Close())
	d = NewHolesLUTestDriver(nil, io.Discard)
	if err := d.Run(); err != ErrBounds || d.Matrix == nil || d.LU != nil {
		t.Fatal("nil settings", err)
	}
	lpStaticMust(t, d.Close())
}
