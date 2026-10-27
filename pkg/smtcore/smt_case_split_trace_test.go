package smtcore

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	z3rand "github.com/glycerine/g3/pkg/util/random"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func caseSplitTraceRecord(tag, function string, line int, body string) string {
	return fmt.Sprintf("-------- [%s] %s src/smt/smt_case_split_queue.cpp:%d ---------\n%s------------------------------------------------\n", tag, function, line, body)
}

func queueTraceFixture(t *testing.T, strategy uint32) (*ast.ASTManager, *SMTInternalizerState, *SMTRelevancyPropagator, *SMTActivityCaseSplitQueue) {
	t.Helper()
	m := astTestManager(t)
	_, p, s := smtInternalizerFixture(t, m, nil)
	params := SMTParamsState{CaseSplitStrategy: strategy, RelevancyLvl: 2, InvDecay: 1.052, RelCaseSplitOrder: 0}
	astTestMust(t, s.configureBooleanHeuristics(params))
	q, err := NewSMTActivityCaseSplitQueue(s)
	astTestMust(t, err)
	astTestMust(t, q.ConfigurePhases(params))
	return m, s, p, q
}

func TestSMTCaseSplitRandomTraceAllLiveConsumersAndReservedVariable(t *testing.T) {
	for _, strategy := range []uint32{0, 1, 2, 4, 6} {
		for _, candidate := range []uint32{0, 1} {
			t.Run(fmt.Sprintf("%d/%d", strategy, candidate), func(t *testing.T) {
				m, s, _, q := queueTraceFixture(t, strategy)
				a := smtRelevancyBool(t, m, "random-trace-a")
				astTestMust(t, s.Internalize(a, true))
				var out bytes.Buffer
				tags := diagnostic.NewTraceTagsWithWriter(&out)
				tags.Enable("random_split")
				s.queueTrace = tags
				frequency := 1.0
				line := 118
				if strategy == 4 {
					frequency = 0.02
					line = 638
				} else if strategy == 6 {
					line = 1171
				}
				q.randomVariableFrequency = frequency
				var seed uint32
				for ; seed < 100000; seed++ {
					r := z3rand.New(seed)
					if int(r.Next()) < int(frequency*float64(z3rand.Max)) && r.Next()%uint32(len(s.boolExpr)) == candidate {
						break
					}
				}
				if seed == 100000 {
					t.Fatal("no branch seed")
				}
				s.random.SetSeed(seed)
				before := s.random
				before.Next()
				before.Next()
				_, _, err := q.NextCaseSplit()
				astTestMust(t, err)
				if s.random != before {
					t.Fatal("trace consumed RNG")
				}
				value := lbool.Undef
				if candidate == 0 {
					value = lbool.True
				}
				want := ""
				if tags.ShouldTrace("random_split") {
					want = caseSplitTraceRecord("random_split", "next_case_split", line, fmt.Sprintf("next: %d get_assignment(next): %s\n", candidate, value))
				}
				if out.String() != want {
					t.Fatal(out.String(), want)
				}
			})
		}
	}
}

func TestSMTCaseSplitCandidateTraceAndIndependentASTStream(t *testing.T) {
	for _, strategy := range []uint32{3, 4, 5} {
		t.Run(fmt.Sprint(strategy), func(t *testing.T) {
			m, s, p, q := queueTraceFixture(t, strategy)
			a := smtRelevancyBool(t, m, "trace-child-a")
			b := smtRelevancyBool(t, m, "trace-child-b")
			parent, err := m.MkOr(a, b)
			astTestMust(t, err)
			astTestMust(t, s.Internalize(parent, true))
			pv, _ := s.BoolVar(parent)
			av, _ := s.BoolVar(a)
			astTestMust(t, s.SetAssignment(pv, lbool.True))
			astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
			t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
			astTestMust(t, p.MarkRelevant(parent))
			pid, err := m.GetASTID(parent)
			astTestMust(t, err)
			aid, err := m.GetASTID(a)
			astTestMust(t, err)
			stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
			astTestMust(t, err)
			m.TraceStream = stream
			t.Cleanup(func() { m.TraceStream = nil; astTestMust(t, stream.Close()) })
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("case_split")
			s.queueTrace = tags
			s.random.SetSeed(10000)
			got, phase, err := q.NextCaseSplit()
			astTestMust(t, err)
			if got != av || phase != lbool.True {
				t.Fatal("candidate changed", got, phase)
			}
			line, displayLine, marker := 427, 431, "[HEAD1]=> "
			if strategy == 4 {
				line, displayLine, marker = 616, 620, "[HEAD]=> "
			} else if strategy == 5 {
				line, displayLine = 883, 887
			}
			want := ""
			if tags.ShouldTrace("case_split") {
				want = caseSplitTraceRecord("case_split", "next_case_split_core", line, fmt.Sprintf("found AND/OR candidate: #%d #%d\n", pid, aid)) + caseSplitTraceRecord("case_split", "next_case_split_core", displayLine, fmt.Sprintf("case-splits:\n%s#%d \n", marker, pid))
			}
			if out.String() != want {
				t.Fatal(out.String(), want)
			}
			record, err := stream.String()
			astTestMust(t, err)
			mandatory := ""
			if strategy != 4 {
				mandatory = fmt.Sprintf("[decide-and-or] #%d #%d\n", pid, aid)
			}
			if record != mandatory {
				t.Fatal("independent AST stream", record, mandatory)
			}
			// Every physical refusal prefix must leave the source decision unchanged.
			for cut := 0; cut < len(want); cut++ {
				w := &smtLRAOutputPrefix{remaining: cut}
				s.queueTrace = diagnostic.NewTraceTagsWithWriter(w)
				s.queueTrace.Enable("case_split")
				s.random.SetSeed(10000)
				got, phase, err = q.NextCaseSplit()
				astTestMust(t, err)
				if got != av || phase != lbool.True || w.text.String() != want[:cut] || !s.queueTrace.IsStreamFailure(io.ErrClosedPipe) {
					t.Fatal("trace refusal changed decision", cut, got, phase, w.text.String())
				}
			}
		})
	}
}

func TestSMTCaseSplitScopeTraceAndCandidateDisplay(t *testing.T) {
	for _, strategy := range []uint32{3, 4, 5} {
		t.Run(fmt.Sprint(strategy), func(t *testing.T) {
			m, s, p, q := queueTraceFixture(t, strategy)
			a := smtRelevancyBool(t, m, "scope-trace-a")
			astTestMust(t, s.Internalize(a, true))
			av, _ := s.BoolVar(a)
			id, err := m.GetASTID(a)
			astTestMust(t, err)
			astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
			t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
			astTestMust(t, p.MarkRelevant(a))
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("case_split")
			s.queueTrace = tags
			astTestMust(t, s.PushScope())
			s.random.SetSeed(10000)
			got, _, err := q.NextCaseSplit()
			astTestMust(t, err)
			if got != av {
				t.Fatal(got, av)
			}
			astTestMust(t, s.AssignLiteral(NewSMTLiteral(av, false), NewSMTBooleanAxiom()))
			s.random.SetSeed(10000)
			_, _, err = q.NextCaseSplit()
			astTestMust(t, err)
			astTestMust(t, s.PopScopes(1))
			push, pop, found, display, marker := 387, 400, 437, 439, "[HEAD1]=> "
			if strategy == 4 {
				push, pop, found, display, marker = 581, 592, 626, 628, "[HEAD]=> "
			} else if strategy == 5 {
				push, pop, found, display = 827, 860, 893, 895
			}
			body := fmt.Sprintf("case-splits:\n%s#%d \n", marker, id)
			want := ""
			if tags.ShouldTrace("case_split") {
				want = caseSplitTraceRecord("case_split", "push_scope", push, "head: 0\n") + caseSplitTraceRecord("case_split", "next_case_split_core", found, fmt.Sprintf("found candidate: #%d\n", id)) + caseSplitTraceRecord("case_split", "next_case_split_core", display, body) + caseSplitTraceRecord("case_split", "pop_scope", pop, body+"head: 0\n")
			}
			if out.String() != want {
				t.Fatal(out.String(), want)
			}
		})
	}
}

type caseSplitPriorityWriter struct {
	bytes.Buffer
	state    *SMTInternalizerState
	variable int32
	seen     bool
	before   float64
	flushes  int
	refuse   bool
}

func (w *caseSplitPriorityWriter) Write(p []byte) (int, error) {
	if !w.seen {
		w.before = w.state.theoryAwareBranching.priorities[w.variable]
		w.seen = true
	}
	if w.refuse {
		return 0, io.ErrClosedPipe
	}
	return w.Buffer.Write(p)
}
func (w *caseSplitPriorityWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func (w *caseSplitPriorityWriter) Flush() error { w.flushes++; return nil }

func TestSMTCaseSplitTheoryPriorityTraceBeforeMutationAndEndl(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m, s, _, q := queueTraceFixture(t, 6)
		a := smtRelevancyBool(t, m, "priority-trace-a")
		b := smtRelevancyBool(t, m, "priority-trace-b")
		astTestMust(t, s.Internalize(a, true))
		astTestMust(t, s.Internalize(b, true))
		bv, _ := s.BoolVar(b)
		w := &caseSplitPriorityWriter{state: s, variable: bv, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("theory_aware_branching")
		s.queueTrace = tags
		astTestMust(t, s.AddTheoryAwareBranchingInfo(bv, 1.23456789, lbool.True))
		got, _, err := q.NextCaseSplit()
		astTestMust(t, err)
		if got != bv {
			t.Fatal("refusal changed priority", got, bv)
		}
		want := ""
		if tags.ShouldTrace("theory_aware_branching") {
			if !w.seen || w.before != 0 {
				t.Fatal("trace moved after priority mutation", w.seen, w.before)
			}
			if !refuse {
				want = caseSplitTraceRecord("theory_aware_branching", "add_theory_aware_branching_info", 1191, fmt.Sprintf("Add theory-aware branching information for l#%d: priority=1.23457\n", bv))
				if w.flushes != 2 {
					t.Fatal("lost source endl or footer flush", w.flushes)
				}
			}
		} else if w.seen {
			t.Fatal("native trace evaluated")
		}
		if w.String() != want {
			t.Fatal(w.String(), want)
		}
	}
}

func TestSMTCaseSplitAvailabilityProbeHasNoDecisionEffects(t *testing.T) {
	for _, strategy := range []uint32{3, 5} {
		m, s, p, q := queueTraceFixture(t, strategy)
		a := smtRelevancyBool(t, m, "probe-a")
		b := smtRelevancyBool(t, m, "probe-b")
		parent, err := m.MkOr(a, b)
		astTestMust(t, err)
		astTestMust(t, s.Internalize(parent, true))
		pv, _ := s.BoolVar(parent)
		astTestMust(t, s.SetAssignment(pv, lbool.True))
		astTestMust(t, p.MarkRelevant(parent))
		q.relCaseSplitOrder = 1
		var out bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		tags.Enable("case_split")
		s.queueTrace = tags
		rng := s.random
		fifo := s.relevanceFIFO.snapshot()
		goal := s.relevanceGoal.snapshot(s.scopeLevel)
		var heapBefore []int32
		if s.relevanceGoal.heap != nil {
			heapBefore, err = s.relevanceGoal.heap.AppendValues(nil)
			astTestMust(t, err)
		}
		available, err := q.hasCaseSplit()
		astTestMust(t, err)
		if !available || s.random != rng || fifo != s.relevanceFIFO.snapshot() || goal != s.relevanceGoal.snapshot(s.scopeLevel) || out.Len() != 0 {
			t.Fatal("availability performed a source decision", strategy, available, out.String())
		}
		if s.relevanceGoal.heap != nil {
			after, err := s.relevanceGoal.heap.AppendValues(nil)
			astTestMust(t, err)
			if fmt.Sprint(after) != fmt.Sprint(heapBefore) || s.relevanceGoal.second[0].lastDecided != -1 {
				t.Fatal("availability consumed goal queue")
			}
		}
	}
}

func TestSMTCaseSplitTraceErasureAndOperandsAfterWriterRefusal(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		m, s, p, q := queueTraceFixture(t, 3)
		a := smtRelevancyBool(t, m, "trace-operand-a")
		astTestMust(t, s.Internalize(a, true))
		astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
		t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
		astTestMust(t, p.MarkRelevant(a))
		// This unreachable malformed display operand must be evaluated only by TRACE.
		s.relevanceFIFO.first = append(s.relevanceFIFO.first, ast.ASTHandle{})
		w := &smtLRAOutputPrefix{}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		if enabled {
			tags.Enable("case_split")
		}
		s.queueTrace = tags
		_, _, err := q.NextCaseSplit()
		if tags.ShouldTrace("case_split") {
			if err == nil || !tags.IsStreamFailure(io.ErrClosedPipe) {
				t.Fatal("bad header skipped later display operand", err)
			}
		} else {
			astTestMust(t, err)
		}
	}
}

type caseSplitObserverWriter struct {
	output  bytes.Buffer
	observe func()
	seen    bool
	refuse  bool
}

func (w *caseSplitObserverWriter) Write(p []byte) (int, error) {
	if !w.seen {
		w.seen = true
		w.observe()
	}
	if w.refuse {
		return 0, io.ErrClosedPipe
	}
	return w.output.Write(p)
}

func TestSMTCaseSplitGoalAssignmentTraceBeforeGoalPublication(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m, s, _, _ := queueTraceFixture(t, 5)
		label, err := m.MkLabelLit([]usymbol.Symbol{astTestSymbol(t, m, "trace-goal-label")})
		astTestMust(t, err)
		a := smtRelevancyBool(t, m, "trace-goal-body")
		goal, err := m.MkAnd(a, label)
		astTestMust(t, err)
		astTestMust(t, s.Internalize(goal, true))
		v, _ := s.BoolVar(goal)
		w := &caseSplitObserverWriter{refuse: refuse, observe: func() {
			if !s.relevanceGoal.current.IsNull() {
				t.Fatal("trace after goal publication")
			}
		}}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("case_split")
		s.queueTrace = tags
		astTestMust(t, s.AssignLiteral(NewSMTLiteral(v, false), NewSMTBooleanAxiom()))
		if s.relevanceGoal.current != goal {
			t.Fatal("lost goal assignment")
		}
		want := ""
		if tags.ShouldTrace("case_split") {
			if !w.seen {
				t.Fatal("missing goal trace")
			}
			if !refuse {
				want = caseSplitTraceRecord("case_split", "assign_lit_eh", 975, "Found goal\n(and trace-goal-body (! true :lblpos trace-goal-label))\n")
			}
		}
		if w.output.String() != want {
			t.Fatal(w.output.String(), want)
		}
	}
}

func TestSMTCaseSplitSecondaryGoalTraceBeforeDecisionAndReinsert(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m, s, p, q := queueTraceFixture(t, 5)
		astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
		t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
		a := smtRelevancyBool(t, m, "secondary-trace-a")
		astTestMust(t, s.Internalize(a, true))
		astTestMust(t, p.MarkRelevant(a))
		av, _ := s.BoolVar(a)
		id, err := m.GetASTID(a)
		astTestMust(t, err)
		w := &caseSplitObserverWriter{refuse: refuse, observe: func() {
			if s.relevanceGoal.second[0].lastDecided != -1 || s.relevanceGoal.heap.Contains(0) {
				t.Fatal("secondary trace moved past decision or before erase")
			}
		}}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("case_split")
		s.queueTrace = tags
		got, _, err := q.NextCaseSplit()
		astTestMust(t, err)
		if got != av || s.relevanceGoal.second[0].lastDecided != -1 || !s.relevanceGoal.heap.Contains(0) {
			t.Fatal("trace refusal lost secondary reinsertion", got, av)
		}
		want := ""
		if tags.ShouldTrace("case_split") {
			if !w.seen {
				t.Fatal("missing secondary trace")
			}
			if !refuse {
				want = caseSplitTraceRecord("case_split", "next_case_split", 915, "q 1 idx 0\n") + caseSplitTraceRecord("case_split", "next_case_split_core", 893, fmt.Sprintf("found candidate: #%d\n", id)) + caseSplitTraceRecord("case_split", "next_case_split_core", 895, "case-splits:\n")
			}
		}
		if w.output.String() != want {
			t.Fatal(w.output.String(), want)
		}
	}
}

func TestSMTCaseSplitASTStreamOwnerRefusalPrecedesLiteralSelection(t *testing.T) {
	for _, strategy := range []uint32{3, 4, 5} {
		m, s, p, q := queueTraceFixture(t, strategy)
		a := smtRelevancyBool(t, m, "stream-refusal-a")
		b := smtRelevancyBool(t, m, "stream-refusal-b")
		parent, err := m.MkOr(a, b)
		astTestMust(t, err)
		astTestMust(t, s.Internalize(parent, true))
		pv, _ := s.BoolVar(parent)
		astTestMust(t, s.SetAssignment(pv, lbool.True))
		astTestMust(t, s.beginSearchPersistence(0, SMTParamsState{}))
		t.Cleanup(func() { astTestMust(t, s.endSearchPersistence()) })
		astTestMust(t, p.MarkRelevant(parent))
		stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
		astTestMust(t, err)
		astTestMust(t, stream.Close())
		m.TraceStream = stream
		s.random.SetSeed(10000)
		_, _, err = q.NextCaseSplit()
		m.TraceStream = nil
		if strategy == 4 {
			astTestMust(t, err)
		} else if err != arena.ErrClosed {
			t.Fatal("missing AST stream owner guard", strategy, err)
		}
	}
}
