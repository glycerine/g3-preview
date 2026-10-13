// Source: src/sat/tactic/goal2sat.cpp:380-391; src/util/memory_manager.cpp:162-171.
package tactic

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSATGoalTranslatorSharedProcessMemoryCheckpoint(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	process := umemory.NewProcessAccounting()
	numericRuntime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: process})
	satOK(t, e)
	integers := unumeral.IntegerConfig{Memory: numericRuntime, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	numerals, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: integers})
	satOK(t, e)
	symbols, e := usymbol.NewManager(usymbol.Config{Budget: b})
	satOK(t, e)
	ctx, e := uparams.NewParamsContext(symbols, numerals, uparams.ParamsConfig{Budget: b})
	satOK(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: b})
	satOK(t, e)
	runtime, e := sat.NewRuntime(sat.RuntimeConfig{Budget: b, ParamsContext: ctx, Process: process})
	satOK(t, e)
	params, e := ctx.NewValue()
	satOK(t, e)
	limit, e := control.NewReslimit(b)
	satOK(t, e)
	solver, e := sat.NewSATSolver(sat.SolverConfig{ParamsContext: ctx, Params: params, RLimit: limit, Budget: b, Runtime: runtime})
	satOK(t, e)

	foreign := astTestManager(t)
	beforeBudget, foreignBudget := b.Used(), foreign.Budget().Used()
	if refused, err := NewGoalTranslator(foreign, solver); refused != nil || !errors.Is(err, sat.ErrSATOwner) || b.Used() != beforeBudget || foreign.Budget().Used() != foreignBudget {
		t.Fatal("mixed AST/SAT process accepted", refused, err, b.Used(), beforeBudget, foreign.Budget().Used(), foreignBudget)
	}
	sort, e := m.MkBoolSort()
	satOK(t, e)
	sortRef, e := m.NewRef(sort)
	satOK(t, e)
	name, e := symbols.Intern("shared_process_atom")
	satOK(t, e)
	atom, e := m.MkConst(name, sort)
	satOK(t, e)
	atomRef, e := m.NewRef(atom)
	satOK(t, e)
	negated, e := m.MkNot(atom)
	satOK(t, e)
	negatedRef, e := m.NewRef(negated)
	satOK(t, e)
	maxMB := uint32((process.AllocationSize()+(1<<20)-1)/(1<<20) + 1)
	key := "max_memory"
	satOK(t, ctx.SetUIntName(&params, &key, maxMB))
	satOK(t, solver.UpdateParams(&params))
	translator, e := NewGoalTranslator(m, solver)
	satOK(t, e)
	if process.AllocationSize() > translator.maxMemory {
		t.Fatal("source process already above configured limit", process.AllocationSize(), translator.maxMemory)
	}
	other, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: process})
	satOK(t, e)
	block, e := other.Allocate(2 << 20)
	satOK(t, e)
	if process.AllocationSize() <= translator.maxMemory {
		t.Fatal("other runtime did not cross SAT limit", process.AllocationSize(), translator.maxMemory)
	}
	if e = translator.Compile([]ast.ASTHandle{negated}); e == nil || !strings.Contains(e.Error(), "max. memory exceeded") {
		t.Fatal("retained goal2sat process checkpoint", e)
	}
	satOK(t, translator.Close())
	satOK(t, other.Deallocate(&block))
	satOK(t, other.Close())
	satOK(t, solver.Close())
	limit.Close()
	satOK(t, ctx.Del(&params))
	satOK(t, runtime.Close())
	satOK(t, m.DelRef(&negatedRef))
	satOK(t, m.DelRef(&atomRef))
	satOK(t, m.DelRef(&sortRef))
	satOK(t, m.Close())
	satOK(t, ctx.Close())
	satOK(t, numerals.Close())
	satOK(t, symbols.Close())
	satOK(t, numericRuntime.Close())
	if process.AllocationSize() != 0 || b.Used() != 0 {
		t.Fatal("retained goal2sat process owners", process.AllocationSize(), b.Used())
	}
}
