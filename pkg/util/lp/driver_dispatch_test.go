package lp

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLPDriverPreparationSourcePrecedence(t *testing.T) {
	branches := []struct {
		args   []string
		action string
		debug  bool
	}{
		{[]string{"--test_mpq"}, "test_rationals", false},
		{[]string{"--test_mpq_np"}, "test_rationals_no_numeric_pairs", false},
		{[]string{"--test_mpq_np_plus"}, "test_rationals_no_numeric_pairs_plus", false},
		{[]string{"--test_int_set"}, "test_int_set", false},
		{[]string{"--bp"}, "test_bound_propagation", false},
		{[]string{"--checklu", "not-opened"}, "check_lu_from_file", false},
		{[]string{"--test_swaps"}, "test_swaps", true},
		{[]string{"--test_perm"}, "test_permutations", false},
		{[]string{"--test_file_directory", ""}, "test_files_from_directory", false},
		{[]string{"--filelist", "not-opened"}, "filelist", false},
		{[]string{"-tbq"}, "test_binary_priority_queue", false},
		{[]string{"--test_lu"}, "test_lu", true},
		{[]string{"--test_small_lu"}, "test_small_lu", true},
		{[]string{"--lar"}, "test_lar_solver", true},
		{[]string{"--test_larger_lu"}, "test_larger_lu", true},
		{[]string{"--test_larger_lu_with_holes"}, "test_larger_lu_with_holes", true},
		{[]string{"--eti"}, "test_evidence_for_total_inf_simple", false},
		{[]string{"--test_lp_0"}, "test_lp_0", false},
		{[]string{"--smap"}, "test_stacked", false},
		{[]string{"--term"}, "test_term", false},
		{[]string{"--file", "not-opened"}, "solve_mps", false},
		{[]string{"--solve_some_mps"}, "solve_some_mps", false},
	}
	for i, first := range branches {
		if first.debug && !sourceDebugEnabled {
			continue
		}
		for j := i; j < len(branches); j++ {
			second := branches[j]
			// Reverse argv order: precedence belongs to source control flow.
			args := append(append([]string{}, second.args...), first.args...)
			p, err := PrepareLPDriver(args, io.Discard)
			if err != nil || p.Action != first.action || p.ExitCode != 0 {
				t.Fatal(i, j, p, err)
			}
		}
	}
}

func TestLPDriverPreparationEmptyOptionsAndDebugStage(t *testing.T) {
	for _, args := range [][]string{nil, {"-xyz_sample"}, {"--checklu", "", "--filelist", "", "--file", ""}} {
		var out bytes.Buffer
		p, err := PrepareLPDriver(args, &out)
		if err != nil || p.Action != "" || !strings.HasPrefix(out.String(), "resize\n") || (p.Settings != nil) != sourceDebugEnabled {
			t.Fatal(p, err, out.String())
		}
	}
	for _, flag := range []string{"--test_swaps", "--test_lu", "--test_small_lu", "--lar", "--test_larger_lu", "--test_larger_lu_with_holes"} {
		p, err := PrepareLPDriver([]string{flag}, io.Discard)
		if err != nil || (p.Action != "") != sourceDebugEnabled {
			t.Fatal(flag, p, err)
		}
	}
	// Early branches bypass all numeric parsing. Later debug settings precede
	// even --term; native source erases this settings block altogether.
	p, err := PrepareLPDriver([]string{"--checklu", "x", "--density", "bad", "--max_iters", "bad"}, io.Discard)
	if err != nil || p.Action != "check_lu_from_file" || p.Settings != nil {
		t.Fatal(p, err)
	}
	p, err = PrepareLPDriver([]string{"--term", "--density", "bad"}, io.Discard)
	if sourceDebugEnabled {
		if err == nil || p.Action != "" {
			t.Fatal(p, err)
		}
	} else if err != nil || p.Action != "test_term" {
		t.Fatal(p, err)
	}
	p, err = PrepareLPDriver([]string{"--file", "", "--solve_some_mps", "--max_iters", "bad"}, io.Discard)
	if err == nil || p.Action != "" {
		t.Fatal(p, err)
	}
}

func TestLPDriverPreparationSolveOptionsAndParseFailure(t *testing.T) {
	args := []string{"--file", "literal/../input", "--solve_some_mps", "--max_iters", "-1", "--time_limit", "23", "--mpq", "--dual", "--min", "--compare_with_primal"}
	p, err := PrepareLPDriver(args, io.Discard)
	if err != nil || p.Action != "solve_mps" || p.File != "literal/../input" || p.MaxIterations != ^uint32(0) || p.TimeLimit != 23 || !p.Exact || !p.Dual || !p.Minimum || !p.CompareWithPrimal {
		t.Fatal(p, err)
	}
	args[1] = "changed"
	if p.Arguments.OptionValue("--file") != p.File {
		t.Fatal("borrowed argv")
	}
	for _, args := range [][]string{{"--unknown"}, {"--file"}} {
		a := NewDriverArguments(args)
		a.Parse()
		var out bytes.Buffer
		p, err = PrepareLPDriver(args, &out)
		want := "resize\n" + a.ErrorMessage + "\n" + a.UsageString()
		if err != nil || p.ExitCode != 1 || p.Action != "" || out.String() != want {
			t.Fatal(p, err, out.String(), want)
		}
	}
	p, err = PrepareLPDriver([]string{"--time_limit", "bad"}, io.Discard)
	if err == nil || p.Action != "" {
		t.Fatal(p, err)
	}
}

func TestLPDriverPreparationReportFailurePrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	worlds := [][]string{nil, {"--unknown"}, {"--file"}, {"--lar"}, {"--density", "25", "--partial_pivot", "40", "--term"}}
	for _, args := range worlds {
		var full bytes.Buffer
		if _, err := PrepareLPDriver(args, &full); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < full.Len(); n++ {
			w := &lpResidualFailWriter{remaining: n}
			p, err := PrepareLPDriver(args, lpResidualWriterOnly{w})
			if !errors.Is(err, lpErrResidualWrite) || !bytes.Equal(w.Bytes(), full.Bytes()[:n]) || p.Action != "" {
				t.Fatal(args, n, p, err, w.String())
			}
		}
		// Every endl failure stops before publishing the selected call.
		good := &lpDriverFlushWriter{}
		if _, err := PrepareLPDriver(args, good); err != nil {
			t.Fatal(err)
		}
		for n := 1; n <= good.flushes; n++ {
			w := &lpDriverFlushWriter{failAt: n}
			p, err := PrepareLPDriver(args, w)
			if !errors.Is(err, lpErrResidualWrite) || p.Action != "" || w.flushes != n {
				t.Fatal(args, n, p, err)
			}
		}
	}
}

func TestLPDriverPreparationShortWritePublication(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, args := range [][]string{{"--test_int_set"}, {"--unknown"}, {"--file"}} {
		var full bytes.Buffer
		good, err := PrepareLPDriver(args, &full)
		if err != nil {
			t.Fatal(err)
		}
		for n := 0; n < full.Len(); n++ {
			w := &larConstraintPrefixWriter{remaining: n}
			p, err := PrepareLPDriver(args, w)
			if !errors.Is(err, io.ErrShortWrite) || w.output.String() != full.String()[:n] {
				t.Fatal(args, n, err, w.output.String())
			}
			// Header failure precedes argument construction. Later refusal
			// retains the parsed arguments, but cannot publish action or exit.
			if (p.Arguments == nil) != (n < len("resize\n")) || p.Action != "" || p.ExitCode != 0 || p.Settings != nil {
				t.Fatal(args, n, p)
			}
			if p.Arguments != nil && p.Arguments.ErrorMessage != good.Arguments.ErrorMessage {
				t.Fatal(args, n, "lost parser error")
			}
		}
		w := &larConstraintPrefixWriter{remaining: full.Len()}
		p, err := PrepareLPDriver(args, w)
		if err != nil || w.output.String() != full.String() || p.Action != good.Action || p.ExitCode != good.ExitCode {
			t.Fatal(args, p, err)
		}
	}
}
