//go:build g3debug

package lp

import (
	"errors"
	"io"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExecutionDebugBranchesAndSourceExit(t *testing.T) {
	for _, flag := range []string{"--test_lu", "--test_small_lu", "--test_larger_lu", "--test_larger_lu_with_holes", "--test_swaps", "--lar"} {
		h := newLPPairHarness(t)
		r := z3rand.New(23)
		d := NewLPTestDriver(h.m, r, MPSDriverConfig{Output: io.Discard, Clock: lpBenchmarkClock()})
		err := d.Run([]string{flag})
		expected := error(nil)
		if flag == "--test_swaps" {
			expected = ErrAssertion
		}
		if err != expected || d.Finished != (expected == nil) || d.closed != (expected == nil) || (d.Preparation.Settings != nil) != (flag != "--test_swaps") {
			t.Fatal(flag, d, err)
		}
		draws := 0
		switch flag {
		case "--test_lu":
			if d.LU == nil || d.LU.CompletedScenarios != 4 {
				t.Fatal(d)
			}
			draws = 72
		case "--test_small_lu":
			if d.SmallLU == nil || d.SmallLU.CompletedPivots != 4 {
				t.Fatal(d)
			}
		case "--test_larger_lu":
			if d.LargerLU == nil || d.LargerLU.CompletedPivots != 1 {
				t.Fatal(d)
			}
		case "--test_larger_lu_with_holes":
			if d.HolesLU == nil || d.HolesLU.CompletedPivots != 1 {
				t.Fatal(d)
			}
		case "--test_swaps":
			if d.Swaps == nil || d.Swaps.Phase != 2 {
				t.Fatal(d)
			}
			draws = 80
		case "--lar":
			if d.LARInput == nil || d.LARInput.Completed != 0 {
				t.Fatal(d)
			}
		}
		if flag != "--test_swaps" && r.Next() != lpScenarioRandomNext(23, draws) {
			t.Fatal("debug RNG", flag)
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	// A real source mismatch propagates the child's exit(1) without killing
	// the embedding Go process, retaining the failed owner for cleanup.
	for _, flag := range []string{"--test_lu", "--test_larger_lu"} {
		var d *LPTestDriver
		w := &lpMPSDriverActionWriter{action: func(text string) error {
			if strings.HasPrefix(text, " test_larger_lu") {
				d.Preparation.Settings.DropTolerance = .1
			}
			return nil
		}}
		d = NewLPTestDriver(nil, z3rand.New(0), MPSDriverConfig{Output: w})
		if err := d.Run([]string{flag}); err != ErrAssertion || d.ExitCode != 1 || d.Finished || d.closed {
			t.Fatal(flag, d, err, w.buffer.String())
		}
		if flag == "--test_lu" && (d.LU.CompletedScenarios != 1 || d.LU.Holes != nil) {
			t.Fatal("parent source exit prefix", d.LU)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLPExecutionDebugLARActualFile(t *testing.T) {
	h := newLPPairHarness(t)
	name := lpMPSDriverFile(t, lpLARFreeMPS(3))
	d := NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: io.Discard, Clock: lpBenchmarkClock()})
	h.must(d.Run([]string{"--lar", "--file", name, "--rep_frq", "0"}))
	if !d.Finished || d.LARInput == nil || d.LARInput.Completed != 1 || d.File != nil || len(d.LARInput.Files) != 1 || d.LARInput.Files[0] != name {
		t.Fatal(d)
	}
	h.must(d.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExecutionDebugReportFailureRetainsCurrentChild(t *testing.T) {
	for _, flag := range []string{"--test_small_lu", "--test_larger_lu", "--test_larger_lu_with_holes", "--test_lu", "--lar"} {
		h := newLPPairHarness(t)
		var d *LPTestDriver
		failure := errors.New("executor debug child report")
		w := &lpMPSDriverActionWriter{action: func(text string) error {
			if d.Preparation != nil {
				return failure
			}
			return nil
		}}
		d = NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: w})
		err := d.Run([]string{flag})
		// --lar has no child report without input, so its preparation report
		// succeeds and the empty input driver returns normally.
		if flag == "--lar" {
			h.must(err)
			if d.LARInput == nil || !d.Finished {
				t.Fatal(d)
			}
		} else {
			if !errors.Is(err, failure) || d.Finished || d.closed || d.Preparation.Action == "" {
				t.Fatal(flag, d, err)
			}
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}
