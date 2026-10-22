package solver

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func solverMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func solverTestManager(t *testing.T) *ast.ASTManager {
	t.Helper()
	budget := arena.NewBudget(1 << 30)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	solverMust(t, err)
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	solverMust(t, err)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	solverMust(t, err)
	t.Cleanup(func() {
		solverMust(t, manager.Close())
		solverMust(t, numerals.Close())
		solverMust(t, symbols.Close())
		if budget.Used() != 0 {
			t.Errorf("solver fixture retained %d bytes", budget.Used())
		}
	})
	return manager
}

func TestSimpleCheckSatResultStatusCoreProofModelAndStatistics(t *testing.T) {
	m := solverTestManager(t)
	r, err := NewSimpleCheckSatResult(m)
	solverMust(t, err)
	t.Cleanup(func() { solverMust(t, r.Close()) })
	if status, err := r.Status(); err != nil || status != lbool.Undef {
		t.Fatal("initial status", status, err)
	}
	boolSort, err := m.MkBoolSort()
	solverMust(t, err)
	name, err := m.Symbols.Intern("solver-core")
	solverMust(t, err)
	atom, err := m.MkConst(name, boolSort)
	solverMust(t, err)
	proof, err := m.MkFalse()
	solverMust(t, err)
	solverMust(t, r.SetUnsatCore([]ast.ASTHandle{atom}))
	solverMust(t, r.SetProof(proof))
	if core, err := r.UnsatCore(); err != nil || len(core) != 0 {
		t.Fatal("non-unsat core must be hidden", core, err)
	}
	solverMust(t, r.SetStatus(lbool.False))
	if core, err := r.UnsatCore(); err != nil || len(core) != 1 || core[0] != atom {
		t.Fatal("unsat core", core, err)
	}
	if got, err := r.Proof(); err != nil || got != proof {
		t.Fatal("unsat proof", got, err)
	}

	md, err := model.NewModel(m)
	solverMust(t, err)
	solverMust(t, r.SetModel(md.Core()))
	if got, err := r.AcquireModel(); err != nil || got != nil {
		t.Fatal("unsat result exposed model", got, err)
	}
	solverMust(t, r.SetStatus(lbool.True))
	got, err := r.AcquireModel()
	solverMust(t, err)
	if got != md.Core() || got.References != 2 {
		t.Fatal("acquired model reference", got, md.Core().References)
	}
	solverMust(t, got.DecRef())

	stats, err := r.Statistics()
	solverMust(t, err)
	stats.UpdateUint("checks", 3)
	out := statistics.New()
	solverMust(t, r.CollectStatistics(out))
	if out.Size() != 1 || out.Key(0) != "checks" || out.UintValue(0) != 3 {
		t.Fatal("statistics copy")
	}
}

func TestSimpleCheckSatResultReasonsLabelsAndReferenceLifetime(t *testing.T) {
	m := solverTestManager(t)
	r, err := NewSimpleCheckSatResult(m)
	solverMust(t, err)
	for caller, want := range map[control.EventCaller]string{
		control.EventCallerControlC:      "interrupted from keyboard",
		control.EventCallerTimeout:       "timeout",
		control.EventCallerResourceLimit: "max. resource limit exceeded",
		control.EventCallerAPIInterrupt:  "interrupted",
	} {
		solverMust(t, r.SetReasonUnknownFromCaller(caller))
		if got, err := r.ReasonUnknown(); err != nil || got != want {
			t.Fatal(caller, got, err)
		}
	}
	solverMust(t, r.SetReasonUnknown("keep"))
	solverMust(t, r.SetReasonUnknownFromCaller(control.EventCallerUnset))
	if got, _ := r.ReasonUnknown(); got != "keep" {
		t.Fatal("unset caller changed reason", got)
	}
	a, err := m.Symbols.Intern("a")
	solverMust(t, err)
	b, err := m.Symbols.Intern("b")
	solverMust(t, err)
	labels := []usymbol.Symbol{a, b}
	solverMust(t, r.SetLabels(labels))
	got, err := r.Labels()
	solverMust(t, err)
	got[0] = b
	again, err := r.Labels()
	solverMust(t, err)
	if len(again) != 2 || again[0] != a || again[1] != b {
		t.Fatal("labels were not copied", again)
	}
	solverMust(t, r.IncRef())
	solverMust(t, r.DecRef())
	if _, err := r.Status(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("final result decrement did not close", err)
	}
}

type recordingProgressCallback struct{ fast, slow uint32 }

func (c *recordingProgressCallback) FastProgressSample() { c.fast++ }
func (c *recordingProgressCallback) SlowProgressSample() { c.slow++ }

func TestProgressCallbackSurface(t *testing.T) {
	var callback ProgressCallback = &recordingProgressCallback{}
	callback.FastProgressSample()
	callback.FastProgressSample()
	callback.SlowProgressSample()
	got := callback.(*recordingProgressCallback)
	if got.fast != 2 || got.slow != 1 {
		t.Fatal("progress callback dispatch", *got)
	}
}
