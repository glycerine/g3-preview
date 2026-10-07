// Licensed under the MIT license.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func gparamsTestConfig(b *arena.Budget) GlobalParamsConfig {
	return GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}
}
func TestGlobalParamsRealModuleSnapshotsAndReset(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	calls := 0
	cfg := gparamsTestConfig(b)
	cfg.Register = func(r GlobalParamsRegistrar) error { calls++; return nil }
	g, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal("registry roots", b.Used())
		}
	}()
	empty, e := g.GetModuleName("sat")
	paramsMust(t, e)
	yes, e := ctx.Empty(&empty)
	paramsMust(t, e)
	if !yes || calls != 0 {
		t.Fatal("GetModule must not trigger registration", calls)
	}
	paramsMust(t, ctx.Del(&empty))
	// Source core get_params creates a real module owner before a setter publishes.
	mod, e := sy.Intern("sat")
	paramsMust(t, e)
	key, e := sy.Intern("seed")
	paramsMust(t, e)
	ref, e := g.GetParams(mod)
	paramsMust(t, e)
	paramsMust(t, ctx.SetUInt(ref, key, 17))
	snap, e := g.GetModule(mod)
	paramsMust(t, e)
	if ctx.Stats().References != 2 {
		t.Fatal("shared module snapshot", ctx.Stats())
	}
	paramsMust(t, g.Reset())
	n, e := ctx.GetUInt(&snap, key, 0)
	paramsMust(t, e)
	if n != 17 || ctx.Stats().References != 1 {
		t.Fatal("reset destroyed retained snapshot", n, ctx.Stats())
	}
	fresh, e := g.GetModuleName("sat")
	paramsMust(t, e)
	yes, e = ctx.Empty(&fresh)
	paramsMust(t, e)
	if !yes {
		t.Fatal("Reset must clear real module registry")
	}
	paramsMust(t, ctx.Del(&fresh))
	paramsMust(t, ctx.Del(&snap))
}
func TestGlobalParamsRegistrationTrueBeforeFailureAndActualMerge(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	calls := 0
	cfg := gparamsTestConfig(b)
	cfg.Register = func(r GlobalParamsRegistrar) error {
		calls++
		d, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		if e != nil {
			return e
		}
		if e = d.InsertName("seed", ParamUInt, pdLiteral("real seed"), pdLiteral("0"), pdLiteral("sat")); e != nil {
			_ = d.Close()
			return e
		}
		if e = r.RegisterModule("sat", d); e != nil {
			_ = d.Close()
			return e
		}
		return arena.ErrInjectedFailure
	}
	g, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	if e = g.ensureRegistered(); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	paramsMust(t, g.ensureRegistered())
	if calls != 1 || !g.modulesRegistered {
		t.Fatal("source true-prefix must suppress retry", calls)
	}
	mod, e := sy.Intern("sat")
	paramsMust(t, e)
	d, e := g.moduleDescriptor(mod)
	paramsMust(t, e)
	exists, e := d.ContainsName("seed")
	paramsMust(t, e)
	if !exists {
		t.Fatal("prior real registration lost")
	}
	incoming, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	paramsMust(t, e)
	paramsMust(t, incoming.InsertName("factor", ParamDouble, pdLiteral("factor"), pdLiteral("1.5"), pdLiteral("sat")))
	paramsMust(t, g.RegisterModule("sat", incoming))
	if _, e = incoming.ContainsName("factor"); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source destroys merged incoming", e)
	}
	exists, e = d.ContainsName("factor")
	paramsMust(t, e)
	if !exists {
		t.Fatal("merge must append actual descriptor")
	}
	paramsMust(t, g.Reset())
	paramsMust(t, g.ensureRegistered())
	if calls != 1 {
		t.Fatal("Reset must not reset registration flag")
	}
}
func TestGlobalParamsConstructorUnwindAndColdOwnerGuards(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	success := false
	for fail := int64(0); fail < 30; fail++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(fail)
		g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
		if e == nil {
			success = true
			copied := &GlobalParams{self: g, ContextField: ctx}
			if _, e = copied.GetModuleName("sat"); !errors.Is(e, ErrGlobalParamsOwner) {
				t.Fatal("shallow owner", e)
			}
			paramsMust(t, g.Close())
			if _, e = g.GetModuleName("sat"); !errors.Is(e, arena.ErrClosed) {
				t.Fatal(e)
			}
			paramsMust(t, g.Close())
		} else if !errors.Is(e, arena.ErrInjectedFailure) || g != nil {
			t.Fatal(fail, e)
		}
		if b.Used() != 0 {
			t.Fatal("unpublished actual owner graph", fail, b.Used())
		}
	}
	if !success {
		t.Fatal("no constructor completion")
	}
}

func TestGlobalParamsResetOwnerDirectoryPlateauAndWarmSnapshots(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
	paramsMust(t, e)
	defer func() {
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	key, e := sy.Intern("sat")
	paramsMust(t, e)
	_, e = g.GetParams(key)
	paramsMust(t, e)
	paramsMust(t, g.Reset())
	baseline := b.Used()
	for i := 0; i < 100; i++ {
		_, e = g.GetParams(key)
		paramsMust(t, e)
		paramsMust(t, g.Reset())
		if b.Used() != baseline {
			t.Fatalf("reset metadata grew iteration%d: %d vs%d", i, b.Used(), baseline)
		}
	}
	_, e = g.GetParams(key)
	paramsMust(t, e)
	b.FailAfter(0)
	var failed error
	allocations := testing.AllocsPerRun(100, func() {
		v, err := g.GetModule(key)
		if err != nil {
			failed = err
			return
		}
		if err = ctx.Del(&v); err != nil {
			failed = err
		}
	})
	b.FailAfter(-1)
	if failed != nil || allocations != 0 {
		t.Fatal(failed, allocations)
	}
}
