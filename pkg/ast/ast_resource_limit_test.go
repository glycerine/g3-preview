// Source: src/ast/ast.h:1455,1549-1550; src/util/rlimit.cpp:23-71;
// src/sat/tactic/goal2sat.cpp:384-392.
package ast_test

import (
	sattactic "github.com/glycerine/g3/pkg/sat/tactic"

	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTResourceLimitSourceCanceledCountsAndScopes(t *testing.T) {
	m := astTestManager(t)
	if m.Limit.HeaderCharged || m.Limit.Steps != 0 || m.Limit.Limit != 0 || m.Limit.CancelFlag() {
		t.Fatal("embedded source member has incorrect constructor state")
	}
	for i := uint64(1); i <= 3; i++ {
		canceled, e := m.Canceled()
		astTestMust(t, e)
		if canceled || m.Limit.Count() != i {
			t.Fatal("unlimited source canceled() must still increment", canceled, m.Limit.Count())
		}
	}
	scope, e := util.NewLimitScope(&m.Limit, 2)
	astTestMust(t, e)
	for i, want := range []bool{false, false, true} {
		got, e := m.Canceled()
		astTestMust(t, e)
		if got != want {
			t.Fatalf("scope step %d: canceled %v, want %v", i, got, want)
		}
	}
	scope.Close()
	if m.Limit.Count() != 5 {
		t.Fatal("source pop must clamp an exhausted scope count")
	}
	m.Limit.Cancel()
	canceled, e := m.Canceled()
	astTestMust(t, e)
	if !canceled || m.Limit.Count() != 6 {
		t.Fatal("source canceled() skipped increment under cancellation")
	}
	suspend := util.NewSuspendLimit(&m.Limit)
	canceled, e = m.Canceled()
	astTestMust(t, e)
	if canceled {
		t.Fatal("source suspension must permit canceled increments")
	}
	suspend.Close()
	m.Limit.ResetCancel()
	canceled, e = m.Canceled()
	astTestMust(t, e)
	if canceled {
		t.Fatal("cancellation reset did not permit work")
	}
}

func TestASTResourceLimitOwnedStorageAndIndependentFormat(t *testing.T) {
	for _, sourceFinalize := range []bool{false, true} {
		t.Run(map[bool]string{false: "close", true: "source-finalize"}[sourceFinalize], func(t *testing.T) {
			m := astTestManager(t)
			child, e := util.NewReslimit(m.BudgetField)
			astTestMust(t, e)
			defer child.Close()
			before := m.BudgetField.Used() - m.Stats().AllocatedBytes
			astTestMust(t, m.Limit.Push(10))
			astTestMust(t, m.Limit.PushChild(child))
			m.Limit.Cancel()
			if !child.CancelFlag() {
				t.Fatal("registered child did not receive cancellation")
			}
			canceled, e := m.Format.Canceled()
			astTestMust(t, e)
			if canceled || m.Format.Limit.Count() != 1 {
				t.Fatal("format manager must construct its own independent source limit")
			}
			astTestMust(t, m.Limit.Push(3))
			if m.BudgetField.Used()-m.Stats().AllocatedBytes != before {
				t.Fatal("AST manager stats omitted owned limit vectors")
			}
			if sourceFinalize {
				astTestMust(t, m.SourceFinalize())
				if !m.Limit.ClosedField || cap(m.Limit.Limits) != 0 || cap(m.Limit.Children) != 0 {
					t.Fatal("source member destructor did not consume limit storage")
				}
			}
			astTestMust(t, m.Close())
			if !m.Limit.ClosedField || child.ClosedField {
				t.Fatal("manager must destroy its limit and preserve borrowed child")
			}
			if _, e := m.Canceled(); e == nil {
				t.Fatal("closed manager accepted canceled()")
			}
		})
	}
}

func TestSATGoalTranslatorUsesASTManagerResourceLimit(t *testing.T) {
	for _, cause := range []string{"cancel", "steps", "solver-only"} {
		t.Run(cause, func(t *testing.T) {
			m := astTestManager(t)
			p := goalTestAtom(t, m, "limit-p")
			h, e := m.MkNot(p)
			astTestMust(t, e)
			astTestPin(t, m, h)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := sattactic.NewGoalTranslator(m, s)
			astTestMust(t, e)
			t.Cleanup(func() { astTestMust(t, tr.Close()) })
			if tr.Limit != &m.Limit {
				t.Fatal("goal2sat borrowed the solver limit instead of AST manager limit")
			}
			switch cause {
			case "cancel":
				m.Limit.Cancel()
			case "steps":
				astTestMust(t, m.Limit.Push(1))
				m.Limit.Inc()
			case "solver-only":
				s.Limit.Cancel()
			}
			e = tr.Compile([]ast.ASTHandle{h})
			if cause == "solver-only" {
				astTestMust(t, e)
			} else {
				want := util.Z3CanceledMessage
				if cause == "steps" {
					want = util.Z3MaxResourceMessage
				}
				if e == nil || !strings.Contains(e.Error(), want) {
					t.Fatalf("source checkpoint: %v, want %s", e, want)
				}
			}
		})
	}
}
