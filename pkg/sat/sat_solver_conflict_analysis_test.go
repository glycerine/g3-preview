package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
)

func satLemmaFixture(t *testing.T, n uint32) (*satSolverConflictAnalysis, *satSolverPropagationState, *satSolverVariableState, *arena.Budget) {
	t.Helper()
	p, v, b := satPropagationFixture(t, n)
	s, e := newSATSolverConflictAnalysis(p, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, p, v, b
}
func satLemmaAudit(t *testing.T, s *satSolverConflictAnalysis) {
	t.Helper()
	p := s.propagation
	v := p.variables
	roots := []util.Block{}
	for _, c := range []*RuntimeUintColumn{&s.lemma, &s.extAntecedents, &s.unmark, &s.minStack} {
		if c.Raw.Block != (util.Block{}) && !c.Raw.PendingReplaces {
			roots = append(roots, c.Raw.Block)
		}
		if c.Raw.Pending != (util.Block{}) {
			roots = append(roots, c.Raw.Pending)
		}
	}
	satScalarAuditRoots(t, v, roots...)
	want := s.ownerBytes + uint64(reflect.TypeOf(satSolverPropagationState{}).Size()) + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(a.ids.FreeCapacity()) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children))
	want += p.memory.Stats().TotalBytes
	if want != s.budget.Used() {
		t.Fatal("actual named once-only graph", want, s.budget.Used())
	}
}
func TestSATSolverLemmaMinimizationOwnedHeaderAndLifetime(t *testing.T) {
	s, p, _, b := satLemmaFixture(t, 1)
	if s.ownerBytes != uint64(reflect.TypeOf(*s).Size()) {
		t.Fatal("actual cold header")
	}
	satLemmaAudit(t, s)
	copy := *s
	if _, e := copy.impliedByMarked(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copy.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.lemma.Push(0))
	satOK(t, s.unmark.Push(0))
	satOK(t, s.minStack.Push(0))
	satOK(t, s.extAntecedents.Push(0))
	satLemmaAudit(t, s)
	before := b.Used()
	satOK(t, s.Close())
	if b.Used() >= before {
		t.Fatal("actual backing/header release")
	}
	if _, e := s.impliedByMarked(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if s.propagation != p {
		t.Fatal("borrowed identity")
	}
}
func TestSATSolverLemmaAntecedentActualAllocationPrefixes(t *testing.T) {
	failures, success := 0, 0
	for fault := int64(0); fault < 64; fault++ {
		ok := false
		t.Run(string(rune('A'+fault)), func(t *testing.T) {
			s, _, v, b := satLemmaFixture(t, 2)
			satOK(t, v.setLevel(BoolVar(1), 1))
			s.levelSet.insert(1)
			b.FailAfter(fault)
			got, e := s.processAntecedentForMinimization(MakeLiteral(1, false))
			mark := v.mark[1]
			un, e1 := s.unmark.Size()
			satOK(t, e1)
			st, e1 := s.minStack.Size()
			satOK(t, e1)
			if !mark || un > 1 || st > un {
				t.Fatal("source mark then unmark then stack", mark, un, st)
			}
			if e != nil {
				if !satRuntimeActualRefusal(v.memory, e) {
					t.Fatal("actual refusal only", e)
				}
				failures++
			} else {
				if !got || un != 1 || st != 1 {
					t.Fatal(got, un, st)
				}
				success++
				ok = true
			}
			satLemmaAudit(t, s) // exact physical graph BEFORE disabling refusal/Close
			b.FailAfter(-1)
			satOK(t, s.Close())
			if !v.mark[1] {
				t.Fatal("Close must not repair source-retained mark")
			}
		})
		if ok {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual refusal prefixes", failures, "success", success)
}

func TestSATSolverLemmaMinimizationWholeOperationFaultPublication(t *testing.T) {
	failures, success := 0, 0
	for fault := int64(0); fault < 128; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, p, v, b := satLemmaFixture(t, 6)
			for id := uint32(0); id < 6; id++ {
				v.scopeLevel = 1
				j := satJustification{}
				if id == 2 || id == 5 {
					j = newSATBinaryJustification(2)
				}
				if id == 3 {
					j = newSATBinaryJustification(8)
				}
				satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
			}
			for _, id := range []uint32{1, 2, 3, 5} {
				v.mark[id] = true
			}
			for _, id := range []uint32{0, 1, 2, 3, 5} {
				satOK(t, s.lemma.Push(2*id+1))
			}
			b.FailAfter(fault)
			e := s.minimizeLemma()
			satLemmaAudit(t, s)
			n, e1 := s.lemma.Size()
			satOK(t, e1)
			if e != nil {
				if !satRuntimeActualRefusal(v.memory, e) {
					t.Fatal(e)
				}
				failures++
				if n != 5 || s.minimizedLits != 0 {
					t.Fatal("no final shrink/stat publication", n, s.minimizedLits)
				}
				want := []uint32{1, 3, 5, 7, 11}
				for i, l := range want {
					got, e1 := s.lemma.Read(uint32(i))
					satOK(t, e1)
					if i == 2 && (got == 5 || got == 7) {
						continue
					}
					if got != l {
						t.Fatal("completed stable-compaction prefix", i, got, l)
					}
				}
				un, e1 := s.unmark.Size()
				satOK(t, e1)
				for i := uint32(0); i < un; i++ {
					id, e1 := s.unmark.Read(i)
					satOK(t, e1)
					if !v.mark[id] {
						t.Fatal("live recorded source mark before interrupted cleanup", id)
					}
				}
				if v.mark[0] || !v.mark[1] || !v.mark[3] {
					t.Fatal("firstUIP/essential marks")
				}
			} else {
				success++
				done = true
				if n != 3 || s.minimizedLits != 2 {
					t.Fatal(n, s.minimizedLits)
				}
				for i, l := range []uint32{1, 3, 7} {
					got, e1 := s.lemma.Read(uint32(i))
					satOK(t, e1)
					if got != l {
						t.Fatal(i, got, l)
					}
				}
				for _, id := range []uint32{2, 4, 5} {
					if v.mark[id] {
						t.Fatal("completed source reset", id)
					}
				}
			}
			b.FailAfter(-1)
		})
		if done {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal("all actual prefixes through first success", failures, success)
	}
	t.Log("full source minimization refusal prefixes", failures, "success", success)
}

func TestSATSolverLemmaMinimizationConstructorBudgetIdentityAndRefusal(t *testing.T) {
	p, _, b := satPropagationFixture(t, 1)
	foreign := arena.NewBudget(^uint64(0))
	before := b.Used()
	s, e := newSATSolverConflictAnalysis(p, foreign)
	if s != nil {
		satOK(t, s.Close())
	}
	if !errors.Is(e, ErrSATOwner) || b.Used() != before || foreign.Used() != 0 {
		t.Fatal("actual borrowed cohort budget identity", e, b.Used(), before, foreign.Used())
	}

	q, e := newSATSolverPropagationState(p.variables, p.clauses, p.converter, p.limit, p.memory, p.config, foreign)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, q.Close())
		if foreign.Used() != 0 {
			t.Fatal("borrowed mismatch fixture exact cleanup", foreign.Used())
		}
	})
	s, e = newSATSolverConflictAnalysis(q, foreign)
	if s != nil {
		satOK(t, s.Close())
	}
	if !errors.Is(e, ErrSATOwner) {
		t.Fatal("actual Runtime budget must match the selected cohort", e)
	}
	b.FailAfter(0)
	s, e = newSATSolverConflictAnalysis(p, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || s != nil || b.Used() != before {
		t.Fatal("cold header refusal before publication", s, e, b.Used(), before)
	}
	b.FailAfter(-1)
}

func TestSATSolverLemmaMinimizationDeepVectorGrowthFaultPrefixes(t *testing.T) {
	failures, success := 0, 0
	for fault := int64(0); fault < 128; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, p, v, b := satLemmaFixture(t, 8)
			c, e := p.clauses.MkClause([]Literal{14, 2, 4, 6, 8, 10, 12}, false)
			satOK(t, e)
			o, e := p.clauses.Offset(c)
			satOK(t, e)
			for id := uint32(0); id < 8; id++ {
				v.scopeLevel = 1
				j := satJustification{}
				if id == 7 {
					j = newSATClauseJustification(o)
				}
				satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
			}
			v.mark[7] = true
			satOK(t, s.lemma.Push(1))
			satOK(t, s.lemma.Push(15))
			b.FailAfter(fault)
			e = s.minimizeLemma()
			satLemmaAudit(t, s)
			n, e1 := s.lemma.Size()
			satOK(t, e1)
			if n != 2 || s.minimizedLits != 0 {
				t.Fatal("essential decision-leaf lemma unchanged", n, s.minimizedLits)
			}
			if e != nil {
				if !satRuntimeActualRefusal(v.memory, e) {
					t.Fatal(e)
				}
				failures++
				un, e1 := s.unmark.Size()
				satOK(t, e1)
				st, e1 := s.minStack.Size()
				satOK(t, e1)
				if un > 6 || st > un {
					t.Fatal("literal-order vector publication", un, st)
				}
				marks := uint32(0)
				gap := false
				for id := uint32(1); id <= 6; id++ {
					m := v.mark[id]
					if !m {
						gap = true
					} else {
						if gap {
							t.Fatal("ascending clause mark prefix", id)
						}
						marks++
					}
				}
				if marks < un || marks > un+1 {
					t.Fatal("mark before actual unmark reservation", marks, un)
				}
				for i := uint32(0); i < un; i++ {
					id, e1 := s.unmark.Read(i)
					satOK(t, e1)
					if id != i+1 {
						t.Fatal("source ascending antecedent owner order", i, id)
					}
				}
			} else {
				done = true
				success++
				un, e1 := s.unmark.Size()
				satOK(t, e1)
				if un != 0 {
					t.Fatal("rejected decision resets complete new suffix", un)
				}
				for id := uint32(1); id <= 6; id++ {
					if v.mark[id] {
						t.Fatal(id)
					}
				}
			}
			if !v.mark[7] {
				t.Fatal("original lemma mark retained")
			}
			b.FailAfter(-1)
			satOK(t, p.clauses.Delete(c))
		})
		if done {
			break
		}
	}
	if failures < 6 || success != 1 {
		t.Fatal("actual geometric growth prefixes through first success", failures, success)
	}
	t.Log("deep source vector refusal prefixes", failures, "success", success)
}
