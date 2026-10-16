package model_test

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func functionInterpretationTraceFixture(t *testing.T) (*ast.ASTManager, *diagnostic.TraceTags, *bytes.Buffer) {
	t.Helper()
	c, symbols, numerals := paramsTestOwner(t)
	budget := arena.NewBudget(1 << 25)
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget})
	astTestMust(t, err)
	globals, err := uparams.NewGlobalParamsService(symbols, c, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, globals.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		if budget.Used() != 0 {
			t.Fatalf("trace owners retained %d bytes", budget.Used())
		}
	})
	output := new(bytes.Buffer)
	tags := diagnostic.NewTraceTagsWithWriter(output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	return m, tags, output
}
