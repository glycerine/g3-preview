package aig

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func aigFixture(t *testing.T) (*ast.ASTManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, e := symbol.NewManager(symbol.Config{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
		if e := q.Close(); e != nil {
			t.Error(e)
		}
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Errorf("owner graph retained %d bytes", b.Used())
		}
	})
	return m, b
}

func aigMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func TestAIGCoreInterningOwnershipAndIDReuse(t *testing.T) {
	m, _ := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	a, e := NewManager(m)
	aigMust(t, e)
	x, e := a.MakeVariable(h)
	aigMust(t, e)
	y, e := a.MakeVariable(h)
	aigMust(t, e)
	xid, _, e := x.ID()
	aigMust(t, e)
	yid, _, e := y.ID()
	aigMust(t, e)
	if xid != 1 || yid != 2 {
		t.Fatalf("variable IDs %d,%d", xid, yid)
	}
	n, e := a.InternAnd(x, y)
	aigMust(t, e)
	same, e := a.InternAnd(x, y)
	aigMust(t, e)
	if !n.Same(same) {
		t.Fatal("equal ordered children did not intern")
	}
	nx, e := x.Not()
	aigMust(t, e)
	different, e := a.InternAnd(nx, y)
	aigMust(t, e)
	if n.Same(different) {
		t.Fatal("inversion bit omitted from key")
	}
	nid, _, e := n.ID()
	aigMust(t, e)
	if nid != firstNodeID {
		t.Fatalf("first AND ID %d", nid)
	}
	count, e := a.NumNodes()
	aigMust(t, e)
	if count != 5 {
		t.Fatalf("node count %d, want true + two vars + two ANDs", count)
	}
	if e := a.Close(); !errors.Is(e, ErrLiveReferences) {
		t.Fatalf("live close: %v", e)
	}
	aigMust(t, x.Close())
	aigMust(t, y.Close())
	aigMust(t, nx.Close())
	count, e = a.NumNodes()
	aigMust(t, e)
	if count != 5 {
		t.Fatalf("children released before owning ANDs: %d", count)
	}
	aigMust(t, same.Close())
	aigMust(t, n.Close())
	aigMust(t, different.Close())
	count, e = a.NumNodes()
	aigMust(t, e)
	if count != 1 {
		t.Fatalf("cascade left %d nodes", count)
	}
	z, e := a.MakeVariable(h)
	aigMust(t, e)
	zid, _, e := z.ID()
	aigMust(t, e)
	if zid != 1 {
		t.Fatalf("source LIFO variable reuse got %d", zid)
	}
	aigMust(t, z.Close())
	aigMust(t, a.Close())
}

func TestAIGCoreDuplicateStillAllocatesAndRefClones(t *testing.T) {
	m, b := aigFixture(t)
	h, e := m.MkTrue()
	aigMust(t, e)
	a, e := NewManager(m)
	aigMust(t, e)
	x, e := a.MakeVariable(h)
	aigMust(t, e)
	y, e := a.MakeVariable(h)
	aigMust(t, e)
	n, e := a.InternAnd(x, y)
	aigMust(t, e)
	clone, e := n.Clone()
	aigMust(t, e)
	if !clone.Same(n) {
		t.Fatal("clone lost node identity")
	}
	before := b.Used()
	b.FailAfter(0)
	if _, e := a.InternAnd(x, y); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("duplicate allocation: %v", e)
	}
	b.FailAfter(-1)
	if b.Used() != before {
		t.Fatalf("failed duplicate retained %d bytes", b.Used()-before)
	}
	aigMust(t, x.Close())
	aigMust(t, y.Close())
	aigMust(t, clone.Close())
	aigMust(t, n.Close())
	aigMust(t, a.Close())
}
