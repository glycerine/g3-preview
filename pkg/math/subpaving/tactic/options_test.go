package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSubpavingTacticOptionSelectionAndDescriptions(t *testing.T) {
	budget := arena.NewBudget(1 << 25)
	sy, err := symbol.NewManager(symbol.Config{Budget: budget, PageSize: 64, DescriptorPageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: budget, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	d, err := params.NewParamDescrs(sy, params.ParamDescrsConfig{Budget: budget, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
		if err := pc.Del(&p); err != nil {
			t.Error(err)
		}
		if err := pc.Close(); err != nil {
			t.Error(err)
		}
		if err := q.Close(); err != nil {
			t.Error(err)
		}
		if err := sy.Close(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("option owner roots", budget.Used())
		}
	})
	read := func(want EngineKind, print bool) {
		t.Helper()
		got, err := ReadOptions(pc, &p)
		if err != nil || got.Engine != want || got.PrintNodes != print {
			t.Fatalf("options %+v: %v; want engine %d print %t", got, err, want, print)
		}
	}
	read(EngineMPQ, false)
	printKey, numeralKey := "print_nodes", "numeral"
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		name string
		kind EngineKind
	}{
		{"mpq", EngineMPQ}, {"mpf", EngineMPF}, {"mpff", EngineMPFF},
		{"mpfx", EngineMPFX}, {"hwf", EngineHWF}, {"unrecognized", EngineHWF},
	} {
		value, err := sy.Intern(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if err := pc.SetSymbolName(&p, &numeralKey, value); err != nil {
			t.Fatal(err)
		}
		read(entry.kind, true)
	}
	if err := pc.SetBoolName(&p, &numeralKey, true); err != nil {
		t.Fatal(err)
	}
	read(EngineMPQ, true) // wrong-kind numeral entry uses the source default
	if err := pc.Reset(&p); err != nil {
		t.Fatal(err)
	}
	read(EngineMPQ, false)
	if got, err := ReadOptions(nil, &p); err == nil || got.Engine != EngineNone {
		t.Fatalf("missing parameter context: %+v, %v", got, err)
	}
	if err := CollectParamDescriptions(d); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name, text string
		kind       params.ParamKind
	}{
		{"max_nodes", "(default: 8192) maximum number of nodes in the subpaving tree.", params.ParamUInt},
		{"max_depth", "(default: 128) maximum depth of the subpaving tree.", params.ParamUInt},
		{"epsilon", "(default: 20) value k s.t. a new lower (upper) bound for x is propagated only new-lower(x) > lower(k) + 1/k * max(min(upper(x) - lower(x), |lower|), 1) (new-upper(x) < upper(x) - 1/k * max(min(upper(x) - lower(x), |lower|), 1)). If k = 0, then this restriction is ignored.", params.ParamUInt},
		{"max_bound", "(default 10) value k s.t. a new upper (lower) bound for x is propagated only if upper(x) > -10^k or lower(x) = -oo (lower(x) < 10^k or upper(x) = oo)", params.ParamUInt},
		{"nth_root_precision", "(default 8192) value k s.t. 1/k is the precision for computing the nth root in the subpaving module.", params.ParamUInt},
		{"numeral", "(default: mpq) options: mpq, mpf, hwf, mpff, mpfx.", params.ParamSymbol},
		{"print_nodes", "(default: false) display subpaving tree leaves.", params.ParamBool},
	}
	size, err := d.Size()
	if err != nil || size != uint32(len(want)) {
		t.Fatalf("description count %d: %v", size, err)
	}
	for i, entry := range want {
		key, err := d.ParamName(uint32(i))
		if err != nil {
			t.Fatal(err)
		}
		name, err := sy.String(key)
		if err != nil || name != entry.name {
			t.Fatalf("description name %d: %q, %v", i, name, err)
		}
		kind, err := d.Kind(key)
		if err != nil || kind != entry.kind {
			t.Fatalf("description kind %s: %d, %v", name, kind, err)
		}
		text, present, err := d.Description(key)
		if err != nil || !present || text != entry.text {
			t.Fatalf("description text %s: %q, %v", name, text, err)
		}
	}
}
