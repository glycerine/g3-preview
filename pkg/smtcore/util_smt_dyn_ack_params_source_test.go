// Source: src/smt/params/dyn_ack_params.h:24-51; src/smt/params/dyn_ack_params.cpp:22-41.
// Source author: Leonardo de Moura (2007-05-18).
// Licensed under the MIT license; see LICENSE.txt.
package smtcore

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func dynAckSourceFixture(t *testing.T) (*util.ParamsContext, *util.GlobalParamsService, *util.ParamsRef, *arena.Budget) {
	t.Helper()
	c, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, e := util.NewGlobalParamsService(sy, c, cfg)
	paramsMust(t, e)
	paramsMust(t, g.InitializeSource())
	local, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, c.Del(&local))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		if b.Used() != 0 {
			t.Fatal("actual service/header roots", b.Used())
		}
	})
	return c, g, &local, b
}

func TestSMTDynAckConstructorLiteralDefaultsAndFallbackOrder(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	d, e := NewSMTDynAckParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, d.Close()) }()
	want := SMTDynAckState{Strategy: 1, Equalities: false, Factor: 0.1, Threshold: 10, GC: 2000, GCInverseDecay: 0.8}
	got, e := d.State()
	paramsMust(t, e)
	if got != want {
		t.Fatal("source defaults", got, want)
	}
	module, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.Current.GetParams(module)
	paramsMust(t, e)
	key, e := g.Symbols.Intern("dack")
	paramsMust(t, e)
	paramsMust(t, c.SetUInt(global, key, 2))
	key, e = g.Symbols.Intern("dack.factor")
	paramsMust(t, e)
	paramsMust(t, c.SetDouble(global, key, 0.25))
	// Wrong local kind falls back to the current module rather than literal default.
	paramsMust(t, c.SetBoolName(p, dynAckName("dack"), true))
	paramsMust(t, c.SetUIntName(p, dynAckName("dack.threshold"), 17))
	paramsMust(t, d.Update(p))
	got, e = d.State()
	paramsMust(t, e)
	if got.Strategy != 2 || got.Factor != 0.25 || got.Threshold != 17 {
		t.Fatal("local/module/default precedence", got)
	}
	// Source enum cast does not validate the numerical range.
	paramsMust(t, c.SetUIntName(p, dynAckName("dack"), 99))
	paramsMust(t, d.Update(p))
	got, e = d.State()
	paramsMust(t, e)
	if got.Strategy != 99 {
		t.Fatal("source cast", got)
	}
}

func dynAckName(s string) *string { return &s }

func dynAckNamedColdGraph(t *testing.T, d *SMTDynAckParams, g *util.GlobalParamsService, b *arena.Budget) {
	t.Helper()
	total := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		total += gparamsGraph(g.Current)
	}
	for v := g.Superseded; v != nil; v = v.RetainedNext {
		total += gparamsGraph(v)
	}
	for v := g.Failed; v != nil; v = v.RetainedNext {
		total += gparamsGraph(v)
	}
	if d != nil {
		if d.self != d || d.Budget != b {
			t.Fatal("actual owner identity")
		}
		total += d.header
		for f := d.Helpers; f != nil; f = f.Next {
			if f.Owner != &d.ModuleParamsOwner {
				t.Fatal("actual helper acquisition owner")
			}
			total += f.Header
		}
	}
	if total != b.Used() {
		t.Fatal("named cold owners", total, b.Used())
	}
}
func TestSMTDynAckEveryActualConstructorReservationAndFailedHeader(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		// A real lookup creates the canonical module symbol before this standalone
		// cold-owner campaign. Separate Symbol/Params budgets remain explicit.
		v, e := g.SourceGetModuleName("smt")
		paramsMust(t, e)
		paramsMust(t, c.Del(&v))
		b.FailAfter(ordinal)
		d, e := NewSMTDynAckParams(c, p, g, b)
		dynAckNamedColdGraph(t, d, g, b) // before lifting the real refusal
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("unexpected real leaf", ordinal, e)
		}
		if d != nil && d.header != uint64(reflect.TypeOf(SMTDynAckParams{}).Size()) {
			t.Fatal("actual reflected header")
		}
		b.FailAfter(-1)
		if d != nil {
			paramsMust(t, d.Close())
			paramsMust(t, d.Close())
		}
		dynAckNamedColdGraph(t, nil, g, b)
		if e == nil {
			first = int(ordinal)
			break
		}
	}
	if first != 2 {
		t.Fatal("actual standalone header then helper", first)
	}
}
func TestSMTDynAckFailedGetterDoesNotPublishFallbackAndModuleReferenceEnds(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, c.SetUIntName(p, dynAckName("dack"), 77))
	d, e := NewSMTDynAckParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, d.Close()) }()
	before, e := d.State()
	paramsMust(t, e)
	refs := c.Stats().References
	// Invalid/ended local reference is a Go guard-domain fixture. The borrowed
	// local is read at the getter, after actual helper module construction.
	invalid, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.Del(&invalid))
	if e = d.Update(&invalid); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("reached local guard", e)
	}
	after, e := d.State()
	paramsMust(t, e)
	if after != before || d.Helpers != nil || c.Stats().References != refs {
		t.Fatal("RHS refusal published defaults or retained module", before, after, c.Stats())
	}
	dynAckNamedColdGraph(t, d, g, b)
	copy := *d
	if _, e = copy.State(); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("copied owner state", e)
	}
	if e = copy.Close(); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("copied owner cleanup", e)
	}
	dynAckNamedColdGraph(t, d, g, b)
}

const dynAckDisplayDefaults = "m_dack=1\nm_dack_eq=0\nm_dack_factor=0.1\nm_dack_threshold=10\nm_dack_gc=2000\nm_dack_gc_inv_decay=0.8\n"

func TestSMTDynAckLiteralDisplayFlagsEveryOutputRefusalAndBadFile(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	d, e := NewSMTDynAckParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, d.Close()) }()
	first := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		outBudget := arena.NewBudget(^uint64(0))
		out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: outBudget})
		paramsMust(t, e)
		outBudget.FailAfter(ordinal)
		e = d.Display(out)
		got, read := out.String()
		paramsMust(t, read)
		if !strings.HasPrefix(dynAckDisplayDefaults, got) {
			t.Fatal("literal output prefix", ordinal, got, e)
		}
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("real output refusal", e)
		}
		if e == nil && got != dynAckDisplayDefaults {
			t.Fatal("complete source output", got)
		}
		outBudget.FailAfter(-1)
		paramsMust(t, out.Close())
		if outBudget.Used() != 0 {
			t.Fatal("output owner refund")
		}
		if e == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 1 {
		t.Fatal("actual output allocation horizon", first)
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, out.SetFloatMode(util.NumericFixedFloat))
	paramsMust(t, out.SetPrecision(2))
	paramsMust(t, d.Display(out))
	text, e := out.String()
	paramsMust(t, e)
	want := "m_dack=1\nm_dack_eq=0\nm_dack_factor=0.10\nm_dack_threshold=10\nm_dack_gc=2000\nm_dack_gc_inv_decay=0.80\n"
	if text != want || out.State().Precision != 2 || out.State().FloatMode != util.NumericFixedFloat {
		t.Fatal("borrowed stream flags", text, out.State())
	}
	paramsMust(t, out.Close())
	file, e := os.CreateTemp(".", "smt-dyn-ack-file-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	actual, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	// The real nonthrowing file write fails on the first label; later source
	// insertions/endl still execute without converting badbit into an exception.
	paramsMust(t, d.Display(actual))
	if !actual.FileBad || actual.FilePosition != 0 {
		t.Fatal("actual failed file stream", actual.FileBad, actual.FilePosition)
	}
	paramsMust(t, actual.Close())
	dynAckNamedColdGraph(t, d, g, b)
}

func TestSMTDynAckEveryActualUnwarmedSymbolAndHelperConstructorPrefix(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		_, _, q := paramsTestOwner(t) // unrelated concrete numeral/context anchors survive
		b := arena.NewBudget(^uint64(0))
		sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b, PageSize: 64, DescriptorPageSize: 2})
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
		d, cause := NewSMTDynAckParams(c, &local, g, b)
		// The explicit current service, catalog, symbol storage and actual helper
		// fields form the entire Budget graph; no allocation-count difference grants
		// any acquired ownership. The numeral and local context providers are borrowed.
		named := g.OwnerBytes + gparamsGraph(g.Current) + catalogGraph(catalog) + astTestSymbolGraph(sy) + sy.Stats().Table.RetainedBytes
		if d != nil {
			named += d.header
			for f := d.Helpers; f != nil; f = f.Next {
				if f.Owner != &d.ModuleParamsOwner {
					t.Fatal("helper owner")
				}
				named += f.Header
			}
		}
		if named != b.Used() {
			t.Fatal("before-lift complete shared cold graph", ordinal, named, b.Used())
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual constructor leaf", ordinal, cause)
		}
		b.FailAfter(-1)
		if d != nil {
			paramsMust(t, d.Close())
		}
		paramsMust(t, c.Del(&local))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if b.Used() != 0 || bc.Used() != 0 {
			t.Fatal("named child owners refund once", ordinal, b.Used(), bc.Used())
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 3 {
		t.Fatal("unwarmed source symbol acquisition was not reached", first)
	}
	t.Log("first successful shared symbol/helper constructor", first)
}

func TestSMTDynAckReachedNamespaceAssertionConstructorPrefix(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, g.FinalizeSource())
	d, cause := NewSMTDynAckParams(c, p, g, b)
	want := util.ErrGlobalParamsUninitialized
	if util.GlobalParamsSourceDebug {
		want = util.ErrGlobalParamsAssertion
	}
	if !errors.Is(cause, want) || d == nil || d.constructorReady || d.constructorCause != cause || d.Helpers != nil {
		t.Fatal("source symbol then namespace assertion ctor stop", cause, d)
	}
	// Actual constructor scalar initialization precedes helper lookup, while the
	// failed object is inspection/own-abort only; no source object was completed.
	state, e := d.State()
	paramsMust(t, e)
	if state != (SMTDynAckState{1, false, 0.1, 10, 2000, 0.8}) {
		t.Fatal("literal initialized prefix", state)
	}
	dynAckNamedColdGraph(t, d, g, b)
	paramsMust(t, g.InitializeSource())
	if e = d.Update(p); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("uncompleted constructor reused as source object", e)
	}
	paramsMust(t, d.Close())
	fresh, e := NewSMTDynAckParams(c, p, g, b)
	paramsMust(t, e)
	paramsMust(t, fresh.Close())
	dynAckNamedColdGraph(t, nil, g, b)
}

func TestSMTDynAckDisplayNumericBoolShowPosSignedLongProfile(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	d, e := NewSMTDynAckParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, d.Close()) }()
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, out.SetShowPos(true))
	for _, equalities := range []bool{false, true} {
		paramsMust(t, c.SetBoolName(p, dynAckName("dack.eq"), equalities))
		paramsMust(t, d.Update(p))
		paramsMust(t, out.ResetText())
		paramsMust(t, d.Display(out))
		eq := "+0"
		if equalities {
			eq = "+1"
		}
		want := "m_dack=+1\nm_dack_eq=" + eq + "\nm_dack_factor=+0.1\nm_dack_threshold=10\nm_dack_gc=2000\nm_dack_gc_inv_decay=+0.8\n"
		got, e := out.String()
		paramsMust(t, e)
		if got != want {
			t.Fatal("numeric bool ostream promotes signed long; unsigned counters omit sign", got, want)
		}
	}
}
