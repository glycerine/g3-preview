package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
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

var _ umemory.ConstructionFailure = (*util.RationalService)(nil)

func catalogTestConfig(b *arena.Budget) util.GlobalParamsCatalogConfig {
	return util.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
}
func paramsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func gparamsTestConfig(b *arena.Budget) util.GlobalParamsConfig {
	return util.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}
}
