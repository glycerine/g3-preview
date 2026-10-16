//go:build g3debug && g3trace

package pattern

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternCollectedInfoMemoryFollowsOwningProcessFinalization(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	process := memory.NewProcessAccounting()
	runtime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, Process: process})
	patternMust(t, err)
	first, err := runtime.BeginSourceInitialize(math.MaxUint32)
	patternMust(t, err)
	if !first {
		t.Fatal("process runtime did not begin source initialization")
	}
	patternMust(t, runtime.CompleteSourceInitialize())
	symbols, err := symbol.NewManager(symbol.Config{Budget: budget})
	patternMust(t, err)
	numerals, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: runtime, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	patternMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	patternMust(t, err)
	boolSort, err := m.MkBoolSort()
	patternMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	patternMust(t, err)
	patternMust(t, m.IncRef(variable))
	c, err := newPatternCandidates(m, 1, nil, false)
	patternMust(t, err)
	c.traceTags = tags
	output.Reset()
	if tags.MemoryFinalizing() {
		t.Fatal("standalone trace tags unexpectedly bound to process")
	}
	active, err := process.BeginFinalize()
	patternMust(t, err)
	if !active {
		t.Fatal("process did not enter finalization")
	}
	patternMust(t, c.collect(variable))
	if got := output.String(); !strings.Contains(got, "src/ast/pattern/pattern_inference.cpp:186 info ") || strings.Contains(got, "dealloc ") {
		t.Fatalf("collector allocation or process-guarded release: %q", got)
	}
	patternMust(t, process.CompleteFinalize())
	patternMust(t, c.close())
	patternMust(t, m.DecRef(variable))
	patternMust(t, m.Close())
	patternMust(t, numerals.Close())
	patternMust(t, symbols.Close())
	patternMust(t, runtime.Close())
	if used := budget.Used(); used != 0 {
		t.Fatalf("owner graph retained %d bytes", used)
	}
}
