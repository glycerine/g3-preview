package params

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func envParamsRuntimeFixture(t *testing.T) (*GlobalParams, *diagnostic.VerboseContext, *umemory.Runtime, *EnvParamsRuntime, *bytes.Buffer) {
	t.Helper()
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(1 << 24)
	cfg := gparamsTestConfig(b)
	cfg.Register = func(r GlobalParamsRegistrar) error {
		d, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		if e != nil {
			return e
		}
		defer d.Close()
		if e = catalogCollectEnvParams(d); e != nil {
			return e
		}
		return r.RegisterGlobal(d)
	}
	g, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	v, e := diagnostic.NewVerboseContext(out, b)
	paramsMust(t, e)
	m, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	paramsMust(t, e)
	stderr := &bytes.Buffer{}
	warnings := diagnostic.NewWarningService(stderr)
	env, e := NewEnvParamsRuntime(g, v, m, warnings)
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, m.Close())
		paramsMust(t, v.Close())
		paramsMust(t, out.Close())
		paramsMust(t, g.Close())
		if b.Used() != 0 {
			t.Fatal("environment owner storage", b.Used())
		}
	})
	return g, v, m, env, stderr
}

func TestEnvParamsRuntimeSourceOrderAndRawWatermark(t *testing.T) {
	g, v, m, env, stderr := envParamsRuntimeFixture(t)
	paramsMust(t, v.SetVerbosityLevel(9))
	paramsMust(t, env.Update())
	level, e := v.VerbosityLevel()
	paramsMust(t, e)
	if level != 9 || !env.WarningsEnabled() {
		t.Fatal("source defaults", level, env.WarningsEnabled())
	}
	paramsMust(t, env.warnings.Warning("default"))
	if stderr.String() != "WARNING: default\n" {
		t.Fatal("default warning route", stderr.String())
	}
	for _, pair := range [][2]string{{"verbose", "17"}, {"warning", "false"}, {"memory_max_size", "1"}, {"memory_max_alloc_count", "3"}, {"memory_high_watermark", "5"}} {
		paramsMust(t, g.SetName(pair[0], pair[1]))
	}
	paramsMust(t, env.Update())
	level, e = v.VerbosityLevel()
	paramsMust(t, e)
	if level != 17 || env.WarningsEnabled() {
		t.Fatal("ordered scalar updates", level, env.WarningsEnabled())
	}
	paramsMust(t, env.warnings.Warning("suppressed"))
	if stderr.String() != "WARNING: default\n" {
		t.Fatal("warning toggle did not suppress output", stderr.String())
	}
	count, e := m.ConfiguredMaxAllocationCount()
	paramsMust(t, e)
	if count != 3 {
		t.Fatal("max allocation count", count)
	}
	block, e := m.Allocate(1)
	paramsMust(t, e)
	above, e := m.AboveWatermark()
	paramsMust(t, e)
	if !above {
		t.Fatal("source watermark is passed as raw uint, not megabytes")
	}
	paramsMust(t, m.Deallocate(&block))
	paramsMust(t, g.SetName("verbose", "23"))
	paramsMust(t, g.SetName("warning", "true"))
	paramsMust(t, m.Close())
	if e := env.Update(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed memory after earlier assignments", e)
	}
	level, e = v.VerbosityLevel()
	paramsMust(t, e)
	if level != 23 || !env.WarningsEnabled() {
		t.Fatal("completed source prefix lost", level, env.WarningsEnabled())
	}
	paramsMust(t, env.warnings.Warning("resumed"))
	if stderr.String() != "WARNING: default\nWARNING: resumed\n" {
		t.Fatal("warning toggle did not resume output", stderr.String())
	}
}

func TestEnvParamsSharedProcessWatermarkAndNumericCohort(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	p := umemory.NewProcessAccounting()
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	paramsMust(t, e)
	other, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	paramsMust(t, e)
	integers := unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: integers})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	paramsMust(t, e)
	ctx, e := NewParamsContext(sy, q, ParamsConfig{Budget: b})
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Register = func(reg GlobalParamsRegistrar) error {
		d, err := NewParamDescrs(sy, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
		if err != nil {
			return err
		}
		defer d.Close()
		if err = catalogCollectEnvParams(d); err != nil {
			return err
		}
		return reg.RegisterGlobal(d)
	}
	g, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	stream, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	verbose, e := diagnostic.NewVerboseContext(stream, b)
	paramsMust(t, e)
	warnings := diagnostic.NewWarningService(&bytes.Buffer{})
	env, e := NewEnvParamsRuntime(g, verbose, r, warnings)
	paramsMust(t, e)
	wrong, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: umemory.NewProcessAccounting(), SizeTBits: 64})
	paramsMust(t, e)
	before := b.Used()
	if refused, err := NewEnvParamsRuntime(g, verbose, wrong, warnings); refused != nil || !errors.Is(err, arena.ErrConfig) || b.Used() != before {
		t.Fatal("different parameter numeral process", refused, err, b.Used(), before)
	}
	paramsMust(t, wrong.Close())
	mark := p.AllocationSize() + 1024
	if mark > uint64(^uint32(0)) {
		t.Fatal("fixture watermark exceeds source unsigned domain", mark)
	}
	paramsMust(t, g.SetName("memory_high_watermark", strconv.FormatUint(mark, 10)))
	paramsMust(t, env.Update())
	if p.AboveWatermark() {
		t.Fatal("process initially above watermark", p.AllocationSize(), mark)
	}
	block, e := other.Allocate(2048)
	paramsMust(t, e)
	above, e := r.AboveWatermark()
	paramsMust(t, e)
	if !above || !p.AboveWatermark() {
		t.Fatal("allocation in second runtime must cross raw process watermark", p.AllocationSize(), mark)
	}
	paramsMust(t, other.Deallocate(&block))
	if p.AboveWatermark() {
		t.Fatal("released second-runtime allocation still above watermark", p.AllocationSize(), mark)
	}
	for _, err := range []error{other.Close(), verbose.Close(), stream.Close(), g.Close(), ctx.Close(), q.Close(), sy.Close(), r.Close()} {
		paramsMust(t, err)
	}
	if b.Used() != 0 {
		t.Fatal("retained shared environment budget", b.Used())
	}
}
