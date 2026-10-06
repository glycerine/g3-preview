// Source: src/ast/ast.h:1455,1549-1550; src/util/rlimit.cpp:23-71;
// src/sat/tactic/goal2sat.cpp:384-392.
// Source authors: Leonardo de Moura and Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"strings"
	"testing"
)

func TestASTResourceLimitSourceCanceledCountsAndScopes(t *testing.T) {
	m := astTestManager(t)
	if m.limit.headerCharged || m.limit.steps != 0 || m.limit.limit != 0 || m.limit.cancelFlag() {
		t.Fatal("embedded source member has incorrect constructor state")
	}
	for i := uint64(1); i <= 3; i++ {
		canceled, e := m.Canceled()
		astTestMust(t, e)
		if canceled || m.limit.count() != i {
			t.Fatal("unlimited source canceled() must still increment", canceled, m.limit.count())
		}
	}
	scope, e := newUtilLimitScope(&m.limit, 2)
	astTestMust(t, e)
	for i, want := range []bool{false, false, true} {
		got, e := m.Canceled()
		astTestMust(t, e)
		if got != want {
			t.Fatalf("scope step %d: canceled %v, want %v", i, got, want)
		}
	}
	scope.close()
	if m.limit.count() != 5 {
		t.Fatal("source pop must clamp an exhausted scope count")
	}
	m.limit.cancel()
	canceled, e := m.Canceled()
	astTestMust(t, e)
	if !canceled || m.limit.count() != 6 {
		t.Fatal("source canceled() skipped increment under cancellation")
	}
	suspend := newUtilSuspendLimit(&m.limit)
	canceled, e = m.Canceled()
	astTestMust(t, e)
	if canceled {
		t.Fatal("source suspension must permit canceled increments")
	}
	suspend.close()
	m.limit.resetCancel()
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
			child, e := newUtilReslimit(m.budget)
			astTestMust(t, e)
			defer child.close()
			before := m.budget.Used() - m.Stats().AllocatedBytes
			astTestMust(t, m.limit.push(10))
			astTestMust(t, m.limit.pushChild(child))
			m.limit.cancel()
			if !child.cancelFlag() {
				t.Fatal("registered child did not receive cancellation")
			}
			canceled, e := m.format.Canceled()
			astTestMust(t, e)
			if canceled || m.format.limit.count() != 1 {
				t.Fatal("format manager must construct its own independent source limit")
			}
			astTestMust(t, m.limit.push(3))
			if m.budget.Used()-m.Stats().AllocatedBytes != before {
				t.Fatal("AST manager stats omitted owned limit vectors")
			}
			if sourceFinalize {
				astTestMust(t, m.SourceFinalize())
				if !m.limit.closed || cap(m.limit.limits) != 0 || cap(m.limit.children) != 0 {
					t.Fatal("source member destructor did not consume limit storage")
				}
			}
			astTestMust(t, m.Close())
			if !m.limit.closed || child.closed {
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
			tr, e := newSATGoalTranslator(m, s)
			astTestMust(t, e)
			t.Cleanup(func() { astTestMust(t, tr.Close()) })
			if tr.limit != &m.limit {
				t.Fatal("goal2sat borrowed the solver limit instead of AST manager limit")
			}
			switch cause {
			case "cancel":
				m.limit.cancel()
			case "steps":
				astTestMust(t, m.limit.push(1))
				m.limit.inc()
			case "solver-only":
				s.limit.cancel()
			}
			e = tr.Compile([]ASTHandle{h})
			if cause == "solver-only" {
				astTestMust(t, e)
			} else {
				want := Z3CanceledMessage
				if cause == "steps" {
					want = Z3MaxResourceMessage
				}
				if e == nil || !strings.Contains(e.Error(), want) {
					t.Fatalf("source checkpoint: %v, want %s", e, want)
				}
			}
		})
	}
}
