// Source: src/test/sorting_network.cpp:150-181.
// Source author: Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPsortASTContextActualFactoriesClausesAndTrail(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	x, e := c.Fresh()
	astTestMust(t, e)
	y, e := c.Fresh()
	astTestMust(t, e)
	n, e := c.Not(x)
	astTestMust(t, e)
	z, e := c.Not(n)
	astTestMust(t, e)
	if z != x {
		t.Fatal("source double-not returns borrowed child", z, x)
	}
	a, e := c.Min(x, y)
	astTestMust(t, e)
	b, e := c.Max(x, y)
	astTestMust(t, e)
	if a == b {
		t.Fatal("actual AND and OR")
	}
	astTestMust(t, c.Clause(0, ASTLiteralVectorSpan{}))
	trail, e := m.RefVectorSize(c.trail)
	astTestMust(t, e)
	clauses, e := m.RefVectorSize(c.clauses)
	astTestMust(t, e)
	if trail != 5 || clauses != 1 {
		t.Fatal("literal factory trail and independent clauses", trail, clauses)
	}
	astTestMust(t, c.SourceFinalize())
	astTestMust(t, c.Close())
	if _, e = m.GetRefCount(x); e == nil {
		t.Fatal("source owned trail reference remained")
	}
}

func TestPsortASTContextUnpublishedClauseConsumedCleanupRetry(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(8000+i), sort)
		astTestMust(t, e)
	}
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget})
	astTestMust(t, e)
	v, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
	astTestMust(t, e)
	astTestMust(t, v.Push(h))
	span, e := v.CPtr()
	astTestMust(t, e)
	external := m.budget.Used() - m.Stats().AllocatedBytes - r.Stats().TotalBytes - c.header - v.header
	graph := func() {
		cold := external + c.header + v.header
		for f := c.pending; f != nil; f = f.next {
			if f.owner != c || f.receipt.owner != m || !f.receipt.acquired || f.receipt.published {
				t.Fatal("exact owned INC frame")
			}
			cold += f.header
		}
		if m.budget.Used() != m.Stats().AllocatedBytes+r.Stats().TotalBytes+cold {
			t.Fatal("named actual owner cold graph", m.budget.Used(), cold)
		}
	}
	// The real cold frame succeeds, then the source reference vector expansion
	// fails after INC, leaving the clause pointer unpublished.
	m.budget.FailAfter(1)
	cause := c.Clause(1, span)
	if !errors.Is(cause, arena.ErrInjectedFailure) || c.pending == nil {
		t.Fatal("actual clause source prefix", cause)
	}
	refs, e := m.GetRefCount(h)
	astTestMust(t, e)
	if refs != 1 {
		t.Fatal("unjoined INC", refs)
	}
	graph()
	astTestMust(t, c.SourceFinalize())
	if c.stage != 2 || c.pending == nil {
		t.Fatal("source members must not consume unjoined INC")
	}
	m.budget.FailAfter(0)
	cleanup := c.Close()
	if !errors.Is(cleanup, arena.ErrInjectedFailure) || c.pending == nil || !c.pending.receipt.consumed {
		t.Fatal("actual consumed deletion warning", cleanup)
	}
	first := c.pending.receipt.cleanupCause
	graph()
	astTestMust(t, c.Close())
	if first != cleanup || !c.closed || c.pending != nil {
		t.Fatal("owning exact consumed ACK")
	}
	graph()
	m.budget.FailAfter(-1)
	astTestMust(t, v.Close())
	astTestMust(t, r.Close())
}

func TestPsortASTContextActualConstructorRefusalPrefixes(t *testing.T) {
	success := int64(-1)
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		m := astTestManager(t)
		external := m.budget.Used() - m.Stats().AllocatedBytes
		m.budget.FailAfter(ordinal)
		c, cause := NewPsortASTContext(m)
		cold := external
		if c != nil {
			cold += c.header
			if c.constructed != (cause == nil) {
				t.Fatal("actual complete constructor publication", ordinal, cause)
			}
		}
		if m.budget.Used() != m.Stats().AllocatedBytes+cold {
			t.Fatal("immediate failed child/header graph", ordinal, m.budget.Used(), cold)
		}
		if c != nil && cause != nil && c.completed != 0 {
			t.Fatal("completed source member unwind", ordinal, c.completed, cause)
		}
		m.budget.FailAfter(-1)
		if c != nil {
			astTestMust(t, c.Close())
		}
		if cause == nil {
			success = ordinal
			break
		}
	}
	if success < 0 {
		t.Fatal("constructor actual first success not reached")
	}
	t.Log("actual constructor first success", success)
}

func TestPsortASTContextStoppedTrailKeepsLaterClausesAndSourceCause(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(10000+i), sort)
		astTestMust(t, e)
	}
	wide, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	f, e := m.MkFalse()
	astTestMust(t, e)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	_, e = c.Max(wide, f)
	astTestMust(t, e)
	astTestMust(t, c.Clause(0, ASTLiteralVectorSpan{}))
	external := m.budget.Used() - m.Stats().AllocatedBytes - c.header
	m.budget.FailAfter(0)
	cause := c.SourceFinalize()
	if !errors.Is(cause, arena.ErrInjectedFailure) || c.stage != 0 || c.destructorCause != cause {
		t.Fatal("actual first reached trail stop", cause, c.stage)
	}
	size, e := m.RefVectorSize(c.clauses)
	astTestMust(t, e)
	if size != 1 {
		t.Fatal("later declared clause member traversed after source stop", size)
	}
	if m.budget.Used() != m.Stats().AllocatedBytes+c.header+external {
		t.Fatal("actual stopped member graph")
	}
	before := m.Stats()
	if c.SourceFinalize() != cause || m.Stats() != before {
		t.Fatal("source teardown replay or changed sticky cause")
	}
	// The real child helper has consumed the dead references but retained its
	// SIZE-zero descriptor. Owning close finishes that child before clauses.
	astTestMust(t, c.Close())
	if c.destructorCause != cause || c.stage != 2 || !c.closed {
		t.Fatal("owning continuation lost source boundary")
	}
	m.budget.FailAfter(-1)
}
