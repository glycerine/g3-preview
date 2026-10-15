package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestFixedEngineSelectedOwnerUpdateAndRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, err := numeral.NewFixedPointManager(2, 1)
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
		for _, e := range []error{pc.Del(&p), pc.Close(), q.Close(), sy.Close(), qm.CloseChecked(), m.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("MPFX tactic owners", b.Used())
		}
	})
	limit := &control.Reslimit{Limit: 1000}
	if e, err := NewFixedEngine(limit, m, qm, pc, &p); e != nil || !errors.Is(err, ErrFixedEngineSelection) {
		t.Fatal("default MPQ selection", e, err)
	}
	name, key := "mpfx", "numeral"
	selected, err := sy.Intern(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.SetSymbolName(&p, &key, selected); err != nil {
		t.Fatal(err)
	}
	printKey, boundKey, depthKey := "print_nodes", "max_bound", "max_depth"
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &boundKey, 2); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &depthKey, 0); err != nil {
		t.Fatal(err)
	}
	e, err := NewFixedEngine(limit, m, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	options, err := e.Options()
	if err != nil || options.Engine != EngineMPFX || !options.PrintNodes {
		t.Fatal("selected MPFX options", options, err)
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
	if err := e.Search(); err != nil {
		t.Fatal("selected MPFX search", err)
	}
	st := statistics.New()
	if err := shell.CollectStatistics(st); err != nil || st.Size() != 2 || st.Key(0) != "nodes" || st.UintValue(0) != 1 || st.Key(1) != "visited" || st.UintValue(1) != 1 {
		t.Fatal("selected MPFX statistics", err)
	}
	if err := shell.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := shell.CollectStatistics(st); err != nil || st.Size() != 1 || st.Key(0) != "nodes" {
		t.Fatal("MPFX statistic reset", err)
	}
	if err := pc.SetBoolName(&p, &printKey, false); err != nil {
		t.Fatal(err)
	}
	if err := e.UpdateParamsRef(pc, &p); err != nil {
		t.Fatal("selected parameter update", err)
	}
	options, err = e.Options()
	if err != nil || options.PrintNodes {
		t.Fatal("updated display option", options, err)
	}
	if err := e.Close(); err != nil {
		t.Fatal("engine close", err)
	}
	if _, err := e.Shell(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed shell borrow", err)
	}
	if err := pc.SetUIntName(&p, &boundKey, 1<<31); err != nil {
		t.Fatal(err)
	}
	if failed, err := NewFixedEngine(limit, m, qm, pc, &p); failed != nil || err == nil {
		t.Fatal("refused selected context", failed, err)
	}
}
