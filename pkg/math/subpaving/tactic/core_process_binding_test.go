package tactic

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestFastCoreTacticNumericProcessCohort(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	process := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: process, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	integers := numeral.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: integers})
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: integers})
	if err != nil {
		t.Fatal(err)
	}
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	if m.NumeralService() != q {
		t.Fatal("AST manager lost borrowed numerical owner")
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	value, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	fastConfig := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	fastConfig.InitialCapacity, fastConfig.Memory = 4, r
	numbers, err := numeral.NewFastFloatManager(fastConfig)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := sy.Intern("mpff")
	if err != nil {
		t.Fatal(err)
	}
	key := "numeral"
	if err := pc.SetSymbolName(&value, &key, selected); err != nil {
		t.Fatal(err)
	}
	core, err := NewFastCoreTactic(m, numbers, qm, pc, &value)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}

	isolated, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	otherParams, err := params.NewParamsContext(sy, isolated, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	otherValue, err := otherParams.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	before := b.Used()
	if rejected, err := NewFastCoreTactic(m, numbers, qm, otherParams, &otherValue); rejected != nil || !errors.Is(err, arena.ErrConfig) || b.Used() != before {
		t.Fatal("unbound parameter numeral service must be refused before snapshot", rejected, err, b.Used(), before)
	}
	for _, err := range []error{otherParams.Del(&otherValue), otherParams.Close(), isolated.Close(), pc.Del(&value), pc.Close(), m.Close(), qm.CloseChecked(), q.Close(), sy.Close(), numbers.CloseChecked(), r.Close()} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if b.Used() != 0 {
		t.Fatal("retained core process cohort", b.Used())
	}
}
