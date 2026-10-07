// Licensed under the MIT license.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func astDependencyExpr(t *testing.T, m *ASTManager, index uint32) ASTHandle {
	t.Helper()
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(index, s)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

// Added exact source-action expectations: there is no registered direct
// AST-valued dependency test in the source suite.
func TestASTDependencySourceLeafJoinCallbacksAndLinearization(t *testing.T) {
	m := astTestManager(t)
	a, b := astDependencyExpr(t, m, 1700), astDependencyExpr(t, m, 1701)
	empty, e := m.DependencyEmpty()
	astTestMust(t, e)
	if !empty.IsNull() {
		t.Fatal(empty)
	}
	null, e := m.DependencyLeaf(ASTHandle{})
	astTestMust(t, e)
	if !null.IsNull() {
		t.Fatal("AST mk_leaf null fastpath")
	}
	x, e := m.DependencyLeaf(a)
	astTestMust(t, e)
	y, e := m.DependencyLeaf(b)
	astTestMust(t, e)
	z, e := m.DependencyLeaf(a) // distinct dependency with equal value.
	astTestMust(t, e)
	if x == z {
		t.Fatal("source leaves are not hash-consed")
	}
	if n, e := m.DependencyRefCount(x); e != nil || n != 0 {
		t.Fatal("fresh leaf RC0", n, e)
	}
	if leaf, e := m.DependencyIsLeaf(x); e != nil || !leaf {
		t.Fatal(leaf, e)
	}
	same, e := m.DependencyJoin(x, x)
	astTestMust(t, e)
	if same != x {
		t.Fatal("same fastpath")
	}
	if n, e := m.DependencyRefCount(x); e != nil || n != 0 {
		t.Fatal("fastpath must not INC", n, e)
	}
	left, e := m.DependencyJoin(x, y)
	astTestMust(t, e)
	root, e := m.DependencyJoin(left, z)
	astTestMust(t, e)
	if leaf, e := m.DependencyIsLeaf(root); e != nil || leaf {
		t.Fatal(leaf, e)
	}
	if n, e := m.GetRefCount(a); e != nil || n != 3 {
		t.Fatal("two actual leaf INCs plus pin", n, e)
	}
	out, e := m.NewExprVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyLinearizeRaw(root, out))
	want := []ASTHandle{a, a, b} // FIFO: right leaf, left's first and second leaf.
	for i, h := range want {
		got, e := m.ExprVectorGet(out, uint32(i))
		astTestMust(t, e)
		if got != h {
			t.Fatal("source FIFO order", i, got, h)
		}
	}
	astTestMust(t, m.ExprVectorReset(out))
	astTestMust(t, m.ExprVectorPush(out, b))
	astTestMust(t, m.ExprVectorPush(out, b))
	astTestMust(t, m.DependencyLinearize(root, out))
	if n, e := m.ExprVectorSize(out); e != nil || n != 2 {
		t.Fatal("dedup entire output", n, e)
	}
	for i, h := range []ASTHandle{b, a} {
		got, e := m.ExprVectorGet(out, uint32(i))
		astTestMust(t, e)
		if got != h {
			t.Fatal("stable compaction", i, got, h)
		}
	}
	for _, h := range []ASTHandle{a, b, ASTHandle{}} {
		got, e := m.DependencyContains(root, h)
		astTestMust(t, e)
		if got == h.IsNull() {
			t.Fatal("contains source equality", h, got)
		}
	}
	astTestMust(t, m.DelExprVector(&out))
	astTestMust(t, m.DependencyInc(root))
	astTestMust(t, m.DependencyDec(root))
	if n, e := m.GetRefCount(a); e != nil || n != 1 {
		t.Fatal("real recursive dependency DEC", n, e)
	}
	if n, e := m.GetRefCount(b); e != nil || n != 1 {
		t.Fatal(n, e)
	}
}

func TestASTDependencySourceReferenceCountZeroAndWrap(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1702)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	e = m.DependencyDec(d)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source dec assertion", e)
		}
		if n, e := m.DependencyRefCount(d); e != nil || n != 0 {
			t.Fatal(n, e)
		}
	} else {
		astTestMust(t, e)
		if n, e := m.DependencyRefCount(d); e != nil || n != (1<<30)-1 {
			t.Fatal("source30-bit underflow", n, e)
		}
		astTestMust(t, m.DependencyInc(d)) // MAX30 -> 0, no overflow assertion.
	}
	astTestMust(t, m.DependencyInc(d))
	astTestMust(t, m.DependencyDec(d))
}

func TestASTDependencyCompletePublicMethodContracts(t *testing.T) {
	var _ func(*ASTManager, []ASTHandle) (ASTDependencyJoinProgress, error) = (*ASTManager).DependencyJoinValues
	var _ func(*ASTManager, *ASTDependencyArray) error = (*ASTManager).DependencyArrayMk
	var _ func(*ASTManager, *ASTDependencyArray) error = (*ASTManager).DependencyArrayDel
	var _ func(*ASTManager, *ASTDependencyArray, *ASTDependencyArray) error = (*ASTManager).DependencyArrayCopy
	var _ func(*ASTManager, *ASTDependencyArray) (uint32, error) = (*ASTManager).DependencyArraySize
	var _ func(*ASTManager, *ASTDependencyArray) (bool, error) = (*ASTManager).DependencyArrayEmpty
	var _ func(*ASTManager, *ASTDependencyArray, uint32) (ASTDependency, error) = (*ASTManager).DependencyArrayGet
	var _ func(*ASTManager, *ASTDependencyArray, uint32, ASTDependency) error = (*ASTManager).DependencyArraySet
	var _ func(*ASTManager, *ASTDependencyArray, uint32, ASTDependency, *ASTDependencyArray) error = (*ASTManager).DependencyArraySetFrom
	var _ func(*ASTManager, *ASTDependencyArray, ASTDependency) error = (*ASTManager).DependencyArrayPush
	var _ func(*ASTManager, *ASTDependencyArray, ASTDependency, *ASTDependencyArray) error = (*ASTManager).DependencyArrayPushFrom
	var _ func(*ASTManager, *ASTDependencyArray) error = (*ASTManager).DependencyArrayPop
	var _ func(*ASTManager, *ASTDependencyArray, *ASTDependencyArray) error = (*ASTManager).DependencyArrayPopFrom
	var _ func(*ASTManager, *ASTDependencyArray) error = (*ASTManager).DependencyArrayUnshare
	var _ func(*ASTManager, *ASTDependencyArray) error = (*ASTManager).DependencyArrayUnfold
	var _ func(*ASTManager, *ASTDependencyArray) error = (*ASTManager).DependencyArrayReroot
}

func TestASTDependencyIndependentDAGOracleProvenanceAndWarmStorage(t *testing.T) {
	m := astTestManager(t)
	foreign := astTestManager(t)
	var hs [12]ASTHandle
	var ds [12]ASTDependency
	for i := range hs {
		hs[i] = astDependencyExpr(t, m, uint32(4000+i))
		var e error
		ds[i], e = m.DependencyLeaf(hs[i])
		astTestMust(t, e)
	}
	root, e := m.DependencyJoin(ds[0], ds[1])
	astTestMust(t, e)
	for i := 2; i < len(ds); i++ {
		x, e := m.DependencyJoin(root, ds[i])
		astTestMust(t, e)
		root = x
	}
	shared, e := m.DependencyJoin(root, ds[3])
	astTestMust(t, e)
	out, e := m.NewExprVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyLinearize(shared, out))
	n, e := m.ExprVectorSize(out)
	astTestMust(t, e)
	if n != uint32(len(hs)) {
		t.Fatal("independent leaf-set cardinality", n)
	}
	seen := map[ASTHandle]bool{}
	for i := uint32(0); i < n; i++ {
		h, e := m.ExprVectorGet(out, i)
		astTestMust(t, e)
		if seen[h] {
			t.Fatal("duplicate", h)
		}
		seen[h] = true
	}
	for _, h := range hs {
		if !seen[h] {
			t.Fatal("missing reachable value", h)
		}
	}
	if _, e := foreign.DependencyJoin(shared, ASTDependency{}); !errors.Is(e, ErrASTHandle) {
		t.Fatal("foreign raw dependency must not pass", e)
	}
	if _, e := m.DependencyLeaf(astDependencyExpr(t, foreign, 5000)); !errors.Is(e, ErrASTHandle) {
		t.Fatal("foreign leaf callback", e)
	}
	astTestMust(t, m.ExprVectorReset(out))
	astTestMust(t, m.DependencyLinearizeRaw(shared, out))
	baseline := m.BudgetField.Used()
	alloc := testing.AllocsPerRun(50, func() {
		if yes, e := m.DependencyContains(shared, hs[8]); e != nil || !yes {
			panic("contains")
		}
		if e := m.ExprVectorReset(out); e != nil {
			panic(e)
		}
		if e := m.DependencyLinearizeRaw(shared, out); e != nil {
			panic(e)
		}
	})
	if alloc != 0 || m.BudgetField.Used() != baseline {
		t.Fatal("warmed concrete FIFO/output scratch", alloc, m.BudgetField.Used(), baseline)
	}
	astTestMust(t, m.DelExprVector(&out))
	astTestMust(t, m.DependencyInc(shared))
	astTestMust(t, m.DependencyDec(shared))
	if m.dependencies.live != 0 {
		t.Fatal("shared-DAG deletion retains orphan", m.dependencies.live)
	}
}

func TestASTDependencyActualLIFOCallbacksNormalAndIntrusiveFailure(t *testing.T) {
	for _, fail := range [2]bool{false, true} {
		m := astTestManager(t)
		s, e := m.MkBoolSort()
		astTestMust(t, e)
		a, e := m.MkVar(6200, s)
		astTestMust(t, e)
		b, e := m.MkVar(6201, s)
		astTestMust(t, e)
		ia, e := m.GetASTID(a)
		astTestMust(t, e)
		ib, e := m.GetASTID(b)
		astTestMust(t, e)
		da, e := m.DependencyLeaf(a)
		astTestMust(t, e)
		db, e := m.DependencyLeaf(b)
		astTestMust(t, e)
		root, e := m.DependencyJoin(da, db)
		astTestMust(t, e)
		astTestMust(t, m.DependencyInc(root))
		start := len(m.ExprIDs.Free)
		if fail {
			m.BudgetField.FailAfter(0)
		}
		e = m.DependencyDec(root)
		m.BudgetField.FailAfter(-1)
		if fail {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
		} else {
			astTestMust(t, e)
		}
		ids := m.ExprIDs.Free[start:]
		if len(ids) != 2 || ids[0] != ib || ids[1] != ia {
			t.Fatal("source left-then-right push => right-then-left actual AST DEC", fail, ids, ia, ib)
		}
		if m.dependencies.live != 0 {
			t.Fatal(m.dependencies.live)
		}
	}
}

func TestASTDependencyScratchActualAndSourceByteBounds(t *testing.T) {
	sourceWords := (uint64(^uint32(0)) - 8) / 8
	old := sourceWords * 2 / 3
	n, e := astDependencyScratchCapacity(old, ^uint64(0)>>1)
	if e != nil || n != sourceWords {
		t.Fatal(n, e, sourceWords)
	}
	if _, e = astDependencyScratchCapacity(old+1, ^uint64(0)>>1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source unsigned32 allocation bytes", e)
	}
	host32 := uint64(1<<31) - 1
	hostWords := host32 / 8
	old = hostWords * 2 / 3
	if _, e = astDependencyScratchCapacity(old, host32); e != nil {
		t.Fatal(e)
	}
	if _, e = astDependencyScratchCapacity(old+2, host32); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual Go host backing bytes", e)
	}
}

func TestASTDependencyWarmDestructionNoReservationOrHeapAllocation(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 6300)
	var roots [102]ASTDependency
	for i := range roots {
		a, e := m.DependencyLeaf(h)
		astTestMust(t, e)
		b, e := m.DependencyLeaf(h)
		astTestMust(t, e)
		roots[i], e = m.DependencyJoin(a, b)
		astTestMust(t, e)
		astTestMust(t, m.DependencyInc(roots[i]))
	}
	astTestMust(t, m.DependencyDec(roots[0]))
	baseline := m.BudgetField.Used()
	i := 1
	var failure error
	m.BudgetField.FailAfter(0)
	alloc := testing.AllocsPerRun(100, func() { failure = m.DependencyDec(roots[i]); i++ })
	m.BudgetField.FailAfter(-1)
	if failure != nil || alloc != 0 || i != 102 || m.dependencies.live != 0 || m.BudgetField.Used() > baseline {
		t.Fatal("source warm LIFO deletion", failure, alloc, i, m.dependencies.live, m.BudgetField.Used(), baseline)
	}
	if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
}
