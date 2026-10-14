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
	integers := numeral.IntegerConfig{Process: process, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
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
	fastConfig.InitialCapacity, fastConfig.Process = 4, process
	numbers, err := numeral.NewFastFloatManager(fastConfig)
	if err != nil {
		t.Fatal(err)
	}
	if q.RuntimeOwner() == qm.RuntimeOwner() || q.RuntimeOwner() == numbers.RuntimeOwner() || qm.RuntimeOwner() == numbers.RuntimeOwner() {
		t.Fatal("tactic owners unexpectedly share one raw runtime")
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
	baselineBytes, baselineProcess := b.Used(), process.AllocationSize()
	owned, err := NewFastCoreTacticOwned(m, pc, &value)
	if err != nil {
		t.Fatal(err)
	}
	if !owned.ownsNumerals || owned.numbers.RuntimeOwner() == q.RuntimeOwner() || owned.rational.RuntimeOwner() == q.RuntimeOwner() || owned.numbers.RuntimeOwner().ProcessOwner() != process || owned.rational.RuntimeOwner().ProcessOwner() != process {
		t.Fatal("owned tactic numeral managers did not join the source process")
	}
	if err := owned.Close(); err != nil || b.Used() != baselineBytes || process.AllocationSize() != baselineProcess {
		t.Fatal("owned tactic did not release its numeral members", err, b.Used(), baselineBytes, process.AllocationSize(), baselineProcess)
	}
	nonFast, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	mpq, err := sy.Intern("mpq")
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.SetSymbolName(&nonFast, &key, mpq); err != nil {
		t.Fatal(err)
	}
	beforeReject, processBeforeReject := b.Used(), process.AllocationSize()
	if rejected, err := NewFastCoreTacticOwned(m, pc, &nonFast); rejected != nil || !errors.Is(err, ErrFastEngineSelection) || b.Used() != beforeReject || process.AllocationSize() != processBeforeReject {
		t.Fatal("owned tactic selection refusal retained numeral members", rejected, err, b.Used(), beforeReject, process.AllocationSize(), processBeforeReject)
	}
	if err := pc.Del(&nonFast); err != nil {
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
	if rejected, err := NewFastCoreTacticOwned(m, otherParams, &otherValue); rejected != nil || !errors.Is(err, arena.ErrConfig) || b.Used() != before {
		t.Fatal("owned tactic must refuse a foreign parameter process before construction", rejected, err, b.Used(), before)
	}
	for _, err := range []error{otherParams.Del(&otherValue), otherParams.Close(), isolated.Close(), pc.Del(&value), pc.Close(), m.Close(), qm.CloseChecked(), q.Close(), sy.Close(), numbers.CloseChecked()} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if b.Used() != 0 {
		t.Fatal("retained core process cohort", b.Used())
	}
}
