package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func basicActivityFixture(t *testing.T) (*ast.ASTManager, *SMTInternalizerState, *SMTActivityCaseSplitQueue) {
	t.Helper()
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	q, err := NewSMTActivityCaseSplitQueue(s)
	astTestMust(t, err)
	astTestMust(t, q.ConfigurePhases(SMTParamsState{}))
	return m, s, q
}

func TestSMTBasicActivityHeapNativeEqualPriorityConsumption(t *testing.T) {
	m, s, q := basicActivityFixture(t)
	var vars []int32
	for _, name := range []string{"basic-heap-a", "basic-heap-b", "basic-heap-c"} {
		x := smtRelevancyBool(t, m, name)
		astTestMust(t, s.Internalize(x, true))
		v, _ := s.BoolVar(x)
		vars = append(vars, v)
	}
	// Native first erases assigned true variable0, moving the last equal
	// entry to the root. Heap order is observably different from an ID scan.
	for _, want := range []int32{vars[2], vars[1], vars[0], SMTNullBoolVar} {
		got, _, err := q.NextCaseSplit()
		astTestMust(t, err)
		if got != want {
			t.Fatal("native equal-priority heap order/consumption", got, want)
		}
		if got != SMTNullBoolVar && s.basicActivity.Contains(got) {
			t.Fatal("selected entry remains present")
		}
	}
	available, err := q.hasCaseSplit()
	astTestMust(t, err)
	if available {
		t.Fatal("zero-frequency availability regenerated consumed entries")
	}
	q.randomVariableFrequency = 0.5 / 32767.0
	available, err = q.hasCaseSplit()
	astTestMust(t, err)
	if available {
		t.Fatal("rounded-zero random threshold claimed a decision")
	}
	q.randomVariableFrequency = 1.0
	rng := s.random
	available, err = q.hasCaseSplit()
	astTestMust(t, err)
	if !available || s.random != rng {
		t.Fatal("reachable random candidate lost or RNG consumed")
	}
}

func TestSMTBasicActivityHeapPriorityUnassignmentAndDeletionCallbacks(t *testing.T) {
	m, s, q := basicActivityFixture(t)
	a := smtRelevancyBool(t, m, "basic-priority-a")
	b := smtRelevancyBool(t, m, "basic-priority-b")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.Internalize(b, true))
	av, _ := s.BoolVar(a)
	astTestMust(t, s.incrementBooleanActivity(av))
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != av || s.basicActivity.Contains(av) {
		t.Fatal("decreased priority callback not applied", got, av)
	}
	astTestMust(t, s.PushScope())
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(av, false), NewSMTBooleanAxiom()))
	child := smtRelevancyBool(t, m, "basic-priority-child")
	astTestMust(t, s.Internalize(child, true))
	cv, _ := s.BoolVar(child)
	if !s.basicActivity.Contains(cv) {
		t.Fatal("creation callback missing")
	}
	astTestMust(t, s.PopScopes(1))
	if !s.basicActivity.Contains(av) || s.basicActivity.Contains(cv) {
		t.Fatal("unassign/delete callbacks missing")
	}
	rng := s.random
	available, err := q.hasCaseSplit()
	astTestMust(t, err)
	if !available || !s.basicActivity.Contains(av) || s.random != rng {
		t.Fatal("availability consumed heap or RNG")
	}
	got, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if got != av {
		t.Fatal("unassigned high-activity entry not revived", got, av)
	}
}

func TestSMTBasicActivityHeapExactDumpPrefixesAndAssignedFiltering(t *testing.T) {
	m, s, q := basicActivityFixture(t)
	a := smtRelevancyBool(t, m, "basic-dump-a")
	b := smtRelevancyBool(t, m, "basic-dump-b")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.Internalize(b, true))
	aid, err := m.GetASTID(a)
	astTestMust(t, err)
	bid, err := m.GetASTID(b)
	astTestMust(t, err)
	want := fmt.Sprintf("remaining case-splits:\n#%d #%d \n", aid, bid)
	var out bytes.Buffer
	astTestMust(t, s.DisplayBasicActivityQueue(&out))
	if out.String() != want {
		t.Fatal(out.String(), want)
	}
	before := m.Stats()
	for cut := 0; cut < len(want); cut++ {
		w := &smtLRAOutputPrefix{remaining: cut}
		if err := s.DisplayBasicActivityQueue(w); !errors.Is(err, io.ErrClosedPipe) || w.text.String() != want[:cut] {
			t.Fatal(cut, err, w.text.String(), want[:cut])
		}
	}
	if err := s.DisplayBasicActivityQueue(smtLRAShortOutput{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	if before != m.Stats() {
		t.Fatal("dump changed AST ownership")
	}
	bv, _ := s.BoolVar(b)
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(bv, false), NewSMTBooleanAxiom()))
	out.Reset()
	astTestMust(t, s.DisplayBasicActivityQueue(&out))
	if want := fmt.Sprintf("remaining case-splits:\n#%d \n", aid); out.String() != want {
		t.Fatal(out.String(), want)
	}
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	av, _ := s.BoolVar(a)
	if got != av {
		t.Fatal(got, av)
	}
	out.Reset()
	astTestMust(t, s.DisplayBasicActivityQueue(&out))
	if out.Len() != 0 {
		t.Fatal(out.String())
	}
}

func TestSMTBasicActivityHeapConstructorRefusalRetryAndReadSideUnassignment(t *testing.T) {
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	q, err := NewSMTActivityCaseSplitQueue(s)
	astTestMust(t, err)
	m.Budget().FailAfter(0)
	err = q.ConfigurePhases(SMTParamsState{})
	m.Budget().FailAfter(-1)
	if err == nil || s.basicActivity != nil {
		t.Fatal("failed constructor published heap", err)
	}
	astTestMust(t, q.ConfigurePhases(SMTParamsState{}))
	a := smtRelevancyBool(t, m, "basic-readside-a")
	astTestMust(t, s.Internalize(a, true))
	v, _ := s.BoolVar(a)
	got, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != v {
		t.Fatal(got, v)
	}
	astTestMust(t, s.SetAssignment(v, lbool.True))
	astTestMust(t, s.SetAssignment(v, lbool.Undef))
	if !s.basicActivity.Contains(v) {
		t.Fatal("public read-side unassignment callback missing")
	}
}
