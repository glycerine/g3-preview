package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/subpaving"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSubpavingEngineProcessBindings(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	integers := numeral.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: integers})
	if err != nil {
		t.Fatal(err)
	}
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	value, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, err := range []error{pc.Del(&value), pc.Close(), q.Close(), sy.Close(), qm.CloseChecked(), r.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("retained tactic process budget", b.Used())
		}
	})

	key, memoryKey := "numeral", "max_memory"
	hwf, err := sy.Intern("hwf")
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.SetSymbolName(&value, &key, hwf); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&value, &memoryKey, 1); err != nil {
		t.Fatal(err)
	}
	limit := &control.Reslimit{Limit: 100}
	hardware, err := NewHardwareEngine(limit, qm, pc, &value)
	if err != nil {
		t.Fatal(err)
	}
	if hardware.context == nil {
		t.Fatal("missing hardware context")
	}
	otherBudget := arena.NewBudget(1 << 22)
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: otherBudget, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := other.Allocate(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if p.AllocationSize() <= 1<<20 {
		t.Fatal("second runtime did not exceed one-megabyte limit")
	}
	shell, err := hardware.Shell()
	if err != nil {
		t.Fatal(err)
	}
	if err := shell.Search(); !errors.Is(err, subpaving.ErrSubpavingMaxMemory) {
		t.Fatal("hardware engine did not bind rational process", err)
	}
	if err := hardware.Close(); err != nil {
		t.Fatal(err)
	}

	otherProcess := memory.NewProcessAccounting()
	wrongRuntime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: otherProcess, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	fastConfig := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	fastConfig.InitialCapacity, fastConfig.Memory = 4, wrongRuntime
	wrongNumbers, err := numeral.NewFastFloatManager(fastConfig)
	if err != nil {
		t.Fatal(err)
	}
	fastName, err := sy.Intern("mpff")
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.SetSymbolName(&value, &key, fastName); err != nil {
		t.Fatal(err)
	}
	if engine, err := NewFastEngine(limit, wrongNumbers, qm, pc, &value); engine != nil || !errors.Is(err, arena.ErrConfig) {
		t.Fatal("divergent numeral process owners", engine, err)
	}
	if err := wrongNumbers.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err := wrongRuntime.Close(); err != nil {
		t.Fatal(err)
	}

	fastConfig.Memory = r
	numbers, err := numeral.NewFastFloatManager(fastConfig)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := NewFastEngine(limit, numbers, qm, pc, &value)
	if err != nil {
		t.Fatal(err)
	}
	fastShell, err := fast.Shell()
	if err != nil {
		t.Fatal(err)
	}
	if err := fastShell.Search(); !errors.Is(err, subpaving.ErrSubpavingMaxMemory) {
		t.Fatal("fast engine did not read shared process", err)
	}
	for _, err := range []error{fast.Close(), numbers.CloseChecked(), other.Deallocate(&extra), other.Close()} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if otherBudget.Used() != 0 {
		t.Fatal("retained second runtime budget", otherBudget.Used())
	}
}
