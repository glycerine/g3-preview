package sat

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func satConfigFixture(t *testing.T) (*util.ParamsContext, *usymbol.Manager, *util.GlobalParams, *arena.Budget, *util.ParamsRef, *satMainConfig) {
	t.Helper()
	c, sy, g, b := satMainConfigTestRuntime(t)
	p, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c.Del(&p)) })
	m, e := newSATMainConfig(g, &p, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, m.Close()) })
	return c, sy, g, b, &p, m
}
func satConfigSymbol(t *testing.T, c *util.ParamsContext, sy *usymbol.Manager, p *util.ParamsRef, key, value string) {
	t.Helper()
	s, e := sy.Intern(value)
	paramsMust(t, e)
	paramsMust(t, c.SetSymbolName(p, &key, s))
}
func TestSATMainConfigEveryPolicyAndDynamicRetention(t *testing.T) {
	c, sy, _, _, p, m := satConfigFixture(t)
	for i, n := range []string{"always_true", "always_false", "caching", "random"} {
		satConfigSymbol(t, c, sy, p, "phase", n)
		paramsMust(t, m.Update(p))
		s, e := m.State()
		paramsMust(t, e)
		if s.Phase != satPhaseSelection(i) {
			t.Fatal(n, s.Phase)
		}
	}
	for i, n := range []string{"geometric", "luby"} {
		satConfigSymbol(t, c, sy, p, "restart", n)
		paramsMust(t, m.Update(p))
		s, e := m.State()
		paramsMust(t, e)
		if s.Restart != satRestartStrategy(i) {
			t.Fatal(n, s.Restart)
		}
	}
	k := "gc.k"
	paramsMust(t, c.SetUIntName(p, &k, 256))
	k = "gc.small_lbd"
	paramsMust(t, c.SetUIntName(p, &k, 0))
	for i, n := range []string{"dyn_psm", "psm", "glue", "glue_psm", "psm_glue"} {
		satConfigSymbol(t, c, sy, p, "gc", n)
		paramsMust(t, m.Update(p))
		s, e := m.State()
		paramsMust(t, e)
		if s.GCStrategy != satGCStrategy(i) || s.GCK != 255 || s.GCSmallLBD != 0 || s.Initialized != satConfigAllFieldsMask {
			t.Fatal(n, s)
		}
	}
	for _, v := range []uint32{0, 1, 254, 255, 256, ^uint32(0)} {
		satConfigSymbol(t, c, sy, p, "gc", "dyn_psm")
		k = "gc.k"
		paramsMust(t, c.SetUIntName(p, &k, v))
		paramsMust(t, m.Update(p))
		s, e := m.State()
		paramsMust(t, e)
		want := v
		if want > 255 {
			want = 255
		}
		if s.GCK != want {
			t.Fatal(v, s.GCK)
		}
	}
}
func TestSATMainConfigAllScalarRoutesAndLocalHiddenFields(t *testing.T) {
	c, sy, g, _, p, m := satConfigFixture(t)
	sat, e := sy.Intern("sat")
	paramsMust(t, e)
	mod, e := g.GetParams(sat)
	paramsMust(t, e)
	for _, key := range []string{"simplify_mult1", "simplify_max"} {
		paramsMust(t, c.SetUIntName(mod, &key, 99))
	}
	key := "simplify_mult2"
	paramsMust(t, c.SetDoubleName(mod, &key, 99))
	paramsMust(t, m.Update(p))
	s, e := m.State()
	paramsMust(t, e)
	if s.SimplifyMult1 != 300 || s.SimplifyMax != 500000 || s.SimplifyMult2 != 1.5 {
		t.Fatal("hidden fields consulted module", s)
	}
	for _, key := range []string{"phase.caching.on", "phase.caching.off", "restart.initial", "restart.max", "random_seed", "burst_search", "max_conflicts", "parallel_threads", "simplify_mult1", "simplify_max", "gc.initial", "gc.increment", "gc.small_lbd", "gc.k"} {
		paramsMust(t, c.SetUIntName(p, &key, 0))
	}
	for _, key := range []string{"restart.factor", "random_freq", "simplify_mult2"} {
		paramsMust(t, c.SetDoubleName(p, &key, -2.5))
	}
	for _, key := range []string{"minimize_lemmas", "dyn_sub_res", "core.minimize", "core.minimize_partial", "dimacs.display"} {
		paramsMust(t, c.SetBoolName(p, &key, false))
	}
	satConfigSymbol(t, c, sy, p, "gc", "dyn_psm")
	paramsMust(t, m.Update(p))
	s, e = m.State()
	paramsMust(t, e)
	if s.PhaseCachingOn|s.PhaseCachingOff|s.RestartInitial|s.RestartMax|s.RandomSeed|s.BurstSearch|s.MaxConflicts|s.NumParallel|s.SimplifyMult1|s.SimplifyMax|s.GCInitial|s.GCIncrement|s.GCSmallLBD|s.GCK != 0 || s.RestartFactor != -2.5 || s.RandomFreq != -2.5 || s.SimplifyMult2 != -2.5 || s.MinimizeLemmas || s.DynSubRes || s.CoreMinimize || s.CoreMinimizePartial || s.DimacsDisplay {
		t.Fatal(s)
	}
	for _, key := range []string{"minimize_lemmas", "dyn_sub_res", "core.minimize", "core.minimize_partial", "dimacs.display"} {
		paramsMust(t, c.SetBoolName(p, &key, true))
	}
	key = "parallel_threads"
	paramsMust(t, c.SetUIntName(p, &key, ^uint32(0)))
	paramsMust(t, m.Update(p))
	s, e = m.State()
	paramsMust(t, e)
	if !s.MinimizeLemmas || !s.DynSubRes || !s.CoreMinimize || !s.CoreMinimizePartial || !s.DimacsDisplay || s.NumParallel != ^uint32(0) {
		t.Fatal(s)
	}
}
func TestSATMainConfigTypedModuleFallbackAndSeedLocalReread(t *testing.T) {
	c, sy, g, _, p, m := satConfigFixture(t)
	sat, e := sy.Intern("sat")
	paramsMust(t, e)
	mod, e := g.GetParams(sat)
	paramsMust(t, e)
	key := "random_freq"
	paramsMust(t, c.SetDoubleName(mod, &key, .75))
	paramsMust(t, c.SetUIntName(p, &key, 1))
	key = "minimize_lemmas"
	paramsMust(t, c.SetBoolName(mod, &key, false))
	paramsMust(t, c.SetUIntName(p, &key, 1))
	satConfigSymbol(t, c, sy, mod, "phase", "random")
	key = "phase"
	paramsMust(t, c.SetUIntName(p, &key, 1))
	key = "random_seed"
	paramsMust(t, c.SetUIntName(mod, &key, 67))
	paramsMust(t, m.Update(p))
	s, e := m.State()
	paramsMust(t, e)
	if s.RandomFreq != .75 || s.MinimizeLemmas || s.Phase != satPhaseRandom || s.RandomSeed != 67 {
		t.Fatal(s)
	}
	paramsMust(t, c.SetUIntName(p, &key, 0))
	paramsMust(t, m.Update(p))
	s, e = m.State()
	paramsMust(t, e)
	if s.RandomSeed != 0 {
		t.Fatal("local seed zero must remain zero", s.RandomSeed)
	}
	paramsMust(t, c.SetBoolName(p, &key, true))
	paramsMust(t, c.SetUIntName(mod, &key, 0))
	paramsMust(t, m.Update(p))
	s, e = m.State()
	paramsMust(t, e)
	if s.RandomSeed != 0 {
		t.Fatal(s.RandomSeed)
	}
}
func TestSATMainConfigInvalidRestartAndGCPrefixes(t *testing.T) {
	c, sy, _, _, p, m := satConfigFixture(t)
	before, e := m.State()
	paramsMust(t, e)
	key := "max_memory"
	paramsMust(t, c.SetUIntName(p, &key, 9))
	satConfigSymbol(t, c, sy, p, "restart", "bad")
	e = m.Update(p)
	if e == nil || e.Error() != "invalid restart strategy" {
		t.Fatal(e)
	}
	after, e := m.State()
	paramsMust(t, e)
	before.MaxMemory = 9 << 20
	if after != before {
		t.Fatal(before, after)
	}
	satConfigSymbol(t, c, sy, p, "restart", "geometric")
	key = "parallel_threads"
	paramsMust(t, c.SetUIntName(p, &key, 41))
	key = "minimize_lemmas"
	paramsMust(t, c.SetBoolName(p, &key, false))
	satConfigSymbol(t, c, sy, p, "gc", "bad")
	roots := c.Stats()
	e = m.Update(p)
	if e == nil || e.Error() != "invalid gc strategy" {
		t.Fatal(e)
	}
	after, e = m.State()
	paramsMust(t, e)
	before.Restart = satRestartGeometric
	before.NumParallel = 41
	if after != before {
		t.Fatal(before, after)
	}
	now := c.Stats()
	if roots.LiveBlocks != now.LiveBlocks || roots.References != now.References {
		t.Fatal("snapshot leaked", roots, now)
	}
}
func TestSATMainConfigIEEEParametersUnrestricted(t *testing.T) {
	c, _, _, _, p, m := satConfigFixture(t)
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.Copysign(0, -1), -1, 2} {
		for _, key := range []string{"restart.factor", "random_freq", "simplify_mult2"} {
			paramsMust(t, c.SetDoubleName(p, &key, v))
		}
		paramsMust(t, m.Update(p))
		s, e := m.State()
		paramsMust(t, e)
		for _, got := range []float64{s.RestartFactor, s.RandomFreq, s.SimplifyMult2} {
			if math.Float64bits(got) != math.Float64bits(v) {
				t.Fatal(v, got)
			}
		}
	}
}
func TestSATMainConfigMegabyteSourceHostWidths(t *testing.T) {
	for _, x := range []struct {
		mb       uint32
		w32, w64 uint64
	}{{0, 0, 0}, {1, 1 << 20, 1 << 20}, {4095, 4095 << 20, 4095 << 20}, {4096, ^uint64(0) >> 32, 4096 << 20}, {^uint32(0) - 1, ^uint64(0) >> 32, uint64(^uint32(0)-1) << 20}, {^uint32(0), ^uint64(0) >> 32, ^uint64(0)}} {
		if got := satConfigMegabytesForBits(x.mb, 32); got != x.w32 {
			t.Fatal(x, got)
		}
		if got := satConfigMegabytesForBits(x.mb, 64); got != x.w64 {
			t.Fatal(x, got)
		}
	}
}
func TestSATMainConfigCloneAssignImplicitScalarCopy(t *testing.T) {
	c, sy, _, b, p, m := satConfigFixture(t)
	satConfigSymbol(t, c, sy, p, "gc", "dyn_psm")
	paramsMust(t, m.Update(p))
	before, e := m.State()
	paramsMust(t, e)
	roots := c.Stats()
	symbols := sy.Stats().Symbols
	used := b.Used()
	n, e := m.Clone(b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, n.Close()) })
	if b.Used() != used+uint64(reflect.TypeOf(satMainConfig{}).Size()) || sy.Stats().Symbols != symbols {
		t.Fatal("clone must charge only header")
	}
	if after, e := n.State(); e != nil || after != before {
		t.Fatal(after, e)
	}
	satConfigSymbol(t, c, sy, p, "phase", "random")
	paramsMust(t, m.Update(p))
	paramsMust(t, n.Assign(m))
	paramsMust(t, n.Assign(n))
	a, e := n.State()
	paramsMust(t, e)
	s, e := m.State()
	paramsMust(t, e)
	if a != s || c.Stats().References != roots.References {
		t.Fatal(a, s)
	}
	paramsMust(t, m.Close())
	if _, e = n.State(); e != nil {
		t.Fatal("clone lifetime independent", e)
	}
	paramsMust(t, n.Close())
	if b.Used() != used-uint64(reflect.TypeOf(satMainConfig{}).Size()) {
		t.Fatal("close accounting", b.Used(), used)
	}
}
func TestSATMainConfigOwnerAndLifetimeGuards(t *testing.T) {
	c, _, _, b, p, m := satConfigFixture(t)
	c2, _, g2, _ := satMainConfigTestRuntime(t)
	p2, e := c2.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c2.Del(&p2)) })
	n, e := newSATMainConfig(g2, &p2, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, n.Close()) })
	before, e := m.State()
	paramsMust(t, e)
	if e = m.Update(&p2); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal(e)
	}
	if e = m.Assign(n); !errors.Is(e, ErrSATMainConfigOwner) {
		t.Fatal(e)
	}
	copyOwner := *m
	if _, e = copyOwner.State(); !errors.Is(e, ErrSATMainConfigOwner) {
		t.Fatal(e)
	}
	if after, e := m.State(); e != nil || after != before {
		t.Fatal(after, e)
	}
	paramsMust(t, m.Close())
	paramsMust(t, m.Close())
	if _, e = m.State(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e = m.Update(p); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	_ = c
}
func TestSATMainConfigConstructorFailureAndWarmZeroAllocation(t *testing.T) {
	c, _, g, b, p, m := satConfigFixture(t)
	before := b.Used()
	b.FailAfter(0)
	n, e := newSATMainConfig(g, p, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || n != nil || b.Used() != before {
		t.Fatal(n, e, b.Used(), before)
	}
	b.FailAfter(-1)
	module, e := g.SymbolsField.Intern("sat")
	paramsMust(t, e)
	ref, e := g.GetParams(module)
	paramsMust(t, e)
	key := "restart.initial"
	paramsMust(t, c.SetUIntName(ref, &key, 231))
	paramsMust(t, m.Update(p))
	roots := c.Stats()
	before = b.Used()
	b.FailAfter(0)
	c.BudgetField.FailAfter(0)
	if allocs := testing.AllocsPerRun(50, func() {
		if e := m.Update(p); e != nil {
			panic(e)
		}
	}); allocs != 0 {
		t.Fatal("warm update allocations", allocs)
	}
	b.FailAfter(-1)
	c.BudgetField.FailAfter(-1)
	now := c.Stats()
	if now.References != roots.References || now.LiveBlocks != roots.LiveBlocks || b.Used() != before {
		t.Fatal("snapshot roots/budget", roots, now)
	}
}

func TestSATMainConfigColdSymbolFaultPrefixes(t *testing.T) {
	completed := false
	for fail := int64(0); fail < 128 && !completed; fail++ {
		t.Run(strconv.FormatInt(fail, 10), func(t *testing.T) {
			c, sy, g, b := satMainConfigTestRuntime(t)
			p, e := c.NewValue()
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, c.Del(&p)) })
			before := b.Used()
			roots := c.Stats()
			c.BudgetField.FailAfter(fail)
			m, e := newSATMainConfig(g, &p, b)
			c.BudgetField.FailAfter(-1)
			now := c.Stats()
			if now.LiveBlocks != roots.LiveBlocks || now.References != roots.References {
				t.Fatal("cold local/snapshot roots", roots, now)
			}
			names := [12]string{"always_true", "always_false", "caching", "random", "geometric", "luby", "dyn_psm", "psm", "glue", "glue_psm", "psm_glue", "sat"}
			published := sy.Stats().Symbols
			if published > uint32(len(names)) {
				t.Fatal("unexpected symbol publications", published)
			}
			// Existing Go string keys are allocation-free at the interning API.
			for i := uint32(0); i < published; i++ {
				s, err := sy.Intern(names[i])
				paramsMust(t, err)
				if m != nil && i < 11 && s != m.cached[i] {
					t.Fatal("cache order", i)
				}
			}
			if e == nil {
				completed = true
				if m == nil || published != 12 {
					t.Fatal("incomplete successful constructor", published)
				}
				paramsMust(t, m.Close())
			} else if !errors.Is(e, arena.ErrInjectedFailure) || m != nil {
				t.Fatal(m, e)
			}
			if b.Used() != before {
				t.Fatal("config cold header was not refunded", b.Used(), before)
			}
		})
	}
	if !completed {
		t.Fatal("constructor sweep never reached actual success")
	}
}

func TestSATMainConfigConstructorPolicyFailureKeepsOnlyBorrowedPrefixes(t *testing.T) {
	c, sy, g, b := satMainConfigTestRuntime(t)
	p, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c.Del(&p)) })
	sat, e := sy.Intern("sat")
	paramsMust(t, e)
	mod, e := g.GetParams(sat)
	paramsMust(t, e)
	key := "restart.initial"
	paramsMust(t, c.SetUIntName(mod, &key, 43))
	satConfigSymbol(t, c, sy, &p, "restart", "bad_restart")
	roots := c.Stats()
	before := b.Used()
	m, e := newSATMainConfig(g, &p, b)
	if m != nil || e == nil || e.Error() != "invalid restart strategy" {
		t.Fatal(m, e)
	}
	now := c.Stats()
	if now.References != roots.References || now.LiveBlocks != roots.LiveBlocks || b.Used() != before {
		t.Fatal("constructor cleanup", roots, now, b.Used(), before)
	}
	for _, name := range []string{"always_true", "always_false", "caching", "random", "geometric", "luby", "dyn_psm", "psm", "glue", "glue_psm", "psm_glue"} {
		c.BudgetField.FailAfter(0)
		_, e = sy.Intern(name)
		c.BudgetField.FailAfter(-1)
		paramsMust(t, e)
	}
}

func TestSATMainConfigInitialMaskAndUnusedDimacsCore(t *testing.T) {
	c, sy, _, b, p, m := satConfigFixture(t)
	paramsMust(t, arena.ValidateRecordType(reflect.TypeOf(satMainConfigState{})))
	paramsMust(t, arena.ValidateRecordType(reflect.TypeOf([11]usymbol.Symbol{})))
	s, e := m.State()
	paramsMust(t, e)
	want := satConfigAllFieldsMask & ^(satConfigGCSmallLBDMask | satConfigGCKMask)
	if s.Initialized != want {
		t.Fatal("source assignment coverage", s.Initialized, want)
	}
	copy, e := m.Clone(b)
	paramsMust(t, e)
	a, e := copy.State()
	paramsMust(t, e)
	if a != s {
		t.Fatal("implicit copy invented dyn-only assignments", a, s)
	}
	paramsMust(t, copy.Close())
	key := "dimacs.core"
	paramsMust(t, c.SetBoolName(p, &key, true))
	paramsMust(t, m.Update(p))
	after, e := m.State()
	paramsMust(t, e)
	if after != s {
		t.Fatal("dimacs.core has no config member", after, s)
	}
	satConfigSymbol(t, c, sy, p, "phase", "random\x00ignored")
	paramsMust(t, m.Update(p))
	s, e = m.State()
	paramsMust(t, e)
	if s.Phase != satPhaseRandom {
		t.Fatal("source C-string symbol", s)
	}
	satConfigSymbol(t, c, sy, p, "phase", "Random")
	e = m.Update(p)
	if e == nil || e.Error() != "invalid phase selection strategy" {
		t.Fatal("case-sensitive source dispatch", e)
	}
}

func TestSATMainConfigCloneFaultAndSameSymbolsAssignment(t *testing.T) {
	c, _, g, b, p, m := satConfigFixture(t)
	before := b.Used()
	b.FailAfter(0)
	n, e := m.Clone(b)
	b.FailAfter(-1)
	if n != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal(n, e, b.Used(), before)
	}
	// A separate registry in the same borrowed symbol domain remains the
	// receiver's registry after implicit source member assignment.
	g2, e := util.NewGlobalParams(g.SymbolsField, c, util.GlobalParamsConfig{Budget: b})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, g2.Close()) })
	n, e = newSATMainConfig(g2, p, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, n.Close()) })
	paramsMust(t, n.Assign(m))
	if n.globals != g2 {
		t.Fatal("assignment copied Go cold bindings")
	}
	roots := c.Stats()
	paramsMust(t, g2.Close())
	if e = n.Update(p); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if c.Stats().References != roots.References {
		t.Fatal("closed-borrow failure touched local roots")
	}
	paramsMust(t, n.Close())
}
