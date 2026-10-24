package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestContextSimplifyConstructorAllocationFrontiers(t *testing.T) {
	success := false
	for stop := int64(0); stop < 1024; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m, c, g, services := solveEqFixture(t)
			simplifyMust(t, m.RegisterDeclarationPlugins())
			m.Budget().FailAfter(stop)
			r, err := NewContextSimplifyTactic(m, c, g, nil, services)
			m.Budget().FailAfter(-1)
			if r != nil {
				simplifyMust(t, r.Close())
			}
			var stopped memory.ConstructionFailure
			if errors.As(err, &stopped) {
				simplifyMust(t, stopped.Cleanup())
			}
			var failure memory.Failure
			if err == nil {
				success = true
			} else if !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &failure) && failure.Kind == memory.FailureMemout) {
				t.Fatal(err)
			}
		})
		if success {
			break
		}
	}
	if !success {
		t.Fatal("no successful constructor frontier")
	}
}

type ctxClosedWriter struct{}

func (ctxClosedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestContextSimplifyTraceScopeRestorationAndProofLifetime(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m, c, g, services := solveEqFixture(t)
			simplifyMust(t, m.RegisterDeclarationPlugins())
			if proofs {
				simplifyMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			}
			r, err := NewContextSimplifyTactic(m, c, g, nil, services)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, r.Close()) })
			p, q, _ := propagateAtoms(t, m)
			shared, err := m.MkOr(p, q)
			simplifyMust(t, err)
			simplifyPin(t, m, shared)
			root, err := m.MkIte(p, shared, shared)
			simplifyMust(t, err)
			simplifyPin(t, m, root)
			goal, err := tactic.NewGoal(m, proofs, true, false)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, goal.Close()) })
			simplifyMust(t, goal.Assert(root))
			var result tactic.TacticResult
			t.Cleanup(func() { simplifyMust(t, result.Close()) })
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			for _, name := range []string{"ctx_simplify_tactic", "ctx_simplify_tactic_cache", "ctx_simplify_tactic_detail", "ctx_simplify_tactic_bug", "ctx_simplify_tactic_ite_bug"} {
				tags.Enable(name)
			}
			simplifyMust(t, r.BindTrace(tags))
			simplifyMust(t, r.Apply(goal, &result))
			value, err := goal.Form(0)
			simplifyMust(t, err)
			if value != shared {
				t.Fatal("ITE scope simplification", value, shared)
			}
			if r.current.simp.ScopeLevel() != 0 {
				t.Fatal("scope leak")
			}
			if tags.ShouldTrace("ctx_simplify_tactic_cache") {
				for _, fragment := range []string{"cache_core src/tactic/core/ctx_simplify_tactic.cpp:234", "uncaching:", "restoring:\n<null>", "processing:", "result:\n", "mk_ite\n"} {
					if !strings.Contains(output.String(), fragment) {
						t.Fatal("missing source trace", fragment, output.String())
					}
				}
				if proofs && !strings.Contains(output.String(), "old size:") {
					t.Fatal("missing per-form proof-mode process trace", output.String())
				}
			} else if output.Len() != 0 {
				t.Fatal("erased trace emitted", output.String())
			}
			simplifyMust(t, r.Cleanup())
			simplifyMust(t, r.Close())
			if proofs {
				pr, err := goal.Proof(0)
				simplifyMust(t, err)
				fact, err := m.GetProofFact(pr)
				simplifyMust(t, err)
				if fact != value {
					t.Fatal("rewritten proof did not survive strategy close")
				}
			}
		})
	}
}

func TestContextSimplifyClosedTraceAndCleanupRefusal(t *testing.T) {
	m, c, g, services := solveEqFixture(t)
	simplifyMust(t, m.RegisterDeclarationPlugins())
	r, err := NewContextSimplifyTactic(m, c, g, nil, services)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, r.Close()) })
	p, q, _ := propagateAtoms(t, m)
	shared, err := m.MkOr(p, q)
	simplifyMust(t, err)
	simplifyPin(t, m, shared)
	root, err := m.MkIte(p, shared, shared)
	simplifyMust(t, err)
	simplifyPin(t, m, root)
	goal, err := tactic.NewGoal(m, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(root))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	tags := diagnostic.NewTraceTagsWithWriter(ctxClosedWriter{})
	tags.EnableAll(true)
	simplifyMust(t, r.BindTrace(tags))
	simplifyMust(t, r.Apply(goal, &result))
	before := r.current
	m.Budget().FailAfter(0)
	err = r.Cleanup()
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.current != before {
		t.Fatal("cleanup replaced the active implementation before successful construction", err)
	}
	simplifyMust(t, r.Cleanup())
}

func TestContextSimplifySoftLimitsCancellationAndFreshCleanup(t *testing.T) {
	m, c, g, services := solveEqFixture(t)
	simplifyMust(t, m.RegisterDeclarationPlugins())
	r, err := NewContextSimplifyTactic(m, c, g, nil, services)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, r.Close()) })
	p, q, _ := propagateAtoms(t, m)
	np, e := m.MkNot(p)
	simplifyMust(t, e)
	simplifyPin(t, m, np)
	inner, e := m.MkAnd(np, q)
	simplifyMust(t, e)
	simplifyPin(t, m, inner)
	root, e := m.MkOr(p, inner)
	simplifyMust(t, e)
	simplifyPin(t, m, root)
	settings, err := c.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, c.Del(&settings)) })
	key := "max_steps"
	simplifyMust(t, c.SetUIntName(&settings, &key, 0))
	simplifyMust(t, r.UpdateParams(&settings))
	goal, err := tactic.NewGoal(m, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(root))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	m.ResourceLimit().IncCancel()
	err = r.Apply(goal, &result)
	m.ResourceLimit().DecCancel()
	simplifyMust(t, err) // Cutoff precedes the checkpoint and returns the original term.
	h, err := goal.Form(0)
	simplifyMust(t, err)
	if h != root {
		t.Fatal("zero steps changed input")
	}
	simplifyMust(t, c.SetUIntName(&settings, &key, ^uint32(0)))
	simplifyMust(t, r.UpdateParams(&settings))
	m.ResourceLimit().IncCancel()
	err = r.Apply(goal, &result)
	m.ResourceLimit().DecCancel()
	var exception tactic.TacticException
	if !errors.As(err, &exception) {
		t.Fatal("cancellation lost tactic failure", err)
	}
	simplifyMust(t, r.Cleanup())
	simplifyMust(t, r.Apply(goal, &result))
	h, err = goal.Form(0)
	simplifyMust(t, err)
	want, err := m.MkOr(p, q)
	simplifyMust(t, err)
	if h != want {
		t.Fatal("scope state leaked after cancellation/cleanup", h, want)
	}
	if r.current.simp.ScopeLevel() != 0 {
		t.Fatal("active scopes after success")
	}
	if r.current.steps == 0 {
		t.Fatal("step accounting was not exercised")
	}
	simplifyMust(t, r.Cleanup())
	if r.current.steps != 0 || len(r.current.cache) != 0 {
		t.Fatal("cleanup did not rebuild fresh implementation")
	}
}

func TestContextSimplifyTranslationParameterSnapshot(t *testing.T) {
	m, c, g, services := solveEqFixture(t)
	simplifyMust(t, m.RegisterDeclarationPlugins())
	settings, err := c.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, c.Del(&settings)) })
	key := "max_steps"
	simplifyMust(t, c.SetUIntName(&settings, &key, 0))
	source, err := NewContextSimplifyTactic(m, c, g, &settings, services)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, source.Close()) })
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, target.Close()) })
	simplifyMust(t, target.RegisterDeclarationPlugins())
	translated, err := source.Translate(target)
	simplifyMust(t, err)
	simplifyMust(t, translated.IncRef())
	t.Cleanup(func() { simplifyMust(t, translated.DecRef()) })
	simplifyMust(t, c.SetUIntName(&settings, &key, ^uint32(0)))
	simplifyMust(t, source.UpdateParams(&settings))
	simplifyMust(t, source.Close())
	p, q, _ := propagateAtoms(t, target)
	np, err := target.MkNot(p)
	simplifyMust(t, err)
	simplifyPin(t, target, np)
	inner, err := target.MkAnd(np, q)
	simplifyMust(t, err)
	simplifyPin(t, target, inner)
	root, err := target.MkOr(p, inner)
	simplifyMust(t, err)
	simplifyPin(t, target, root)
	goal, err := tactic.NewGoal(target, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(root))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, translated.Apply(goal, &result))
	value, err := goal.Form(0)
	simplifyMust(t, err)
	if value != root {
		t.Fatal("source update changed translated soft-limit snapshot")
	}
	simplifyMust(t, translated.UpdateParams(&settings))
	simplifyMust(t, translated.Cleanup())
	simplifyMust(t, translated.Apply(goal, &result))
	value, err = goal.Form(0)
	simplifyMust(t, err)
	want, err := target.MkOr(p, q)
	simplifyMust(t, err)
	if value != want {
		t.Fatal("translated strategy did not use target manager", value, want)
	}
}
