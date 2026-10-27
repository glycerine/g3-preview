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

func relevanceFIFOFixture(t *testing.T) (*ast.ASTManager, *SMTInternalizerState, *SMTRelevancyPropagator, *SMTActivityCaseSplitQueue) {
	t.Helper()
	m := astTestManager(t)
	_, p, s := smtInternalizerFixture(t, m, nil)
	q, err := NewSMTActivityCaseSplitQueue(s)
	astTestMust(t, err)
	astTestMust(t, q.ConfigurePhases(SMTParamsState{CaseSplitStrategy: 3, RelevancyLvl: 2}))
	return m, s, p, q
}

func TestSMTRelevanceFIFOActualAdmissionSearchPartitionAndPop(t *testing.T) {
	m, s, p, q := relevanceFIFOFixture(t)
	a := smtRelevancyBool(t, m, "fifo-before-search")
	b := smtRelevancyBool(t, m, "fifo-base-during-search")
	for _, term := range []ast.ASTHandle{a, b} {
		astTestMust(t, s.Internalize(term, true))
	}
	av, _ := s.BoolVar(a)
	bv, _ := s.BoolVar(b)
	astTestMust(t, p.MarkRelevant(a))
	if len(s.relevanceFIFO.first) != 0 || len(s.relevanceFIFO.second) != 1 || s.relevanceFIFO.baseBooleanLimit != -1 {
		t.Fatal("native pre-search signed sentinel partition lost")
	}
	astTestMust(t, s.beginSearchPersistence(s.scopeLevel, SMTParamsState{}))
	t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
	astTestMust(t, p.MarkRelevant(b))
	astTestMust(t, s.PushScope())
	c := smtRelevancyBool(t, m, "fifo-new-search-variable")
	astTestMust(t, s.Internalize(c, true))
	astTestMust(t, p.MarkRelevant(c))
	cv, _ := s.BoolVar(c)
	if len(s.relevanceFIFO.first) != 1 || len(s.relevanceFIFO.second) != 2 {
		t.Fatal("search admission partition", s.relevanceFIFO)
	}
	selected, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if selected != bv {
		t.Fatal("primary queue must precede older secondary event", selected, bv)
	}
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(bv, false), NewSMTBooleanAxiom()))
	selected, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if selected != av || s.relevanceFIFO.firstHead != 1 || s.relevanceFIFO.secondHead != 0 {
		t.Fatal("head advancement/order", selected, av, s.relevanceFIFO)
	}
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(av, false), NewSMTBooleanAxiom()))
	selected, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if selected != cv || s.relevanceFIFO.secondHead != 1 {
		t.Fatal("new variable lost", selected, cv)
	}
	astTestMust(t, s.PopScopes(1))
	if len(s.relevanceFIFO.second) != 1 || s.relevanceFIFO.firstHead != 0 || s.relevanceFIFO.secondHead != 0 {
		t.Fatal("scope did not restore actual queue snapshot", s.relevanceFIFO)
	}
	selected, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if selected != bv {
		t.Fatal("popped decision cursor not restored", selected, bv)
	}
}

func TestSMTRelevanceFIFOAdmissionFiltersAndSuccessfulHeadRetention(t *testing.T) {
	m, s, p, q := relevanceFIFOFixture(t)
	a := smtRelevancyBool(t, m, "fifo-true-atom")
	b := smtRelevancyBool(t, m, "fifo-undefined-atom")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.Internalize(b, true))
	av, _ := s.BoolVar(a)
	bv, _ := s.BoolVar(b)
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(av, false), NewSMTBooleanAxiom()))
	astTestMust(t, p.MarkRelevant(a))
	astTestMust(t, p.MarkRelevant(b))
	if len(s.relevanceFIFO.second) != 1 || s.relevanceFIFO.second[0] != b {
		t.Fatal("assigned atom must be filtered at the event")
	}
	for i := 0; i < 2; i++ {
		selected, _, err := q.NextCaseSplit()
		astTestMust(t, err)
		if selected != bv || s.relevanceFIFO.secondHead != 0 {
			t.Fatal("successful candidate consumed its own head", selected, bv)
		}
	}
	astTestMust(t, s.PushScope())
	astTestMust(t, s.AssignLiteral(NewSMTLiteral(bv, true), NewSMTBooleanAxiom()))
	selected, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if selected != SMTNullBoolVar || s.relevanceFIFO.secondHead != 1 {
		t.Fatal("exhausted entry not consumed", selected)
	}
	astTestMust(t, s.PopScopes(1))
	selected, _, err = q.NextCaseSplit()
	astTestMust(t, err)
	if selected != bv || s.relevanceFIFO.secondHead != 0 {
		t.Fatal("backtrack did not revive head", selected, bv)
	}
}

func TestSMTRelevanceFIFOExactDumpHeadsEveryWriterPrefixAndBorrowing(t *testing.T) {
	m, s, p, q := relevanceFIFOFixture(t)
	a := smtRelevancyBool(t, m, "fifo-dump-a")
	b := smtRelevancyBool(t, m, "fifo-dump-b")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.Internalize(b, true))
	astTestMust(t, p.MarkRelevant(a))
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
	astTestMust(t, p.MarkRelevant(b))
	aid, err := m.GetASTID(a)
	astTestMust(t, err)
	bid, err := m.GetASTID(b)
	astTestMust(t, err)
	want := fmt.Sprintf("case-splits:\n[HEAD1]=> #%d \n[HEAD2]=> #%d \n", bid, aid)
	var out bytes.Buffer
	astTestMust(t, s.DisplayRelevanceCaseSplitQueue(&out))
	if out.String() != want {
		t.Fatal(out.String(), want)
	}
	before, snapshot := m.Stats(), s.relevanceFIFO.snapshot()
	for cut := 0; cut < len(want); cut++ {
		w := &smtLRAOutputPrefix{remaining: cut}
		if err := s.DisplayRelevanceCaseSplitQueue(w); !errors.Is(err, io.ErrClosedPipe) || w.text.String() != want[:cut] {
			t.Fatal(cut, err, w.text.String(), want[:cut])
		}
	}
	if err := s.DisplayRelevanceCaseSplitQueue(smtLRAShortOutput{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	if m.Stats() != before || s.relevanceFIFO.snapshot() != snapshot {
		t.Fatal("dump mutated owner or queue state")
	}
	for _, term := range []ast.ASTHandle{a, b} {
		v, _ := s.BoolVar(term)
		astTestMust(t, s.AssignLiteral(NewSMTLiteral(v, false), NewSMTBooleanAxiom()))
	}
	selected, _, err := q.NextCaseSplit()
	astTestMust(t, err)
	if selected != SMTNullBoolVar {
		t.Fatal(selected)
	}
	out.Reset()
	astTestMust(t, s.DisplayRelevanceCaseSplitQueue(&out))
	want = fmt.Sprintf("case-splits:\n#%d \n#%d \n", bid, aid)
	if out.String() != want {
		t.Fatal("consumed entries disappear or end HEAD invented", out.String(), want)
	}
	if err := s.DisplayRelevanceCaseSplitQueue(nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
}

func TestSMTRelevanceFIFOEndSearchResetsAdmissionSentinel(t *testing.T) {
	m, s, p, _ := relevanceFIFOFixture(t)
	a := smtRelevancyBool(t, m, "fifo-after-search")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
	astTestMust(t, s.endSearchPersistence())
	astTestMust(t, p.MarkRelevant(a))
	if s.relevanceFIFO.baseBooleanLimit != -1 || len(s.relevanceFIFO.first) != 0 || len(s.relevanceFIFO.second) != 1 {
		t.Fatal(s.relevanceFIFO)
	}
	value, err := s.Assignment(NewSMTLiteral(1, false))
	astTestMust(t, err)
	if value != lbool.Undef {
		t.Fatal(value)
	}
}
