// Source: src/smt/params/theory_bv_params.cpp:23-29;
// scripts/mk_genfile_common.py:895-939.
package smtcore

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTBVSimultaneousHelperOlderReadAndReverseDestruction(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	a, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	defer func() { a.Running = false; paramsMust(t, a.Close()) }()
	paramsMust(t, a.Bind(c, g))
	a.Running = true
	older, e := a.BeginHelper(p, "smt")
	paramsMust(t, e)
	younger, e := a.BeginHelper(p, "rewriter")
	paramsMust(t, e)
	if a.Helpers != younger || younger.Next != older {
		t.Fatal("actual declaration order")
	}
	if _, e = a.FrameBool(older, "bv.reflect", true); e != nil {
		t.Fatal("older p remains readable while younger rp is alive", e)
	}
	paramsMust(t, a.FinishHelper(younger, nil))
	if a.Helpers != older || older.DestructorStarted {
		t.Fatal("reverse lexical destructor order")
	}
	paramsMust(t, a.FinishHelper(older, nil))
	a.Running = false
	paramsMust(t, a.Close())
	arrayNamedColdGraph(t, nil, g, b)
}

func TestSMTBVSevenDefaultsThenActualUpdate(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTBVParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	got, e := v.State()
	paramsMust(t, e)
	want := SMTBVState{Mode: 1, HiDiv0: true, Reflect: true, BlastMaxSize: 2147483647, EnableInt2BV: true}
	if got != want {
		t.Fatal("literal initializers plus reached default helper update", got, want)
	}
}

func bvNamedColdGraph(t *testing.T, v *SMTBVParams, g *util.GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		n += gparamsGraph(g.Current)
	}
	for p := g.Superseded; p != nil; p = p.RetainedNext {
		n += gparamsGraph(p)
	}
	for p := g.Failed; p != nil; p = p.RetainedNext {
		n += gparamsGraph(p)
	}
	if v != nil {
		if v.self != v || v.Budget != b || v.ModuleParamsOwner.Self != &v.ModuleParamsOwner {
			t.Fatal("exact wrapper identity")
		}
		n += v.header
		for f := v.Helpers; f != nil; f = f.Next {
			if f.Self != f || f.Owner != &v.ModuleParamsOwner || f.Invocation == 0 || f.Invocation > v.Serial {
				t.Fatal("actual named frame")
			}
			n += f.Header
		}
	}
	if n != b.Used() {
		t.Fatal("all named cold bytes before fault lift", n, b.Used())
	}
}
func TestSMTBVIndependentBooleanPrecedenceAndUntouchedMembers(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	smt, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	sg, e := g.Current.GetParams(smt)
	paramsMust(t, e)
	rw, e := g.Symbols.Intern("rewriter")
	paramsMust(t, e)
	rg, e := g.Current.GetParams(rw)
	paramsMust(t, e)
	v, e := NewSMTBVParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	// Each of the three getter paths independently uses local bool, module bool,
	// then true; a non-bool local falls through. Exhaust all 8 module triples
	// and 64 local absent/false/true/wrong-kind triples.
	keys := []string{"hi_div0", "bv.reflect", "bv.enable_int2bv"}
	for modules := 0; modules < 8; modules++ {
		for i, k := range keys {
			target := sg
			if i == 0 {
				target = rg
			}
			paramsMust(t, c.SetBoolName(target, &k, modules&(1<<i) != 0))
		}
		for local := 0; local < 64; local++ {
			paramsMust(t, c.Reset(p))
			want := [3]bool{}
			for i, k := range keys {
				mode := (local >> (2 * i)) & 3
				want[i] = modules&(1<<i) != 0
				if mode == 1 || mode == 2 {
					want[i] = mode == 2
					paramsMust(t, c.SetBoolName(p, &k, want[i]))
				} else if mode == 3 {
					paramsMust(t, c.SetUIntName(p, &k, 9))
				}
			}
			v.state.Mode = 0
			v.state.LazyLE = true
			v.state.CC = true
			v.state.BlastMaxSize = 7
			paramsMust(t, v.Update(p))
			state, e := v.State()
			paramsMust(t, e)
			if state.HiDiv0 != want[0] || state.Reflect != want[1] || state.EnableInt2BV != want[2] || state.Mode != 0 || !state.LazyLE || !state.CC || state.BlastMaxSize != 7 {
				t.Fatal("literal three getter-only update", modules, local, state, want)
			}
		}
	}
	bvNamedColdGraph(t, v, g, b)
}
func TestSMTBVEveryActualWarmConstructorReservationAndDeclarationPrefix(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		for _, k := range []string{"smt", "rewriter", "hi_div0", "bv.reflect", "bv.enable_int2bv"} {
			_, e := g.Symbols.Intern(k)
			paramsMust(t, e)
		}
		b.FailAfter(ordinal)
		v, cause := NewSMTBVParams(c, p, g, b)
		bvNamedColdGraph(t, v, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual constructor reservation", ordinal, cause)
		}
		if v != nil {
			if v.header != uint64(reflect.TypeOf(SMTBVParams{}).Size()) || v.Helpers != nil {
				t.Fatal("actual wrapper header and completed source unwind", ordinal)
			}
			got, e := v.State()
			paramsMust(t, e)
			if cause != nil {
				if got.HiDiv0 || v.constructorReady || v.constructorCause != cause {
					t.Fatal("failed constructor initialized prefix", ordinal, got)
				}
				if e = v.Update(p); !errors.Is(e, util.ErrParamsPrecondition) {
					t.Fatal("incomplete constructor guard", e)
				}
			} else if !got.HiDiv0 || !v.constructorReady {
				t.Fatal("source rewriter true default", got)
			}
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		bvNamedColdGraph(t, nil, g, b)
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 3 {
		t.Fatal("wrapper then smt frame then rewriter frame", first)
	}
}
func TestSMTBVUnwarmedSharedSymbolAndHelperAcquisitionPrefixes(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t)
		b := arena.NewBudget(^uint64(0))
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
		paramsMust(t, e)
		bc := arena.NewBudget(^uint64(0))
		c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
		paramsMust(t, e)
		catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		cfg := gparamsTestConfig(b)
		cfg.Catalog = catalog
		g, e := util.NewGlobalParamsService(sy, c, cfg)
		paramsMust(t, e)
		paramsMust(t, g.InitializeSource())
		local, e := c.NewValue()
		paramsMust(t, e)
		b.FailAfter(ordinal)
		v, cause := NewSMTBVParams(c, &local, g, b)
		n := g.OwnerBytes + catalogGraph(catalog) + gparamsGraph(g.Current) + astTestSymbolGraph(sy) + sy.Stats().Table.RetainedBytes
		if v != nil {
			n += v.header
			for f := v.Helpers; f != nil; f = f.Next {
				n += f.Header
			}
		}
		if n != b.Used() {
			t.Fatal("unwarmed complete named cold graph", ordinal, n, b.Used())
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("real symbol/frame allocation", ordinal, cause)
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		paramsMust(t, c.Del(&local))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bc.Used() != 0 {
			t.Fatal("once-only source and own refunds", ordinal, b.Used(), bc.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 3 {
		t.Fatal("wrapper and two helper frames; native symbols add no reservations", first)
	}
	t.Log("first successful unwarmed constructor", first)
}

const bvDefaultDisplay = "m_bv_mode=1\nm_hi_div0=1\nm_bv_reflect=1\nm_bv_lazy_le=0\nm_bv_cc=0\nm_bv_blast_max_size=2147483647\nm_bv_enable_int2bv2int=1\n"

func TestSMTBVSevenDisplayChainsShowPosGoBufferAndBadbit(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTBVParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	first := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		sb := arena.NewBudget(^uint64(0))
		out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		cause := v.Display(out)
		got, e := out.String()
		paramsMust(t, e)
		if !strings.HasPrefix(bvDefaultDisplay, got) {
			t.Fatal("independent exact insertion prefix", ordinal, got, cause)
		}
		if cause == nil && got != bvDefaultDisplay {
			t.Fatal("seven actual fields", got)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("stream owners", sb.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 0 {
		t.Fatal("Go buffer output did not complete", first)
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, v.Display(out))
	got, e := out.String()
	paramsMust(t, e)
	want := "m_bv_mode=+1\nm_hi_div0=+1\nm_bv_reflect=+1\nm_bv_lazy_le=+0\nm_bv_cc=+0\nm_bv_blast_max_size=2147483647\nm_bv_enable_int2bv2int=+1\n"
	if got != want {
		t.Fatal("signed bool/enum vs unsigned", got, want)
	}
	paramsMust(t, out.Close())
	file, e := os.CreateTemp(".", "smt-bv-display-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e = util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, v.Display(out))
	if !out.FileBad || out.FilePosition != 0 {
		t.Fatal("real failed file ostream", out.FileBad, out.FilePosition)
	}
	paramsMust(t, out.Close())
	bvNamedColdGraph(t, v, g, b)
}

func TestSMTBVYoungerRealNumeralDestructorStopProtectsOlderHelper(t *testing.T) {
	bn := arena.NewBudget(1 << 26)
	bc := arena.NewBudget(1 << 24)
	bs := arena.NewBudget(1 << 24)
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unreached polynomial checkpoint") }, Cooperation: func() error { panic("unreached polynomial cooperation") }})
	paramsMust(t, e)
	catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, e := util.NewGlobalParamsService(sy, c, cfg)
	paramsMust(t, e)
	paramsMust(t, g.InitializeSource())
	local, e := c.NewValue()
	paramsMust(t, e)
	// Each concrete lexical result has an actual module block. Dropping the
	// original module reference leaves the younger helper as the final owner.
	smt, e := sy.Intern("smt")
	paramsMust(t, e)
	sg, e := g.Current.GetParams(smt)
	paramsMust(t, e)
	paramsMust(t, c.SetBoolName(sg, dynAckName("bv.reflect"), true))
	rw, e := sy.Intern("rewriter")
	paramsMust(t, e)
	rg, e := g.Current.GetParams(rw)
	paramsMust(t, e)
	scoped, e := util.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, digits))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	k, e := sy.Intern("large-real-leaf")
	paramsMust(t, e)
	paramsMust(t, c.SetRat(rg, k, &big))
	paramsMust(t, q.Del(&big))
	v, e := NewSMTBVParams(c, &local, g, b)
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		b.FailAfter(-1)
		v.Running = false
		paramsDecFinish(t, v.Close)
		paramsDecFinish(t, g.Close)
		paramsDecFinish(t, catalog.Close)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 {
			t.Fatal("all named owner refunds", b.Used(), bn.Used(), bc.Used(), bs.Used())
		}
	})
	v.Running = true
	older, e := v.BeginHelper(&local, "smt")
	paramsMust(t, e)
	younger, e := v.BeginHelper(&local, "rewriter")
	paramsMust(t, e)
	empty, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Assign(rg, &empty))
	paramsMust(t, c.Del(&empty))
	oldSlot, oldGen := older.Value.Slot, older.Value.Generation
	oldRefs := c.Blocks[oldSlot-1].Refs
	extras := upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &older.Value}, {c, &younger.Value}}}
	before := upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, extras)
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	b.FailAfter(0)
	cause := v.FinishHelper(younger, nil)
	if !errors.Is(cause, arena.ErrBounds) || !younger.DestructorStarted || younger.SourceDone || younger.Receipt.InvocationField == 0 || v.DestructorCause != younger.DestructorCause {
		t.Fatal("actual younger source stop", cause, younger.Receipt)
	}
	outer := v.FinishHelper(older, nil)
	if !errors.Is(outer, arena.ErrBounds) || older.DestructorStarted || older.SourceDone || older.Value.Slot != oldSlot || older.Value.Generation != oldGen || c.Blocks[oldSlot-1].Refs != oldRefs || v.Helpers != younger || younger.Next != older {
		t.Fatal("older source destructor reached after younger stopped", outer)
	}
	after := upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, extras)
	if after.Named != before.Named || after.Objects+1 != before.Objects {
		t.Fatal("exact consumed raw and retained leaf descriptor", before, after)
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	bvNamedColdGraph(t, v, g, b)
	v.Running = false
	// Callback/foreign owning requests cannot replace or adopt the exact frame.
	copyOwner := *v
	if e := copyOwner.Close(); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("foreign copied outer owner", e)
	}
	paramsDecFinish(t, v.Close)
	if !errors.Is(younger.DestructorCause, arena.ErrBounds) || younger.Header != 0 || older.Header != 0 || !younger.SourceDone || !older.SourceDone || !errors.Is(v.DestructorCause, arena.ErrBounds) {
		t.Fatal("immutable source cause and matched own completion")
	}
	bvNamedColdGraph(t, v, g, b)
}

func TestSMTBVIncompleteSecondHelperConstructorDisplayRefusesBeforeInsertion(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	for _, k := range []string{"smt", "rewriter"} {
		_, e := g.Symbols.Intern(k)
		paramsMust(t, e)
	}
	sb := arena.NewBudget(^uint64(0))
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	defer func() { sb.FailAfter(-1); paramsMust(t, out.Close()) }()
	b.FailAfter(2)
	v, cause := NewSMTBVParams(c, p, g, b)
	if !errors.Is(cause, arena.ErrInjectedFailure) || v == nil || v.Serial != 1 || v.constructorReady || v.Helpers != nil {
		t.Fatal("actual second helper declaration refusal/unwind", v, cause)
	}
	defer func() { b.FailAfter(-1); paramsMust(t, v.Close()) }()
	bvNamedColdGraph(t, v, g, b)
	before := sb.Used()
	sb.FailAfter(0)
	if e := v.Display(out); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("incomplete source object entered display", e)
	}
	got, e := out.String()
	paramsMust(t, e)
	if got != "" || sb.Used() != before || v.constructorCause != cause || v.state.HiDiv0 {
		t.Fatal("incomplete display reached insertion/publication", got, sb.Used(), before)
	}
	bvNamedColdGraph(t, v, g, b)
}
func TestSMTBVCompletedHelperHistoryAndNewInvocationDoNotReplay(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewSMTBVParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	serial := v.Serial
	v.Running = true
	older, e := v.BeginHelper(p, "smt")
	paramsMust(t, e)
	younger, e := v.BeginHelper(p, "rewriter")
	paramsMust(t, e)
	paramsMust(t, v.FinishHelper(younger, nil))
	paramsMust(t, v.FinishHelper(older, nil))
	v.Running = false
	if !older.SourceDone || !younger.SourceDone || !older.DestructorStarted || !younger.DestructorStarted || older.Header != 0 || younger.Header != 0 || v.Helpers != nil || v.Serial != serial+2 {
		t.Fatal("literal completed source frame history")
	}
	paramsMust(t, v.Update(p))
	if v.Serial != serial+4 || v.Helpers != nil || !older.SourceDone || !younger.SourceDone || older.Invocation == younger.Invocation {
		t.Fatal("fresh invocation replayed/overwrote prior frames")
	}
	bvNamedColdGraph(t, v, g, b)
}

func TestSMTBVNullGlobalsConstructorPrefixAndForeignLocalGetter(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, g.FinalizeSource())
	v, cause := NewSMTBVParams(c, p, g, b)
	if v == nil {
		t.Fatal("actual initialized scalar wrapper before source helper", cause)
	}
	if util.GlobalParamsSourceDebug {
		if !errors.Is(cause, util.ErrGlobalParamsAssertion) {
			t.Fatal("actual source namespace assertion", cause)
		}
	} else if !errors.Is(cause, util.ErrGlobalParamsUninitialized) {
		t.Fatal("null-global Go guard", cause)
	}
	want := SMTBVState{Mode: 1, Reflect: true, BlastMaxSize: 2147483647, EnableInt2BV: true}
	got, e := v.State()
	paramsMust(t, e)
	if got != want || v.Helpers != nil || v.constructorReady || v.Serial != 1 {
		t.Fatal("seven source initializers, incomplete first helper", got, v.Serial)
	}
	bvNamedColdGraph(t, v, g, b)
	paramsMust(t, v.Close())
	paramsMust(t, g.InitializeSource())
	v, e = NewSMTBVParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	foreignC, _, _ := paramsTestOwner(t)
	foreign, e := foreignC.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, foreignC.Del(&foreign)) }()
	before, _ := v.State()
	serial := v.Serial
	if e = v.Update(&foreign); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("reached local ParamsRef owner guard", e)
	}
	after, _ := v.State()
	if before != after || v.Helpers != nil || v.Serial != serial+2 {
		t.Fatal("getter error publication or helper unwind", after, v.Serial)
	}
	paramsMust(t, v.Update(p))
	bvNamedColdGraph(t, v, g, b)
}
