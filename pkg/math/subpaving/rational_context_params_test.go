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

func TestRationalContextNamedParamsSinglePassAndDefaults(t *testing.T) {
	_, m, _, _, _ := rationalClauseFixture(t)
	b := arena.NewBudget(1 << 25)
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
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	d, err := params.NewParamDescrs(sy, params.ParamDescrsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{d.Close(), pc.Del(&p), pc.Close(), q.Close(), sy.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("parameter owners retained budget", b.Used())
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
	c, err := NewRationalContextFromParams(m, &control.Reslimit{Limit: 100}, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	if c.zeroEpsilon || c.maxDepth != 7 || c.maxNodes != 9 || c.maxMemory != 3*1024*1024 {
		t.Fatal("named scalar policy", c.maxDepth, c.maxNodes, c.maxMemory)
	}
	want := numeral.NewRational()
	t.Cleanup(func() {
		if err := m.Del(&want); err != nil {
			t.Error(err)
		}
	})
	check := func(got *numeral.Rational, n int64, den uint64) {
		t.Helper()
		if err := m.SetFraction(&want, n, den); err != nil {
			t.Fatal(err)
		}
		equal, err := m.Equal(got, &want)
		if err != nil || !equal {
			t.Fatal("named numeral", n, den, equal, err)
		}
	}
	check(&c.epsilon, 1, 4)
	check(&c.maxBound, 100, 1)
	check(&c.minusMaxBound, -100, 1)
	check(&c.rootPrecision, 1, 16)
	if err := CollectRationalParamDescrs(d); err != nil {
		t.Fatal(err)
	}
	if size, err := d.Size(); err != nil || size != 5 {
		t.Fatal("source descriptor count", size, err)
	}
	if err := pc.Reset(&p); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal(err)
	}
	if c.zeroEpsilon || c.maxDepth != 128 || c.maxNodes != 8192 || c.maxMemory != math.MaxUint64 {
		t.Fatal("default scalar policy", c.maxDepth, c.maxNodes, c.maxMemory)
	}
	check(&c.epsilon, 1, 20)
	check(&c.maxBound, 10_000_000_000, 1)
	check(&c.rootPrecision, 1, 8192)
	set("epsilon", 0)
	set("max_bound", 0)
	set("nth_root_precision", 0)
	if err := c.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal(err)
	}
	if !c.zeroEpsilon {
		t.Fatal("zero epsilon")
	}
	check(&c.maxBound, 1, 1)
	check(&c.rootPrecision, 1, 1)
	partial, err := NewRationalContextFromParams(m, &control.Reslimit{Limit: 100}, nil, &p)
	if !errors.Is(err, arena.ErrInvalidSlot) || partial == nil || partial.maxDepth != 0 || partial.maxNodes != 0 || partial.maxMemory != 0 {
		t.Fatal("refused named pass must return its partial owner", partial, err)
	}
	if err := partial.Close(); err != nil {
		t.Fatal(err)
	}
}
