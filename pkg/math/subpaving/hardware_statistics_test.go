package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func TestHardwareShellStatisticsCountersAndReset(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	child, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	var zero, one numeral.HardwareFloat
	if err := m.SetFloat64(&zero, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.SetFloat64(&one, 1); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(child, x, &one, true, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(child, x, &zero, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(child, x, &zero, true, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("bound after conflict: %v", err)
	}
	st := statistics.New()
	budget := arena.NewBudget(1 << 20)
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewHardwareShellCoefficients(c, qm)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CollectStatistics(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("nil statistics: %v", err)
	}
	if err := w.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key statistics.Key
		val uint32
	}{{"conflicts", 1}, {"new bounds", 2}, {"nodes", 2}}
	if st.Size() != len(want) {
		t.Fatalf("statistics size %d, want %d", st.Size(), len(want))
	}
	for i, entry := range want {
		if st.Key(i) != entry.key || st.UintValue(i) != entry.val {
			t.Fatalf("statistic %d: %s=%d, want %s=%d", i, st.Key(i), st.UintValue(i), entry.key, entry.val)
		}
	}
	if err := w.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := c.CollectStatistics(st); err != nil {
		t.Fatal(err)
	}
	if st.Size() != 1 || st.Key(0) != "nodes" || st.UintValue(0) != 2 {
		t.Fatal("reset must retain live node count")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := qm.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatal("rational manager roots", budget.Used())
	}
	if err := c.CollectStatistics(st); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed context: %v", err)
	}
}
