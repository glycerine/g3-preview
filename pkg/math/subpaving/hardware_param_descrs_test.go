package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestHardwareShellParamDescriptionsSourceOrder(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	sy, err := symbol.NewManager(symbol.Config{Budget: budget, PageSize: 64, DescriptorPageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	d, err := params.NewParamDescrs(sy, params.ParamDescrsConfig{Budget: budget, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewHardwareShellCoefficients(c, qm)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := qm.CloseChecked(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
		if err := sy.Close(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("descriptor roots", budget.Used())
		}
	})
	if err := CollectHardwareParamDescrs(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("nil descriptions: %v", err)
	}
	if err := w.CollectParamDescrs(d); err != nil {
		t.Fatal(err)
	}
	want := []struct{ name, description string }{
		{"max_nodes", "(default: 8192) maximum number of nodes in the subpaving tree."},
		{"max_depth", "(default: 128) maximum depth of the subpaving tree."},
		{"epsilon", "(default: 20) value k s.t. a new lower (upper) bound for x is propagated only new-lower(x) > lower(k) + 1/k * max(min(upper(x) - lower(x), |lower|), 1) (new-upper(x) < upper(x) - 1/k * max(min(upper(x) - lower(x), |lower|), 1)). If k = 0, then this restriction is ignored."},
		{"max_bound", "(default 10) value k s.t. a new upper (lower) bound for x is propagated only if upper(x) > -10^k or lower(x) = -oo (lower(x) < 10^k or upper(x) = oo)"},
		{"nth_root_precision", "(default 8192) value k s.t. 1/k is the precision for computing the nth root in the subpaving module."},
	}
	check := func() {
		t.Helper()
		size, err := d.Size()
		if err != nil || size != uint32(len(want)) {
			t.Fatalf("descriptor size %d: %v", size, err)
		}
		for i, entry := range want {
			key, err := d.ParamName(uint32(i))
			if err != nil {
				t.Fatal(err)
			}
			name, err := sy.String(key)
			if err != nil || name != entry.name {
				t.Fatalf("descriptor %d name %q: %v", i, name, err)
			}
			kind, err := d.Kind(key)
			if err != nil || kind != params.ParamUInt {
				t.Fatalf("descriptor %s kind %d: %v", name, kind, err)
			}
			text, present, err := d.Description(key)
			if err != nil || !present || text != entry.description {
				t.Fatalf("descriptor %s text %q: %v", name, text, err)
			}
		}
	}
	check()
	if err := w.CollectParamDescrs(d); err != nil {
		t.Fatal(err)
	}
	check() // source insert of matching keys is idempotent
}
