package subpaving

import (
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestFixedContextNamedParamsSourceOrderAndStoppedOwner(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	b := arena.NewBudget(1 << 24)
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
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{d.Close(), pc.Del(&p), pc.Close(), q.Close(), qm.CloseChecked(), sy.Close(), m.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("parameter owner leak", b.Used())
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
	c, err := NewFixedContextFromParams(m, &control.Reslimit{Limit: 1000}, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	check := func(v *numeral.FixedPoint, want *big.Rat) {
		t.Helper()
		got, err := m.Rational(v)
		if err != nil || got.Cmp(want) != 0 {
			t.Fatal("fixed policy value", got, want, err)
		}
	}
	if c.zeroEpsilon || c.maxDepth != 7 || c.maxNodes != 9 || c.maxMemory != 3*1024*1024 || c.root == nil {
		t.Fatal("named policy fields")
	}
	check(c.epsilon, big.NewRat(1, 4))
	check(c.maxBound, big.NewRat(100, 1))
	check(c.minusMaxBound, big.NewRat(-100, 1))
	check(c.rootPrecision, big.NewRat(1, 16))
	if err := CollectFixedParamDescrs(d); err != nil {
		t.Fatal(err)
	}
	if size, err := d.Size(); err != nil || size != 5 {
		t.Fatal("source descriptors", size, err)
	}
	set("epsilon", 2)
	set("max_bound", 1<<31)
	partial, err := NewFixedContextFromParams(m, &control.Reslimit{Limit: 1000}, pc, &p)
	if err == nil || partial == nil {
		t.Fatal("stopped constructor", partial, err)
	}
	if partial.root != nil || partial.maxDepth != 0 || partial.maxNodes != 0 || partial.maxMemory != 0 {
		t.Fatal("later fields changed before power failure")
	}
	check(partial.epsilon, big.NewRat(1, 2))
	if err := partial.Close(); err != nil {
		t.Fatal("partial owner close", err)
	}
	if err := pc.Reset(&p); err != nil {
		t.Fatal(err)
	}
	set("epsilon", 0)
	set("max_bound", 0)
	set("nth_root_precision", 0)
	if err := c.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal(err)
	}
	if !c.zeroEpsilon || c.maxDepth != 128 || c.maxNodes != 8192 || c.maxMemory != math.MaxUint64 {
		t.Fatal("zero/default parameter fields")
	}
	check(c.maxBound, big.NewRat(1, 1))
	check(c.rootPrecision, big.NewRat(1, 1))
	w, err := NewFixedShellFromParams(m, &control.Reslimit{Limit: 1000}, qm, pc, &p)
	if err != nil {
		t.Fatal("owning MPFX wrapper", err)
	}
	if _, err := w.MkVar(true); err != nil {
		t.Fatal("factory wrapper variable", err)
	}
	if err := w.Close(); err != nil || !w.context.closed {
		t.Fatal("wrapper and context cleanup", err)
	}
	set("epsilon", 2)
	set("max_bound", 1<<31)
	if failed, err := NewFixedShellFromParams(m, &control.Reslimit{Limit: 1000}, qm, pc, &p); err == nil || failed != nil {
		t.Fatal("failed base constructor cleanup", failed, err)
	}
}
