package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestFastEngineSelectedOwnerAndPartialCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	cfg.InitialCapacity = 4
	numbers, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
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
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{pc.Del(&p), pc.Close(), q.Close(), sy.Close(), qm.CloseChecked(), numbers.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("retained MPFF tactic owners", b.Used())
		}
	})
	limit := &control.Reslimit{Limit: 1000}
	if e, err := NewFastEngine(limit, numbers, qm, pc, &p); e != nil || !errors.Is(err, ErrFastEngineSelection) {
		t.Fatal("default MPQ selection", e, err)
	}
	selected, err := sy.Intern("mpff")
	if err != nil {
		t.Fatal(err)
	}
	engineKey, depthKey, printKey := "numeral", "max_depth", "print_nodes"
	if err := pc.SetSymbolName(&p, &engineKey, selected); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &depthKey, 0); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	e, err := NewFastEngine(limit, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	options, err := e.Options()
	if err != nil || options.Engine != EngineMPFF || !options.PrintNodes {
		t.Fatal("selected options", options, err)
	}
	shell, err := e.Shell()
	if err != nil {
		t.Fatal(err)
	}
	x, err := shell.MkVar(true)
	if err != nil || x != 0 {
		t.Fatal("selected variable", x, err)
	}
	one := numeral.NewInt(1)
	if _, err := shell.MkSum(&one, []*numeral.Int{&one}, []uint32{x}); err != nil {
		t.Fatal("selected sum", err)
	}
	if err := e.Close(); err != nil {
		t.Fatal("engine close", err)
	}
	if _, err := e.Shell(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed shell borrow", err)
	}
	var stillBorrowed numeral.FastFloat
	if err := numbers.SetInt32(&stillBorrowed, 1); err != nil {
		t.Fatal("closed borrowed manager", err)
	}
	if err := numbers.Del(&stillBorrowed); err != nil {
		t.Fatal(err)
	}
	maxBound := "max_bound"
	if err := pc.SetUIntName(&p, &maxBound, 1<<31); err != nil {
		t.Fatal(err)
	}
	partial, err := NewFastEngine(limit, numbers, qm, pc, &p)
	if err == nil || partial == nil {
		t.Fatal("refused selected context", partial, err)
	}
	if err := partial.Close(); err != nil {
		t.Fatal("partial engine cleanup", err)
	}
}
