package params

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func gparamsVectorGraph(s arena.VectorStats) uint64 {
	d := s.Descriptors
	return s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func gparamsGraph(g *GlobalParams) uint64 {
	n := g.OwnerBytes
	count := func(d *ParamDescrs) uint64 {
		if d == nil || d.Budget != g.Budget {
			return 0
		}
		s := d.Stats()
		return s.OwnerBytes + s.TableBytes + s.NamesBytes
	}
	n += count(g.GlobalDescriptors)
	for _, d := range g.Descriptors {
		n += count(d)
	}
	return n
}
func TestGlobalParamsSetterUsesGoNormalizationStorage(t *testing.T) {
	successes := 0
	for prefix := int64(0); prefix < 45; prefix++ {
		g, ctx, sy, b := gparamsRealRegistry(t)
		paramsMust(t, g.ensureRegistered())
		// Intern source names beforehand; the registry budget remains independent
		// of the genuinely borrowed Symbol/Params owners.
		key, e := sy.Intern("text")
		paramsMust(t, e)
		b.FailAfter(prefix)
		e = g.SetName("sat.text", "a genuinely borrowed string")
		b.FailAfter(-1)
		if e != nil {
			t.Fatal(prefix, e)
		}
		successes++
		if b.Used() != gparamsGraph(g) {
			t.Fatalf("prefix%d owned%d actual%d", prefix, gparamsGraph(g), b.Used())
		}
		snap, err := g.GetModuleName("sat")
		paramsMust(t, err)
		if e == nil {
			value, _, err := ctx.GetString(&snap, key, nil)
			paramsMust(t, err)
			if value != "a genuinely borrowed string" {
				t.Fatal(prefix, value)
			}
		}
		paramsMust(t, ctx.Del(&snap))
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal(prefix, b.Used())
		}
	}
	if successes != 45 {
		t.Fatal(successes)
	}
}
func TestGlobalParamsActualSATCollectorCurrentSharedRegistry(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	cfg := gparamsTestConfig(b)
	cfg.Register = func(r GlobalParamsRegistrar) error {
		d, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b})
		if e != nil {
			return e
		}
		if e = catalogCollector("CollectSATConfigParamDescrs")(d); e != nil {
			d.Close()
			return e
		}
		if e = r.RegisterModule("sat", d); e != nil {
			d.Close()
		}
		return e
	}
	g, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	text, e := g.GetValueName("sat.max_memory")
	paramsMust(t, e)
	if text != "4294967295" {
		t.Fatal(text)
	}
	paramsMust(t, g.SetName("sat.restart", "geometric"))
	first, e := g.GetModuleName("sat")
	paramsMust(t, e)
	defer func() { paramsMust(t, ctx.Del(&first)) }()
	paramsMust(t, g.SetName("sat.restart", "luby"))
	second, e := g.GetModuleName("sat")
	paramsMust(t, e)
	defer func() { paramsMust(t, ctx.Del(&second)) }()
	key, e := sy.Intern("restart")
	paramsMust(t, e)
	a, e := ctx.GetSymbol(&first, key, usymbol.Symbol{})
	paramsMust(t, e)
	z, e := ctx.GetSymbol(&second, key, usymbol.Symbol{})
	paramsMust(t, e)
	if a == z {
		t.Fatal("current per-call registry snapshot must observe replacement")
	}
	ad, e := sy.String(a)
	paramsMust(t, e)
	zd, e := sy.String(z)
	paramsMust(t, e)
	if ad != "geometric" || zd != "luby" {
		t.Fatal(ad, zd)
	}
	if len(g.Descriptors) != 1 || g.Descriptors[0].Stats().Live != 24 {
		t.Fatal("actual collector", g.Descriptors)
	}
}

func TestGlobalParamsNativeDescriptorMergeConsumesCallerOwner(t *testing.T) {
	g, _, sy, budget := gparamsRealRegistry(t)
	paramsMust(t, g.ensureRegistered())
	incoming, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: budget})
	paramsMust(t, e)
	paramsMust(t, incoming.InsertName("new_parameter", ParamUInt, pdLiteral("native copied metadata"), pdLiteral("17"), nil))
	budget.FailAfter(0)
	paramsMust(t, g.RegisterModule("sat", incoming))
	budget.FailAfter(-1)
	if !incoming.closed || budget.Used() != gparamsGraph(g) {
		t.Fatal("successful merge consumes incoming owner", incoming.closed, budget.Used())
	}
	key, e := sy.Intern("new_parameter")
	paramsMust(t, e)
	found := false
	for _, d := range g.Descriptors {
		text, present, e := d.Description(key)
		paramsMust(t, e)
		if present {
			found = true
			if text != "native copied metadata" {
				t.Fatal(text)
			}
		}
	}
	if !found {
		t.Fatal("merged metadata lost after incoming close")
	}
	paramsMust(t, g.Close())
	if budget.Used() != 0 {
		t.Fatal("owner refund", budget.Used())
	}
}
func TestGlobalParamsDescriptionBindingAndMapPublication(t *testing.T) {
	g, _, sy, b := gparamsRealRegistry(t)
	paramsMust(t, g.ensureRegistered())
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		paramsMust(t, g.RegisterModuleDescription(name, nil, arena.BytesRegionRef{}, 0))
	}
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	paramsMust(t, e)
	defer r.Close()
	ref := gparamsDescriptionBytes(t, r, "actual borrowed text")
	// The borrowed binding and map publication are ordinary Go storage.
	b.FailAfter(0)
	e = g.RegisterModuleDescription("sat", r, ref, 0)
	b.FailAfter(-1)
	if e != nil || len(g.Bindings) != 1 || g.ModuleDescriptions.size() != 8 {
		t.Fatal(e, g.Bindings, g.ModuleDescriptions.size())
	}
	key, e := sy.Intern("sat")
	paramsMust(t, e)
	if _, ok := g.ModuleDescriptions.get(key); !ok || b.Used() != gparamsGraph(g) {
		t.Fatal("publication/ownership", ok, b.Used(), gparamsGraph(g))
	}
}
