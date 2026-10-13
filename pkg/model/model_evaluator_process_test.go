// Source: src/model/model_evaluator.cpp:282-290; src/util/memory_manager.cpp:162-171.
package model_test

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/cooperate"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/messages"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestModelEvaluatorSharedProcessMemoryAndCohort(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	process := umemory.NewProcessAccounting()
	numericRuntime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: process})
	astTestMust(t, e)
	integers := unumeral.IntegerConfig{Memory: numericRuntime, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	numerals, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: integers})
	astTestMust(t, e)
	symbols, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	ctx, e := uparams.NewParamsContext(symbols, numerals, uparams.ParamsConfig{Budget: b})
	astTestMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{Descriptors: arena.Config{Budget: b}}, Text: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	astTestMust(t, e)
	globals, e := uparams.NewGlobalParamsService(symbols, ctx, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog})
	astTestMust(t, e)
	astTestMust(t, globals.InitializeSource())
	params, e := ctx.NewValue()
	astTestMust(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: b})
	astTestMust(t, e)
	md, e := model.NewModel(m)
	astTestMust(t, e)
	memory, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: process})
	astTestMust(t, e)
	cooperation, e := cooperate.NewCooperation(b)
	astTestMust(t, e)
	config := model.ModelEvaluatorConfig{Context: ctx, Globals: globals, Params: &params, Memory: memory, Cooperation: cooperation}
	wrong, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: umemory.NewProcessAccounting()})
	astTestMust(t, e)
	config.Memory = wrong
	before := b.Used()
	if refused, err := model.NewModelEvaluator(md.Core(), config); refused != nil || !errors.Is(err, uparams.ErrParamsOwner) || b.Used() != before {
		t.Fatal("mixed evaluator process accepted", refused, err, b.Used(), before)
	}
	astTestMust(t, wrong.Close())
	config.Memory = memory
	foreignRuntime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: umemory.NewProcessAccounting()})
	astTestMust(t, e)
	foreignIntegers := unumeral.IntegerConfig{Memory: foreignRuntime, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	foreignNumerals, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: foreignIntegers})
	astTestMust(t, e)
	foreignAST, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: foreignNumerals, Budget: b})
	astTestMust(t, e)
	foreignModel, e := model.NewModel(foreignAST)
	astTestMust(t, e)
	before = b.Used()
	if refused, err := model.NewModelEvaluator(foreignModel.Core(), config); refused != nil || !errors.Is(err, uparams.ErrParamsOwner) || b.Used() != before {
		t.Fatal("mixed AST numeral process accepted", refused, err, b.Used(), before)
	}
	astTestMust(t, foreignModel.Close())
	astTestMust(t, foreignAST.Close())
	astTestMust(t, foreignNumerals.Close())
	astTestMust(t, foreignRuntime.Close())
	ev, e := model.NewModelEvaluator(md.Core(), config)
	astTestMust(t, e)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	sortRef, e := m.NewRef(sort)
	astTestMust(t, e)
	name, e := symbols.Intern("model_process_atom")
	astTestMust(t, e)
	atom, e := m.MkConst(name, sort)
	astTestMust(t, e)
	atomRef, e := m.NewRef(atom)
	astTestMust(t, e)
	term, e := m.MkNot(atom)
	astTestMust(t, e)
	termRef, e := m.NewRef(term)
	astTestMust(t, e)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	maxMB := uint32((process.AllocationSize()+(1<<20)-1)/(1<<20) + 1)
	key := "max_memory"
	astTestMust(t, ctx.SetUIntName(&params, &key, maxMB))
	astTestMust(t, ev.UpdtParams(&params))
	if process.AllocationSize() > uint64(maxMB)*(1<<20) {
		t.Fatal("evaluator already exceeds configured source limit", process.AllocationSize(), maxMB)
	}
	other, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: process})
	astTestMust(t, e)
	block, e := other.Allocate(2 << 20)
	astTestMust(t, e)
	if process.AllocationSize() <= uint64(maxMB)*(1<<20) {
		t.Fatal("other runtime did not cross evaluator limit", process.AllocationSize(), maxMB)
	}
	if e = ev.Evaluate(term, out); e == nil || e.Error() != messages.MaxMemory {
		t.Fatal("retained model evaluator process checkpoint", e)
	}
	astTestMust(t, other.Deallocate(&block))
	astTestMust(t, other.Close())
	astTestMust(t, ev.Close())
	astTestMust(t, cooperation.Close())
	astTestMust(t, memory.Close())
	astTestMust(t, ctx.Del(&params))
	astTestMust(t, globals.Close())
	astTestMust(t, catalog.Close())
	astTestMust(t, md.Close())
	astTestMust(t, m.DelRef(&out))
	astTestMust(t, m.DelRef(&termRef))
	astTestMust(t, m.DelRef(&atomRef))
	astTestMust(t, m.DelRef(&sortRef))
	astTestMust(t, m.Close())
	astTestMust(t, ctx.Close())
	astTestMust(t, numerals.Close())
	astTestMust(t, symbols.Close())
	astTestMust(t, numericRuntime.Close())
	if process.AllocationSize() != 0 || b.Used() != 0 {
		t.Fatal("retained model evaluator process owners", process.AllocationSize(), b.Used())
	}
}
