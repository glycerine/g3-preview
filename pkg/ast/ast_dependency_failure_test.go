package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTDependencyDeletionRefusalCompletesPhysicalGraphImmediately(t *testing.T) {
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
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("first actual todo reserve", e)
	}
	if m.dependencies.live != 0 || m.allocator.Stats().Live != baseline {
		t.Fatal("lost intrusive deletion roots before Close", m.dependencies.live, m.allocator.Stats().Live, baseline)
	}
	if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	m.BudgetField.FailAfter(-1)
}

func TestASTDependencyTraversalFailureUnmarksAndRetainsOutputPrefix(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1901)
	leaf, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	out, e := m.NewExprVector()
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	e = m.DependencyLinearizeRaw(leaf, out)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	c, e := m.dependencies.node(leaf.record)
	astTestMust(t, e)
	if c.marked || len(m.dependencies.todo) != 0 {
		t.Fatal("unsafe allocation mark adapter", c.marked, len(m.dependencies.todo))
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.DependencyLinearizeRaw(leaf, out))
	n, e := m.ExprVectorSize(out)
	astTestMust(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	astTestMust(t, m.DelExprVector(&out))
	astTestMust(t, m.DependencyInc(leaf))
	astTestMust(t, m.DependencyDec(leaf))
}

func TestASTDependencyArrayDeletionASTFailureConsumesAllRoots(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	baseline := m.liveNodes
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(2000+i), b)
		astTestMust(t, e)
	}
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	leaf, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, leaf))
	m.BudgetField.FailAfter(0)
	e = m.DependencyArrayDel(&a)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if a != (ASTDependencyArray{}) || m.dependencies.live != 0 || m.dependencyArrays.liveCells != 0 || m.dependencyArrays.liveBlocks != 0 || m.liveNodes != baseline {
		t.Fatal("immediate owned callback graph", a, m.dependencies.live, m.dependencyArrays.liveCells, m.dependencyArrays.liveBlocks, m.liveNodes, baseline)
	}
	m.BudgetField.FailAfter(-1)
}

func TestASTDependencyArraySourceThirtyBitReleaseCounter(t *testing.T) {
	m := astTestManager(t)
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayMk(&a))
	c, e := m.dependencyArrays.cell(a.cell)
	astTestMust(t, e)
	c.rc = 0
	astTestMust(t, m.dependencyArrays.store(a.cell, c))
	old := a
	e = m.DependencyArrayDel(&a)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) || a != old {
			t.Fatal(e, a, old)
		}
		c.rc = 1
		astTestMust(t, m.dependencyArrays.store(a.cell, c))
		astTestMust(t, m.DependencyArrayDel(&a))
	} else {
		astTestMust(t, e)
		if a != (ASTDependencyArray{}) {
			t.Fatal(a)
		}
		c, e = m.dependencyArrays.cell(old.cell)
		astTestMust(t, e)
		if c.rc != astDependencyMaxRC {
			t.Fatal(c.rc)
		}
		// Explicit recovery of an unreachable-to-caller retained source counter.
		c.rc = 1
		astTestMust(t, m.dependencyArrays.store(old.cell, c))
		astTestMust(t, m.DependencyArrayDel(&old))
	}
}

func TestASTDependencyStableDedupFailurePreservesSourceDirtyMarkAndCompaction(t *testing.T) {
	m := astTestManager(t)
	out, e := m.NewExprVector()
	astTestMust(t, e)
	var hs [17]ASTHandle
	for i := range hs {
		hs[i] = astDependencyExpr(t, m, uint32(2100+i))
		astTestMust(t, m.ExprVectorPush(out, hs[i]))
		if i == 0 {
			astTestMust(t, m.ExprVectorPush(out, hs[i]))
		}
	}
	physical := m.allocator.Stats().Live
	m.BudgetField.FailAfter(1) // real cold FastMark owner succeeds; inline16 overflow refuses.
	e = m.DependencyLinearize(ASTDependency{}, out)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if m.allocator.Stats().Live != physical {
		t.Fatal("lost output/owner allocation")
	}
	if n, e := m.ExprVectorSize(out); e != nil || n != 18 {
		t.Fatal("size publishes after all compaction", n, e)
	}
	for i := 0; i < 16; i++ {
		x, e := m.ExprVectorGet(out, uint32(i))
		astTestMust(t, e)
		if x != hs[i] {
			t.Fatal("completed source set(j) prefix", i, x)
		}
		if m.nodeAt(hs[i].slot).mark1 {
			t.Fatal("listed mark not reset", i)
		}
	}
	last := m.nodeAt(hs[16].slot)
	if !last.mark1 {
		t.Fatal("source unlisted mark-before-buffer-push prefix was silently repaired")
	}
	owner := last.markOwner1
	m.BudgetField.FailAfter(-1)
	// Explicit caller recovery, rather than an invented wrapper rollback.
	astTestMust(t, m.ResetNodeMark1(hs[16], owner))
	astTestMust(t, m.DelExprVector(&out))
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

func TestASTDependencyArrayEveryOperationFaultOwnership(t *testing.T) {
	for op := 0; op < 6; op++ {
		finished := false
		for fault := int64(0); fault < 80; fault++ {
			m := astTestManager(t)
			h := astDependencyExpr(t, m, uint32(2300+op))
			d, e := m.DependencyLeaf(h)
			astTestMust(t, e)
			astTestMust(t, m.DependencyInc(d))
			physical := m.allocator.Stats().Live
			var a, b ASTDependencyArray
			for i := 0; i < 8; i++ {
				astTestMust(t, m.DependencyArrayPush(&a, d))
			}
			astTestMust(t, m.DependencyArrayCopy(&a, &b))
			if op >= 3 {
				astTestMust(t, m.DependencyArraySet(&a, 0, ASTDependency{}))
			}
			m.BudgetField.FailAfter(fault)
			switch op {
			case 0:
				e = m.DependencyArraySet(&a, 1, ASTDependency{})
			case 1:
				e = m.DependencyArrayPush(&a, d)
			case 2:
				e = m.DependencyArrayPop(&a)
			case 3:
				e = m.DependencyArrayUnshare(&b)
			case 4:
				e = m.DependencyArrayUnfold(&b)
			case 5:
				e = m.DependencyArrayReroot(&b)
			}
			m.BudgetField.FailAfter(-1)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(op, fault, e)
			}
			astTestMust(t, m.DependencyArrayDel(&a))
			astTestMust(t, m.DependencyArrayDel(&b))
			if m.dependencyArrays.liveCells != 0 || m.dependencyArrays.liveBlocks != 0 || m.dependencies.live != 1 || m.allocator.Stats().Live != physical {
				t.Fatal("lost physical array prefix before Close", op, fault, m.dependencyArrays.liveCells, m.dependencyArrays.liveBlocks, m.dependencies.live, m.allocator.Stats().Live, physical)
			}
			astTestMust(t, m.DependencyDec(d))
			if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
				t.Fatal(op, fault, rc, e)
			}
			if e == nil {
				finished = true
				break
			}
		}
		if !finished {
			t.Fatal("fault sweep did not reach success", op)
		}
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

func TestASTDependencyArrayRetainedFreedValueSetPreservesIncomingIncrement(t *testing.T) {
	m := astTestManager(t)
	old, e := m.DependencyLeaf(astDependencyExpr(t, m, 6100))
	astTestMust(t, e)
	incoming, e := m.DependencyLeaf(astDependencyExpr(t, m, 6101))
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(incoming))
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, old))
	astTestMust(t, m.DependencyArrayPop(&a))
	e = m.DependencyArraySet(&a, 0, incoming)
	rc, x := m.DependencyRefCount(incoming)
	astTestMust(t, x)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) || rc != 1 {
			t.Fatal(e, rc)
		}
	} else {
		if !errors.Is(e, ErrASTHandle) || rc != 2 {
			t.Fatal("INC-before-invalid old callback prefix", e, rc)
		}
		astTestMust(t, m.DependencyDec(incoming))
	}
	astTestMust(t, m.DependencyArrayDel(&a))
	astTestMust(t, m.DependencyDec(incoming))
	if m.dependencies.live != 0 {
		t.Fatal("explicit caller prefix recovery", m.dependencies.live)
	}
}

func TestASTDependencyArrayWrappedValueCounterRefusalPreservesSlot(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 2400)
	leaf, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	c, e := m.dependencies.node(leaf.record)
	astTestMust(t, e)
	// Seed the independently tested 30-bit boundary instead of executing a
	// billion INC calls; the actual array callback performs the wrapping INC.
	c.rc = astDependencyMaxRC
	astTestMust(t, m.dependencies.store(leaf.record, c))
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, leaf))
	c, e = m.dependencies.node(leaf.record)
	astTestMust(t, e)
	if c.rc != 0 {
		t.Fatal("actual INC did not wrap", c.rc)
	}
	physical := m.allocator.Stats().Live
	e = m.DependencyArraySet(&a, 0, ASTDependency{})
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
		got, e := m.DependencyArrayGet(&a, 0)
		astTestMust(t, e)
		if got != leaf {
			t.Fatal("refused DEC replaced source slot", got, leaf)
		}
		if m.allocator.Stats().Live != physical {
			t.Fatal("refused DEC consumed physical storage")
		}
		c.rc = 1
		astTestMust(t, m.dependencies.store(leaf.record, c))
		astTestMust(t, m.DependencyArrayDel(&a))
	} else {
		astTestMust(t, e)
		got, e := m.DependencyArrayGet(&a, 0)
		astTestMust(t, e)
		if got != (ASTDependency{}) {
			t.Fatal(got)
		}
		c, e = m.dependencies.node(leaf.record)
		astTestMust(t, e)
		if c.rc != astDependencyMaxRC {
			t.Fatal(c.rc)
		}
		astTestMust(t, m.DependencyArrayDel(&a))
		c.rc = 1
		astTestMust(t, m.dependencies.store(leaf.record, c))
		astTestMust(t, m.DependencyDec(leaf))
	}
}

func TestASTDependencyArrayMalformedCopyReadDropsAcquiredPrefix(t *testing.T) {
	m := astTestManager(t)
	first, e := m.DependencyLeaf(astDependencyExpr(t, m, 2401))
	astTestMust(t, e)
	second, e := m.DependencyLeaf(astDependencyExpr(t, m, 2402))
	astTestMust(t, e)
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, first))
	astTestMust(t, m.DependencyArrayPush(&a, second))
	c, e := m.dependencyArrays.cell(a.cell)
	astTestMust(t, e)
	// Deliberately malformed adapter metadata, not a valid source root: only
	// the first read can succeed. Check acquired references before teardown.
	astTestMust(t, m.dependencyArrays.write(c.link, util.PArrayCapacityHeader, 4, 1))
	live := m.allocator.Stats().Live
	blocks := m.dependencyArrays.liveBlocks
	block, _, e := m.dependencyArrays.getValues(a.cell)
	if !errors.Is(e, arena.ErrBounds) || !util.PArrayNull(block) {
		t.Fatal(block, e)
	}
	if rc, e := m.DependencyRefCount(first); e != nil || rc != 1 {
		t.Fatal("lost copied-prefix INC", rc, e)
	}
	if m.allocator.Stats().Live != live || m.dependencyArrays.liveBlocks != blocks {
		t.Fatal("lost temporary backing before Close")
	}
	astTestMust(t, m.dependencyArrays.write(c.link, util.PArrayCapacityHeader, 4, 2))
	astTestMust(t, m.DependencyArrayDel(&a))
}

func TestASTDependencyArrayRetainedFreedGetReturnsRawToken(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 2403))
	astTestMust(t, e)
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, d))
	astTestMust(t, m.DependencyArrayPop(&a))
	got, e := m.DependencyArrayGet(&a, 0)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else {
		astTestMust(t, e)
		if got != d {
			t.Fatal("copy-only source getter lost retained scalar", got, d)
		}
		if _, e = m.DependencyRefCount(got); !errors.Is(e, ErrASTHandle) {
			t.Fatal("actual dereference must validate lifetime", e)
		}
		if _, e = m.DependencyIsLeaf(got); !errors.Is(e, ErrASTHandle) {
			t.Fatal(e)
		}
	}
	astTestMust(t, m.DependencyArrayDel(&a))
}
