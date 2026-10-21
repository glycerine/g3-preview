// Source: src/ast/rewriter/rewriter.h:365-408;
// src/ast/rewriter/rewriter_def.h:659-750; src/model/model_evaluator.cpp:282-289.
package rewriter

import (
	"errors"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/messages"
	"testing"
)

type traversalClient struct {
	DefaultConfiguration
	manager                         *ast.ASTManager
	decl, result                    ast.ASTHandle
	resourceError                   error
	limit                           uint32
	calls, checks, resets, cleanups uint32
}

func (c *traversalClient) ReduceApp(d ast.ASTHandle, args ast.AppArguments, out, proof ast.ASTRef) (RewriteStatus, error) {
	c.calls++
	if d == c.decl {
		return RewriteDone, c.manager.SetRef(out, c.result)
	}
	return RewriteFailed, nil
}
func (c *traversalClient) MaxStepsExceeded(n uint32) (bool, error) {
	c.checks++
	return c.limit != 0 && n > c.limit, c.resourceError
}
func (c *traversalClient) Reset()   { c.resets++ }
func (c *traversalClient) Cleanup() { c.cleanups++ }

func TestTraversalClientResourceFailureAndResume(t *testing.T) {
	m := astTestManager(t)
	a, b := goalTestAtom(t, m, "public-a"), goalTestAtom(t, m, "public-b")
	d := rewriterTestDecl(t, m, "public-f")
	term := rewriterTestApp(t, m, d, a)
	out := astTestPin(t, m, b)
	failure := errors.New(messages.MaxMemory)
	cfg := traversalClient{manager: m, decl: d, result: a, resourceError: failure}
	rw, e := NewTraversal(m, &cfg)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	if e = rw.Rewrite(term, out); !errors.Is(e, failure) {
		t.Fatal("resource exception lost", e)
	}
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != b || cfg.calls != 0 || cfg.checks != 1 {
		t.Fatal("resource check must precede reduction", got, cfg.calls, cfg.checks)
	}
	cfg.resourceError = nil
	satOK(t, rw.Resume(out))
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != a || cfg.calls != 2 {
		t.Fatal("resume", got, cfg.calls)
	}
	satOK(t, rw.Reset())
	cfg.limit = 1
	if e = rw.Rewrite(rewriterTestApp(t, m, d, term), out); e == nil || e.Error() != messages.MaxSteps {
		t.Fatal("source step limit", e)
	}
	cfg.limit = 0
	satOK(t, rw.Resume(out))
	satOK(t, rw.Cleanup())
	if cfg.resets != 1 || cfg.cleanups != 1 {
		t.Fatal("configuration lifecycle", cfg.resets, cfg.cleanups)
	}
	copyOwner := *rw
	if e = copyOwner.Rewrite(term, out); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied owner", e)
	}
	satOK(t, rw.Close())
	if e = rw.Rewrite(term, out); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed owner", e)
	}
}

func TestTraversalAtomicRootDoesNotCheckResourceCallback(t *testing.T) {
	m := astTestManager(t)
	v := shiftTestVar(t, m, 0)
	out := astTestPin(t, m, ast.ASTHandle{})
	cfg := traversalClient{manager: m, resourceError: errors.New("no frame")}
	rw, e := NewTraversal(m, &cfg)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	satOK(t, rw.Rewrite(v, out))
	if cfg.checks != 0 {
		t.Fatal("source checks resources only in frame loop")
	}
}

func TestTraversalBindingsAndDisabledCancellation(t *testing.T) {
	m := astTestManager(t)
	v0, v1 := shiftTestVar(t, m, 0), shiftTestVar(t, m, 1)
	a, b := goalTestAtom(t, m, "bound-a"), goalTestAtom(t, m, "bound-b")
	term := goalTestApp(t, m, ast.ASTOpOr, v0, v1)
	out := astTestPin(t, m, ast.ASTHandle{})
	cfg := traversalClient{manager: m}
	rw, e := NewTraversal(m, &cfg)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	satOK(t, rw.SetBindings([]ast.ASTHandle{a, b}))
	satOK(t, rw.SetCancelCheck(false))
	m.Limit.IncCancel()
	e = rw.Rewrite(term, out)
	m.Limit.DecCancel()
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != goalTestApp(t, m, ast.ASTOpOr, a, b) {
		t.Fatal("direct bindings", got)
	}
	n, e := rw.NumSteps()
	satOK(t, e)
	if n == 0 {
		t.Fatal("source frame step counter")
	}
	satOK(t, rw.Reset())
	satOK(t, rw.SetInvBindings([]ast.ASTHandle{a, b}))
	satOK(t, rw.Rewrite(term, out))
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != goalTestApp(t, m, ast.ASTOpOr, b, a) {
		t.Fatal("inverse bindings", got)
	}
}

func TestTraversalColdConstructionCleanup(t *testing.T) {
	m := astTestManager(t)
	cfg := traversalClient{manager: m}
	warm, e := NewTraversal(m, &cfg)
	satOK(t, e)
	satOK(t, warm.Close())
	baseline := m.Budget().Used()
	for fault := int64(0); fault < 128; fault++ {
		m.Budget().FailAfter(fault)
		rw, e := NewTraversal(m, &cfg)
		m.Budget().FailAfter(-1)
		if rw != nil {
			satOK(t, rw.Close())
		}
		if m.Budget().Used() != baseline {
			t.Fatal("partial owner leak", fault, baseline, m.Budget().Used())
		}
		if e == nil {
			return
		}
	}
	t.Fatal("constructor never completed")
}

func TestTraversalProofCongruenceRewriteAndReflexivity(t *testing.T) {
	m := astTestManager(t)
	satOK(t, m.ToggleProofMode(ast.ASTProofEnabled))
	a, b := goalTestAtom(t, m, "proof-a"), goalTestAtom(t, m, "proof-b")
	f, g := rewriterTestDecl(t, m, "proof-f"), rewriterTestDecl(t, m, "proof-g")
	inner := rewriterTestApp(t, m, g, a)
	term := rewriterTestApp(t, m, f, inner)
	want := rewriterTestApp(t, m, f, b)
	cfg := traversalClient{manager: m, decl: g, result: b}
	rw, e := NewProofTraversal(m, &cfg)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	out := astTestPin(t, m, ast.ASTHandle{})
	proof := astTestPin(t, m, ast.ASTHandle{})
	satOK(t, rw.RewriteWithProof(term, out, proof))
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != want {
		t.Fatal("proof traversal value", got, want)
	}
	p, e := m.RefGet(proof)
	satOK(t, e)
	fact, e := m.GetProofFact(p)
	satOK(t, e)
	if fact != goalTestApp(t, m, ast.ASTOpIff, term, want) {
		t.Fatal("proof traversal fact", fact)
	}
	satOK(t, rw.Reset())
	satOK(t, rw.RewriteWithProof(a, out, proof))
	p, e = m.RefGet(proof)
	satOK(t, e)
	yes, e := m.IsProofKind(p, ast.ASTPrReflexivity)
	satOK(t, e)
	if !yes {
		t.Fatal("unchanged root needs explicit reflexivity")
	}
	if e = rw.SetBindings([]ast.ASTHandle{a}); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("proof traversal bindings guard", e)
	}
}
