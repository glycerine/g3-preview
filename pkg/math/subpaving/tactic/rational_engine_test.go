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

func TestRationalEngineSelectionSearchAndCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	integers := numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: integers})
	if err != nil {
		t.Fatal(err)
	}
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: integers})
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
		for _, err := range []error{pc.Del(&p), pc.Close(), q.Close(), sy.Close(), qm.CloseChecked()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("rational engine owner roots", b.Used())
		}
	})
	limit := &control.Reslimit{Limit: 1000}
	depth := "max_depth"
	if err := pc.SetUIntName(&p, &depth, 0); err != nil {
		t.Fatal(err)
	}
	e, err := NewRationalEngine(limit, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	options, err := e.Options()
	if err != nil || options.Engine != EngineMPQ {
		t.Fatal("default selected engine", options, err)
	}
	ctx, err := e.Context()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ctx.NewVariable(false); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Search(); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Context(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context borrow", err)
	}
	var borrowed numeral.Rational
	if err := qm.SetInt32(&borrowed, 1); err != nil {
		t.Fatal("borrowed manager closed", err)
	}
	if err := qm.Del(&borrowed); err != nil {
		t.Fatal(err)
	}

	name, err := sy.Intern("mpf")
	if err != nil {
		t.Fatal(err)
	}
	selector := "numeral"
	if err := pc.SetSymbolName(&p, &selector, name); err != nil {
		t.Fatal(err)
	}
	if got, err := NewRationalEngine(limit, qm, pc, &p); got != nil || !errors.Is(err, ErrRationalEngineSelection) {
		t.Fatal("wrong engine selection", got, err)
	}
}
