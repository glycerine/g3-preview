// Source: generated gparams_register_modules.cpp and mem_initializer.cpp.
package engine

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/process"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestGeneratedProcessLinksFullParameterCatalog(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	b := arena.NewBudget(1 << 28)
	p := memory.NewProcessAccounting()
	m, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	must(err)
	r, err := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: m, Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	must(err)
	s, err := symbol.NewService(symbol.Config{Budget: b})
	must(err)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	must(err)
	d, err := diagnostic.NewDebugContext(region, b)
	must(err)
	cfg := process.InitializationConfig{Runtime: m, Rational: r, Symbols: s, Debug: d,
		Params:  params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Catalog: params.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Globals: params.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}}
	ready, err := InitializeGenerated(cfg, math.MaxUint32)
	must(err)
	g, err := ready.Globals.Manager()
	must(err)
	var output bytes.Buffer
	must(g.DisplayModulesInto(&output))
	if len(g.Descriptors) != 19 || strings.Count(output.String(), "[module] ") != 19 {
		t.Fatal("generated catalog module count", len(g.Descriptors), output.String())
	}
	for _, name := range []string{"ackermannization", "rewriter", "pi", "sat", "smt", "lp", "opt"} {
		if !strings.Contains(output.String(), "[module] "+name) {
			t.Fatal("generated parameter catalog omitted module", name)
		}
	}
	must(process.FinalizeGenerated(p, d, ready.Globals, r, s, diagnostic.NewTraceTags()))
	must(ready.Close())
	must(r.Close())
	must(s.Close())
	must(d.Close())
	must(region.Close())
	must(m.Close())
	if b.Used() != 0 {
		t.Fatal("generated catalog retained owners", b.Used())
	}
}
