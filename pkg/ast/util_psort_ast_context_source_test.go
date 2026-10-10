// Source: src/test/sorting_network.cpp:150-181.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
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

func TestPsortASTContextUnpublishedClauseNativeCleanup(t *testing.T) {
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
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	v, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	astTestMust(t, v.Push(h))
	span, e := v.Span()
	astTestMust(t, e)
	external := m.BudgetField.Used() - m.Stats().AllocatedBytes - r.Stats().TotalBytes - c.header - v.header
	graph := func() {
		cold := external + c.header + v.header
		for f := c.pending; f != nil; f = f.next {
			if f.owner != c || f.receipt.owner != m || !f.receipt.acquired || f.receipt.published {
				t.Fatal("exact owned INC frame")
			}
			cold += f.header
		}
		if m.BudgetField.Used() != m.Stats().AllocatedBytes+r.Stats().TotalBytes+cold {
			t.Fatal("named actual owner cold graph", m.BudgetField.Used(), cold)
		}
	}
	// The real cold frame succeeds, then the source reference vector expansion
	// fails after INC, leaving the clause pointer unpublished.
	m.BudgetField.FailAfter(1)
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
	m.BudgetField.FailAfter(0)
	cleanup := c.Close()
	astTestMust(t, cleanup)
	if !c.closed || c.pending != nil {
		t.Fatal("native cleanup did not acknowledge the acquired reference")
	}
	graph()
	astTestMust(t, c.Close())
	graph()
	m.BudgetField.FailAfter(-1)
	astTestMust(t, v.Close())
	astTestMust(t, r.Close())
}

func TestPsortASTContextActualConstructorRefusalPrefixes(t *testing.T) {
	success := int64(-1)
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		m := astTestManager(t)
		external := m.BudgetField.Used() - m.Stats().AllocatedBytes
		m.BudgetField.FailAfter(ordinal)
		c, cause := NewPsortASTContext(m)
		cold := external
		if c != nil {
			cold += c.header
			if c.constructed != (cause == nil) {
				t.Fatal("actual complete constructor publication", ordinal, cause)
			}
		}
		if m.BudgetField.Used() != m.Stats().AllocatedBytes+cold {
			t.Fatal("immediate failed child/header graph", ordinal, m.BudgetField.Used(), cold)
		}
		if c != nil && cause != nil && c.completed != 0 {
			t.Fatal("completed source member unwind", ordinal, c.completed, cause)
		}
		m.BudgetField.FailAfter(-1)
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

func TestPsortASTContextNativeWideTrailAndClauseFinalization(t *testing.T) {
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
	external := m.BudgetField.Used() - m.Stats().AllocatedBytes - c.header
	m.BudgetField.FailAfter(0)
	cause := c.SourceFinalize()
	astTestMust(t, cause)
	if c.stage != 2 || c.destructorCause != nil {
		t.Fatal("native trail and clause finalization", c.stage, c.destructorCause)
	}
	if m.BudgetField.Used() != m.Stats().AllocatedBytes+c.header+external {
		t.Fatal("actual finalized member graph")
	}
	before := m.Stats()
	astTestMust(t, c.SourceFinalize())
	if m.Stats() != before {
		t.Fatal("repeated finalization changed AST ownership")
	}
	astTestMust(t, c.Close())
	if c.destructorCause != nil || c.stage != 2 || !c.closed {
		t.Fatal("native owning closure")
	}
	m.BudgetField.FailAfter(-1)
}
