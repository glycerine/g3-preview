package ast

import (
	"errors"
	"testing"
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
	var out []ASTHandle
	out, e = m.DependencyLinearizeRaw(root, out)
	astTestMust(t, e)
	want := []ASTHandle{a, a, b} // FIFO: right leaf, left's first and second leaf.
	for i, h := range want {
		got := out[i]
		if got != h {
			t.Fatal("source FIFO order", i, got, h)
		}
	}
	out = out[:0]
	out = append(out, b)
	out = append(out, b)
	out, e = m.DependencyLinearize(root, out)
	astTestMust(t, e)
	if n := uint32(len(out)); n != 2 {
		t.Fatal("dedup entire output", n, e)
	}
	for i, h := range []ASTHandle{b, a} {
		got := out[i]
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
	var out []ASTHandle
	out, e = m.DependencyLinearize(shared, out)
	astTestMust(t, e)
	n := uint32(len(out))
	if n != uint32(len(hs)) {
		t.Fatal("independent leaf-set cardinality", n)
	}
	seen := map[ASTHandle]bool{}
	for i := uint32(0); i < n; i++ {
		h := out[i]
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
	out = out[:0]
	out, e = m.DependencyLinearizeRaw(shared, out)
	astTestMust(t, e)
	baseline := m.BudgetField.Used()
	alloc := testing.AllocsPerRun(50, func() {
		if yes, e := m.DependencyContains(shared, hs[8]); e != nil || !yes {
			panic("contains")
		}
		out, e = m.DependencyLinearizeRaw(shared, out[:0])
		if e != nil {
			panic(e)
		}
	})
	if alloc != 0 || m.BudgetField.Used() != baseline {
		t.Fatal("warmed concrete FIFO/output scratch", alloc, m.BudgetField.Used(), baseline)
	}
	astTestMust(t, m.DependencyInc(shared))
	astTestMust(t, m.DependencyDec(shared))
	if m.dependencies.live != 0 {
		t.Fatal("shared-DAG deletion retains orphan", m.dependencies.live)
	}
}

func TestASTDependencyNativeLIFOCallbacksWithReservationsDisabled(t *testing.T) {
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
		start := m.ExprIDs.FreeLen()
		if fail {
			m.BudgetField.FailAfter(0)
		}
		e = m.DependencyDec(root)
		m.BudgetField.FailAfter(-1)
		astTestMust(t, e)
		ids := m.ExprIDs.FreeIDs()[start:]
		if len(ids) != 2 || ids[0] != ib || ids[1] != ia {
			t.Fatal("source left-then-right push => right-then-left actual AST DEC", fail, ids, ia, ib)
		}
		if m.dependencies.live != 0 {
			t.Fatal(m.dependencies.live)
		}
	}
}

func TestASTDependencyNativeColdFIFOAndCompleteUnmarking(t *testing.T) {
	m := astTestManager(t)
	absent := astDependencyExpr(t, m, 19000)
	var root ASTDependency
	var nodes []ASTHandle
	var leaves []ASTDependency
	for i := 0; i < 257; i++ {
		h := astDependencyExpr(t, m, uint32(20000+i))
		nodes = append(nodes, h)
		leaf, e := m.DependencyLeaf(h)
		astTestMust(t, e)
		leaves = append(leaves, leaf)
		root, e = m.DependencyJoin(root, leaf)
		astTestMust(t, e)
	}
	astTestMust(t, m.DependencyInc(root))
	var out []ASTHandle
	for _, h := range nodes {
		out = append(out, h)
	}
	out = out[:0]
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	yes, e := m.DependencyContains(root, absent)
	astTestMust(t, e)
	if yes {
		t.Fatal("absent value found")
	}
	out, e = m.DependencyLinearizeRaw(root, out)
	astTestMust(t, e)
	if n := uint32(len(out)); n != 257 {
		t.Fatal("native FIFO cardinality", n, e)
	}
	for i := 0; i < 257; i++ {
		want := 256 - i
		if i == 255 {
			want = 0
		}
		if i == 256 {
			want = 1
		}
		got := out[i]
		if got != nodes[want] {
			t.Fatal("left/right FIFO traversal order", i, got, nodes[want])
		}
	}
	for _, leaf := range leaves {
		c, e := m.dependencies.node(leaf.record)
		astTestMust(t, e)
		if c.marked {
			t.Fatal("traversal retained a dependency mark")
		}
	}
	if len(m.dependencies.todo) != 0 || m.BudgetField.Used() != before {
		t.Fatal("native scratch changed object storage")
	}
	astTestMust(t, m.DependencyDec(root))
	m.BudgetField.FailAfter(-1)
	for _, h := range nodes {
		rc, e := m.GetRefCount(h)
		astTestMust(t, e)
		if rc != 1 {
			t.Fatal("dependency deletion lost caller ownership", rc)
		}
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
