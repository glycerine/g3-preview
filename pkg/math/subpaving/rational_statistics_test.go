package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func TestRationalStatisticsConflictResetAndNodeRetention(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	child, err := c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(child, x, value, true, false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 4); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(child, x, value, false, false); err != nil {
		t.Fatal(err)
	}
	st := statistics.New()
	if err := c.CollectStatistics(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil destination", err)
	}
	if err := c.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key statistics.Key
		val uint32
	}{{"conflicts", 1}, {"new bounds", 2}, {"nodes", 2}}
	if st.Size() != len(want) {
		t.Fatal("source nonzero count", st.Size())
	}
	for i, v := range want {
		if st.Key(i) != v.key || st.UintValue(i) != v.val {
			t.Fatal("source order and value", i, st.Key(i), st.UintValue(i))
		}
	}
	if err := c.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := c.CollectStatistics(st); err != nil || st.Size() != 1 || st.Key(0) != "nodes" || st.UintValue(0) != 2 {
		t.Fatal("reset keeps live nodes", err)
	}
}
