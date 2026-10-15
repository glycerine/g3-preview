//go:build g3debug && g3trace

package process

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
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestGeneratedProcessBindsSymbolMemoryTraceBeforeInitialize(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	p := memory.NewProcessAccounting()
	runtime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	must(t, err)
	rational, err := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: runtime, Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	must(t, err)
	symbols, err := symbol.NewService(symbol.Config{Budget: b})
	must(t, err)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	must(t, err)
	debug, err := diagnostic.NewDebugContext(region, b)
	must(t, err)
	var out bytes.Buffer
	trace := diagnostic.NewTraceTagsWithWriter(&out)
	trace.Enable("memory")
	config := InitializationConfig{Runtime: runtime, Rational: rational, Symbols: symbols, Debug: debug, Trace: trace,
		Params:  params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Catalog: params.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Globals: params.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}}
	init, err := InitializeGenerated(config, math.MaxUint32)
	must(t, err)
	rationalAlloc := strings.Index(out.String(), " src/util/rational.cpp:66 synch_mpq_manager ")
	symbolAlloc := strings.Index(out.String(), " src/util/symbol.cpp:67 internal_symbol_table ")
	gparamsAlloc := strings.Index(out.String(), " src/util/gparams.cpp:635 imp ")
	if rationalAlloc < 0 || symbolAlloc <= rationalAlloc || gparamsAlloc <= symbolAlloc {
		t.Fatal("generated memory trace missed source rational-symbol-gparams initialization order", out.String())
	}
	must(t, FinalizeGenerated(p, debug, init.Globals, rational, symbols, trace))
	if strings.Contains(out.String(), " src/util/symbol.cpp:72\n") || strings.Contains(out.String(), " src/util/rational.cpp:82\n") {
		t.Fatal("generated process finalizing guard emitted manager deallocation", out.String())
	}
	must(t, init.Close())
	must(t, rational.Close())
	must(t, symbols.Close())
	must(t, debug.Close())
	must(t, region.Close())
	must(t, runtime.Close())
	if b.Used() != 0 {
		t.Fatal("generated process retained symbol trace owner charge", b.Used())
	}
}
