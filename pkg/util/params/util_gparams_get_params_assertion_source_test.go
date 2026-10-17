package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestGlobalParamsGetParamsReadsSourceAssertionAfterModulePublication(t *testing.T) {
	ctx, symbols, _ := paramsTestOwner(t)
	budget := arena.NewBudget(1 << 20)
	g, err := NewGlobalParams(symbols, ctx, gparamsTestConfig(budget))
	paramsMust(t, err)
	module, err := symbols.Intern("source-get-params-assertion")
	paramsMust(t, err)
	d, _, _ := debugFixture(t)
	old, err := udiag.BindProcessDebugContext(d)
	paramsMust(t, err)
	t.Cleanup(func() {
		_, e := udiag.BindProcessDebugContext(old)
		paramsMust(t, e)
	})
	paramsMust(t, d.Close())
	ref, err := g.GetParams(module)
	if GlobalParamsSourceDebug {
		if ref != nil || !errors.Is(err, arena.ErrClosed) {
			t.Fatal("source assertion switch after module insertion", ref, err)
		}
	} else if ref == nil || err != nil {
		t.Fatal("erased source assertion after module insertion", ref, err)
	}
	if g.ModuleParams.size() != 1 || len(g.Cells) != 1 || g.Cells[0] == nil {
		t.Fatal("source module cell was not published before assertion", g.ModuleParams.size(), len(g.Cells))
	}
	ref, err = g.GetParams(module)
	if GlobalParamsSourceDebug {
		if ref != nil || !errors.Is(err, arena.ErrClosed) {
			t.Fatal("source assertion switch after module lookup", ref, err)
		}
	} else if ref != &g.Cells[0].value || err != nil {
		t.Fatal("erased source assertion after module lookup", ref, err)
	}
	global, err := g.GetParams(usymbol.Symbol{})
	if err != nil || global != &g.globalParams {
		t.Fatal("global parameters bypass module assertion", global, err)
	}
	_, err = udiag.BindProcessDebugContext(old)
	paramsMust(t, err)
	paramsMust(t, g.SourceFinalize())
	if budget.Used() != 0 {
		t.Fatal("retained module cell after source finalization", budget.Used())
	}
}
