package solver

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/params"
)

func combinedSolverGlobals(t *testing.T, c *params.ParamsContext) *params.GlobalParamsService {
	t.Helper()
	b := c.Budget()
	catalog, err := params.NewGlobalParamsCatalog(c.Symbols(), params.GlobalParamsCatalogConfig{Budget: b})
	solverMust(t, err)
	service, err := params.NewGlobalParamsService(c.Symbols(), c, params.GlobalParamsConfig{Budget: b, Catalog: catalog})
	solverMust(t, err)
	solverMust(t, service.InitializeSource())
	t.Cleanup(func() {
		solverMust(t, service.Close())
		solverMust(t, catalog.Close())
	})
	return service
}

func TestCombinedSolverParamsDefaultsGlobalAndLocalPrecedence(t *testing.T) {
	m := solverTestManager(t)
	c := solverTestParams(t, m)
	g := combinedSolverGlobals(t, c)
	p, err := NewCombinedSolverParams(c, g, nil)
	solverMust(t, err)
	if timeout, e := p.Solver2Timeout(); e != nil || timeout != ^uint32(0) {
		t.Fatal("timeout default", timeout, e)
	}
	if ignore, e := p.IgnoreSolver1(); e != nil || ignore {
		t.Fatal("ignore default", ignore, e)
	}
	if unknown, e := p.Solver2Unknown(); e != nil || unknown != 1 {
		t.Fatal("unknown default", unknown, e)
	}
	solverMust(t, p.Close())

	manager, err := g.Manager()
	solverMust(t, err)
	module, err := m.Symbols.Intern("combined_solver")
	solverMust(t, err)
	global, err := manager.GetParams(module)
	solverMust(t, err)
	keyTimeout, err := m.Symbols.Intern("solver2_timeout")
	solverMust(t, err)
	keyIgnore, err := m.Symbols.Intern("ignore_solver1")
	solverMust(t, err)
	keyUnknown, err := m.Symbols.Intern("solver2_unknown")
	solverMust(t, err)
	solverMust(t, c.SetUInt(global, keyTimeout, 19))
	solverMust(t, c.SetBool(global, keyIgnore, true))
	solverMust(t, c.SetUInt(global, keyUnknown, 2))
	local, err := c.NewValue()
	solverMust(t, err)
	defer func() { solverMust(t, c.Del(&local)) }()
	solverMust(t, c.SetUInt(&local, keyTimeout, 7))
	p, err = NewCombinedSolverParams(c, g, &local)
	solverMust(t, err)
	if timeout, _ := p.Solver2Timeout(); timeout != 7 {
		t.Fatal("local precedence", timeout)
	}
	if ignore, _ := p.IgnoreSolver1(); !ignore {
		t.Fatal("global Boolean fallback")
	}
	if unknown, _ := p.Solver2Unknown(); unknown != 2 {
		t.Fatal("global uint fallback", unknown)
	}
	solverMust(t, p.Close())
	if _, err = p.Solver2Timeout(); !errors.Is(err, params.ErrParamsOwner) {
		t.Fatal("closed facade", err)
	}
}

func TestCombinedSolverParamDescriptors(t *testing.T) {
	m := solverTestManager(t)
	d, err := params.NewParamDescrs(m.Symbols, params.ParamDescrsConfig{Budget: m.Budget(), Text: arena.VectorConfig{Descriptors: arena.Config{Budget: m.Budget()}}})
	solverMust(t, err)
	defer func() { solverMust(t, d.Close()) }()
	solverMust(t, CollectCombinedSolverParamDescrs(d))
	if size, err := d.Size(); err != nil || size != 3 {
		t.Fatal("descriptor count", size, err)
	}
}
