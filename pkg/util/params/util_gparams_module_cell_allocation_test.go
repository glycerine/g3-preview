package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestGlobalParamsModuleCellAllocationAndRelease(t *testing.T) {
	ctx, symbols, _ := paramsTestOwner(t)
	budget := arena.NewBudget(1 << 20)
	g, err := NewGlobalParams(symbols, ctx, gparamsTestConfig(budget))
	paramsMust(t, err)
	module, err := symbols.Intern("module-cell")
	paramsMust(t, err)
	base := budget.Used()
	budget.FailAfter(0)
	_, err = g.GetParams(module)
	budget.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || budget.Used() != base || g.ModuleParams.size() != 0 || len(g.Cells) != 0 {
		t.Fatal("failed placement published module cell", err, budget.Used(), base, g.ModuleParams.size(), len(g.Cells))
	}
	first, err := g.GetParams(module)
	paramsMust(t, err)
	if budget.Used() != base+gparamsRefCellBytes {
		t.Fatal("source alloc(params_ref) charge", budget.Used(), base, gparamsRefCellBytes)
	}
	again, err := g.GetParams(module)
	paramsMust(t, err)
	if again != first || budget.Used() != base+gparamsRefCellBytes {
		t.Fatal("lookup allocated a second module cell", again, first, budget.Used())
	}
	paramsMust(t, g.Reset())
	if budget.Used() != base || g.ModuleParams.size() != 0 {
		t.Fatal("reset retained module placement", budget.Used(), base, g.ModuleParams.size())
	}
	_, err = g.GetParams(module)
	paramsMust(t, err)
	if budget.Used() != base+gparamsRefCellBytes {
		t.Fatal("fresh module allocation after reset", budget.Used())
	}
	paramsMust(t, g.SourceFinalize())
	if budget.Used() != 0 {
		t.Fatal("source destructor retained module cell", budget.Used())
	}
}
