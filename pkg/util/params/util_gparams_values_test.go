package params

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func gparamsRealRegistry(t *testing.T) (*GlobalParams, *ParamsContext, *usymbol.Manager, *arena.Budget) {
	t.Helper()
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	cfg := gparamsTestConfig(b)
	cfg.Register = func(r GlobalParamsRegistrar) error {
		d, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		if e != nil {
			return e
		}
		for _, x := range []struct {
			name string
			kind ParamKind
			def  string
		}{{"seed", ParamUInt, "0"}, {"factor", ParamDouble, "1.5"}, {"text", ParamString, "default text"}, {"phase", ParamSymbol, "caching"}, {"enabled", ParamBool, "true"}, {"unsupported", ParamNumeral, "0"}} {
			if e = d.InsertName(x.name, x.kind, pdLiteral("description"), pdLiteral(x.def), pdLiteral("sat")); e != nil {
				_ = d.Close()
				return e
			}
		}
		if e = r.RegisterModule("sat", d); e != nil {
			_ = d.Close()
			return e
		}
		global, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		if e != nil {
			return e
		}
		defer global.Close()
		if e = global.InsertName("feature_flag", ParamBool, pdLiteral("feature"), pdLiteral("false"), nil); e != nil {
			return e
		}
		return r.RegisterGlobal(global)
	}
	g, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal("registry graph", b.Used())
		}
	})
	return g, ctx, sy, b
}
func TestGlobalParamsActualNormalizeAndScalarSetGetRoutes(t *testing.T) {
	g, ctx, sy, _ := gparamsRealRegistry(t)
	paramsMust(t, g.SetName(":SaT.SeEd", "4294967296"))
	snap, e := g.GetModuleName("sat")
	paramsMust(t, e)
	defer func() { paramsMust(t, ctx.Del(&snap)) }()
	seed, e := sy.Intern("seed")
	paramsMust(t, e)
	n, e := ctx.GetUInt(&snap, seed, 99)
	paramsMust(t, e)
	if n != 0 {
		t.Fatal("long64 then uint32 narrow", n)
	}
	paramsMust(t, g.SetName("SAT.FACTOR", "2.3.4"))
	text, e := g.GetValueName("sat.factor")
	paramsMust(t, e)
	if text != "2.3" {
		t.Fatal(text)
	}
	paramsMust(t, g.SetName("sat.factor", "1/2"))
	text, e = g.GetValueName("sat.factor")
	paramsMust(t, e)
	if text != "1" {
		t.Fatal("strtod prefix", text)
	}
	paramsMust(t, g.SetName(":FEATURE-FLAG\x00ignored", "true\x00false"))
	text, e = g.GetValueName("feature_flag")
	paramsMust(t, e)
	if text != "true" {
		t.Fatal(text)
	}
	name, e := sy.Intern("SAT.phase")
	paramsMust(t, e)
	paramsMust(t, g.SetSymbol(name, "random"))
	text, e = g.GetValueSymbol(name)
	paramsMust(t, e)
	if text != "random" {
		t.Fatal(text)
	}
	// A stored empty UInt is valid source conversion, not a parse failure.
	paramsMust(t, g.SetName("sat.seed", ""))
	text, e = g.GetValueName("sat.seed")
	paramsMust(t, e)
	if text != "0" {
		t.Fatal(text)
	}
}
func TestGlobalParamsValidationErrorDomainsAndPublishedEmptyModule(t *testing.T) {
	g, ctx, sy, _ := gparamsRealRegistry(t)
	for _, x := range []struct{ name, value, want string }{
		{"sat.seed", "-1", "Expected values for parameter seed is an unsigned integer. It was given argument '-1'"},
		{"sat.factor", "1e2", "Expected values for parameter factor is a double. It was given argument '1e2'"},
		{"sat.enabled", "TRUE", "Expected values for parameter enabled are 'true' or 'false'. It was given argument 'TRUE'"},
		{"other.seed", "1", "invalid parameter, unknown module 'other'"},
	} {
		e := g.SetName(x.name, x.value)
		if e == nil || e.Error() != x.want {
			t.Fatal(x, e)
		}
	}
	if g.ModuleParams.size() != 0 {
		t.Fatal("validation precedes get_params publication")
	}
	e := g.SetName("sat.not_known", "1")
	if e == nil || g.ModuleParams.size() != 1 {
		t.Fatal("invalid kind must keep newly published empty module", e)
	}
	mod, e := sy.Intern("sat")
	paramsMust(t, e)
	v, e := g.GetModule(mod)
	paramsMust(t, e)
	empty, e := ctx.Empty(&v)
	paramsMust(t, e)
	paramsMust(t, ctx.Del(&v))
	if !empty {
		t.Fatal("unknown key populated values")
	}
	if e = g.SetName("sat.unsupported", "123"); e == nil || e.Error() != "unsupported parameter type 'unsupported'" {
		t.Fatal(e)
	}
}
func TestGlobalParamsBorrowedStringAndSharedSnapshotPublication(t *testing.T) {
	g, ctx, sy, _ := gparamsRealRegistry(t)
	paramsMust(t, g.SetName("sat.text", "real\x00ignored"))
	mod, e := sy.Intern("sat")
	paramsMust(t, e)
	raw, e := g.GetParams(mod)
	paramsMust(t, e)
	x := ctx.Entry(&ctx.Blocks[raw.Slot-1], 0)
	if x.Kind != ParamString || !x.text.borrowed || x.text.owned != nil {
		t.Fatal("actual borrowed interned pointer", *x)
	}
	snapshot, e := g.GetModule(mod)
	paramsMust(t, e)
	defer func() { paramsMust(t, ctx.Del(&snapshot)) }()
	paramsMust(t, g.SetName("sat.text", "new"))
	key, e := sy.Intern("text")
	paramsMust(t, e)
	old, present, e := ctx.GetString(&snapshot, key, nil)
	paramsMust(t, e)
	if !present || old != "real" {
		t.Fatal("module snapshot not shared COW", old)
	}
	got, e := g.GetValueName("sat.text")
	paramsMust(t, e)
	if got != "new" {
		t.Fatal(got)
	}
	alias, e := g.GetRef()
	paramsMust(t, e)
	paramsMust(t, g.SetName("feature_flag", "true"))
	key, e = sy.Intern("feature_flag")
	paramsMust(t, e)
	flag, e := ctx.GetBool(alias, key, false)
	paramsMust(t, e)
	if !flag {
		t.Fatal("get_ref must remain actual borrowed alias")
	}
	paramsMust(t, g.Reset())
	flag, e = ctx.GetBool(alias, key, true)
	paramsMust(t, e)
	if !flag {
		t.Fatal("reset source alias should observe empty/default")
	}
}
func TestGlobalParamsDefaultLookupAndSourceHistoricalMessages(t *testing.T) {
	g, _, _, _ := gparamsRealRegistry(t)
	got, e := g.GetValueName("sat.factor")
	paramsMust(t, e)
	if got != "1.5" {
		t.Fatal(got)
	}
	for _, x := range []struct{ name, part string }{
		{"pp_decimal", "'z3 -pp:pp.decimal'"}, {"arith_adaptive", "this is an old parameter name"}, {"other.factor", "unknown module 'other'"},
	} {
		_, e = g.GetValueName(x.name)
		if e == nil || !gparamsTestContains(e.Error(), x.part) {
			t.Fatal(x, e)
		}
	}
	paramsMust(t, g.SetName("sat.factor", "-0"))
	got, e = g.GetValueName("sat.factor")
	paramsMust(t, e)
	if got != "-0" {
		t.Fatal("negativezero", got)
	}
	_, e = g.GetValueSymbol(usymbol.Numerical(4))
	if GlobalParamsSourceDebug {
		if e != ErrGlobalParamsOwner {
			t.Fatal("source bare-str assertion", e)
		}
	} else if e == nil || !gparamsTestContains(e.Error(), "unknown parameter ''") {
		t.Fatal("source release bare-str empty", e)
	}
	if _, e = g.GetValueSymbol(usymbol.Symbol{}); e == nil {
		t.Fatal("null bare pointer")
	}
}
func gparamsTestContains(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
