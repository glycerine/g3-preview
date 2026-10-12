//go:build g3debug

package lp

import (
	"io"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func lpLUParentReport(t *testing.T, seed uint32) (string, int, []int) {
	t.Helper()
	s, err := NewSettings()
	lpStaticMust(t, err)
	var r z3rand.Generator
	r.SetSeed(seed)
	w := &lpDriverFlushWriter{}
	ends := make([]int, 0, 4)
	a := NewSmallLUTestDriver(s, w)
	lpStaticMust(t, a.Run())
	ends = append(ends, w.Len())
	b := NewLargerLUTestDriver(s, w, false)
	lpStaticMust(t, b.Run(&r))
	ends = append(ends, w.Len())
	c := NewHolesLUTestDriver(s, w)
	lpStaticMust(t, c.Run())
	ends = append(ends, w.Len())
	d := NewLargerLUTestDriver(s, w, true)
	lpStaticMust(t, d.Run(&r))
	ends = append(ends, w.Len())
	return w.String(), w.flushes, ends
}

func TestLPLUTestDriverSourceParentSequence(t *testing.T) {
	for seed := uint32(0); seed < 20; seed++ {
		expected, flushes, _ := lpLUParentReport(t, seed)
		s, err := NewSettings()
		lpStaticMust(t, err)
		var r z3rand.Generator
		r.SetSeed(seed)
		var d *LUTestDriver
		boundaries := 0
		w := &lpMPSDriverActionWriter{action: func(text string) error {
			switch text {
			case " test_larger_lu\n":
				if d.CompletedScenarios != 1 || !d.Small.closed {
					t.Fatal("small child not closed before larger scenario")
				}
				boundaries++
			case " test_larger_lu_with_holes\n":
				if d.CompletedScenarios != 2 || !d.Larger.closed {
					t.Fatal("larger child not closed before holes scenario")
				}
				boundaries++
			case " test_larger_lu_exp\n":
				if d.CompletedScenarios != 3 || !d.Holes.closed {
					t.Fatal("holes child not closed before random scenario")
				}
				boundaries++
			}
			return nil
		}}
		d = NewLUTestDriver(s, w)
		lpStaticMust(t, d.Run(&r))
		if !d.closed || d.CompletedScenarios != 4 || d.ExitCode != 0 || !d.Randomized.closed || boundaries != 3 || w.buffer.String() != expected || w.flushes != flushes {
			t.Fatal("source parent sequence/report", seed, d.CompletedScenarios, boundaries, w.flushes, flushes)
		}
		state := seed
		for i := 0; i < 73; i++ {
			state = state*214013 + 2531011
		}
		if got := r.Next(); got != (state>>16)&32767 {
			t.Fatal("parent borrowed shared random stream", seed, got, state)
		}
		if d.Run(&r) != ErrExactVectorPending {
			t.Fatal("parent repeated run")
		}
		lpStaticMust(t, d.Close())
	}
}

func TestLPLUTestDriverParentReportFailurePrefixes(t *testing.T) {
	report, flushes, ends := lpLUParentReport(t, 17)
	s, err := NewSettings()
	lpStaticMust(t, err)
	for at := 0; at < len(report); at++ {
		var r z3rand.Generator
		r.SetSeed(17)
		w := &larConstraintPrefixWriter{remaining: at, err: lpErrResidualWrite}
		d := NewLUTestDriver(s, w)
		if err := d.Run(&r); err != lpErrResidualWrite || w.output.String() != report[:at] || d.closed {
			t.Fatal("parent report prefix", at, err)
		}
		want := uint32(0)
		for _, end := range ends {
			if at >= end {
				want++
			}
		}
		if d.CompletedScenarios != want || (d.Larger != nil) != (want >= 1) || (d.Holes != nil) != (want >= 2) || (d.Randomized != nil) != (want >= 3) {
			t.Fatal("parent completed/constructed prefix", at, d.CompletedScenarios, want)
		}
		if want >= 1 && !d.Small.closed || want >= 2 && !d.Larger.closed || want >= 3 && !d.Holes.closed {
			t.Fatal("completed child kept a lexical owner", at)
		}
		state := uint32(17)
		draws := 0
		if at >= ends[2]+len(" test_larger_lu_exp\n") {
			draws = 72
		}
		for i := 0; i <= draws; i++ {
			state = state*214013 + 2531011
		}
		if got := r.Next(); got != (state>>16)&32767 {
			t.Fatal("parent failure random prefix", at, got, state)
		}
		lpStaticMust(t, d.Close())
		lpStaticMust(t, d.Close())
	}
	for at := 1; at <= flushes; at++ {
		var r z3rand.Generator
		r.SetSeed(17)
		w := &lpDriverFlushWriter{failAt: at}
		d := NewLUTestDriver(s, w)
		if err := d.Run(&r); err != lpErrResidualWrite || w.flushes != at || !strings.HasPrefix(report, w.String()) {
			t.Fatal("parent endl failure", at, err, w.flushes)
		}
		lpStaticMust(t, d.Close())
	}
}

func TestLPLUTestDriverStopsOnChildAndExitFailures(t *testing.T) {
	s, err := NewSettings()
	lpStaticMust(t, err)
	d := NewLUTestDriver(s, io.Discard)
	if err := d.Run(nil); err != ErrBounds || d.CompletedScenarios != 3 || d.Randomized == nil || d.Randomized.RandomDraws != 0 || !d.Holes.closed {
		t.Fatal("nil random provider stops last child", err, d.CompletedScenarios)
	}
	lpStaticMust(t, d.Close())
	d = NewLUTestDriver(nil, io.Discard)
	if err := d.Run(nil); err != ErrBounds || d.CompletedScenarios != 0 || d.Larger != nil {
		t.Fatal("first child failure stops parent", err)
	}
	lpStaticMust(t, d.Close())
	w := &lpMPSDriverActionWriter{action: func(text string) error {
		if text == " test_larger_lu\n" {
			s.DropTolerance = .1 // Live borrowed settings, after the small child.
		}
		return nil
	}}
	d = NewLUTestDriver(s, w)
	if err := d.Run(nil); err != ErrAssertion || d.ExitCode != 1 || d.CompletedScenarios != 1 || d.Larger.closed || d.Holes != nil || d.Randomized != nil {
		t.Fatal("child source exit stops parent", err, d.ExitCode, d.CompletedScenarios)
	}
	lpStaticMust(t, d.Close())
}
