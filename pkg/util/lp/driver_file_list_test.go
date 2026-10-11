package lp

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lpDriverListFile(t *testing.T, text string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "models.list")
	if err := os.WriteFile(name, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestLPDriverFileListLiteralLines(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"", nil}, {"\nignored\n", nil},
		{"one", []string{"one"}}, {"one\n", []string{"one"}},
		{"one\ntwo", []string{"one", "two"}},
		{"one\n\nignored", []string{"one"}},
		{" a \n\t\n\r\n./x\n./x\n", []string{" a ", "\t", "\r", "./x", "./x"}},
		{"a\r\nb\x00c\n", []string{"a\r", "b\x00c"}},
		{strings.Repeat("x", 100000) + "\nend", []string{strings.Repeat("x", 100000), "end"}},
	} {
		var out bytes.Buffer
		got, err := ReadDriverFileList(lpDriverListFile(t, tc.text), &out)
		if err != nil || !reflect.DeepEqual(got, tc.want) || out.Len() != 0 {
			t.Fatalf("got %q, %v, report %q; want %q", got, err, out.String(), tc.want)
		}
	}
}

type lpDriverListReadFailure struct {
	text string
}

func (r *lpDriverListReadFailure) Read(p []byte) (int, error) {
	n := copy(p, r.text)
	r.text = r.text[n:]
	return n, lpErrResidualWrite
}

func TestLPDriverFileListReadAndReportFailures(t *testing.T) {
	for _, text := range []string{"", "unfinished", "ok\nunfinished", "ok\n\nunfinished"} {
		got, err := readDriverFileList(&lpDriverListReadFailure{text: text})
		var want []string
		if strings.HasPrefix(text, "ok\n") {
			want = []string{"ok"}
		}
		wantErr := lpErrResidualWrite
		if strings.Contains(text, "\n\n") {
			wantErr = nil
		}
		if !reflect.DeepEqual(got, want) || err != wantErr {
			t.Fatal(text, got, err)
		}
	}
	name := filepath.Join(t.TempDir(), "absent")
	want := "cannot open " + name + "\n"
	for accepted := 0; accepted <= len(want); accepted++ {
		w := &lpResidualFailWriter{remaining: accepted}
		got, err := ReadDriverFileList(name, lpResidualWriterOnly{writer: w})
		if len(got) != 0 || w.String() != want[:accepted] || (accepted < len(want) && err != lpErrResidualWrite) || (accepted == len(want) && err != nil) {
			t.Fatal(accepted, got, err, w.String())
		}
	}
	w := &lpMPSDriverActionWriter{failFlush: 1}
	if _, err := ReadDriverFileList(name, w); err != lpErrResidualWrite || w.buffer.String() != want || w.flushes != 1 {
		t.Fatal(err, w.buffer.String(), w.flushes)
	}
	// Opening a native directory succeeds; reading it must report the real I/O
	// error rather than treating it as a successfully empty file list.
	if _, err := ReadDriverFileList(t.TempDir(), io.Discard); err == nil {
		t.Fatal("directory read succeeded")
	}
}

func TestLPDriverFileListActualSolvesAndLifetime(t *testing.T) {
	for _, exact := range []bool{false, true} {
		h := newLPPairHarness(t)
		first := lpMPSDriverFile(t, lpMPSDriverModel(3, 2, 2, 4, 3))
		second := lpMPSDriverFile(t, lpMPSDriverModel(5, 4, 1, 4, 2))
		// Duplicate input is retained; the blank line stops before a missing file.
		list := lpDriverListFile(t, first+"\n"+second+"\n"+first+"\n\nmissing\n")
		var out bytes.Buffer
		clock := &lpMPSDriverClock{}
		d := NewMPSFileListDriver(list, lpMPSDriverArguments(t, exact, false, false, false), h.m, MPSDriverConfig{Output: &out, Clock: clock.read})
		h.must(d.Run())
		want := []string{first, second, first}
		if !reflect.DeepEqual(d.Files, want) || d.Completed != 3 || d.Current != nil || !d.closed || strings.Count(out.String(), "Status: OPTIMAL\n") != 3 || strings.Count(out.String(), "cost = 8\n") != 2 || strings.Count(out.String(), "cost = 9\n") != 1 {
			t.Fatal(exact, d.Files, d.Completed, out.String())
		}
		wantClock := 6
		if exact {
			wantClock = 9
		}
		if clock.calls != wantClock || d.Run() != ErrExactVectorPending {
			t.Fatal(clock.calls)
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPDriverFileListFailedChildCleanup(t *testing.T) {
	for _, exact := range []bool{false, true} {
		h := newLPPairHarness(t)
		model := lpMPSDriverFile(t, lpMPSDriverModel(3, 2, 2, 4, 3))
		list := lpDriverListFile(t, model+"\n"+model+"\n"+model)
		// Fail the second solver's first clock read, after full file population
		// and setup, so the parent must retain an actual partial owning child.
		failAt := 3
		if exact {
			failAt = 4
		}
		clock := &lpMPSDriverClock{failAt: failAt}
		d := NewMPSFileListDriver(list, lpMPSDriverArguments(t, exact, false, false, false), h.m, MPSDriverConfig{Output: io.Discard, Clock: clock.read})
		if err := d.Run(); !errors.Is(err, lpErrMPSDriverClock) || d.Completed != 1 || d.Current == nil || d.closed || d.Current.closed {
			t.Fatal(exact, err, d.Completed, d.Current)
		}
		if d.Run() != ErrExactVectorPending {
			t.Fatal("failed list retried")
		}
		h.must(d.Close())
		h.must(d.Close())
		if d.Current != nil || clock.calls != failAt {
			t.Fatal(d.Current, clock.calls)
		}
		lpExactVectorNamedCensus(t, h, nil)
	}
	var out bytes.Buffer
	d := NewMPSFileListDriver(filepath.Join(t.TempDir(), "missing"), nil, nil, MPSDriverConfig{Output: &out})
	if err := d.Run(); err != nil || d.Completed != 0 || !d.closed || !strings.HasPrefix(out.String(), "cannot open ") {
		t.Fatal(err, out.String())
	}
	d = NewMPSFileListDriver("unused", nil, nil, MPSDriverConfig{})
	if d.config.Output != os.Stdout {
		t.Fatal("missing native default output")
	}
	if err := d.Close(); err != nil || d.Run() != ErrExactVectorPending {
		t.Fatal(err)
	}
}

func TestLPDriverFileListIOFailurePrefixes(t *testing.T) {
	d := NewMPSFileListDriver(t.TempDir(), nil, nil, MPSDriverConfig{Output: io.Discard})
	if err := d.Run(); err == nil || d.Current != nil || d.Completed != 0 || d.closed {
		t.Fatal(err, d.Completed, d.Current)
	}
	lpStaticMust(t, d.Close())
	for _, exact := range []bool{false, true} {
		h := newLPPairHarness(t)
		model := lpMPSDriverFile(t, lpMPSDriverModel(3, 2, 2, 4, 3))
		list := lpDriverListFile(t, model+"\n"+model)
		var d *MPSFileListDriver
		calls := 0
		clock := func() (uint64, error) {
			calls++
			var err error
			if exact {
				err = d.Current.rationalReader.file.Close()
			} else {
				err = d.Current.floatReader.file.Close()
			}
			h.must(err)
			return 0, lpErrMPSDriverClock
		}
		d = NewMPSFileListDriver(list, lpMPSDriverArguments(t, exact, false, false, false), h.m, MPSDriverConfig{Output: io.Discard, Clock: clock})
		if err := d.Run(); err != lpErrMPSDriverClock || d.Completed != 0 || d.Current == nil {
			t.Fatal(exact, err, d.Completed)
		}
		child := d.Current
		if err := d.Close(); !errors.Is(err, os.ErrClosed) || d.Current != child || d.closed || child.closed {
			t.Fatal(exact, err, d.Current, d.closed)
		}
		h.must(d.Close())
		if calls != 1 || d.Current != nil || !d.closed {
			t.Fatal(calls, d.Current)
		}
		lpExactVectorNamedCensus(t, h, nil)
	}
}
