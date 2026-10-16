package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestGlobalParamsModuleMergeDuplicateKindSourceAssertion(t *testing.T) {
	if paramDescrsStreamSourceDebug {
		bindProcessAssertionRefusal(t)
	}
	ctx, symbols, _ := paramsTestOwner(t)
	budget := arena.NewBudget(^uint64(0))
	cfg := gparamsTestConfig(budget)
	cfg.Register = func(GlobalParamsRegistrar) error { return nil }
	g, err := NewGlobalParams(symbols, ctx, cfg)
	paramsMust(t, err)
	t.Cleanup(func() {
		paramsMust(t, g.Close())
		if budget.Used() != 0 {
			t.Fatal("module descriptor owners", budget.Used())
		}
	})
	first, err := NewParamDescrs(symbols, ParamDescrsConfig{Budget: budget})
	paramsMust(t, err)
	paramsMust(t, first.InsertName("seed", ParamUInt, pdstr("first"), pdstr("7"), nil))
	paramsMust(t, g.RegisterModule("sat", first))
	incoming, err := NewParamDescrs(symbols, ParamDescrsConfig{Budget: budget})
	paramsMust(t, err)
	paramsMust(t, incoming.InsertName("seed", ParamBool, pdstr("replacement"), pdstr("false"), nil))
	err = g.RegisterModule("sat", incoming)
	if paramDescrsStreamSourceDebug {
		if !errors.Is(err, udiag.ErrDebugAssertion) {
			t.Fatal("checked duplicate-kind assertion", err)
		}
		paramsMust(t, incoming.Close())
	} else if err != nil {
		t.Fatal("release duplicate module merge", err)
	}
	if got, err := g.GetValueName("sat.seed"); err != nil || got != "7" {
		t.Fatal("merge changed first descriptor", got, err)
	}
	if n, err := first.Size(); err != nil || n != 1 {
		t.Fatal("merge appended duplicate name", n, err)
	}
}
