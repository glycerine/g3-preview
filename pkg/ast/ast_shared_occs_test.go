// No registered shared_occs test exists; these are additional finite source models.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func sharedTestApp(t *testing.T, m *ASTManager, args ...ASTHandle) ASTHandle {
	t.Helper()
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args)
	astTestMust(t, e)
	return h
}
func TestASTSharedOccurrencesActualMultiplicityAndDefaults(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 2)
	sub := sharedTestApp(t, m, h[0], h[1])
	root := sharedTestApp(t, m, sub, sub)
	collect, e := NewASTSharedOccurrences(m)
	astTestMust(t, e)
	defer collect.Close()
	cfg := DefaultASTSharedOccurrencesConfig()
	if cfg.TrackAtomic || !cfg.VisitQuantifiers || cfg.VisitPatterns {
		t.Fatal("source defaults", cfg)
	}
	before, e := m.GetRefCount(sub)
	astTestMust(t, e)
	astTestMust(t, collect.Collect(root))
	yes, e := collect.IsShared(sub)
	astTestMust(t, e)
	if !yes || collect.NumShared() != 1 {
		t.Fatal("shared composite")
	}
	after, e := m.GetRefCount(sub)
	astTestMust(t, e)
	if after != before+1 {
		t.Fatal("actual unique-set reference retain", before, after)
	}
	astTestMust(t, collect.Collect(root))
	again, e := m.GetRefCount(sub)
	astTestMust(t, e)
	if again != after {
		t.Fatal("default reset then discovery")
	}
	astTestMust(t, collect.Reset())
	after, e = m.GetRefCount(sub)
	astTestMust(t, e)
	if after != before {
		t.Fatal("real reset DEC", after, before)
	}
	atomicRoot := sharedTestApp(t, m, h[0], h[0])
	astTestMust(t, collect.Collect(atomicRoot))
	if collect.NumShared() != 0 {
		t.Fatal("default ignores atomic vars")
	}
	atom, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{TrackAtomic: true, VisitQuantifiers: true})
	astTestMust(t, e)
	defer atom.Close()
	astTestMust(t, atom.Collect(atomicRoot))
	yes, e = atom.IsShared(h[0])
	astTestMust(t, e)
	if !yes || atom.NumShared() != 1 {
		t.Fatal("actual atomic specialization")
	}
}
func TestASTSharedOccurrencesExternalVisitedAndPlaneIndependence(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 1)[0]
	pins, e := m.NewRefVectorFrom([]ASTHandle{h, h})
	astTestMust(t, e)
	defer m.DelRefVector(&pins)
	visited, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer visited.Close()
	collect, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{TrackAtomic: true, VisitQuantifiers: true})
	astTestMust(t, e)
	defer collect.Close()
	astTestMust(t, m.MarkNode1(h, true, 111))
	astTestMust(t, m.MarkNode2(h, true, 222))
	astTestMust(t, collect.CollectWithMark(h, visited))
	if collect.NumShared() != 0 {
		t.Fatal("RC>1 alone is not repeated occurrence")
	}
	astTestMust(t, collect.CollectWithMark(h, visited))
	if collect.NumShared() != 1 {
		t.Fatal("external visited overload must persist visitation")
	}
	astTestMust(t, visited.Reset())
	for plane := 1; plane <= 2; plane++ {
		var yes bool
		if plane == 1 {
			yes, e = m.IsNodeMarked1(h, 111)
		} else {
			yes, e = m.IsNodeMarked2(h, 222)
		}
		astTestMust(t, e)
		if !yes {
			t.Fatal("shared flag must be separate from public mark planes")
		}
	}
	astTestMust(t, m.ResetNodeMark1(h, 111))
	astTestMust(t, m.ResetNodeMark2(h, 222))
}
func TestASTSharedOccurrencesMarkInlineFailurePrefix(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 17)
	visited, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer visited.Close()
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	for _, n := range h[:16] {
		astTestMust(t, visited.Mark(n, true))
	}
	err := visited.Mark(h[16], true)
	m.BudgetField.FailAfter(-1)
	yes, e := visited.IsMarked(h[16])
	astTestMust(t, e)
	if !errors.Is(err, arena.ErrInjectedFailure) || !yes || m.BudgetField.Used() != before {
		t.Fatal("private shared mark before overflow allocation", err, yes)
	}
	astTestMust(t, visited.Reset())
	yes, e = visited.IsMarked(h[16])
	astTestMust(t, e)
	if !yes {
		t.Fatal("unlisted failed prefix must remain")
	}
	astTestMust(t, visited.ResetMark(h[16]))
}

func TestASTSharedOccurrencesPhysicalBucketIterator(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 12)
	pins, e := m.NewRefVectorFrom(append(append([]ASTHandle{}, h...), h...))
	astTestMust(t, e)
	defer m.DelRefVector(&pins)
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer v.Close()
	c, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{TrackAtomic: true})
	astTestMust(t, e)
	defer c.Close()
	for _, n := range h {
		astTestMust(t, v.Mark(n, true))
		astTestMust(t, c.CollectWithMark(n, v))
	}
	it, e := c.BeginShared()
	astTestMust(t, e)
	next := 0
	for {
		done, e := it.Done()
		astTestMust(t, e)
		if done {
			break
		}
		for next < len(c.table.entries) && !astExprUsed(c.table.entries[next]) {
			next++
		}
		got, e := it.Value()
		astTestMust(t, e)
		if got != m.external(c.table.entries[next]) {
			t.Fatal("source physical bucket order")
		}
		next++
		astTestMust(t, it.Next())
	}
	astTestMust(t, v.Reset())
}

func TestASTSharedOccurrencesAllTraversalPolicies(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 2)
	sub := sharedTestApp(t, m, h[0], h[1])
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{sub})
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []util.Symbol{util.NumericalSymbol(0)}, Body: sub, Patterns: []ASTHandle{pattern}})
	astTestMust(t, e)
	// sub occurs once as body and once under the pattern. RC alone does not
	// qualify it: actual traversal must encounter it twice.
	for bits := 0; bits < 8; bits++ {
		cfg := ASTSharedOccurrencesConfig{TrackAtomic: bits&1 != 0, VisitQuantifiers: bits&2 != 0, VisitPatterns: bits&4 != 0}
		c, e := NewASTSharedOccurrencesWithConfig(m, cfg)
		astTestMust(t, e)
		astTestMust(t, c.Collect(q))
		yes, e := c.IsShared(sub)
		astTestMust(t, e)
		if yes != (cfg.VisitQuantifiers && cfg.VisitPatterns) {
			t.Fatal("actual quantifier/pattern child routing", cfg, yes)
		}
		astTestMust(t, c.Close())
	}
	// Repeated quantifiers qualify before the VisitQuantifiers test.
	outer := sharedTestApp(t, m, q, q)
	c, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{})
	astTestMust(t, e)
	defer c.Close()
	astTestMust(t, c.Collect(outer))
	yes, e := c.IsShared(q)
	astTestMust(t, e)
	if !yes || c.NumShared() != 1 {
		t.Fatal("quantifier sharing before descent policy")
	}
}

func TestASTSharedOccurrencesWarmInlineAndHeaderRefusal(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 2)
	sub := sharedTestApp(t, m, h[0], h[1])
	root := sharedTestApp(t, m, sub, sub)
	c, e := NewASTSharedOccurrences(m)
	astTestMust(t, e)
	defer c.Close()
	astTestMust(t, c.Collect(root))
	if n := testing.AllocsPerRun(100, func() {
		if e := c.Collect(root); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("local inline16 marker/frame warm operation allocated", n)
	}
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	if n := testing.AllocsPerRun(20, func() {
		_, e := NewASTSharedOccurrences(m)
		if !errors.Is(e, arena.ErrInjectedFailure) {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("cold header allocation before budget refusal", n)
	}
	m.BudgetField.FailAfter(-1)
	if m.BudgetField.Used() != before {
		t.Fatal("refusal changed owner graph")
	}
}

func TestASTSharedOccurrencesFrameFailureCursorPrefix(t *testing.T) {
	m := astTestManager(t)
	leaf := markTestNodes(t, m, 1)[0]
	root := leaf
	for i := 0; i < 5; i++ {
		root = sharedTestApp(t, m, root)
	}
	c, e := NewASTSharedOccurrences(m)
	astTestMust(t, e)
	defer c.Close()
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer v.Close()
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(1)
	e = c.CollectWithMark(root, v)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || c.stackSize != 2 || len(c.stack) != 2 || c.stack[0].next != 1 || c.stack[1].next != 1 || m.BudgetField.Used() != before+24 {
		t.Fatal("parent cursor advances before child frame allocation", e, c.stackSize, c.stack)
	}
	// reset does not touch traversal frames. In source release an atomic
	// process returns before draining the old stack; debug rejects nonempty.
	astTestMust(t, c.Reset())
	e = c.CollectWithMark(leaf, v)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source stack.empty diagnostic", e)
		}
	} else {
		astTestMust(t, e)
	}
	if c.stackSize != 2 {
		t.Fatal("reset or atomic early return silently cleared frames")
	}
	astTestMust(t, c.Cleanup())
	if c.stackSize != 0 || c.stack != nil || c.stackBytes != 0 {
		t.Fatal("source stack.finalize")
	}
}

func TestASTSharedOccurrencesResetDeletesAscendingPhysicalBuckets(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 10)
	pins, e := m.NewRefVectorFrom(append(append([]ASTHandle{}, h...), h...))
	astTestMust(t, e)
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer v.Close()
	c, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{TrackAtomic: true})
	astTestMust(t, e)
	defer c.Close()
	for _, n := range h {
		astTestMust(t, v.Mark(n, true))
		astTestMust(t, c.CollectWithMark(n, v))
	}
	astTestMust(t, v.Reset())             // borrowed visitation list must outlive its nodes
	astTestMust(t, m.DelRefVector(&pins)) // leave the collector's sole retain
	var expected []uint32
	for _, r := range c.table.entries {
		if astExprUsed(r) {
			expected = append(expected, m.nodeAt(r.slot).id)
		}
	}
	astTestMust(t, c.Reset())
	// Each erase/recycle occurs in bucket order. The genuine source ID helper
	// returns these IDs in reverse, independently checking actual DEC callbacks.
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	for i := len(expected) - 1; i >= 0; i-- {
		n, e := m.MkVar(uint32(80000+i), b)
		astTestMust(t, e)
		id, e := m.GetASTID(n)
		astTestMust(t, e)
		if id != expected[i] {
			t.Fatal("real bucket ASC destruction/LIFO ID reuse", i, id, expected[i])
		}
	}
}

func TestASTSharedOccurrencesResetConsumedDeletionFailureIsNotRepeated(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 20)
	root := sharedTestApp(t, m, h...)
	pins, e := m.NewRefVectorFrom([]ASTHandle{root, root})
	astTestMust(t, e)
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer v.Close()
	c, e := NewASTSharedOccurrences(m)
	astTestMust(t, e)
	defer c.Close()
	astTestMust(t, v.Mark(root, true))
	astTestMust(t, c.CollectWithMark(root, v))
	astTestMust(t, v.Reset())
	astTestMust(t, m.DelRefVector(&pins))
	r, e := m.internal(root)
	astTestMust(t, e)
	before := m.liveNodes
	m.BudgetField.FailAfter(0)
	e = c.Reset()
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("actual source inline16 deletion overflow fault", e)
	}
	if m.validNode(r) == nil || m.liveNodes != before-21 || c.table.size != 1 || c.resetPhase != 1 {
		t.Fatal("consumed callback prefix immediately before retry", m.liveNodes, before, c.table.size, c.resetPhase)
	}
	astTestMust(t, c.Reset())
	if c.table.size != 0 || c.resetPhase != 0 || m.liveNodes != before-21 {
		t.Fatal("retry DEC already consumed node")
	}
}

func TestASTSharedOccurrencesForcedManagerCloseAndIteratorBorrow(t *testing.T) {
	m := astTestManager(t)
	c, e := NewASTSharedOccurrences(m)
	astTestMust(t, e)
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	it, e := c.BeginShared()
	astTestMust(t, e)
	charged := c.owner.ownerBytes + c.table.bytes + v.owner.ownerBytes + s.owner.ownerBytes + s.table.bytes
	astTestMust(t, m.Close())
	if _, e = it.Done(); e == nil {
		t.Fatal("iterator must reject forced closed manager lifetime")
	}
	before := m.BudgetField.Used()
	astTestMust(t, c.Close())
	astTestMust(t, v.Close())
	astTestMust(t, s.Close())
	if m.BudgetField.Used() != before-charged {
		t.Fatal("forced teardown refunds only external storage, with no dead AST callbacks")
	}
}

func TestASTSharedOccurrencesIteratorSurvivesInPlaceTableChanges(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 2)
	pins, e := m.NewRefVectorFrom([]ASTHandle{h[0], h[0], h[1], h[1]})
	astTestMust(t, e)
	defer m.DelRefVector(&pins)
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer v.Close()
	c, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{TrackAtomic: true})
	astTestMust(t, e)
	defer c.Close()
	astTestMust(t, v.Mark(h[0], true))
	astTestMust(t, c.CollectWithMark(h[0], v))
	it, e := c.BeginShared()
	astTestMust(t, e)
	end, e := c.EndShared()
	astTestMust(t, e)
	index := it.index
	epoch := c.table.epoch
	astTestMust(t, v.Mark(h[1], true))
	astTestMust(t, c.CollectWithMark(h[1], v))
	got, e := it.Value()
	astTestMust(t, e)
	if got != h[0] || it.index != index || c.table.capacity != 8 || c.table.epoch != epoch {
		t.Fatal("source captures existing entry pointer through insertion")
	}
	done, e := end.Done()
	astTestMust(t, e)
	if !done {
		t.Fatal("captured endpoint remains endpoint after in-place insert")
	}
	astTestMust(t, c.Reset())
	done, e = it.Done()
	astTestMust(t, e)
	if done {
		t.Fatal("reset does not advance captured current pointer")
	}
	got, e = it.Value()
	astTestMust(t, e)
	if !got.IsNull() {
		t.Fatal("captured FREE slot returns source null")
	}
	astTestMust(t, it.Next())
	same, e := it.Equal(end)
	astTestMust(t, e)
	if !same {
		t.Fatal("Next scans cleared backing to captured endpoint")
	}
	astTestMust(t, v.Reset())
}

func TestASTSharedOccurrencesFrameSourceExtentAndHostBounds(t *testing.T) {
	const hostMax = uint64(^uint64(0) >> 1)
	for _, tc := range []struct {
		old  uint64
		want uint32
	}{{0, 2}, {2, 3}, {3, 5}, {5, 8}, {178956970, 268435455}} {
		n, e := astSharedFrameNextCapacity(tc.old, hostMax)
		astTestMust(t, e)
		if n != tc.want {
			t.Fatal("source capacity schedule/LP64 extent endpoint", tc, n)
		}
	}
	if _, e := astSharedFrameNextCapacity(178956971, hostMax); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source pair16 extent must refuse even though actual Go frame12 fits", e)
	}
	if _, e := astSharedFrameNextCapacity(^uint64(0), hostMax); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source unsigned width", e)
	}
	if _, e := astSharedFrameNextCapacity(uint64(^uint32(0)), hostMax); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source unsigned capacity arithmetic wraps and fails growth", e)
	}
	if _, e := astSharedFrameNextCapacity(0, 23); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual frame12 host domain", e)
	}
	n, e := astSharedFrameNextCapacity(0, 24)
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("actual host endpoint")
	}
}

func TestASTSharedOccurrencesIteratorActualBackingLifetime(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 8)
	pins, e := m.NewRefVectorFrom(append(append([]ASTHandle{}, h...), h...))
	astTestMust(t, e)
	defer m.DelRefVector(&pins)
	v, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer v.Close()
	c, e := NewASTSharedOccurrencesWithConfig(m, ASTSharedOccurrencesConfig{TrackAtomic: true})
	astTestMust(t, e)
	defer c.Close()
	for _, n := range h {
		astTestMust(t, v.Mark(n, true))
	}
	for _, n := range h[:7] {
		astTestMust(t, c.CollectWithMark(n, v))
	}
	it, e := c.BeginShared()
	astTestMust(t, e)
	original, e := it.Value()
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	e = c.CollectWithMark(h[7], v)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("actual growth reserve refusal", e)
	}
	got, e := it.Value()
	astTestMust(t, e)
	if got != original {
		t.Fatal("failed reserve leaves entry pointer alive")
	}
	astTestMust(t, c.CollectWithMark(h[7], v))
	if _, e = it.Value(); !errors.Is(e, ErrASTHandle) {
		t.Fatal("successful relocation invalidates old backing", e)
	}
	it, e = c.BeginShared()
	astTestMust(t, e)
	original, e = it.Value()
	astTestMust(t, e)
	r, e := m.internal(original)
	astTestMust(t, e)
	epoch := c.table.epoch
	astTestMust(t, c.table.erase(m, r)) // actual source base-table erase, no RC callback
	got, e = it.Value()
	astTestMust(t, e)
	if c.table.epoch != epoch || got == original {
		t.Fatal("erased current pointer remains a live slot view")
	}
	// FREE yields null; DELETED yields the explicit non-node sentinel token.
	if !got.IsNull() && (got.slot != 1 || got.generation != 0) {
		t.Fatal("source erased slot representation", got)
	}
	astTestMust(t, m.DecRef(original)) // balance the erased collector-owned retain
	astTestMust(t, v.Reset())
}

func TestASTSparseSharedExpressionDomainsAndASTWideVisited(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	truth, e := m.MkFamilyApp(0, ASTOpTrue, nil, nil)
	astTestMust(t, e)
	decl, e := m.GetDecl(truth)
	astTestMust(t, e)
	sparse, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer sparse.Close()
	c, e := NewASTSharedOccurrences(m)
	astTestMust(t, e)
	defer c.Close()
	visited, e := NewASTSharedOccurrencesMark(m)
	astTestMust(t, e)
	defer visited.Close()
	for _, h := range []ASTHandle{sort, decl} {
		if e = sparse.Mark(h, true); !errors.Is(e, ErrASTDomain) {
			t.Error("source expr* mark facade accepts nonexpression", e)
		}
		if e = sparse.Mark(h, false); !errors.Is(e, ErrASTDomain) {
			t.Error("source expr* erase facade accepts nonexpression", e)
		}
		if _, e = sparse.IsMarked(h); !errors.Is(e, ErrASTDomain) {
			t.Error("source expr* contains facade accepts nonexpression", e)
		}
		if _, e = c.IsShared(h); !errors.Is(e, ErrASTDomain) {
			t.Error("source expr* shared facade accepts nonexpression", e)
		}
		astTestMust(t, visited.Mark(h, true))
		yes, e := visited.IsMarked(h)
		astTestMust(t, e)
		if !yes {
			t.Fatal("shared_occs_mark intentionally takes ast*, not expr*")
		}
	}
	if sparse.table.size != 0 {
		t.Fatal("rejected domains publish no sparse keys")
	}
	astTestMust(t, visited.Reset())
}
