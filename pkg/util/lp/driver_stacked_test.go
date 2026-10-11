package lp

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

func lpStackedScenarioReport() string {
	text := "test_stacked_map()\n0\n0\n0\n0\ntest stacked unsigned\nbefore push v=4\nv = 6\n4\ntest_stacked_vector\n"
	if sourceDebugEnabled {
		text += "test_stacked_set\n"
	}
	return text
}

func TestLPDriverStackedIndependentResults(t *testing.T) {
	var d StackedTestDriver
	w := &lpDriverFlushWriter{}
	if err := d.Run(w); err != nil || w.String() != lpStackedScenarioReport() || w.flushes != strings.Count(w.String(), "\n") {
		t.Fatal(err, w)
	}
	if d.Completed != 4 || !reflect.DeepEqual(d.Map.values, map[int32]float64{0: 3, 2: 3}) || len(d.Map.stack) != 0 || d.Value.Value != 4 || d.Value.StackSize() != 0 {
		t.Fatal(d.Completed, d.Map, d.Value)
	}
	if !reflect.DeepEqual(d.Vector.Values(), []int32{0, 1, 2, 3, 7}) || !reflect.DeepEqual(d.Vector.vectorSizes, []uint32{0, 2}) {
		t.Fatal(d.Vector)
	}
	if sourceDebugEnabled {
		if d.Set == nil || !reflect.DeepEqual(d.Set.Values(), map[int32]struct{}{1: {}, 2: {}, 3: {}}) || len(d.Set.stack) != 0 {
			t.Fatal(d.Set)
		}
	} else if d.Set != nil {
		t.Fatal("native source erases set test")
	}
	var out bytes.Buffer
	if err := d.Run(&out); err != ErrExactVectorPending || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	var other StackedTestDriver
	if err := other.Run(io.Discard); err != nil {
		t.Fatal(err)
	}
	d.Map.Set(0, 99)
	d.Value.Value = 99
	if err := d.Vector.Set(0, 99); err != nil {
		t.Fatal(err)
	}
	if other.Map.values[0] != 3 || other.Value.Value != 4 || other.Vector.Values()[0] != 0 {
		t.Fatal("independent scenario owners")
	}
	if d.Set != nil {
		d.Set.Insert(99)
		if _, ok := other.Set.Values()[99]; ok {
			t.Fatal("shared set")
		}
	}
}

func lpCheckStackedScenarioPrefix(t *testing.T, d *StackedTestDriver, line int) {
	t.Helper()
	wantCompleted := uint32(0)
	if line >= 5 {
		wantCompleted = 1
	}
	if line >= 9 {
		wantCompleted = 2
	}
	if line >= 10 {
		wantCompleted = 3
	}
	if d.Completed != wantCompleted || d.Set != nil {
		t.Fatal(line, d.Completed, wantCompleted, d.Set)
	}
	if line < 5 {
		if d.Map.values != nil {
			t.Fatal(line, d.Map)
		}
	} else if !reflect.DeepEqual(d.Map.values, map[int32]float64{0: 3, 2: 3}) {
		t.Fatal(line, d.Map)
	}
	wantValue, wantDepth := uint32(0), uint32(0)
	if line == 6 {
		wantValue = 4
	}
	if line == 7 {
		wantValue, wantDepth = 6, 2
	}
	if line >= 8 {
		wantValue = 4
	}
	if d.Value.Value != wantValue || d.Value.StackSize() != wantDepth {
		t.Fatal(line, d.Value, wantValue, wantDepth)
	}
	if line < 10 {
		if d.Vector.Size() != 0 || len(d.Vector.vectorSizes) != 0 {
			t.Fatal(line, d.Vector)
		}
	} else if !reflect.DeepEqual(d.Vector.Values(), []int32{0, 1, 2, 3, 7}) || !reflect.DeepEqual(d.Vector.vectorSizes, []uint32{0, 2}) {
		t.Fatal(line, d.Vector)
	}
}

func TestLPDriverStackedReportAndPublicationFailures(t *testing.T) {
	message := lpStackedScenarioReport()
	for at := 0; at < len(message); at++ {
		var d StackedTestDriver
		w := &lpResidualFailWriter{remaining: at}
		if err := d.Run(lpResidualWriterOnly{w}); err != lpErrResidualWrite || w.String() != message[:at] {
			t.Fatal(at, err, w.String())
		}
		lpCheckStackedScenarioPrefix(t, &d, strings.Count(message[:at], "\n"))
		if err := d.Run(io.Discard); err != ErrExactVectorPending {
			t.Fatal(at, err)
		}
	}
	for at := 1; at <= strings.Count(message, "\n"); at++ {
		var d StackedTestDriver
		w := &lpDriverFlushWriter{failAt: at}
		if err := d.Run(w); err != lpErrResidualWrite || w.flushes != at || strings.Count(w.String(), "\n") != at {
			t.Fatal(at, err, w)
		}
		lpCheckStackedScenarioPrefix(t, &d, at-1)
	}
}

func TestLPDriverStackedSetEquality(t *testing.T) {
	worlds := []map[int32]struct{}{nil, {}, {1: {}}, {2: {}}, {1: {}, 2: {}}}
	for i, a := range worlds {
		for j, b := range worlds {
			want := i == j || (i < 2 && j < 2)
			if driverStackedSetsEqual(a, b) != want {
				t.Fatal(i, j)
			}
		}
	}
}
