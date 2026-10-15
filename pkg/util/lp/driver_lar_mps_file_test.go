package lp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lpLARFreeMPS(count int) string {
	text := "NAME LARFREE\nROWS\n N COST\nCOLUMNS\n"
	for j := 0; j < count; j++ {
		text += lpExactMPSPair(fmt.Sprintf("x%d", j), "COST", fmt.Sprint(j+1))
	}
	text += "RHS\nBOUNDS\n"
	for j := 0; j < count; j++ {
		text += lpMPSBoundLine("FR", fmt.Sprintf("x%d", j))
	}
	return text + "ENDATA\n"
}

func TestLPLARMPSFileIndependentFreeModels(t *testing.T) {
	for count := 1; count <= 40; count++ {
		h := newLPPairHarness(t)
		var out bytes.Buffer
		name := lpMPSDriverFile(t, lpLARFreeMPS(count))
		var d *LARFileDriver
		clock := &lpMPSDriverClock{}
		read := func() (uint64, error) {
			if d.creator != nil || d.reader == nil || d.reader.file == nil || d.Solver == nil || len(d.Solver.GetListOfAllVarIndices()) != count {
				t.Fatal("creation reader outlived successful population or missing second reader", count)
			}
			return clock.read()
		}
		d = NewLARFileDriver(name, lpDriverArgs(t, "lp-test", "--randomize_lar"), h.m, MPSDriverConfig{Output: &out, Clock: read})
		h.must(d.Run())
		if !d.closed || d.Status != StatusOptimal || d.Iterations != 1 || clock.calls != 3 || d.Solver != nil || d.Runner != nil || d.reader != nil || out.String() != "status is OPTIMAL, processed for 2 seconds, and 1 iterations\nchecking randomize\n" {
			t.Fatal(count, d.Status, d.Iterations, out.String())
		}
		if d.Run() != ErrExactVectorPending {
			t.Fatal("file driver retried")
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPLARMPSInputDispatchAndFileList(t *testing.T) {
	first := lpMPSDriverFile(t, lpLARFreeMPS(2))
	second := lpMPSDriverFile(t, lpLARFreeMPS(3))
	list := lpDriverListFile(t, first+"\n"+second+"\n"+first+"\n\nmissing")
	for _, profile := range []string{"file-priority", "list", "empty-file-falls-back", "empty", "missing-list"} {
		h := newLPPairHarness(t)
		args := []string{"lp-test"}
		var want []string
		switch profile {
		case "file-priority":
			args = append(args, "--file", first, "--filelist", list+".missing")
			want = []string{first}
		case "list":
			args = append(args, "--filelist", list)
			want = []string{first, second, first}
		case "empty-file-falls-back":
			args = append(args, "--file", "", "--filelist", list)
			want = []string{first, second, first}
		case "missing-list":
			args = append(args, "--filelist", list+".missing")
		}
		var out bytes.Buffer
		clock := &lpMPSDriverClock{}
		d := NewLARInputDriver(lpDriverArgs(t, args...), h.m, MPSDriverConfig{Output: &out, Clock: clock.read})
		h.must(d.Run())
		if !reflect.DeepEqual(d.Files, want) || d.Completed != len(want) || d.Current != nil || !d.closed || clock.calls != 3*len(want) || strings.Count(out.String(), "status is OPTIMAL") != len(want) {
			t.Fatal(profile, d.Files, d.Completed, clock.calls, out.String())
		}
		if d.Run() != ErrExactVectorPending {
			t.Fatal("list driver retried")
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPLARMPSFileSourceBoundaries(t *testing.T) {
	for _, profile := range []string{"constraints", "missing", "nil-args", "nil-manager", "closed", "comparison"} {
		h := newLPPairHarness(t)
		input := lpLARFreeMPS(2)
		if profile == "constraints" {
			input = lpMPSDriverModel(3, 2, 2, 4, 3)
		}
		name := lpMPSDriverFile(t, input)
		if profile == "missing" {
			name += ".missing"
		}
		var out bytes.Buffer
		args := lpDriverArgs(t, "lp-test")
		if profile == "comparison" {
			args = lpDriverArgs(t, "lp-test", "--compare_with_primal")
		}
		clock := &lpMPSDriverClock{}
		d := NewLARFileDriver(name, args, h.m, MPSDriverConfig{Output: &out, Clock: clock.read})
		if profile == "nil-args" {
			d.Arguments = nil
		} else if profile == "nil-manager" {
			d.manager = nil
		} else if profile == "closed" {
			h.must(d.Close())
		}
		err := d.Run()
		switch profile {
		case "constraints":
			if sourceDebugEnabled {
				if err != ErrAssertion || d.Solver == nil || d.creator == nil || d.reader != nil {
					t.Fatal(profile, err)
				}
			} else {
				h.must(err)
				if d.Status != StatusOptimal || !d.closed {
					t.Fatal(profile, d.Status)
				}
			}
		case "missing":
			h.must(err)
			if out.String() != "setting m_is_OK to false\ncannot process "+name+"\nsetting m_is_OK to false\n" || clock.calls != 0 || !d.closed {
				t.Fatal(out.String())
			}
		case "nil-args":
			if err != ErrBounds {
				t.Fatal(err)
			}
		case "nil-manager":
			if err != ErrExactVectorOwner {
				t.Fatal(err)
			}
		case "closed":
			if err != ErrExactVectorPending {
				t.Fatal(err)
			}
		case "comparison":
			h.must(err)
			if out.String() != "status = OPTIMAL\n" || clock.calls != 0 || !d.closed {
				t.Fatal(out.String())
			}
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPLARMPSFileFailedSecondChildAndClose(t *testing.T) {
	for _, closeFile := range []bool{false, true} {
		h := newLPPairHarness(t)
		name := lpMPSDriverFile(t, lpLARFreeMPS(2))
		list := lpDriverListFile(t, name+"\n"+name+"\n"+name)
		clock := &lpMPSDriverClock{failAt: 4}
		var d *LARInputDriver
		read := func() (uint64, error) {
			if closeFile && clock.calls == 3 {
				h.must(d.Current.reader.file.Close())
			}
			return clock.read()
		}
		d = NewLARInputDriver(lpDriverArgs(t, "lp-test", "--filelist", list), h.m, MPSDriverConfig{Output: io.Discard, Clock: read})
		if err := d.Run(); err != lpErrMPSDriverClock || d.Completed != 1 || d.Current == nil || d.Current.Runner == nil || d.Current.Solver == nil || d.Current.closed {
			t.Fatal(err, d.Completed, d.Current)
		}
		child := d.Current
		if closeFile {
			if err := d.Close(); !errors.Is(err, os.ErrClosed) || d.Current != child || d.closed {
				t.Fatal(err, d.Current, d.closed)
			}
		}
		h.must(d.Close())
		h.must(d.Close())
		if d.Current != nil || clock.calls != 4 || d.Run() != ErrExactVectorPending {
			t.Fatal(d.Current, clock.calls)
		}
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPLARMPSInputEntryGuards(t *testing.T) {
	for _, profile := range []string{"nil-args", "directory-list", "closed", "default-output"} {
		args := lpDriverArgs(t, "lp-test", "--filelist", t.TempDir())
		d := NewLARInputDriver(args, nil, MPSDriverConfig{})
		switch profile {
		case "nil-args":
			d.Arguments = nil
			if err := d.Run(); err != ErrBounds {
				t.Fatal(err)
			}
		case "directory-list":
			if err := d.Run(); err == nil || d.Current != nil || d.Completed != 0 {
				t.Fatal(err)
			}
		case "closed":
			lpStaticMust(t, d.Close())
			if d.Run() != ErrExactVectorPending {
				t.Fatal("closed list ran")
			}
		case "default-output":
			if d.config.Output != os.Stdout || NewLARFileDriver("", args, nil, MPSDriverConfig{}).config.Output != os.Stdout {
				t.Fatal("missing stdout")
			}
		}
		lpStaticMust(t, d.Close())
	}
	// Missing individual files return normally, so the loop continues.
	h := newLPPairHarness(t)
	missing := filepath.Join(t.TempDir(), "missing.mps")
	d := NewLARInputDriver(lpDriverArgs(t, "lp-test", "--filelist", lpDriverListFile(t, missing+"\n"+missing)), h.m, MPSDriverConfig{Output: io.Discard})
	h.must(d.Run())
	if d.Completed != 2 {
		t.Fatal(d.Completed)
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPLARMPSFileReportFailures(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := lpMPSDriverFile(t, lpLARFreeMPS(2))
		want := "status is OPTIMAL, processed for 2 seconds, and 1 iterations\nchecking randomize\n"
		if missing {
			name += ".missing"
			want = "setting m_is_OK to false\ncannot process " + name + "\nsetting m_is_OK to false\n"
		}
		for accepted := 0; accepted <= len(want); accepted++ {
			h := newLPPairHarness(t)
			w := &lpResidualFailWriter{remaining: accepted}
			d := NewLARFileDriver(name, lpDriverArgs(t, "lp-test", "--randomize_lar"), h.m, MPSDriverConfig{Output: lpResidualWriterOnly{writer: w}, Clock: (&lpMPSDriverClock{}).read})
			err := d.Run()
			if w.String() != want[:accepted] || accepted < len(want) && err != lpErrResidualWrite || accepted == len(want) && err != nil {
				t.Fatal(missing, accepted, err, w.String())
			}
			h.must(d.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPLARMPSFilePhysicalOwnership(t *testing.T) {
	// The reader itself parses this value on every fault ordinal. Twenty
	// blocks still force multi-limb arithmetic and physical allocations.
	large := strings.Repeat("1234567890", 20) + "7/13"
	input := strings.Replace(lpLARFreeMPS(2), lpExactMPSPair("x0", "COST", "1"), lpExactMPSPair("x0", "COST", large), 1)
	name := lpMPSDriverFile(t, input)
	for _, compare := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 6000 && !success; ordinal++ {
				if !t.Run(fmt.Sprintf("compare%t/direct%t/%d", compare, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					flags := []string{"lp-test", "--randomize_lar"}
					if compare {
						flags = append(flags, "--compare_with_primal")
					}
					d := NewLARFileDriver(name, lpDriverArgs(t, flags...), h.m, MPSDriverConfig{Output: io.Discard, Clock: (&lpMPSDriverClock{}).read})
					h.budget.FailAfter(ordinal)
					err := d.Run()
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						if !d.closed || d.Status != StatusOptimal {
							t.Fatal(d.Status, d.closed)
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
					}
					h.must(d.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}) {
					t.FailNow()
				}
			}
			if !success || direct && failures == 0 {
				t.Fatal(compare, direct, success, failures)
			}
		}
	}
}

func TestLPLARMPSFileOlderCallerComparison(t *testing.T) {
	name := lpMPSDriverFile(t, lpLARFreeMPS(2))
	for _, compare := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			flags := []string{"lp-test"}
			if compare {
				flags = append(flags, "--compare_with_primal")
			}
			d := NewLARFileDriver(name, lpDriverArgs(t, flags...), h.m, MPSDriverConfig{Output: io.Discard, Clock: (&lpMPSDriverClock{}).read})
			v := h.pairExactLarge(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_ = d.Run()
				if d.creator != nil && d.creator.calls.comparison != 0 || d.reader != nil && d.reader.calls.comparison != 0 || d.Solver != nil && d.Solver.calls.comparison != 0 {
					t.Fatal("file owner adopted caller comparison")
				}
			}
			h.must(d.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("file cleanup consumed caller receipt")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !matched {
			t.Fatal("no actual older caller comparison", compare)
		}
	}
}
