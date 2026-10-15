package aig

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func aigTestBindSMT2Params(t *testing.T, symbols *symbol.Manager, numbers *numeral.RationalService, budget *arena.Budget) func() {
	t.Helper()
	ctx, err := uparams.NewParamsContext(symbols, numbers, uparams.ParamsConfig{Budget: budget,
		Text: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	aigMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget,
		Descriptions: arena.BytesRegionConfig{Descriptors: arena.Config{Budget: budget}},
		Text:         arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	aigMust(t, err)
	globals, err := uparams.NewGlobalParamsService(symbols, ctx, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	aigMust(t, err)
	aigMust(t, globals.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	aigMust(t, err)
	return func() {
		_, err := uparams.BindProcessGlobalParamsService(previous)
		aigMust(t, err)
		aigMust(t, globals.Close())
		aigMust(t, catalog.Close())
		aigMust(t, ctx.Close())
	}
}
