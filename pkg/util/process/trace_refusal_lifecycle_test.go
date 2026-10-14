//go:build g3trace

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

type processTraceRefusal struct {
	calls int
	cause error
}

func (w *processTraceRefusal) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= 10 {
		return 0, w.cause
	}
	return len(p), nil
}
func (w *processTraceRefusal) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestGeneratedMemoryLifecycleContinuesAfterGParamsTraceRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	p := memory.NewProcessAccounting()
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	must(t, e)
	rational, e := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: runtime, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	must(t, e)
	symbols, e := symbol.NewService(symbol.Config{Budget: b})
	must(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	must(t, e)
	debug, e := diagnostic.NewDebugContext(region, b)
	must(t, e)
	cause := errors.New("trace refused")
	writer := &processTraceRefusal{cause: cause}
	trace := diagnostic.NewTraceTagsWithWriter(writer)
	trace.Enable("gparams")
	cfg := InitializationConfig{Runtime: runtime, Rational: rational, Symbols: symbols, Debug: debug, Trace: trace,
		Params:  params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Catalog: params.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Globals: params.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16},
	}
	initialized, e := InitializeGenerated(cfg, math.MaxUint32)
	must(t, e)
	if initialized == nil || initialized.Globals == nil || !p.InitializationStarted() || writer.calls != 10 {
		t.Fatal("source init must publish after trace body refusal", initialized, writer.calls)
	}
	if _, e = initialized.Globals.SourceGetRef(); e != nil {
		t.Fatal("get_ref after sticky trace failbit", e)
	}
	must(t, FinalizeGenerated(p, debug, initialized.Globals, rational, symbols, trace))
	// The source ofstream keeps badbit after the init refusal. Finalization
	// still runs its trace body but cannot write another physical byte.
	if p.InitializationStarted() || trace.IsEnabled("gparams") || writer.calls != 10 {
		t.Fatal("generated finalization after sticky trace failbit", p.InitializationStarted(), writer.calls)
	}
	must(t, initialized.Close())
	must(t, rational.Close())
	must(t, symbols.Close())
	must(t, debug.Close())
	must(t, region.Close())
	must(t, runtime.Close())
	if b.Used() != 0 {
		t.Fatal("retained process owners", b.Used())
	}
}
