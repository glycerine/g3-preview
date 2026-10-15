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

func TestFixedCoreTacticRationalProcessBindingAndRefusedOwners(t *testing.T) {
	b := arena.NewBudget(1 << 25)
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
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	selected, err := sy.Intern("mpfx")
	if err != nil {
		t.Fatal(err)
	}
	key := "numeral"
	if err := pc.SetSymbolName(&p, &key, selected); err != nil {
		t.Fatal(err)
	}
	numbers, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	core, err := NewFixedCoreTactic(m, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	baselineBytes, baselineProcess := b.Used(), process.AllocationSize()
	owned, err := NewFixedCoreTacticOwned(m, pc, &p)
	if err != nil || owned == nil || !owned.ownsNumerals || owned.rational.RuntimeOwner().ProcessOwner() != process {
		t.Fatal("owned rational manager process", owned, err)
	}
	if err := owned.Close(); err != nil || b.Used() != baselineBytes || process.AllocationSize() != baselineProcess {
		t.Fatal("owned MPFX process cleanup", err, b.Used(), baselineBytes, process.AllocationSize(), baselineProcess)
	}
	other, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := params.NewParamsContext(sy, other, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	foreignValue, err := foreign.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	before := b.Used()
	if rejected, err := NewFixedCoreTactic(m, numbers, qm, foreign, &foreignValue); rejected != nil || !errors.Is(err, arena.ErrConfig) || b.Used() != before {
		t.Fatal("foreign parameter process", rejected, err)
	}
	if rejected, err := NewFixedCoreTacticOwned(m, foreign, &foreignValue); rejected != nil || !errors.Is(err, arena.ErrConfig) || b.Used() != before {
		t.Fatal("owned foreign parameter process", rejected, err)
	}
	wrong, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	before, beforeProcess := b.Used(), process.AllocationSize()
	if rejected, err := NewFixedCoreTacticOwned(m, pc, &wrong); rejected != nil || !errors.Is(err, ErrFixedEngineSelection) || b.Used() != before || process.AllocationSize() != beforeProcess {
		t.Fatal("wrong selected engine leaked owners", rejected, err)
	}
	for _, e := range []error{pc.Del(&wrong), foreign.Del(&foreignValue), foreign.Close(), other.Close(), pc.Del(&p), pc.Close(), m.Close(), qm.CloseChecked(), q.Close(), sy.Close(), numbers.Close()} {
		if e != nil {
			t.Fatal(e)
		}
	}
	if b.Used() != 0 {
		t.Fatal("source process owner leak", b.Used())
	}
}
