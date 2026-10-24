package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestSATSolverBorrowedGlobalServiceCurrentModuleAndLocalOverrides(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	b := c.Budget()
	catalog, err := uparams.NewGlobalParamsCatalog(sy, uparams.GlobalParamsCatalogConfig{Budget: b})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, catalog.Close()) })
	globals, err := uparams.NewGlobalParamsService(sy, c, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, globals.Close()) })
	satOK(t, globals.InitializeSource())
	module, err := sy.Intern("sat")
	satOK(t, err)
	values, err := globals.Current.GetParams(module)
	satOK(t, err)
	key := "max_conflicts"
	satOK(t, c.SetUIntName(values, &key, 0))
	for _, name := range []string{"resolution", "scc", "asymm_branch"} {
		satOK(t, c.SetBoolName(values, &name, false))
	}
	key = "dimacs.display"
	satOK(t, c.SetBoolName(values, &key, true))
	runtime, err := NewRuntime(RuntimeConfig{Budget: b, ParamsContext: c, Globals: globals})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, runtime.Close()) })
	if runtime.globals != nil || runtime.globalService != globals {
		t.Fatal("the provided registry must stay borrowed")
	}
	limit, err := control.NewReslimit(b)
	satOK(t, err)
	t.Cleanup(limit.Close)
	p, err := c.NewValue()
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Del(&p)) })
	s, err := NewSATSolver(SolverConfig{ParamsContext: c, Params: p, RLimit: limit, Budget: b, Runtime: runtime})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, s.Close()) })
	state, err := s.config.State()
	satOK(t, err)
	if state.MaxConflicts != 0 || !state.DimacsDisplay || s.simplifier.resolution || s.scc.enabled || s.asymm.enabled {
		t.Fatal("global options missing from real solver children", state)
	}
	key = "max_conflicts"
	satOK(t, c.SetUIntName(&p, &key, 17))
	satOK(t, s.UpdateParams(&p))
	state, err = s.config.State()
	satOK(t, err)
	if state.MaxConflicts != 17 {
		t.Fatal("local value must override the global registry", state.MaxConflicts)
	}
	clone, err := s.config.Clone(b)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, clone.Close()) })
	satOK(t, c.Reset(&p))
	satOK(t, globals.FinalizeSource())
	satOK(t, globals.InitializeSource())
	satOK(t, s.UpdateParams(&p))
	satOK(t, clone.Update(&p))
	state, err = s.config.State()
	satOK(t, err)
	if state.MaxConflicts != ^uint32(0) || state.DimacsDisplay || !s.simplifier.resolution || !s.scc.enabled || !s.asymm.enabled {
		t.Fatal("updates must read the service's new current registry", state)
	}
	copy, err := clone.State()
	satOK(t, err)
	if copy.MaxConflicts != state.MaxConflicts || copy.DimacsDisplay != state.DimacsDisplay {
		t.Fatal("clone retained the old global registry")
	}
	satOK(t, s.Close())
	satOK(t, runtime.Close())
	satOK(t, globals.CheckOpen())
}
