package sat

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATIntegrityMissingBinaryReciprocalSourceTrace(t *testing.T) {
	g, _ := satGCFixture(t, 2)
	l := g.lifecycle()
	watch, err := newSATBinaryWatch(2, false)
	satOK(t, err)
	list, err := l.propagation.variables.watches.Get(1)
	satOK(t, err)
	satOK(t, list.Push(watch))
	defer func() { satOK(t, list.SetEnd(0)) }()
	c, err := newSATIntegrityChecker(l, g)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	l.traceTags = tags
	_, err = c.checkWatches()
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("missing reciprocal assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled source trace emitted %q", output.String())
	}
	tags.Enable("sat_watched_bug")
	_, err = c.checkWatches()
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("enabled missing reciprocal assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	want := ""
	if tags.ShouldTrace("sat_watched_bug") && satStorageSourceDebug {
		want = "-------- [sat_watched_bug] check_watches src/sat/sat_integrity_checker.cpp:172 ---------\n" +
			"l: 0 l2: 1\nwas_eliminated1: 0 was_eliminated2: 0 learned: 0\n1\n\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("missing reciprocal source trace: got %q, want %q", got, want)
	}
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.EnableAssertions(false))
	output.Reset()
	yes, err := c.checkWatches()
	satOK(t, err)
	if !yes || output.String() != want {
		t.Fatalf("trace with assertions disabled: result=%v, got %q, want %q", yes, output.String(), want)
	}
	// The source writes the scalar fields before either watch-list display.
	// Refusing the first formatter owner must preserve that reached prefix.
	output.Reset()
	l.budget.FailAfter(0)
	defer l.budget.FailAfter(-1)
	yes, err = c.checkWatches()
	satOK(t, err)
	want = ""
	if tags.ShouldTrace("sat_watched_bug") && satStorageSourceDebug {
		want = "-------- [sat_watched_bug] check_watches src/sat/sat_integrity_checker.cpp:172 ---------\n" +
			"l: 0 l2: 1\nwas_eliminated1: 0 was_eliminated2: 0 learned: 0\n"
	}
	if !yes || output.String() != want {
		t.Fatalf("refused watch-list formatter prefix: result=%v, got %q, want %q", yes, output.String(), want)
	}
	l.budget.FailAfter(-1)
	reciprocal, err := l.propagation.variables.watches.Get(3)
	satOK(t, err)
	ternary, err := newSATTernaryWatch(0, 2)
	satOK(t, err)
	satOK(t, reciprocal.Push(ternary))
	defer func() { satOK(t, reciprocal.SetEnd(0)) }()
	for ordinal, body := range []string{
		"l: 0 l2: 1\nwas_eliminated1: 0 was_eliminated2: 0 learned: 0\n",
		"l: 0 l2: 1\nwas_eliminated1: 0 was_eliminated2: 0 learned: 0\n1\n",
		"l: 0 l2: 1\nwas_eliminated1: 0 was_eliminated2: 0 learned: 0\n1\n(0 1)\n------------------------------------------------\n",
	} {
		output.Reset()
		l.budget.FailAfter(int64(ordinal))
		yes, err = c.checkWatches()
		satOK(t, err)
		want = ""
		if tags.ShouldTrace("sat_watched_bug") && satStorageSourceDebug {
			want = "-------- [sat_watched_bug] check_watches src/sat/sat_integrity_checker.cpp:172 ---------\n" + body
		}
		if !yes || output.String() != want {
			t.Fatalf("two-list trace at formatter reservation %d: result=%v, got %q, want %q", ordinal, yes, output.String(), want)
		}
	}
}
