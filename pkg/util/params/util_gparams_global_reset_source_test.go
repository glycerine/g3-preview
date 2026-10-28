package params

// Source: src/util/gparams.cpp:106-123; src/util/params.cpp:325-339,510-513.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Actual imp global ParamsRef + generic ParamsRef::set supplies the stored
// numeral. This is an internal imp lifecycle route, not credit for public
// gparams string parsing. Public realloc reaches the signed-host undefined
// domain guard; no allocation-exhaustion or ordinary free-failure claim.
func TestGlobalParamsSourceGlobalResetRetainsInlinePrefixBeforeChildStop(t *testing.T) {
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
	g, e := NewGlobalParams(sy, c, GlobalParamsConfig{Budget: bg, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	paramsMust(t, e)
	var v *ParamsRef
	t.Cleanup(func() {
		// Explicit existing owning paths retire a failing fixture without replaying
		// the parent's source destructor. No private state/counter mutation.
		if v != nil && v.Slot != 0 && c.Blocks[v.Slot-1].resetActive && c.Blocks[v.Slot-1].Refs == 0 {
			paramsMust(t, c.CleanupDecrement(v))
		}
		if !g.closed {
			paramsDecFinish(t, g.closeLegacy)
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
	if bg.Used() != gparamsGraph(g) {
		t.Fatal("parent graph before stop", bg.Used(), gparamsGraph(g))
	}
	paramsDecTLSNearMinimum(t, r)
	e = g.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || g.sourceStage != 0 || g.sourceCause != e || v.Slot != oldSlot || v.Generation != oldGeneration || c.Blocks[oldSlot-1].Refs != 1 {
		t.Fatal("actual parent/child stop", e, g.sourceStage, *v)
	}
	after := numericAuditPhysical(t, m, extras)
	if before.Named != after.Named || before.Objects != after.Objects+1 {
		t.Fatal("retained numeral and consumed raw", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	if bg.Used() != gparamsGraph(g) {
		t.Fatal("parent graph after stop", bg.Used(), gparamsGraph(g))
	}
	if e = d.open(); e != nil {
		t.Fatal("older descriptor destroyed", e)
	}
	stoppedIndex := g.sourceIndex
	if e = g.SourceFinalize(); e != g.sourceCause || g.sourceStage != 0 || g.sourceIndex != stoppedIndex {
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
	if bg.Used() != gparamsGraph(g) {
		t.Fatal("parent coldgraph after wrong receipt")
	}
	if e = g.CleanupStopped(); e != nil {
		t.Fatal("owning continuation did not acknowledge exact child", e)
	}
	if !g.closed || bg.Used() != 0 || g.sourceCompleted {
		t.Fatal("owning suffix", g.closed, bg.Used(), *v, g.sourceCompleted)
	}
	if e = g.SourceFinalize(); e != g.sourceCause {
		t.Fatal("source cause lost after owning cleanup", e)
	}
}
