package api

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/process"
	"github.com/glycerine/g3/pkg/util/symbol"
)

// Source: src/api/api_config_params.cpp:30-69. The public operations borrow
// the generated process owner, publish environment updates and use the shared
// warning stream for invalid names.
func TestAPIGlobalParamsGeneratedProcessSetGetResetAndWarnings(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	p := memory.NewProcessAccounting()
	m, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p})
	satOK(t, err)
	r, err := numeral.NewRationalGlobalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: m, Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	satOK(t, err)
	sy, err := symbol.NewService(symbol.Config{Budget: b})
	satOK(t, err)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	satOK(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	satOK(t, err)
	cfg := process.InitializationConfig{Runtime: m, Rational: r, Symbols: sy, Debug: d,
		Params:  params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Catalog: params.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}},
		Globals: params.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}}
	init, err := process.InitializeGenerated(cfg, math.MaxUint32)
	satOK(t, err)
	warnings := diagnostic.BorrowProcessWarningService()
	oldWarnings := warnings.Enabled()
	oldVerbosity, err := diagnostic.ProcessVerbosityLevel()
	satOK(t, err)
	var warningOutput bytes.Buffer
	satOK(t, warnings.SetWarningStream(&warningOutput))
	t.Cleanup(func() {
		satOK(t, process.FinalizeGenerated(p, d, init.Globals, r, sy, init.Trace))
		satOK(t, init.Close())
		satOK(t, diagnostic.SetProcessVerbosityLevel(oldVerbosity))
		satOK(t, warnings.SetWarningStream(nil))
		satOK(t, warnings.Enable(oldWarnings))
		satOK(t, r.Close())
		satOK(t, sy.Close())
		satOK(t, d.Close())
		satOK(t, region.Close())
		satOK(t, m.Close())
		if b.Used() != 0 {
			t.Errorf("retained budget %d", b.Used())
		}
	})
	satOK(t, warnings.Enable(true))
	satOK(t, GlobalParamSet("verbose", "3"))
	value, ok, err := GlobalParamGet("verbose")
	satOK(t, err)
	if !ok || value != "3" {
		t.Fatalf("set/get: %q %v warning=%q", value, ok, warningOutput.String())
	}
	level, err := diagnostic.ProcessVerbosityLevel()
	satOK(t, err)
	if level != 3 {
		t.Fatalf("process verbosity %d", level)
	}
	satOK(t, GlobalParamSet("not_a_known_parameter", "7"))
	if !strings.Contains(warningOutput.String(), "WARNING: unknown parameter 'not_a_known_parameter'") {
		t.Fatalf("setter warning %q", warningOutput.String())
	}
	warningOutput.Reset()
	value, ok, err = GlobalParamGet("not_a_known_parameter")
	satOK(t, err)
	if ok || value != "" || !strings.Contains(warningOutput.String(), "WARNING: unknown parameter 'not_a_known_parameter'") {
		t.Fatalf("getter warning %q %v %q", value, ok, warningOutput.String())
	}
	satOK(t, GlobalParamSet("warning", "false"))
	warningOutput.Reset()
	satOK(t, GlobalParamSet("not_a_known_parameter", "7"))
	if warnings.Enabled() || warningOutput.Len() != 0 {
		t.Fatalf("disabled process warnings: enabled=%v output=%q", warnings.Enabled(), warningOutput.String())
	}
	satOK(t, GlobalParamResetAll())
	if !warnings.Enabled() {
		t.Fatal("reset did not restore the source warning default")
	}
	value, ok, err = GlobalParamGet("verbose")
	satOK(t, err)
	if !ok || value != "0" {
		t.Fatalf("reset default %q %v", value, ok)
	}
	warningOutput.Reset()
	current := init.Globals.Current
	init.Globals.Current = nil
	setErr := GlobalParamSet("sat.random_seed", "7")
	_, ok, getErr := GlobalParamGet("sat.random_seed")
	init.Globals.Current = current
	if (!errors.Is(setErr, params.ErrGlobalParamsAssertion) && !errors.Is(setErr, params.ErrGlobalParamsUninitialized)) || warningOutput.Len() != 0 {
		t.Fatalf("nonexception setter failure: %v warning=%q", setErr, warningOutput.String())
	}
	if ok || (!errors.Is(getErr, params.ErrGlobalParamsAssertion) && !errors.Is(getErr, params.ErrGlobalParamsUninitialized)) || warningOutput.Len() != 0 {
		t.Fatalf("nonexception getter failure: %v %v warning=%q", ok, getErr, warningOutput.String())
	}
}

func TestAPIGlobalParamFirstCallInitializesGeneratedProcess(t *testing.T) {
	if params.BorrowProcessGlobalParamsService() != nil {
		t.Fatal("previous test retained a process global parameter binding")
	}
	t.Cleanup(func() { satOK(t, engine.FinalizeDefaultProcess()) })
	value, ok, err := GlobalParamGet("smt.random_seed")
	satOK(t, err)
	if !ok || value != "0" || params.BorrowProcessGlobalParamsService() == nil {
		t.Fatalf("first-call default and binding: %q %v", value, ok)
	}
	first := params.BorrowProcessGlobalParamsService()
	satOK(t, GlobalParamSet("smt.random_seed", "17"))
	value, ok, err = GlobalParamGet("smt.random_seed")
	satOK(t, err)
	if !ok || value != "17" || params.BorrowProcessGlobalParamsService() != first {
		t.Fatalf("repeated-call value/binding: %q %v", value, ok)
	}
	satOK(t, engine.FinalizeDefaultProcess())
	if params.BorrowProcessGlobalParamsService() != nil {
		t.Fatal("default finalization retained a process global parameter binding")
	}
}
