package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestArbitraryEngineSelectedContextAndPartialCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	core, err := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{
		Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	})
	if err != nil {
		t.Fatal(err)
	}
	numbers, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	sy, err := symbol.NewManager(symbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, err := range []error{pc.Del(&p), pc.Close(), q.Close(), sy.Close(), numbers.Close(), core.Close(), r.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("MPF tactic owner roots", b.Used())
		}
	})
	limit := &control.Reslimit{Limit: 1000}
	if e, err := NewArbitraryEngine(limit, numbers, pc, &p); e != nil || !errors.Is(err, ErrArbitraryEngineSelection) {
		t.Fatal("default MPQ selection", e, err)
	}
	name := "mpf"
	selected, err := sy.Intern(name)
	if err != nil {
		t.Fatal(err)
	}
	numeralKey, depthKey, boundKey, printKey := "numeral", "max_depth", "max_bound", "print_nodes"
	if err := pc.SetSymbolName(&p, &numeralKey, selected); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &depthKey, 0); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &boundKey, 2); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	engine, err := NewArbitraryEngine(limit, numbers, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	options, err := engine.Options()
	if err != nil || options.Engine != EngineMPF || !options.PrintNodes {
		t.Fatal("selected MPF options", options, err)
	}
	shell, err := engine.Shell()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shell.MkVar(false); err != nil {
		t.Fatal(err)
	}
	if err := shell.Search(); err != nil {
		t.Fatal(err)
	}
	stats := statistics.New()
	if err := shell.CollectStatistics(stats); err != nil || stats.Size() != 1 || stats.Key(0) != "nodes" || stats.UintValue(0) != 1 {
		t.Fatal("selected search", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Shell(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed engine borrow", err)
	}
	var stillBorrowed numeral.ArbitraryFloat
	if err := numbers.SetInt32(&stillBorrowed, 1); err != nil {
		t.Fatal("engine closed borrowed manager", err)
	}
	if err := numbers.Del(&stillBorrowed); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &boundKey, 1<<31); err != nil {
		t.Fatal(err)
	}
	partial, err := NewArbitraryEngine(limit, numbers, pc, &p)
	if err == nil || partial == nil {
		t.Fatal("refused MPF context prefix", partial, err)
	}
	if err := partial.Close(); err != nil {
		t.Fatal("partial engine cleanup", err)
	}
}
