package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func envParamsRuntimeFixture(t *testing.T) (*GlobalParams, *diagnostic.VerboseContext, *umemory.Runtime, *EnvParamsRuntime) {
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
	env, e := NewEnvParamsRuntime(g, v, m)
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
	return g, v, m, env
}

func TestEnvParamsRuntimeSourceOrderAndRawWatermark(t *testing.T) {
	g, v, m, env := envParamsRuntimeFixture(t)
	paramsMust(t, v.SetVerbosityLevel(9))
	paramsMust(t, env.Update())
	level, e := v.VerbosityLevel()
	paramsMust(t, e)
	if level != 9 || !env.WarningsEnabled() {
		t.Fatal("source defaults", level, env.WarningsEnabled())
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
}
