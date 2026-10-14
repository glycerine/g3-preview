// Source: src/tactic/tactic.cpp:26-61; src/tactic/tactic.h:108.
package tactic

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/text"
)

func TestTacticReportReadsProcessMemoryAndWritesSourceOrder(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	b := arena.NewBudget(1 << 25)
	process := memory.NewProcessAccounting()
	runtime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: process})
	must(err)
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: process})
	must(err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: runtime, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	must(err)
	s, err := symbol.NewManager(symbol.Config{Budget: b})
	must(err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	must(err)
	g, err := NewGoal(m, false, false, false)
	must(err)
	out, err := text.NewNumericTextStream(text.NumericTextStreamConfig{Budget: b})
	must(err)
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	must(err)
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	must(err)
	must(diagnostic.SetProcessVerboseStream(out))
	t.Cleanup(func() {
		must(diagnostic.SetProcessVerboseStream(oldStream))
		must(diagnostic.SetProcessVerbosityLevel(oldLevel))
		must(out.Close())
		must(g.Close())
		must(m.Close())
		must(s.Close())
		must(q.Close())
		must(other.Close())
		must(runtime.Close())
		if b.Used() != 0 || process.AllocationSize() != 0 {
			t.Fatal("report retained process owners", b.Used(), process.AllocationSize())
		}
	})
	must(diagnostic.SetProcessVerbosityLevel(9))
	if report, err := newTacticReport("max-bv-sharing", g, nil); err != nil || report != nil {
		t.Fatal("disabled source report must erase construction", report, err)
	}
	must(diagnostic.SetProcessVerbosityLevel(10))
	startBytes := process.AllocationSize()
	clockReads := 0
	clock := func() (uint64, error) {
		clockReads++
		if clockReads == 1 {
			return 1_000_000_000, nil
		}
		return 1_250_000_000, nil
	}
	report, err := newTacticReport("max-bv-sharing", g, clock)
	must(err)
	block, err := other.Allocate(2 << 20)
	must(err)
	defer func() {
		if block != (memory.Block{}) {
			must(other.Deallocate(&block))
		}
	}()
	endBytes := process.AllocationSize()
	nodes := m.Stats().LiveNodes
	must(report.Close())
	must(report.Close())
	got, err := out.String()
	must(err)
	want := fmt.Sprintf("(max-bv-sharing :num-exprs 0 :num-asts %d :time 0.25 :before-memory %.2f :after-memory %.2f)\n", nodes, float64(startBytes)/float64(1024*1024), float64(endBytes)/float64(1024*1024))
	if got != want || clockReads != 2 {
		t.Fatal("source tactic report", got, want, clockReads)
	}
	must(other.Deallocate(&block))
}

func TestTacticProgressPositiveVerboseGateAndSourceFormat(t *testing.T) {
	b := arena.NewBudget(1024)
	out, err := text.NewNumericTextStream(text.NumericTextStreamConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	if err != nil {
		t.Fatal(err)
	}
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	if err != nil {
		t.Fatal(err)
	}
	if err := diagnostic.SetProcessVerboseStream(out); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, err := range []error{diagnostic.SetProcessVerboseStream(oldStream), diagnostic.SetProcessVerbosityLevel(oldLevel), out.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
	})
	if err := diagnostic.SetProcessVerbosityLevel(9); err != nil {
		t.Fatal(err)
	}
	if err := ReportTacticProgress(":bound-propagations", 7); err != nil {
		t.Fatal(err)
	}
	if err := diagnostic.SetProcessVerbosityLevel(10); err != nil {
		t.Fatal(err)
	}
	for _, value := range []uint32{0, 7} {
		if err := ReportTacticProgress(":bound-propagations", value); err != nil {
			t.Fatal(err)
		}
	}
	got, err := out.String()
	if err != nil || got != "(:bound-propagations 7)\n" {
		t.Fatal("source progress output", got, err)
	}
}
