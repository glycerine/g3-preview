package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParserParamsDefaultsSnapshotAndLocalPrecedence(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget, PageSize: 64, DescriptorPageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	context, err := uparams.NewParamsContext(symbols, numerals, uparams.ParamsConfig{Budget: budget, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	if err != nil {
		t.Fatal(err)
	}
	globals, err := uparams.NewGlobalParamsService(symbols, context, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	if err != nil {
		t.Fatal(err)
	}
	if err = globals.InitializeSource(); err != nil {
		t.Fatal(err)
	}
	local, err := context.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Del(&local); err != nil {
			t.Error(err)
		}
		if err := globals.Close(); err != nil {
			t.Error(err)
		}
		if err := catalog.Close(); err != nil {
			t.Error(err)
		}
		if err := context.Close(); err != nil {
			t.Error(err)
		}
		if err := numerals.Close(); err != nil {
			t.Error(err)
		}
		if err := symbols.Close(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Errorf("owners retained %d", budget.Used())
		}
	})
	defaults, err := NewParserParams(context, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	ignoreUser, _ := defaults.IgnoreUserPatterns()
	ignoreBad, _ := defaults.IgnoreBadPatterns()
	visual, _ := defaults.ErrorForVisualStudio()
	if ignoreUser || !ignoreBad || visual {
		t.Fatal("defaults", ignoreUser, ignoreBad, visual)
	}
	if err = defaults.Close(); err != nil {
		t.Fatal(err)
	}
	params, err := NewParserParams(context, globals, &local)
	if err != nil {
		t.Fatal(err)
	}
	key := "ignore_user_patterns"
	if err = context.SetBoolName(&local, &key, true); err != nil {
		t.Fatal(err)
	}
	key = "error_for_visual_studio"
	if err = context.SetBoolName(&local, &key, true); err != nil {
		t.Fatal(err)
	}
	ignoreUser, _ = params.IgnoreUserPatterns()
	visual, _ = params.ErrorForVisualStudio()
	if !ignoreUser || !visual {
		t.Fatal("precedence", ignoreUser, visual)
	}
	copy := *params
	if _, err = copy.IgnoreBadPatterns(); err == nil {
		t.Fatal("copied facade accepted")
	}
	if err = params.Close(); err != nil {
		t.Fatal(err)
	}
}
