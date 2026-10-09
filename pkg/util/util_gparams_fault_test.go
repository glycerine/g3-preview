package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
		return s.OwnerBytes + s.TableBytes + s.NamesBytes + gparamsVectorGraph(s.Text)
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
	a, e := ctx.GetSymbol(&first, key, Symbol{})
	paramsMust(t, e)
	z, e := ctx.GetSymbol(&second, key, Symbol{})
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

func TestGlobalParamsDescriptorMergeFaultRetainsCallerOwner(t *testing.T) {
	failed, passed := 0, 0
	for prefix := int64(0); prefix < 45; prefix++ {
		g, _, sy, b := gparamsRealRegistry(t)
		paramsMust(t, g.ensureRegistered())
		incoming, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		paramsMust(t, e)
		paramsMust(t, incoming.InsertName("new_parameter", ParamUInt, pdLiteral("a considerably longer description that requires real copied payload pages"), pdLiteral("17"), nil))
		b.FailAfter(prefix)
		e = g.RegisterModule("sat", incoming)
		b.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(prefix, e)
			}
			failed++
			if incoming.closed {
				t.Fatal("source deletes incoming only after complete Copy")
			}
		} else {
			passed++
			if !incoming.closed {
				t.Fatal("successful merge did not destroy incoming")
			}
		}
		s := incoming.Stats()
		owned := gparamsGraph(g) + s.OwnerBytes + s.TableBytes + s.NamesBytes + gparamsVectorGraph(s.Text)
		if b.Used() != owned {
			t.Fatal(prefix, b.Used(), owned)
		}
		paramsMust(t, incoming.Close())
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal(prefix, b.Used())
		}
	}
	if failed == 0 || passed == 0 {
		t.Fatal(failed, passed)
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
