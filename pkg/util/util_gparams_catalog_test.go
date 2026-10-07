// Licensed under the MIT license.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func catalogTestConfig(b *arena.Budget) GlobalParamsCatalogConfig {
	return GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
}
func TestGlobalParamsCatalogCompleteMetadataAndExactDispatcher(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	c, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	g, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c})
	paramsMust(t, e)
	defer func() {
		paramsMust(t, g.Close())
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	paramsMust(t, g.ensureRegistered())
	// Complete static source fixture is independent of production collectors.
	seen := map[string]bool{}
	global, module := 0, 0
	for _, row := range catalogSourceMetadata {
		qualified := row.group + "/" + row.key
		if seen[qualified] {
			continue
		}
		seen[qualified] = true
		var d *ParamDescrs
		if row.group == "" {
			d = g.GlobalDescriptors
			global++
		} else {
			key, e := sy.Intern(row.group)
			paramsMust(t, e)
			d, e = g.moduleDescriptor(key)
			paramsMust(t, e)
			module++
		}
		kind, e := d.KindName(row.key)
		paramsMust(t, e)
		if kind != row.kind {
			t.Fatal(row, kind)
		}
		descr, present, e := d.DescriptionName(row.key)
		paramsMust(t, e)
		if !present || descr != row.descr {
			t.Fatal(row, descr, present)
		}
		def, present, e := d.DefaultName(row.key)
		paramsMust(t, e)
		if !present || def != row.def {
			t.Fatal(row, def, present)
		}
		mod, present, e := d.ModuleName(row.key)
		paramsMust(t, e)
		if mod != row.group || present != (row.group != "") {
			t.Fatal(row, mod, present)
		}
	}
	if len(catalogSourceMetadata) != 458 || global != 20 || module != 430 || g.ModuleDescriptors.size != 19 || g.ModuleDescriptions.size != 15 {
		t.Fatal(len(catalogSourceMetadata), global, module, g.ModuleDescriptors.size, g.ModuleDescriptions.size)
	}
	// Source new-module publication order, distinct from hash bucket order.
	modules := []string{"ackermannization", "rewriter", "nnf", "pi", "pp", "interp", "algebraic", "rcf", "model_evaluator", "model", "fixedpoint", "nlsat", "opt", "parser", "sat", "smt", "combined_solver", "sls", "lp"}
	if len(g.Descriptors) != len(modules) {
		t.Fatal(len(g.Descriptors))
	}
	for i, moduleName := range modules {
		key, e := sy.Intern(moduleName)
		paramsMust(t, e)
		hash, e := sy.Hash(key)
		paramsMust(t, e)
		bucket := g.ModuleDescriptors.find(key, hash)
		if bucket < 0 || g.ModuleDescriptors.Entries[bucket].descriptor != uint32(i+1) {
			t.Fatal("source publication order", i, moduleName)
		}
	}
	for _, row := range catalogSourceDescriptions {
		// Last assignment for a repeated module wins independently of dictionary layout.
		last := row.text
		for _, later := range catalogSourceDescriptions {
			if later.module == row.module {
				last = later.text
			}
		}
		key, e := sy.Intern(row.module)
		paramsMust(t, e)
		hash, e := sy.Hash(key)
		paramsMust(t, e)
		i := g.ModuleDescriptions.find(key, hash)
		if i < 0 {
			t.Fatal(row)
		}
		out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
		paramsMust(t, e)
		paramsMust(t, g.descriptionInto(out, g.ModuleDescriptions.Entries[i].text))
		text, e := out.CStr()
		paramsMust(t, e)
		paramsMust(t, out.Close())
		if text != last {
			t.Fatal(row, text, last)
		}
	}
}
func TestGlobalParamsCatalogConcreteHookGuardsAndIndependentReuse(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	c, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	base := b.Used()
	g, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c, Register: func(GlobalParamsRegistrar) error { return nil }})
	if g != nil || !errors.Is(e, arena.ErrConfig) || b.Used() != base {
		t.Fatal(g, e, b.Used(), base)
	}
	first, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c})
	paramsMust(t, e)
	paramsMust(t, first.ensureRegistered())
	second, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c})
	paramsMust(t, e)
	paramsMust(t, second.ensureRegistered())
	if first.Descriptors[0] == second.Descriptors[0] || first.Bindings[0].region != second.Bindings[0].region {
		t.Fatal("owning descriptors independent, descriptions truly borrowed")
	}
	paramsMust(t, first.Close())
	text, e := second.GetValueName("sat.asymm_branch.rounds")
	paramsMust(t, e)
	if text != "32" {
		t.Fatal(text)
	}
	paramsMust(t, second.Close())
	paramsMust(t, c.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	g, e = NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c})
	if g != nil || !errors.Is(e, arena.ErrClosed) || b.Used() != 0 {
		t.Fatal(g, e, b.Used())
	}
}

func TestGlobalParamsCatalogDescriptionBorrowReadAndClosedPublicationPrefix(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	c, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	g, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c})
	paramsMust(t, e)
	defer func() {
		paramsMust(t, g.Close())
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	paramsMust(t, g.ensureRegistered())
	paramsMust(t, c.Region.Write(c.descriptions[15], 0, 'T'))
	out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	defer out.Close()
	key, e := sy.Intern("ackermannization")
	paramsMust(t, e)
	hash, e := sy.Hash(key)
	paramsMust(t, e)
	i := g.ModuleDescriptions.find(key, hash)
	paramsMust(t, g.descriptionInto(out, g.ModuleDescriptions.Entries[i].text))
	text, e := out.CStr()
	paramsMust(t, e)
	if text != "Tactics based on solving UF-theories via ackermannization (see also ackr module)" {
		t.Fatal("borrowed final literal", text)
	}
	paramsMust(t, out.Reset())
	paramsMust(t, c.Close())
	if e = g.DisplayModuleInto(out, "ackermannization"); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	text, e = out.CStr()
	paramsMust(t, e)
	if text != "[module] ackermannization, description: " {
		t.Fatal(text)
	}
	// Closing between construction and first lazy check must preserve TRUE,
	// then fail the actual borrowed catalog invocation without fake defaults.
	c2, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	other, e := NewGlobalParams(sy, ctx, GlobalParamsConfig{Budget: b, Catalog: c2})
	paramsMust(t, e)
	paramsMust(t, c2.Close())
	if e = other.ensureRegistered(); !errors.Is(e, arena.ErrClosed) || !other.modulesRegistered || other.ModuleDescriptors.size != 0 {
		t.Fatal(e, other.modulesRegistered, other.ModuleDescriptors.size)
	}
	paramsMust(t, other.Close())
}
