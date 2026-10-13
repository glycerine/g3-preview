package process

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestGeneratedInitializeRejectsForeignRationalMemoryProcess(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	one, two := memory.NewProcessAccounting(), memory.NewProcessAccounting()
	m1, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: one})
	must(t, err)
	m2, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: two})
	must(t, err)
	r, err := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: m2, Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	must(t, err)
	s, err := symbol.NewService(symbol.Config{Budget: b})
	must(t, err)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	must(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	must(t, err)
	ready, err := InitializeGenerated(InitializationConfig{Runtime: m1, Rational: r, Symbols: s, Debug: d,
		Params: params.ParamsConfig{Budget: b}, Catalog: params.GlobalParamsCatalogConfig{Budget: b},
		Globals: params.GlobalParamsConfig{Budget: b}}, math.MaxUint32)
	if ready != nil || !errors.Is(err, arena.ErrConfig) {
		t.Fatal("foreign rational process accepted", ready, err)
	}
	if active, err := one.BeginFinalize(); err != nil || active {
		t.Fatal("rejected initialization changed source process", active, err)
	}
	must(t, r.Close())
	must(t, s.Close())
	must(t, d.Close())
	must(t, region.Close())
	must(t, m2.Close())
	must(t, m1.Close())
	if b.Used() != 0 {
		t.Fatal("retained budget", b.Used())
	}
}
