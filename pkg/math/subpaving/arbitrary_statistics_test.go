package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func TestArbitraryStatisticsConflictLeafAndTreeCleanup(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	child, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(child, f.x, &f.five, true, false, true); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(child, f.x, &f.four, false, false, true); err != nil {
		t.Fatal(err)
	}
	if child.conflictVar != f.x || child.queuedLeaf || c.leafHead != root || c.leafTail != root {
		t.Fatal("conflict must retire the child leaf without changing the parent")
	}
	if err := c.PublishBound(child, f.x, &f.four, true, false, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("conflicting node refuses another bound", err)
	}
	st := statistics.New()
	if err := c.CollectStatistics(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil statistics target", err)
	}
	if err := c.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key statistics.Key
		val uint32
	}{{"conflicts", 1}, {"new bounds", 2}, {"nodes", 2}}
	if st.Size() != len(want) {
		t.Fatal("source nonzero statistics count", st.Size())
	}
	for i, entry := range want {
		if st.Key(i) != entry.key || st.UintValue(i) != entry.val {
			t.Fatal("source statistic order/value", i, st.Key(i), st.UintValue(i))
		}
	}
	if err := c.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := c.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	if st.Size() != 1 || st.Key(0) != "nodes" || st.UintValue(0) != 2 {
		t.Fatal("reset retains live node count")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if !child.closed || !root.closed || root.children != 0 || c.leafHead != nil || len(c.nodes) != 0 {
		t.Fatal("tree cleanup must retire descendants before the root")
	}
	if err := c.CollectStatistics(st); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed statistics refusal", err)
	}
}
