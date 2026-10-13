package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	umm "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type arbitraryClauseFixture struct {
	context *ArbitraryContext
	numbers *numeral.ArbitraryFloatNumeralManager
	x, y    uint32
	four    numeral.ArbitraryFloat
	five    numeral.ArbitraryFloat
}

func newArbitraryClauseFixture(t *testing.T) *arbitraryClauseFixture {
	return newArbitraryClauseFixtureWithPolicy(t, 0, false)
}

func newArbitraryClauseFixtureWithPolicy(t *testing.T, epsilon uint32, yInteger bool) *arbitraryClauseFixture {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	r, err := umm.NewRuntime(umm.RuntimeConfig{Budget: b, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	config := numeral.IntegerConfig{
		Memory: r, Profile: umm.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}
	core, err := numeral.NewArbitraryFloatManager(config)
	if err != nil {
		t.Fatal(err)
	}
	numbers, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewArbitraryContext(numbers, &control.Reslimit{Limit: 100}, epsilon, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	f := &arbitraryClauseFixture{context: c, numbers: numbers}
	if f.x, err = c.NewVariable(false); err != nil {
		t.Fatal(err)
	}
	if f.y, err = c.NewVariable(yInteger); err != nil {
		t.Fatal(err)
	}
	if err := numbers.SetInt32(&f.four, 4); err != nil {
		t.Fatal(err)
	}
	if err := numbers.SetInt32(&f.five, 5); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{c.Close(), numbers.Del(&f.four), numbers.Del(&f.five), numbers.Close(), core.Close(), r.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("MPF clause fixture retained budget", b.Used())
		}
	})
	return f
}

func TestArbitraryClauseSourceTruthAndPropagation(t *testing.T) {
	t.Run("open ties", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		loClosed, _ := c.NewInequality(f.x, &f.five, true, false)
		loOpen, _ := c.NewInequality(f.x, &f.five, true, true)
		hiClosed, _ := c.NewInequality(f.x, &f.five, false, false)
		hiOpen, _ := c.NewInequality(f.x, &f.five, false, true)
		root, err := c.Root()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, f.x, &f.five, false, false, true); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			a    *ArbitraryInequality
			want arbitraryTruth
		}{{loClosed, arbitraryUnknown}, {loOpen, arbitraryFalse}, {hiClosed, arbitraryTrue}, {hiOpen, arbitraryUnknown}} {
			got, err := c.inequalityValue(tc.a, root)
			if err != nil || got != tc.want {
				t.Fatalf("closed upper tie: %v, want %v, err %v", got, tc.want, err)
			}
		}
		if err := c.PublishBound(root, f.x, &f.five, false, true, true); err != nil {
			t.Fatal(err)
		}
		if got, err := c.inequalityValue(hiOpen, root); err != nil || got != arbitraryTrue {
			t.Fatalf("open upper tie: %v, %v", got, err)
		}
	})
	t.Run("sole unknown", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		ax, _ := c.NewInequality(f.x, &f.five, true, false)
		ay, _ := c.NewInequality(f.y, &f.five, true, false)
		clause, err := c.AddClause([]*ArbitraryInequality{ay, ax})
		if err != nil {
			t.Fatal(err)
		}
		root, err := c.Root()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateClause(clause, root); err != nil || len(c.queue) != 0 {
			t.Fatalf("two unknown atoms: %v, %d", err, len(c.queue))
		}
		if err := c.PublishBound(root, f.x, &f.four, false, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateClause(clause, root); err != nil {
			t.Fatal(err)
		}
		if root.lower[f.y] == nil || root.lower[f.y].reasonClause != clause || len(c.queue) != 2 || root.conflictVar != math.MaxUint32 || clause.visited != c.timestamp {
			t.Fatalf("sole unknown clause publication: %+v, %d, %d, %d", root.lower[f.y], len(c.queue), root.conflictVar, clause.visited)
		}
		if err := c.PropagateClause(clause, root); err != nil || len(c.queue) != 2 {
			t.Fatalf("satisfied clause does not republish: %v, %d", err, len(c.queue))
		}
	})
	t.Run("all false", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		ax, _ := c.NewInequality(f.x, &f.five, true, false)
		ay, _ := c.NewInequality(f.y, &f.five, true, false)
		clause, err := c.AddClause([]*ArbitraryInequality{ay, ax})
		if err != nil {
			t.Fatal(err)
		}
		root, err := c.Root()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, f.x, &f.four, false, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, f.y, &f.four, false, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateClause(clause, root); err != nil {
			t.Fatal(err)
		}
		if root.conflictVar != f.x || root.lower[f.x] == nil || root.lower[f.x].reasonClause != clause || len(c.queue) != 3 {
			t.Fatalf("all-false first atom conflict: %d, %+v, %d", root.conflictVar, root.lower[f.x], len(c.queue))
		}
	})
}

func TestArbitraryClauseQueuedWatchOrderAndCancellation(t *testing.T) {
	t.Run("cancel and resume queued clause", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		ax, err := c.NewInequality(f.x, &f.five, true, false)
		if err != nil {
			t.Fatal(err)
		}
		ay, err := c.NewInequality(f.y, &f.five, true, false)
		if err != nil {
			t.Fatal(err)
		}
		clause, err := c.AddClause([]*ArbitraryInequality{ay, ax})
		if err != nil {
			t.Fatal(err)
		}
		root, err := c.Root()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, f.x, &f.four, false, false, true); err != nil {
			t.Fatal(err)
		}
		if !c.limit.Inc() {
			t.Fatal("resource fixture setup")
		}
		c.limit.Limit = c.limit.Steps
		if err := c.PropagateQueued(root); !errors.Is(err, ErrArbitraryContextCanceled) || c.queueHead != 0 || len(c.queue) != 1 {
			t.Fatalf("checkpoint before first bound: %v, %d, %d", err, c.queueHead, len(c.queue))
		}
		c.limit.Limit = 0
		if err := c.PropagateQueued(root); err != nil {
			t.Fatal(err)
		}
		if root.lower[f.y] == nil || root.lower[f.y].reasonClause != clause || len(c.queue) != 0 || c.queueHead != 0 || c.numVisited != 1 {
			t.Fatalf("resumed watched clause and generated bound: %+v, %d, %d, %d", root.lower[f.y], len(c.queue), c.queueHead, c.numVisited)
		}
	})
	t.Run("monomial definition watch", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		z, err := c.DefineMonomial([]ArbitraryPower{{Variable: f.x, Degree: 2}})
		if err != nil || z != 2 {
			t.Fatalf("monomial setup: %d, %v", z, err)
		}
		root, err := c.Root()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, f.x, &f.four, false, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, f.x, &f.four, true, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateQueued(root); err != nil || root.lower[z] == nil || root.upper[z] == nil ||
			!root.lower[z].definitionReason || root.upper[z].reasonVariable != z || c.queueHead != 0 || len(c.queue) != 0 {
			t.Fatalf("queued monomial definition: %v, %+v/%+v, %d, %d", err, root.lower[z], root.upper[z], c.queueHead, len(c.queue))
		}
	})
}
