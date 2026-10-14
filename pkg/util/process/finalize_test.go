package process

import (
	"bytes"
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

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFinalizeInvalidOwnersLeaveProcessReady(t *testing.T) {
	b := arena.NewBudget(1 << 16)
	p := memory.NewProcessAccounting()
	m, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	must(t, err)
	first, err := m.BeginSourceInitialize(math.MaxUint32)
	if err != nil || !first {
		t.Fatal("begin source initialization", first, err)
	}
	must(t, m.CompleteSourceInitialize())
	if err := FinalizeGenerated(p, nil, nil, nil, nil, nil); !errors.Is(err, arena.ErrConfig) {
		t.Fatal("missing generated owner", err)
	}
	active, err := p.BeginFinalize()
	if err != nil || !active {
		t.Fatal("invalid call consumed source finalization guard", active, err)
	}
	must(t, p.CompleteFinalize())
	must(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("retained budget", b.Used())
	}
}

func TestGeneratedInitializeAndFinalizeConcreteServices(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	p := memory.NewProcessAccounting()
	m, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	must(t, err)
	r, err := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: m, Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	must(t, err)
	s, err := symbol.NewService(symbol.Config{Budget: b})
	must(t, err)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	must(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	must(t, err)
	previousDebug := diagnostic.BorrowProcessDebugContext()
	var traceOutput bytes.Buffer
	tr := diagnostic.NewTraceTagsWithWriter(&traceOutput)
	tr.Enable("gparams")
	traceCompiled := tr.ShouldTrace("gparams")
	cfg := InitializationConfig{Runtime: m, Rational: r, Symbols: s, Debug: d, Trace: tr,
		Params:  params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Catalog: params.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Globals: params.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}}
	init, err := InitializeGenerated(cfg, math.MaxUint32)
	must(t, err)
	if diagnostic.BorrowProcessDebugContext() != d {
		t.Fatal("generated process did not bind its debug owner")
	}
	if init == nil || init.Context == nil || init.Catalog == nil || init.Globals == nil || init.Trace != tr {
		t.Fatal("generated initialization did not publish its parameter owners")
	}
	g := init.Globals
	if again, err := InitializeGenerated(InitializationConfig{Runtime: m}, 1<<25); err != nil || again != nil {
		t.Fatal("repeated generated initialization", again, err)
	}
	symbols, err := s.Manager()
	must(t, err)
	must(t, d.EnableAssertions(false))
	if bare, present, err := symbols.BareString(symbol.Numerical(7)); err != nil || !present || bare != "" {
		t.Fatal("generated symbol table missed disabled process assertions", bare, present, err)
	}
	if enabled, err := g.NamespaceAssertionsEnabled(); err != nil || enabled {
		t.Fatal("global params missed disabled process assertions", enabled, err)
	}
	must(t, d.EnableAssertions(true))
	if bare, present, err := symbols.BareString(symbol.Numerical(7)); params.GlobalParamsSourceDebug {
		if !errors.Is(err, symbol.ErrInvalid) || present || bare != "" {
			t.Fatal("generated symbol table missed re-enabled process assertion", bare, present, err)
		}
	} else if err != nil || !present || bare != "" {
		t.Fatal("native bare_str numerical continuation", bare, present, err)
	}
	if enabled, err := g.NamespaceAssertionsEnabled(); err != nil || enabled != params.GlobalParamsSourceDebug {
		t.Fatal("global params did not reread process assertions", enabled, err)
	}
	must(t, d.EnableDebug("memory"))
	tr.Enable("memory")
	must(t, FinalizeGenerated(p, d, g, r, s, tr))
	wantTrace := ""
	if traceCompiled {
		wantTrace = "-------- [gparams] init src/util/gparams.cpp:634 ---------\ngparams::init()\n------------------------------------------------\n" +
			"-------- [gparams] finalize src/util/gparams.cpp:639 ---------\ngparams::finalize()\n------------------------------------------------\n"
	}
	if traceOutput.String() != wantTrace {
		t.Fatal("generated process trace owner and source order", traceOutput.String())
	}
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
	if p.InitializationStarted() {
		t.Fatal("finalization retained initialized guard")
	}
	must(t, FinalizeGenerated(p, nil, nil, nil, nil, nil))
	must(t, init.Close())
	if diagnostic.BorrowProcessDebugContext() != previousDebug {
		t.Fatal("generated close did not restore the prior process debug owner")
	}
	if init.Trace != nil {
		t.Fatal("closed generated owner retained borrowed trace")
	}
	must(t, r.Close())
	must(t, s.Close())
	must(t, d.Close())
	must(t, region.Close())
	must(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("retained budget", b.Used())
	}
}
