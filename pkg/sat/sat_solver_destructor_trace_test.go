package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satDestructorTraceObserver struct {
	output                    bytes.Buffer
	allocator                 *ClauseAllocator
	original, learned         StoredClauseRef
	originalLive, learnedLive bool
}

func (w *satDestructorTraceObserver) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("Delete clauses")) {
		w.originalLive = w.allocator.IsLive(w.original)
	}
	if bytes.Contains(p, []byte("Delete learned")) {
		w.learnedLive = w.allocator.IsLive(w.learned)
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:60-67; both records precede their respective
// deletion loops even when the clause vectors are empty.
func TestSATSolverDestructorClauseTraceSourceOrder(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	compiled := tags.ShouldTrace("sat")
	s.traceTags = tags
	satOK(t, s.Close())
	if !s.Lifecycle.originalDone || !s.Lifecycle.learnedDone {
		t.Fatal("source clause deletion stages did not finish")
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased destructor trace emitted", output.String())
		}
		return
	}
	want := "-------- [sat] ~solver src/sat/sat_solver.cpp:62 ---------\n" +
		"Delete clauses\n------------------------------------------------\n" +
		"-------- [sat] ~solver src/sat/sat_solver.cpp:64 ---------\n" +
		"Delete learned\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source destructor trace order: got %q, want %q", got, want)
	}
	satOK(t, s.Close())
	if output.String() != want {
		t.Fatal("completed destructor replayed trace")
	}
}

func TestSATDestructorTracePrecedesEachLiveClauseDeletion(t *testing.T) {
	l, p, _, _ := satLifecycleFixture(t, 4)
	original, err := p.clauses.MkClause([]Literal{0, 2, 4, 6}, false)
	satOK(t, err)
	learned, err := p.clauses.MkClause([]Literal{1, 3, 5, 7}, true)
	satOK(t, err)
	satOK(t, l.pushClause(false, original))
	satOK(t, l.pushClause(true, learned))
	w := &satDestructorTraceObserver{allocator: p.clauses, original: original, learned: learned}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat")
	satOK(t, l.deleteClausesBody(tags))
	if p.clauses.IsLive(original) || p.clauses.IsLive(learned) {
		t.Fatal("source destructor retained clauses")
	}
	if tags.ShouldTrace("sat") && (!w.originalLive || !w.learnedLive) {
		t.Fatal("TRACE ran after its clause deletion")
	}
	before := w.output.String()
	satOK(t, l.deleteClausesBody(tags))
	if w.output.String() != before {
		t.Fatal("completed deletion replayed source TRACE")
	}
}
