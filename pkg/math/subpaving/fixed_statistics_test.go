package subpaving

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func TestFixedStatisticsSourceEventsOrderAndLiveNodes(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	v, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	child, err := c.NewChild(c.root)
	if err != nil || c.numNodes != 2 {
		t.Fatal("source child count", c.numNodes, err)
	}
	if err := child.Close(); err != nil || c.numNodes != 1 {
		t.Fatal("source child retirement", c.numNodes, err)
	}
	if err := m.SetInt64(v, 1); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(c.root, 0, v, true, false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(v, 0); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(c.root, 0, v, false, false); err != nil {
		t.Fatal(err)
	}
	if !c.root.inconsistent || c.numBounds != 2 || c.numConflicts != 1 {
		t.Fatal("bound and conflict events", c.numBounds, c.numConflicts)
	}
	st := statistics.New()
	if err := c.CollectStatistics(st); err != nil || st.Size() != 3 {
		t.Fatal("source statistic count", err)
	}
	for i, tc := range []struct {
		name statistics.Key
		want uint32
	}{{"conflicts", 1}, {"new bounds", 2}, {"nodes", 1}} {
		if st.Key(i) != tc.name || st.UintValue(i) != tc.want {
			t.Fatal("source statistic order/value", i, st.Key(i), st.UintValue(i), tc)
		}
	}
	if err := c.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st = statistics.New()
	if err := c.CollectStatistics(st); err != nil || st.Size() != 1 || st.Key(0) != "nodes" || st.UintValue(0) != 1 {
		t.Fatal("reset keeps live node count", err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil || c.numNodes != 0 {
		t.Fatal("root retirement count", c.numNodes, err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}
