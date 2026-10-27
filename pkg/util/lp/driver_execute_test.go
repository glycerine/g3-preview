package lp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExecutionEntryContainersAndPrecedence(t *testing.T) {
	for _, tc := range []struct {
		args   []string
		action string
		exit   uint32
	}{
		{nil, "", 0}, {[]string{"--unknown"}, "", 1}, {[]string{"--file"}, "", 1},
		{[]string{"--test_int_set", "--file", "absent", "--max_iters", "bad"}, "test_int_set", 0},
		{[]string{"--test_perm", "--term"}, "test_permutations", 0},
		{[]string{"--test_file_directory", "absent"}, "test_files_from_directory", 0},
		{[]string{"-tbq"}, "test_binary_priority_queue", 0},
		{[]string{"--smap", "--term"}, "test_stacked", 0},
	} {
		h := newLPPairHarness(t)
		var out bytes.Buffer
		r := z3rand.New(73)
		d := NewLPTestDriver(h.m, r, MPSDriverConfig{Output: &out})
		entry := append([]string{"ignored executable", "ignored lp selector"}, tc.args...)
		h.must(d.RunTestEntry(entry))
		if !d.Finished || !d.closed || d.ExitCode != tc.exit || d.Preparation.Action != tc.action {
			t.Fatal(tc, d, out.String())
		}
		if strings.Contains(out.String(), "ignored") {
			t.Fatal("entry did not skip exactly two arguments", out.String())
		}
		if tc.action == "test_int_set" && !strings.HasSuffix(out.String(), "2 \n2 1 \n1 \n") {
			t.Fatal(out.String())
		}
		if tc.action == "test_permutations" && (d.Permutations == nil || len(d.Permutations.Products) != 2) {
			t.Fatal("permutation action")
		}
		if tc.action == "test_stacked" && (d.Stacked == nil || d.Stacked.Completed != 4) {
			t.Fatal("stacked action")
		}
		draws := 0
		if tc.action == "test_binary_priority_queue" {
			draws = 300
		}
		if r.Next() != lpScenarioRandomNext(73, draws) {
			t.Fatal("wrong shared RNG draws", tc.action)
		}
		if d.Run(nil) != ErrExactVectorPending || d.RunTestEntry(nil) != ErrExactVectorPending {
			t.Fatal("repeated entry")
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, args := range [][]string{nil, {"one"}} {
		d := NewLPTestDriver(nil, nil, MPSDriverConfig{Output: io.Discard})
		if d.RunTestEntry(args) != ErrBounds || d.Preparation != nil || d.Finished {
			t.Fatal("short entry", d)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLPExecutionActualExamplesAndSourceFailures(t *testing.T) {
	for _, flag := range []string{"--bp", "--eti", "--term", "--test_lp_0"} {
		h := newLPPairHarness(t)
		d := NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: io.Discard})
		err := d.Run([]string{flag})
		expected := error(nil)
		if flag == "--test_lp_0" {
			expected = ErrBounds
		}
		if sourceDebugEnabled {
			expected = ErrAssertion
		}
		if err != expected || d.Finished != (expected == nil) || d.closed != (expected == nil) {
			t.Fatal(flag, d, err, expected)
		}
		switch flag {
		case "--bp":
			if d.BoundPropagation == nil {
				t.Fatal("missing bound driver")
			}
			if expected == nil && len(d.BoundPropagation.Completed) != 7 {
				t.Fatal("missing cases")
			}
		case "--eti", "--term":
			if d.Example == nil {
				t.Fatal("missing example")
			}
			if expected == nil && d.Example.Status != StatusOptimal {
				t.Fatal("example status")
			}
		case "--test_lp_0":
			if d.Zero == nil || d.Zero.Core == nil {
				t.Fatal("failed zero driver not retained")
			}
		}
		h.must(d.Close())
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExecutionMPSIndependentObjectivesAndFileList(t *testing.T) {
	for world := int64(0); world < 8; world++ {
		cx, cy, u, v, rhs := 1+world%7, 1+world%5, 2+world%3, 4+world%4, 3+world%6
		x, y := int64(0), int64(0)
		if cx >= cy {
			x = min(u, rhs)
			y = min(v, rhs-x)
		} else {
			y = min(v, rhs)
			x = min(u, rhs-y)
		}
		name := lpMPSDriverFile(t, lpMPSDriverModel(cx, cy, u, v, rhs))
		for _, exact := range []bool{false, true} {
			for _, minimum := range []bool{false, true} {
				h := newLPPairHarness(t)
				var out bytes.Buffer
				args := []string{"--file", name, "--rep_frq", "0", "--max_iters", "1000", "--time_limit", "0"}
				if exact {
					args = append(args, "--mpq")
				}
				if minimum {
					args = append(args, "--min")
				}
				d := NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: &out, Clock: lpBenchmarkClock()})
				h.must(d.Run(args))
				want := cx*x + cy*y
				if minimum {
					want = 0
				}
				text := fmt.Sprint(want)
				if minimum && !exact {
					text = "-0"
				}
				if d.File == nil || d.File.Status != StatusOptimal || !d.File.closed || !d.Finished || !strings.Contains(out.String(), "cost = "+text+"\n") || d.Preparation.MaxIterations != 1000 {
					t.Fatal(world, exact, minimum, d, out.String())
				}
				h.must(d.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
	h := newLPPairHarness(t)
	name := lpMPSDriverFile(t, lpMPSDriverModel(3, 2, 2, 4, 3))
	list := lpDriverListFile(t, name+"\n"+name+"\n\nabsent\n")
	d := NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: io.Discard, Clock: lpBenchmarkClock()})
	h.must(d.Run([]string{"--filelist", list, "--file", "not opened", "--mpq", "--rep_frq", "0"}))
	if !d.Finished || d.FileList == nil || d.FileList.Completed != 2 || d.File != nil {
		t.Fatal("file-list precedence/completion", d)
	}
	h.must(d.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExecutionLUFileAndNumericBenchmarkSetup(t *testing.T) {
	h := newLPPairHarness(t)
	matrix := make([]float64, 27)
	matrix[0] = 2
	matrix[26] = 6
	name := lpMPSDriverFile(t, lpLUCheckText([][]float64{matrix}, []uint32{0}))
	d := NewLPTestDriver(h.m, nil, MPSDriverConfig{Output: io.Discard})
	h.must(d.Run([]string{"--checklu", name, "--max_iters", "bad"}))
	if d.LUCheck == nil || d.LUCheck.Column.Data[0] != 3 || !d.Finished {
		t.Fatal("actual LU check", d)
	}
	h.must(d.Close())
	lpExactVectorNamedCensus(t, h, nil)
	failure := errors.New("executor benchmark start clock")
	for _, flag := range []string{"--test_mpq", "--test_mpq_np", "--test_mpq_np_plus"} {
		h := newLPPairHarness(t)
		r := z3rand.New(19)
		d := NewLPTestDriver(h.m, r, MPSDriverConfig{Output: io.Discard, Clock: func() (uint64, error) { return 0, failure }})
		if err := d.Run([]string{flag}); err != failure || d.Finished || d.closed {
			t.Fatal(flag, d, err)
		}
		if flag == "--test_mpq" {
			if d.PairBenchmark == nil || d.PairBenchmark.Coefficients != 10 || d.PairBenchmark.Values != 10 || d.PairBenchmark.CompletedProducts != 0 {
				t.Fatal("pair setup", d)
			}
		} else {
			if d.RationalBenchmark == nil || d.RationalBenchmark.Plus != (flag == "--test_mpq_np_plus") || d.RationalBenchmark.Coefficients != 10 || d.RationalBenchmark.Values != 10 || d.RationalBenchmark.CompletedIterations != 0 {
				t.Fatal("scalar setup", d)
			}
		}
		if r.Next() != lpScenarioRandomNext(19, 40) {
			t.Fatal("benchmark shared RNG")
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExecutionEveryReportByteAndFlushPrefix(t *testing.T) {
	for _, args := range [][]string{nil, {"--unknown"}, {"--test_int_set"}, {"--test_perm"}, {"-tbq"}, {"--smap"}} {
		good := &lpDriverFlushWriter{}
		d := NewLPTestDriver(nil, z3rand.New(73), MPSDriverConfig{Output: good})
		if err := d.Run(args); err != nil {
			t.Fatal(args, err)
		}
		var preparation bytes.Buffer
		if _, err := PrepareLPDriver(args, &preparation); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < good.Len(); n++ {
			w := &larConstraintPrefixWriter{remaining: n, err: lpErrResidualWrite}
			d := NewLPTestDriver(nil, z3rand.New(73), MPSDriverConfig{Output: w})
			err := d.Run(args)
			if !errors.Is(err, lpErrResidualWrite) || w.output.String() != good.String()[:n] || d.Finished || d.closed {
				t.Fatal(args, n, err, w.output.String())
			}
			if n < preparation.Len() && (d.Permutations != nil || d.Stacked != nil || d.Preparation.Action != "") {
				t.Fatal("action before preparation completes", args, n, d)
			}
			if err = d.Close(); err != nil {
				t.Fatal(err)
			}
		}
		for n := 1; n <= good.flushes; n++ {
			w := &lpDriverFlushWriter{failAt: n}
			d := NewLPTestDriver(nil, z3rand.New(73), MPSDriverConfig{Output: w})
			if err := d.Run(args); !errors.Is(err, lpErrResidualWrite) || d.Finished || w.flushes != n {
				t.Fatal(args, n, err)
			}
			if err := d.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestLPExecutionMPSPhysicalFailureOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	input := strings.Replace(lpMPSDriverModel(3, 2, 2, 4, 3), lpExactMPSPair("x", "COST", "3"), lpExactMPSPair("x", "COST", strings.Repeat("1234567890", 8)+"7/13"), 1)
	name := lpMPSDriverFile(t, input)
	for _, direct := range []bool{false, true} {
		done, failures := false, 0
		for ordinal := int64(0); ordinal < 1000 && !done; ordinal++ {
			if !t.Run(fmt.Sprintf("direct=%v/refusal=%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				d := NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: io.Discard, Clock: lpBenchmarkClock()})
				h.budget.FailAfter(ordinal)
				err := d.Run([]string{"--file", name, "--mpq", "--rep_frq", "0"})
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
					if !d.Finished || d.File.Status != StatusOptimal {
						t.Fatal(d)
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(err) || d.File == nil || d.Finished || d.closed {
						t.Fatal(err, d)
					}
					failures++
				}
				h.must(d.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}) {
				t.FailNow()
			}
		}
		if !done || direct && failures == 0 {
			t.Fatal("missing executor physical completion", direct, done, failures)
		}
	}
}
