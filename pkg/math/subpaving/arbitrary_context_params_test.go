package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestArbitraryContextNamedParamsSourceOrderAndDescriptions(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	if c.maxMemory != math.MaxUint64 {
		t.Fatal("default memory limit", c.maxMemory)
	}
	w, err := NewArbitraryShellCoefficients(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	b := arena.NewBudget(1 << 24)
	sy, err := symbol.NewManager(symbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	d, err := params.NewParamDescrs(sy, params.ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, err := range []error{d.Close(), pc.Del(&p), pc.Close(), q.Close(), sy.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("parameter owners", b.Used())
		}
	})
	set := func(name string, value uint32) {
		t.Helper()
		if err := pc.SetUIntName(&p, &name, value); err != nil {
			t.Fatal(err)
		}
	}
	set("epsilon", 4)
	set("max_bound", 2)
	set("max_depth", 7)
	set("max_nodes", 9)
	set("max_memory", 3)
	set("nth_root_precision", 16)
	if err := w.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal(err)
	}
	core, err := f.numbers.BorrowCore()
	if err != nil {
		t.Fatal(err)
	}
	value := func(v *numeral.ArbitraryFloat) float64 {
		t.Helper()
		x, err := core.ToDouble(v)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	if c.zeroEpsilon || value(c.epsilon) != .25 || value(c.maxBound) != 100 || value(c.minusMaxBound) != -100 || value(c.rootPrecision) != 1.0/16 || c.maxDepth != 7 || c.maxNodes != 9 || c.maxMemory != 3*1024*1024 {
		t.Fatal("named MPF policy")
	}
	set("epsilon", 2)
	set("max_bound", 1<<31)
	if err := w.UpdateParamsRef(pc, &p); err == nil {
		t.Fatal("late MPF exponent refusal")
	}
	if value(c.epsilon) != .5 || value(c.maxBound) != 1 || value(c.minusMaxBound) != -100 || c.maxDepth != 7 || c.maxNodes != 9 || c.maxMemory != 3*1024*1024 {
		t.Fatal("source update prefix after exponent failure", value(c.epsilon), value(c.maxBound), value(c.minusMaxBound), c.maxDepth, c.maxNodes, c.maxMemory)
	}
	if err := pc.Reset(&p); err != nil {
		t.Fatal(err)
	}
	if err := w.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal(err)
	}
	if c.zeroEpsilon || value(c.maxBound) != 1e10 || value(c.rootPrecision) != 1.0/8192 || c.maxDepth != 128 || c.maxNodes != 8192 || c.maxMemory != math.MaxUint64 {
		t.Fatal("source defaults")
	}
	set("epsilon", 0)
	set("max_bound", 0)
	set("nth_root_precision", 0)
	if err := c.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal(err)
	}
	if !c.zeroEpsilon || value(c.maxBound) != 1 || value(c.rootPrecision) != 1 {
		t.Fatal("zero parameter branches")
	}
	set("epsilon", 4)
	set("max_bound", 2)
	set("max_depth", 7)
	set("max_nodes", 9)
	set("max_memory", 3)
	set("nth_root_precision", 16)
	fresh, err := NewArbitraryContextFromParams(f.numbers, &control.Reslimit{Limit: 100}, pc, &p)
	if err != nil {
		t.Fatal("single-pass factory", err)
	}
	if fresh.zeroEpsilon || value(fresh.epsilon) != .25 || value(fresh.maxBound) != 100 || fresh.maxDepth != 7 || fresh.maxNodes != 9 || fresh.maxMemory != 3*1024*1024 {
		t.Fatal("factory policy")
	}
	if err := fresh.Close(); err != nil {
		t.Fatal(err)
	}
	set("epsilon", 2)
	set("max_bound", 1<<31)
	partial, err := NewArbitraryContextFromParams(f.numbers, &control.Reslimit{Limit: 100}, pc, &p)
	if err == nil || partial == nil {
		t.Fatal("refused factory owner", partial, err)
	}
	if value(partial.epsilon) != .5 || value(partial.maxBound) != 1 || partial.maxDepth != 0 || partial.maxNodes != 0 {
		t.Fatal("single pass stopped at power", partial.maxDepth, partial.maxNodes)
	}
	if err := partial.Close(); err != nil {
		t.Fatal("partial factory cleanup", err)
	}
	if missing, err := NewArbitraryContextFromParams(nil, &control.Reslimit{Limit: 100}, pc, &p); missing != nil || !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("missing factory manager", missing, err)
	}
	if err := w.CollectParamDescrs(d); err != nil {
		t.Fatal(err)
	}
	if size, err := d.Size(); err != nil || size != 5 {
		t.Fatal("source descriptor count", size, err)
	}
	if err := w.UpdateParamsRef(nil, &p); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("missing params context", err)
	}
	if err := CollectArbitraryParamDescrs(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("missing descriptions", err)
	}
}
