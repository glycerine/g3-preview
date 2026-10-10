package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTDependencyNativeDeletionCompletesPhysicalGraphImmediately(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1900)
	baseline := m.allocator.Stats().Live
	var root ASTDependency
	for i := 0; i < 80; i++ {
		leaf, e := m.DependencyLeaf(h)
		astTestMust(t, e)
		root, e = m.DependencyJoin(root, leaf)
		astTestMust(t, e)
	}
	astTestMust(t, m.DependencyInc(root))
	m.BudgetField.FailAfter(0)
	e := m.DependencyDec(root)
	astTestMust(t, e)
	if m.dependencies.live != 0 || m.allocator.Stats().Live != baseline {
		t.Fatal("native deletion retained dependency nodes", m.dependencies.live, m.allocator.Stats().Live, baseline)
	}
	if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	m.BudgetField.FailAfter(-1)
}

func TestASTDependencyNativeOutputGrowthUnmarksAndBorrowsReferences(t *testing.T) {

	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1901)
	leaf, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	before := m.BudgetField.Used()
	rc, e := m.GetRefCount(h)
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	out, e := m.DependencyLinearizeRaw(leaf, []ASTHandle{h})
	astTestMust(t, e)
	if len(out) != 2 || out[0] != h || out[1] != h {
		t.Fatal(out)
	}
	c, e := m.dependencies.node(leaf.record)
	astTestMust(t, e)
	if c.marked || len(m.dependencies.todo) != 0 || m.BudgetField.Used() != before {
		t.Fatal("native append/traversal cleanup", c.marked, len(m.dependencies.todo))
	}
	if got, e := m.GetRefCount(h); e != nil || got != rc {
		t.Fatal("output must not retain nodes", got, rc, e)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.DependencyInc(leaf))
	astTestMust(t, m.DependencyDec(leaf))

}

func TestASTDependencyNativeStableDedupPreservesNodeMarks(t *testing.T) {

	m := astTestManager(t)
	var hs [257]ASTHandle
	var out []ASTHandle
	for i := range hs {
		hs[i] = astDependencyExpr(t, m, uint32(2100+i))
		out = append(out, hs[i], hs[i])
	}
	// A caller's mark plane belongs to that caller, independently of deduplication.
	mark, e := newASTFastMark1Locked(m)
	astTestMust(t, e)
	astTestMust(t, mark.markLocked(m, hs[16], true))
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	out, e = m.DependencyLinearize(ASTDependency{}, out)
	astTestMust(t, e)
	if len(out) != len(hs) || m.BudgetField.Used() != before {
		t.Fatal(len(out), m.BudgetField.Used(), before)
	}
	for i, h := range out {
		if h != hs[i] {
			t.Fatal("first-occurrence order", i, h, hs[i])
		}
	}
	if marked, e := mark.nodeMarkedLocked(m, hs[16]); e != nil || !marked {
		t.Fatal("borrowed caller mark lost", marked, e)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, mark.closeLocked(m))

}

func TestASTDependencyJoinValuesFailurePublishesBothScalarRoots(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 2200)
	var values [600]ASTHandle
	for i := range values {
		values[i] = h
	}
	sawPending := false
	sawSuccess := false
	for fault := int64(0); fault < 100; fault++ {
		physical := m.allocator.Stats().Live
		m.BudgetField.FailAfter(fault)
		n := len(values)
		if ASTSourceDebug {
			n = 8
		}
		p, e := m.DependencyJoinValues(values[:n])
		m.BudgetField.FailAfter(-1)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fault, e)
		}
		if !p.PendingLeaf.IsNull() {
			sawPending = true
		}
		for _, d := range [2]ASTDependency{p.Completed, p.PendingLeaf} {
			if !d.IsNull() {
				astTestMust(t, m.DependencyInc(d))
				astTestMust(t, m.DependencyDec(d))
			}
		}
		if m.dependencies.live != 0 || m.allocator.Stats().Live != physical {
			t.Fatal("lost constructor prefix before Close", fault, m.dependencies.live, m.allocator.Stats().Live, physical)
		}
		if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
			t.Fatal("actual leaf ownership", fault, rc, e)
		}
		if e == nil {
			sawSuccess = true
			break
		}
	}
	if !sawSuccess {
		t.Fatal("fault sweep never reached successful full operation")
	}
	// Pooled reuse can remove a later join reservation; debug's direct allocator
	// necessarily exercises the completed+pending-leaf publication boundary.
	if ASTSourceDebug && !sawPending {
		t.Fatal("no unjoined pending leaf failure observed")
	}
}

func TestASTDependencyEmptyOwnersConstructorFailurePrefix(t *testing.T) {
	s, q, b := astTestOwners(t)
	baseline := b.Used()
	b.FailAfter(1) // AST cold header completes; allocator constructor then refuses.
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	b.FailAfter(-1)
	if m != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != baseline {
		t.Fatal("empty dependency owner teardown", m, e, b.Used(), baseline)
	}
}

func TestASTDependencyJoinValuesLaterInvalidInputRetainsEarlierLeafPrefix(t *testing.T) {
	m := astTestManager(t)
	foreign := astTestManager(t)
	good := astDependencyExpr(t, m, 6000)
	bad := astDependencyExpr(t, foreign, 6001)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	stale, e := m.MkVar(6002, s)
	astTestMust(t, e)
	pin, e := m.NewRef(stale)
	astTestMust(t, e)
	astTestMust(t, m.DelRef(&pin))
	for _, invalid := range [2]ASTHandle{bad, stale} {
		p, e := m.DependencyJoinValues([]ASTHandle{good, invalid})
		if !errors.Is(e, ErrASTHandle) || p.Completed.IsNull() || !p.PendingLeaf.IsNull() {
			t.Fatal("earlier actual leaf allocation/INC publication", p, e)
		}
		if rc, e := m.GetRefCount(good); e != nil || rc != 2 || m.dependencies.live != 1 {
			t.Fatal("source completed leaf prefix", rc, e, m.dependencies.live)
		}
		astTestMust(t, m.DependencyInc(p.Completed))
		astTestMust(t, m.DependencyDec(p.Completed))
		if rc, e := m.GetRefCount(good); e != nil || rc != 1 || m.dependencies.live != 0 {
			t.Fatal(rc, e, m.dependencies.live)
		}
	}
}

func TestASTDependencyNativeOutputInvalidHandleRetainsCompactedPrefix(t *testing.T) {
	m := astTestManager(t)
	foreign := astTestManager(t)
	a := astDependencyExpr(t, m, 2600)
	b := astDependencyExpr(t, m, 2601)
	bad := astDependencyExpr(t, foreign, 2602)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	stale, e := m.MkVar(2603, sort)
	astTestMust(t, e)
	pin, e := m.NewRef(stale)
	astTestMust(t, e)
	astTestMust(t, m.DelRef(&pin))
	for _, tc := range []struct {
		value ASTHandle
		want  error
	}{
		{bad, ErrASTHandle}, {stale, ErrASTHandle}, {sort, ErrASTDomain},
	} {
		out := []ASTHandle{a, a, b, tc.value}
		out, e = m.DependencyLinearize(ASTDependency{}, out)
		if !errors.Is(e, tc.want) || len(out) != 4 || out[0] != a || out[1] != b || out[3] != tc.value {
			t.Fatal("checked input retains completed compaction and original length", out, e)
		}
	}
}
