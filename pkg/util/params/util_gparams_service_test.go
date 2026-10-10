// Source: src/util/gparams.cpp:81-105,633-644; src/util/params.cpp:55-77,210-217.
package params

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Independent source branches: no dedicated source gparams initialization test.
func TestGlobalParamsServiceUnconditionalInitAndCurrentPointer(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(1 << 26)
	catalogBudget := arena.NewBudget(1 << 26)
	catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(catalogBudget))
	paramsMust(t, e)
	defer func() {
		paramsMust(t, catalog.Close())
		if catalogBudget.Used() != 0 {
			t.Fatal("catalog roots", catalogBudget.Used())
		}
	}()
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	s, e := NewGlobalParamsService(sy, ctx, cfg)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("service roots", b.Used())
		}
	}()
	if _, e = s.Manager(); !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("null source g_imp", e)
	}
	paramsMust(t, s.InitializeSource())
	first, e := s.Manager()
	paramsMust(t, e)
	if first.modulesRegistered {
		t.Fatal("constructor invoked module registration")
	}
	paramsMust(t, s.InitializeSource())
	second, e := s.Manager()
	paramsMust(t, e)
	if first == second || first.closed || s.Superseded != first {
		t.Fatal("source unconditional overwrite must retain old actual owner")
	}
	// Source finalize only deletes the current pointer, not the lost predecessor.
	paramsMust(t, s.FinalizeSource())
	if s.Current != nil || !second.closed || first.closed {
		t.Fatal("source dealloc/pointer-clear boundary")
	}
	paramsMust(t, s.FinalizeSource())
	paramsMust(t, s.InitializeSource())
	third, e := s.Manager()
	paramsMust(t, e)
	if third == first || third == second {
		t.Fatal("source reinit did not construct fresh imp")
	}
}

func gparamsServiceAudit(t *testing.T, s *GlobalParamsService, b *arena.Budget) {
	t.Helper()
	if s.self != s || s.Config.Budget != b || s.closed {
		t.Fatal("actual service identity")
	}
	n := s.OwnerBytes
	seen := map[*GlobalParams]bool{}
	add := func(g *GlobalParams) {
		if seen[g] || g.self != g || g.Budget != b {
			t.Fatal("duplicate/foreign declared service node")
		}
		seen[g] = true
		n += gparamsGraph(g)
		if d := g.constructorPendingDescriptor; d != nil {
			if d.Self != d || d.Budget != b || d.constructorReady || d == g.GlobalDescriptors {
				t.Fatal("actual uncompleted descriptor")
			}
			n += d.OwnerBytes
		}
	}
	if s.Current != nil {
		add(s.Current)
	}
	for g := s.Superseded; g != nil; g = g.RetainedNext {
		if !g.constructorReady {
			t.Fatal("uncompleted source predecessor")
		}
		add(g)
	}
	for g := s.Failed; g != nil; g = g.RetainedNext {
		if g.constructorReady || g.constructorCause == nil || g.sourceStarted {
			t.Fatal("failed source constructor mistaken for live imp")
		}
		add(g)
	}
	if n != b.Used() {
		t.Fatal("exact declared global/failed/predecessor cold graph", n, b.Used())
	}
	if s.OwnerBytes != uint64(reflect.TypeOf(GlobalParamsService{}).Size()) {
		t.Fatal("charged service header")
	}
}
func TestGlobalParamsServiceCtorRefusalPreservesCurrentAndFailedHeaders(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	success := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			cb := arena.NewBudget(1 << 26)
			catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(cb))
			paramsMust(t, e)
			defer func() {
				paramsMust(t, catalog.Close())
				if cb.Used() != 0 {
					t.Fatal("catalog roots")
				}
			}()
			cfg := gparamsTestConfig(b)
			cfg.Catalog = catalog
			s, e := NewGlobalParamsService(sy, ctx, cfg)
			paramsMust(t, e)
			defer func() {
				b.FailAfter(-1)
				paramsMust(t, s.Close())
				if b.Used() != 0 {
					t.Fatal("actual service roots", b.Used())
				}
			}()
			paramsMust(t, s.InitializeSource())
			old, e := s.Manager()
			paramsMust(t, e)
			b.FailAfter(ordinal)
			e = s.InitializeSource()
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) || s.Current != old || old.closed || s.Superseded != nil {
					t.Fatal("failed ctor changed old source pointer", e)
				}
				gparamsServiceAudit(t, s, b)
				// A real repeated source init can allocate another failed placement header,
				// without overwriting the first actual failed root or the old g_imp.
				b.FailAfter(1)
				before := s.Failed
				e = s.InitializeSource()
				if !errors.Is(e, arena.ErrInjectedFailure) || s.Current != old || s.Failed == nil || s.Failed == before || s.Failed.RetainedNext != before {
					t.Fatal("source unconditional retry/failed-root retention", e)
				}
				gparamsServiceAudit(t, s, b)
				b.FailAfter(-1)
				paramsMust(t, s.InitializeSource())
				fresh, e := s.Manager()
				paramsMust(t, e)
				if fresh == old || old.closed || s.Superseded != old {
					t.Fatal("completed new source publication")
				}
			} else {
				success = true
				if s.Current == old || s.Superseded != old {
					t.Fatal("completed overwrite")
				}
			}
			gparamsServiceAudit(t, s, b)
		})
		if success {
			t.Logf("first actual overwrite constructor success %d", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("actual source constructor did not succeed")
	}
}
func TestGlobalParamsServiceFinalizeRetainsCurrentBeforeActualConsumedNUMStop(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	bg := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m := q.Manager.Z
	catalogBudget := arena.NewBudget(1 << 24)
	catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(catalogBudget))
	paramsMust(t, e)
	service, e := NewGlobalParamsService(sy, c, GlobalParamsConfig{Budget: bg, DescriptorPageSize: 2, DescriptorWordsPageSize: 16, Catalog: catalog})
	paramsMust(t, e)
	paramsMust(t, service.InitializeSource())
	g, e := service.Manager()
	paramsMust(t, e)
	var v *ParamsRef
	t.Cleanup(func() {
		// Explicit existing owning paths retire a failing fixture without replaying
		// the parent's source destructor. No private state/counter mutation.
		if v != nil && v.Slot != 0 && c.Blocks[v.Slot-1].resetActive && c.Blocks[v.Slot-1].Refs == 0 {
			paramsMust(t, c.CleanupDecrement(v))
		}
		paramsDecFinish(t, service.Close)
		paramsMust(t, catalog.Close())
		if catalogBudget.Used() != 0 {
			t.Error("catalog roots", catalogBudget.Used())
		}
		paramsDecFinish(t, c.Close)

		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 || bg.Used() != 0 {
			t.Error("remaining real owners", bn.Used(), bc.Used(), bs.Used(), bg.Used())
		}
	})
	d, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: bg, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bg}}})
	paramsMust(t, e)
	paramsMust(t, g.RegisterModule("child", d))
	module, e := sy.Intern("child")
	paramsMust(t, e)
	key, e := sy.Intern("stored-numeral")
	paramsMust(t, e)
	_ = module
	v = &g.globalParams
	prefix, e := sy.Intern("completed-prefix")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(v, prefix, 42))
	scoped, e := unumeral.NewScopedInt(m)
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	paramsMust(t, m.SetLimbs(num, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	paramsMust(t, c.SetRat(v, key, &big))
	paramsMust(t, q.Del(&big))
	oldSlot, oldGeneration := v.Slot, v.Generation
	extras := numericPhysicalExtraOwners{RationalServices: []*unumeral.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, v}}}
	before := numericAuditPhysical(t, m, extras)
	gparamsServiceAudit(t, service, bg)
	paramsDecTLSNearMinimum(t, r)
	e = service.FinalizeSource()
	if !errors.Is(e, arena.ErrBounds) || g.sourceStage != 0 || g.sourceCause != e || v.Slot != oldSlot || v.Generation != oldGeneration || c.Blocks[oldSlot-1].Refs != 1 {
		t.Fatal("actual parent/child stop", e, g.sourceStage, *v)
	}
	after := numericAuditPhysical(t, m, extras)
	if before.Named != after.Named || before.Objects != after.Objects+1 {
		t.Fatal("retained numeral and consumed raw", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	gparamsServiceAudit(t, service, bg)
	if e = d.open(); e != nil {
		t.Fatal("older descriptor destroyed", e)
	}
	stoppedIndex := g.sourceIndex
	if e = service.FinalizeSource(); e != g.sourceCause || g.sourceStage != 0 || g.sourceIndex != stoppedIndex {
		t.Fatal("source retry changed stopped prefix", e)
	}
	if c.Blocks[oldSlot-1].Size != 2 {
		t.Fatal("source Reset vector extent changed before finalize")
	}
	got, err := c.GetUInt(v, prefix, 0)
	paramsMust(t, err)
	if got != 42 {
		t.Fatal("source inline prefix disappeared while global reset object remains live", got)
	}
	if g.sourceReset.invocation == 0 {
		t.Fatal("actual parent live reset receipt missing")
	}
	wrong := g.sourceReset
	wrong.invocation++
	if err := c.CompleteReset(v, wrong); !errors.Is(err, ErrParamsPrecondition) {
		t.Fatal("foreign reset receipt adopted parent child", err)
	}
	if g.sourceStage != 0 || g.sourceIndex != stoppedIndex || v.Slot != oldSlot || v.Generation != oldGeneration || c.Blocks[oldSlot-1].Refs != 1 {
		t.Fatal("foreign reset receipt changed actual parent/child")
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	gparamsServiceAudit(t, service, bg)
	if service.Current != g {
		t.Fatal("source cleared stopped actual current")
	}
	if e = service.Close(); e != nil {
		t.Fatal("owning continuation did not acknowledge exact child", e)
	}
	if !g.closed || service.Current != nil || !service.closed || g.sourceCompleted {
		t.Fatal("owning suffix", g.closed, bg.Used(), *v, g.sourceCompleted)
	}
	if e = g.SourceFinalize(); e != g.sourceCause {
		t.Fatal("source cause lost after owning cleanup", e)
	}
}

func TestGlobalParamsServiceBorrowedLifetimeAndIdentity(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b, cb := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
	catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(cb))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	s, e := NewGlobalParamsService(sy, ctx, cfg)
	paramsMust(t, e)
	copyOwner := new(GlobalParamsService)
	reflect.ValueOf(copyOwner).Elem().Set(reflect.ValueOf(s).Elem())
	before := b.Used()
	if e := copyOwner.InitializeSource(); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal("copied service", e)
	}
	if e := copyOwner.FinalizeSource(); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal("copied finalize", e)
	}
	if e := copyOwner.Close(); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal("copied owning close", e)
	}
	if b.Used() != before || s.Current != nil {
		t.Fatal("copied owner touched current/cold charge")
	}
	paramsMust(t, catalog.Close())
	// Null source finalization never dereferences unused catalog or children.
	paramsMust(t, s.FinalizeSource())
	if e := s.InitializeSource(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("actual reached dead catalog", e)
	}
	if b.Used() != before || s.Current != nil || s.Failed != nil {
		t.Fatal("dead borrow published failed constructor")
	}
	paramsMust(t, s.Close())
	paramsMust(t, s.Close())
	if b.Used() != 0 || cb.Used() != 0 {
		t.Fatal("actual owner refunds", b.Used(), cb.Used())
	}
	if _, e := s.Manager(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed pointer access", e)
	}
	// A concrete constructor must reject the foreign symbol/catalog cohort
	// before reserving its own cold header. Existing owners remain untouched.
	foreignCtx, foreignSy, _ := paramsTestOwner(t)
	foreignCatalog, e := NewGlobalParamsCatalog(foreignSy, catalogTestConfig(cb))
	paramsMust(t, e)
	bad := cfg
	bad.Catalog = foreignCatalog
	if _, e := NewGlobalParamsService(sy, ctx, bad); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal("foreign catalog", e)
	}
	if _, e := NewGlobalParamsService(sy, foreignCtx, bad); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal("foreign context", e)
	}
	if b.Used() != 0 {
		t.Fatal("foreign borrow cold reservation")
	}
	paramsMust(t, foreignCatalog.Close())
}
