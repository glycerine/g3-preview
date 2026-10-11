package lp

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func lpScenarioRandomNext(seed uint32, draws int) uint32 {
	for i := 0; i <= draws; i++ {
		seed = seed*214013 + 2531011
	}
	return (seed >> 16) & 0x7fff
}

func TestLPContainerDriverLiteralReports(t *testing.T) {
	var out bytes.Buffer
	if err := RunDriverIntSetTest(&out); err != nil || out.String() != "2 \n2 1 \n1 \n" {
		t.Fatal(err, out.String())
	}
	out.Reset()
	r := z3rand.New(0)
	if err := RunDriverPriorityQueueTest(&out, r); err != nil {
		t.Fatal(err)
	}
	want := "testing binary_heap_priority_queue...0\n1\n2\n3\n4\n5\n6\n7\n8\n9\n"
	if sourceDebugEnabled {
		want += "0\n1\n2\n3\n4\n5\n6\n7\n8\n9\n"
	}
	want += " done\n"
	if out.String() != want || r.Next() != lpScenarioRandomNext(0, 300) {
		t.Fatal(out.String(), want)
	}
}

func TestLPContainerDriverReportFailurePrefixes(t *testing.T) {
	for _, priority := range []bool{false, true} {
		var full bytes.Buffer
		if priority {
			if err := RunDriverPriorityQueueTest(&full, z3rand.New(73)); err != nil {
				t.Fatal(err)
			}
		} else if err := RunDriverIntSetTest(&full); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < full.Len(); n++ {
			w := &lpResidualFailWriter{remaining: n}
			r := z3rand.New(73)
			var err error
			if priority {
				err = RunDriverPriorityQueueTest(lpResidualWriterOnly{w}, r)
			} else {
				err = RunDriverIntSetTest(lpResidualWriterOnly{w})
			}
			if !errors.Is(err, lpErrResidualWrite) || !bytes.Equal(w.Bytes(), full.Bytes()[:n]) {
				t.Fatal(priority, n, err, w.String())
			}
			draws := 0
			if priority && n >= full.Len()-len(" done\n") {
				draws = 300
			}
			if r.Next() != lpScenarioRandomNext(73, draws) {
				t.Fatal("draw publication", priority, n, draws)
			}
		}
		good := &lpDriverFlushWriter{}
		if priority {
			if err := RunDriverPriorityQueueTest(good, z3rand.New(73)); err != nil {
				t.Fatal(err)
			}
		} else if err := RunDriverIntSetTest(good); err != nil {
			t.Fatal(err)
		}
		wantFlushes := 3
		if priority {
			wantFlushes = 11
			if sourceDebugEnabled {
				wantFlushes = 21
			}
		}
		if good.flushes != wantFlushes {
			t.Fatal(priority, good.flushes, wantFlushes)
		}
		for at := 1; at <= wantFlushes; at++ {
			w := &lpDriverFlushWriter{failAt: at}
			r := z3rand.New(73)
			var err error
			if priority {
				err = RunDriverPriorityQueueTest(w, r)
			} else {
				err = RunDriverIntSetTest(w)
			}
			if !errors.Is(err, lpErrResidualWrite) || w.flushes != at || strings.Count(w.String(), "\n") != at {
				t.Fatal(priority, at, err, w)
			}
			draws := 0
			if priority && at == wantFlushes {
				draws = 300
			}
			if r.Next() != lpScenarioRandomNext(73, draws) {
				t.Fatal(priority, at, "flush draw order")
			}
		}
	}
}

func TestLPContainerDriverSharedRandomAndGuards(t *testing.T) {
	if err := RunDriverPairQueueTest(nil); err != ErrBounds {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RunDriverPriorityQueueTest(&out, nil); err != ErrBounds || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	for seed := uint32(0); seed < 100; seed++ {
		r := z3rand.New(seed)
		if err := RunDriverPairQueueTest(r); err != nil {
			t.Fatal(seed, err)
		}
		if err := RunDriverPriorityQueueTest(io.Discard, r); err != nil {
			t.Fatal(seed, err)
		}
		if r.Next() != lpScenarioRandomNext(seed, 600) {
			t.Fatal(seed, "shared random reset")
		}
	}
}

func TestLPIntSetPrintActualEndl(t *testing.T) {
	s, err := NewIntSet(4)
	if err != nil {
		t.Fatal(err)
	}
	for _, populated := range []bool{false, true} {
		if populated {
			if err = s.Insert(3); err != nil {
				t.Fatal(err)
			}
		}
		w := &lpDriverFlushWriter{failAt: 1}
		if err = s.Print(w); err != lpErrResidualWrite || w.flushes != 1 || !strings.HasSuffix(w.String(), "\n") {
			t.Fatal(err, w)
		}
		w = &lpDriverFlushWriter{}
		if err = s.Print(w); err != nil || w.flushes != 1 {
			t.Fatal(err, w)
		}
	}
}
