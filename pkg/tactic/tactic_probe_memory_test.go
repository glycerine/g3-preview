package tactic

// Source: src/tactic/probe.cpp:29-37; src/util/memory_manager.cpp:162-171.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestMemoryProbeReadsSharedProcessAllocation(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	b := arena.NewBudget(1 << 25)
	process := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: process})
	must(err)
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: process})
	must(err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: r, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	must(err)
	s, err := symbol.NewManager(symbol.Config{Budget: b})
	must(err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	must(err)
	g, err := NewGoal(m, false, false, false)
	must(err)
	p, err := NewProbe(b, ProbeMemory, 999)
	must(err)
	t.Cleanup(func() {
		must(p.Close())
		must(g.Close())
		must(m.Close())
		must(s.Close())
		must(q.Close())
		must(other.Close())
		must(r.Close())
		if b.Used() != 0 || process.AllocationSize() != 0 {
			t.Fatal("memory probe retained process owners", b.Used(), process.AllocationSize())
		}
	})
	check := func() {
		t.Helper()
		got, err := p.Eval(g)
		must(err)
		want := float64(process.AllocationSize()) / float64(1024*1024)
		if got != want {
			t.Fatal("source memory probe", got, want)
		}
	}
	check()
	block, err := other.Allocate(2 << 20)
	must(err)
	defer func() {
		if block != (memory.Block{}) {
			must(other.Deallocate(&block))
		}
	}()
	check()
	must(other.Deallocate(&block))
	check()
}
