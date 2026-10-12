//go:build g3debug

package lp

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func lpLargerLUOracle(seed uint32, randomized bool) ([6][12]float64, string, uint32) {
	var a [6][12]float64
	state := seed
	for i := range a {
		for j := range a[i] {
			if randomized {
				state = state*214013 + 2531011
				a[i][j] = float64(((state >> 16) & 32767) % 20)
			} else {
				k := (j - i + 12) % 12
				a[i][j] = float64(k * k)
			}
		}
	}
	if randomized {
		report := " test_larger_lu_exp\n"
		for i := range a {
			report += strconv.Itoa(int(a[i][8])) + "\n"
		}
		return a, report, state
	}
	var widths [12]int
	for j := range widths {
		for i := range a {
			if n := len(strconv.Itoa(int(a[i][j]))); n > widths[j] {
				widths[j] = n
			}
		}
	}
	report := " test_larger_lu\n"
	for i := range a {
		for j, v := range a[i] {
			text := strconv.Itoa(int(v))
			report += strings.Repeat(" ", widths[j]-len(text)) + text + " "
		}
		report += "\n"
	}
	return a, report, state
}

func TestLPLargerLUDriversIndependentLiteralAndRandomOracle(t *testing.T) {
	for _, randomized := range []bool{false, true} {
		for world := uint32(0); world < 30; world++ {
			t.Run(fmt.Sprintf("random=%v/seed=%d", randomized, world), func(t *testing.T) {
				s, err := NewSettings()
				lpStaticMust(t, err)
				// Exercise sparse and dense factorization with identical source input.
				s.DensityThreshold = []float64{0, .7, math.Inf(1)}[world%3]
				var r z3rand.Generator
				r.SetSeed(world)
				var out bytes.Buffer
				d := NewLargerLUTestDriver(s, &out, randomized)
				t.Cleanup(func() { lpStaticMust(t, d.Close()) })
				if err := d.Run(&r); err != nil {
					t.Fatalf("source scenario %v; pivots=%d report=%q", err, d.CompletedPivots, out.String())
				}
				a, report, state := lpLargerLUOracle(world, randomized)
				wantBasis, wantCount, wantDraws := []uint32{1, 3, 9, 4, 5, 6}, uint32(1), uint32(0)
				if randomized {
					wantBasis, wantCount, wantDraws = []uint32{1, 8, 11, 4, 5, 6}, 2, 72
				}
				if !d.closed || d.ExitCode != 0 || d.CompletedPivots != wantCount || d.RandomDraws != wantDraws || !reflect.DeepEqual(d.Basis, wantBasis) || out.String() != report {
					t.Fatal("source publication/report", d.Basis, d.CompletedPivots, d.RandomDraws, out.String(), report)
				}
				for i := range a {
					for j, want := range a[i] {
						got, err := d.Matrix.Element(uint32(i), uint32(j))
						lpStaticMust(t, err)
						if got != want {
							t.Fatal("literal matrix", i, j, got, want)
						}
					}
				}
				for i, j := range d.Basis {
					if d.Heading[j] != int32(i) {
						t.Fatal("basis heading", i, j, d.Heading)
					}
				}
				for i, j := range d.NBasis {
					if d.Heading[j] != -int32(i)-1 {
						t.Fatal("nonbasis heading", i, j, d.Heading)
					}
				}
				if !randomized {
					state = world
				}
				state = state*214013 + 2531011
				if got := r.Next(); got != (state>>16)&32767 {
					t.Fatal("borrowed random stream draw count", got, state)
				}
				if d.Run(&r) != ErrExactVectorPending {
					t.Fatal("repeated scenario")
				}
			})
		}
	}
}

func TestLPLargerLUDriversReportFailurePrefixes(t *testing.T) {
	s, err := NewSettings()
	lpStaticMust(t, err)
	for _, randomized := range []bool{false, true} {
		_, report, state := lpLargerLUOracle(17, randomized)
		header := strings.IndexByte(report, '\n') + 1
		for at := 0; at < len(report); at++ {
			var r z3rand.Generator
			r.SetSeed(17)
			w := &larConstraintPrefixWriter{remaining: at, err: lpErrResidualWrite}
			d := NewLargerLUTestDriver(s, w, randomized)
			if err := d.Run(&r); err != lpErrResidualWrite || w.output.String() != report[:at] {
				t.Fatal("report prefix", randomized, at, err, w.output.String())
			}
			if d.CompletedPivots != 0 || d.closed || (d.Matrix != nil) != (at >= header) {
				t.Fatal("report-before-algorithm publication", randomized, at)
			}
			wantState := uint32(17)
			if randomized && at >= header {
				wantState = state
				if d.RandomDraws != 72 || d.LU == nil || d.Left == nil || d.Right == nil {
					t.Fatal("random fill precedes scalar reporting", at)
				}
			}
			wantState = wantState*214013 + 2531011
			if got := r.Next(); got != (wantState>>16)&32767 {
				t.Fatal("failed report advanced random stream", randomized, at, got, wantState)
			}
			lpStaticMust(t, d.Close())
			lpStaticMust(t, d.Close())
		}
		for at := 1; at <= 7; at++ {
			var r z3rand.Generator
			r.SetSeed(17)
			w := &lpDriverFlushWriter{failAt: at}
			d := NewLargerLUTestDriver(s, w, randomized)
			if err := d.Run(&r); err != lpErrResidualWrite || w.flushes != at || !strings.HasPrefix(report, w.String()) {
				t.Fatal("endl refusal", randomized, at, err, w.String())
			}
			lpStaticMust(t, d.Close())
		}
	}
	w := &larConstraintPrefixWriter{remaining: 2}
	d := NewLargerLUTestDriver(s, w, true)
	if err := d.Run(nil); err != io.ErrShortWrite || d.Matrix != nil {
		t.Fatal("short initial report", err)
	}
	lpStaticMust(t, d.Close())
	d = NewLargerLUTestDriver(s, io.Discard, true)
	if err := d.Run(nil); err != ErrBounds || d.RandomDraws != 0 {
		t.Fatal("nil borrowed RNG", err)
	}
	lpStaticMust(t, d.Close())
	d = NewLargerLUTestDriver(nil, io.Discard, false)
	if err := d.Run(nil); err != ErrBounds || d.LU != nil {
		t.Fatal("nil settings", err)
	}
	lpStaticMust(t, d.Close())
}

func TestLPLargerLUDriverDropProfiles(t *testing.T) {
	for _, random := range []bool{false, true} {
		for _, tolerance := range []float64{.1, 1, 10, 100} {
			s, err := NewSettings()
			lpStaticMust(t, err)
			s.DropTolerance = tolerance
			var r z3rand.Generator
			r.SetSeed(17)
			var out bytes.Buffer
			d := NewLargerLUTestDriver(s, &out, random)
			err = d.Run(&r)
			if err != ErrAssertion || d.LU == nil || d.Left == nil || d.Right == nil || d.Workspace != nil || d.CompletedPivots != 0 || d.closed {
				t.Fatal("source initial mismatch and retained lexical owners", random, tolerance, err)
			}
			if random {
				if out.String() != " test_larger_lu_exp\n" || d.ExitCode != 0 || d.RandomDraws != 72 {
					t.Fatal("random initial assertion precedes scalar reports", tolerance, out.String())
				}
			} else {
				_, initial, _ := lpLargerLUOracle(17, false)
				if d.ExitCode != 1 || !strings.HasPrefix(out.String(), initial+"left side\n") || !strings.Contains(out.String(), "\n\nright side\n") || !strings.HasSuffix(out.String(), "\n\ndifferent sides\ninitial factorization is incorrect\n") {
					t.Fatal("source exit diagnostics", tolerance, d.ExitCode, out.String())
				}
			}
			lpStaticMust(t, d.Close())
		}
	}
}

func TestLPLargerLUDriverMismatchDiagnosticFailurePrefixes(t *testing.T) {
	s, err := NewSettings()
	lpStaticMust(t, err)
	s.DropTolerance = .1 // Real elimination drops create the initial mismatch.
	var full bytes.Buffer
	d := NewLargerLUTestDriver(s, &full, false)
	if err := d.Run(nil); err != ErrAssertion || d.ExitCode != 1 {
		t.Fatal("mismatch fixture", err)
	}
	lpStaticMust(t, d.Close())
	report := full.String()
	for at := 0; at < len(report); at++ {
		w := &larConstraintPrefixWriter{remaining: at, err: lpErrResidualWrite}
		d := NewLargerLUTestDriver(s, w, false)
		if err := d.Run(nil); err != lpErrResidualWrite || w.output.String() != report[:at] || d.ExitCode != 0 || d.CompletedPivots != 0 || d.closed {
			t.Fatal("mismatch report prefix before exit", at, err, d.ExitCode)
		}
		lpStaticMust(t, d.Close())
	}
	for at := 1; at <= 25; at++ {
		w := &lpDriverFlushWriter{failAt: at}
		d := NewLargerLUTestDriver(s, w, false)
		if err := d.Run(nil); err != lpErrResidualWrite || w.flushes != at || !strings.HasPrefix(report, w.String()) || d.ExitCode != 0 {
			t.Fatal("mismatch endl prefix before exit", at, err, w.flushes, d.ExitCode)
		}
		lpStaticMust(t, d.Close())
	}
}
