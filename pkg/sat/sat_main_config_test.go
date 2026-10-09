package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func satMainConfigTestRuntime(t *testing.T) (*util.ParamsContext, *usymbol.Manager, *util.GlobalParams, *arena.Budget) {
	t.Helper()
	c, s, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	g, e := util.NewGlobalParams(s, c, util.GlobalParamsConfig{Budget: b, Register: func(util.GlobalParamsRegistrar) error {
		t.Fatal("configuration GetModule must not lazily register descriptors")
		return nil
	}})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal("registry/config graph", b.Used())
		}
	})
	return c, s, g, b
}

func TestSATMainConfigActualDefaultsAndIndeterminateGCFields(t *testing.T) {
	c, _, g, b := satMainConfigTestRuntime(t)
	local, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c.Del(&local)) })
	m, e := newSATMainConfig(g, &local, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, m.Close()) })
	s, e := m.State()
	paramsMust(t, e)
	if s.MaxMemory != uint64(^uint(0)) || s.Phase != satPhaseCaching || s.Restart != satRestartLuby || s.GCStrategy != satGCGluePSM {
		t.Fatal("source policy defaults", s)
	}
	if s.PhaseCachingOn != 400 || s.PhaseCachingOff != 100 || s.RestartInitial != 100 || s.RestartFactor != 1.5 || s.RestartMax != ^uint32(0) || s.RandomFreq != .01 || s.RandomSeed != 0 || s.BurstSearch != 100 || s.MaxConflicts != ^uint32(0) || s.NumParallel != 1 {
		t.Fatal("source search defaults", s)
	}
	if s.SimplifyMult1 != 300 || s.SimplifyMult2 != 1.5 || s.SimplifyMax != 500000 || s.GCInitial != 20000 || s.GCIncrement != 500 || !s.MinimizeLemmas || !s.DynSubRes || s.CoreMinimize || s.CoreMinimizePartial || s.DimacsDisplay {
		t.Fatal("source simplification defaults", s)
	}
	if s.Initialized&(satConfigGCSmallLBDMask|satConfigGCKMask) != 0 {
		t.Fatal("source glue_psm does not initialize dyn-only fields", s.Initialized)
	}
}

func TestSATMainConfigCurrentRealModuleAndWrongKindFallback(t *testing.T) {
	c, sy, g, b := satMainConfigTestRuntime(t)
	local, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c.Del(&local)) })
	module, e := sy.Intern("sat")
	paramsMust(t, e)
	// Populate the actual source get_params cell. GetModule must retain it
	// anew on every Update; this is not a caller-injected fallback snapshot.
	ref, e := g.GetParams(module)
	paramsMust(t, e)
	name := "restart.initial"
	paramsMust(t, c.SetUIntName(ref, &name, 321))
	paramsMust(t, c.SetBoolName(&local, &name, false))
	m, e := newSATMainConfig(g, &local, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, m.Close()) })
	s, e := m.State()
	paramsMust(t, e)
	if s.RestartInitial != 321 {
		t.Fatal("wrong local kind must fall back", s.RestartInitial)
	}
	paramsMust(t, c.SetUIntName(ref, &name, 654))
	paramsMust(t, m.Update(&local))
	s, e = m.State()
	paramsMust(t, e)
	if s.RestartInitial != 654 {
		t.Fatal("stale module snapshot", s.RestartInitial)
	}
	paramsMust(t, c.SetUIntName(&local, &name, 77))
	paramsMust(t, m.Update(&local))
	s, e = m.State()
	paramsMust(t, e)
	if s.RestartInitial != 77 {
		t.Fatal("local priority", s.RestartInitial)
	}
	paramsMust(t, c.Reset(&local))
	paramsMust(t, g.Reset())
	paramsMust(t, m.Update(&local))
	s, e = m.State()
	paramsMust(t, e)
	if s.RestartInitial != 100 {
		t.Fatal("removed real module must use source literal", s.RestartInitial)
	}
}

func TestSATMainConfigSequentialInvalidPhasePrefix(t *testing.T) {
	c, sy, g, b := satMainConfigTestRuntime(t)
	local, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c.Del(&local)) })
	m, e := newSATMainConfig(g, &local, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, m.Close()) })
	before, e := m.State()
	paramsMust(t, e)
	name := "max_memory"
	paramsMust(t, c.SetUIntName(&local, &name, 5))
	name = "restart"
	geometric, e := sy.Intern("geometric")
	paramsMust(t, e)
	paramsMust(t, c.SetSymbolName(&local, &name, geometric))
	name = "phase"
	invalid, e := sy.Intern("invalid_source_phase")
	paramsMust(t, e)
	paramsMust(t, c.SetSymbolName(&local, &name, invalid))
	e = m.Update(&local)
	if e == nil || e.Error() != "invalid phase selection strategy" {
		t.Fatal(e)
	}
	after, e := m.State()
	paramsMust(t, e)
	before.MaxMemory = 5 * 1024 * 1024
	before.Restart = satRestartGeometric
	if before != after {
		t.Fatal("completed prefix or untouched suffix", before, after)
	}
}
