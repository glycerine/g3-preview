package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func TestFastStatisticsConflictResetAndSearch(t *testing.T) {
	m, c, v, root := fastMonomialFixture(t, []bool{false})
	st := statistics.New()
	if err := c.CollectStatistics(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil statistics target", err)
	}
	if err := m.SetInt32(v, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, true, false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(v, 4); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, false, false); err != nil {
		t.Fatal(err)
	}
	if !root.inconsistent || root.queuedLeaf || len(c.leafQueue) != 0 {
		t.Fatal("conflict must remove root leaf")
	}
	if err := c.PropagateBound(root, 0, v, true, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("conflicting node refuses another bound", err)
	}
	if err := c.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key statistics.Key
		val uint32
	}{{"conflicts", 1}, {"new bounds", 2}, {"nodes", 1}}
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
	if err := c.CollectStatistics(st); err != nil || st.Size() != 1 || st.Key(0) != "nodes" || st.UintValue(0) != 1 {
		t.Fatal("reset must retain live node count", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.CollectStatistics(st); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context refusal", err)
	}

	_, search, _, _ := fastMonomialFixture(t, []bool{false})
	search.maxDepth = 1
	if err := search.Search(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := search.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	if st.Size() != 3 || st.Key(0) != "new bounds" || st.UintValue(0) != 2 || st.Key(1) != "splits" || st.UintValue(1) != 1 || st.Key(2) != "nodes" || st.UintValue(2) != 3 {
		t.Fatal("search decision bounds, split and live-node counters", st.Size())
	}
}
