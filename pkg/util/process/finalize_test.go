package process

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFinalizeConcreteServices(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	p := memory.NewProcessAccounting()
	m, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	must(t, err)
	first, err := m.BeginSourceInitialize(math.MaxUint32)
	must(t, err)
	if !first {
		t.Fatal("first process initialization was skipped")
	}
	r, err := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: m, Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	must(t, err)
	must(t, r.InitializeSource())
	s, err := symbol.NewService(symbol.Config{Budget: b})
	must(t, err)
	must(t, s.InitializeSource())
	sy, err := s.Manager()
	must(t, err)
	q, err := r.Service()
	must(t, err)
	c, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b, PageEntries: 1,
		Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	must(t, err)
	cat, err := params.NewGlobalParamsCatalog(sy, params.GlobalParamsCatalogConfig{Budget: b,
		Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}},
		Text:         arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	must(t, err)
	g, err := params.NewGlobalParamsService(sy, c, params.GlobalParamsConfig{Budget: b, Catalog: cat, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	must(t, err)
	must(t, g.InitializeSource())
	must(t, m.CompleteSourceInitialize())
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	must(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	must(t, err)
	must(t, d.EnableDebug("memory"))
	tr := diagnostic.NewTraceTags()
	tr.Enable("memory")
	must(t, FinalizeGenerated(p, d, g, r, s, tr))
	if enabled, err := d.IsDebugEnabled("memory"); err != nil || enabled || tr.IsEnabled("memory") {
		t.Fatal("diagnostic finalizers did not clear tags", enabled, err)
	}
	if _, err := g.Manager(); err != params.ErrGlobalParamsUninitialized {
		t.Fatal("global params pointer not cleared", err)
	}
	if _, err := s.Manager(); err != symbol.ErrServiceUninitialized {
		t.Fatal("symbol table pointer not cleared", err)
	}
	if active, err := p.BeginFinalize(); err != nil || active {
		t.Fatal("process guard not completed", active, err)
	}
	must(t, g.Close())
	must(t, cat.Close())
	must(t, c.Close())
	must(t, r.Close())
	must(t, s.Close())
	must(t, d.Close())
	must(t, region.Close())
	must(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("retained budget", b.Used())
	}
}
