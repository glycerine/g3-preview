package ast

import (
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

func dynAckName(s string) *string { return &s }

func astParamsNamedColdGraph(t *testing.T, g *util.GlobalParamsService, b *arena.Budget) {
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
	if total != b.Used() {
		t.Fatal("named cold owners", total, b.Used())
	}
}
