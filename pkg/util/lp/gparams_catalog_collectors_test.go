package lp

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func lpParamFixture(t *testing.T) (*uparams.ParamDescrs, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	sy, err := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 1, DescriptorPageSize: 1, DirectOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	d, err := uparams.NewParamDescrs(sy, uparams.ParamDescrsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		if err := d.Close(); err != nil {
			t.Error(err)
		}
		if err := sy.Close(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Error("parameter fixture retained", b.Used())
		}
	})
	return d, b
}

// Source: src/util/lp/lp_params.pyg; independent literal schema oracle.
func TestLPParameterCatalogSchema(t *testing.T) {
	d, _ := lpParamFixture(t)
	if err := catalogCollectLpParams(d); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key        string
		kind       uparams.ParamKind
		def, descr string
	}{
		{"rep_freq", uparams.ParamUInt, "0", "the report frequency, in how many iterations print the cost and other info "},
		{"min", uparams.ParamBool, "false", "minimize cost"},
		{"print_stats", uparams.ParamBool, "false", "print statistic"},
		{"simplex_strategy", uparams.ParamUInt, "0", "simplex strategy for the solver"},
		{"bprop_on_pivoted_rows", uparams.ParamBool, "true", "propagate bounds on rows changed by the pivot operation"},
	}
	n, err := d.Size()
	if err != nil || int(n) != len(want) {
		t.Fatal(n, err)
	}
	for _, w := range want {
		if k, err := d.KindName(w.key); err != nil || k != w.kind {
			t.Fatal(w.key, k, err)
		}
		if v, p, err := d.DefaultName(w.key); err != nil || !p || v != w.def {
			t.Fatal(w.key, v, p, err)
		}
		if v, p, err := d.DescriptionName(w.key); err != nil || !p || v != w.descr {
			t.Fatal(w.key, v, p, err)
		}
		if v, p, err := d.ModuleName(w.key); err != nil || !p || v != "lp" {
			t.Fatal(w.key, v, p, err)
		}
	}
	if err := catalogInsert(d, "local", uparams.ParamBool, "description", "true", ""); err != nil {
		t.Fatal(err)
	}
	if _, p, err := d.ModuleName("local"); err != nil || p {
		t.Fatal(p, err)
	}
}

func TestLPParameterCatalogReachedFailurePrefixes(t *testing.T) {
	keys := []string{"rep_freq", "min", "print_stats", "simplex_strategy", "bprop_on_pivoted_rows"}
	for i, key := range keys {
		t.Run(key, func(t *testing.T) {
			d, _ := lpParamFixture(t)
			if err := d.InsertName(key, uparams.ParamString, nil, nil, nil); err != nil {
				t.Fatal(err)
			}
			if err := catalogCollectLpParams(d); err != uparams.ErrParamDescrsPrecondition {
				t.Fatal("actual kind collision", err)
			}
			n, err := d.Size()
			if err != nil || n != uint32(i+1) {
				t.Fatal("completed prefix", n, err)
			}
			for j, name := range keys {
				present, err := d.ContainsName(name)
				if err != nil || present != (j <= i) {
					t.Fatal(name, present, err)
				}
			}
		})
	}
}

func TestLPParameterCatalogNativeStorageIgnoresRuntimeReservationFault(t *testing.T) {
	d, b := lpParamFixture(t)
	before := b.Used()
	b.FailAfter(0)
	if err := catalogCollectLpParams(d); err != nil {
		t.Fatal(err)
	}
	if b.Used() != before {
		t.Fatal("Go-managed maps and slices invented source allocation", b.Used(), before)
	}
}
