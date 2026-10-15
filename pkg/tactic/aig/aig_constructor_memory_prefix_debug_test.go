//go:build g3debug && g3trace

package aig

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGTacticConstructorFailureMemoryTracePrefixes(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
	symbols, err := symbol.NewManager(symbol.Config{Budget: budget})
	aigMust(t, err)
	numbers, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	aigMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numbers, Budget: budget, TraceTags: tags})
	aigMust(t, err)
	tags.Enable("memory")
	warm, err := newManagerWithGateEncoding(m, true, true)
	aigMust(t, err)
	aigMust(t, warm.Close())
	baseline := budget.Used()
	for prefix := int64(0); prefix < 3; prefix++ {
		output.Reset()
		budget.FailAfter(prefix)
		got, e := newManagerWithGateEncoding(m, true, true)
		budget.FailAfter(-1)
		if got != nil || !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatalf("prefix %d: manager=%v err=%v", prefix, got, e)
		}
		if used := budget.Used(); used != baseline {
			t.Fatalf("prefix %d retained %d bytes", prefix, used-baseline)
		}
		var records []string
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.HasPrefix(line, "alloc ") &&
				(strings.Contains(line, "src/tactic/aig/aig_tactic.cpp:34") || strings.Contains(line, "src/tactic/aig/aig.cpp:1684")) ||
				strings.HasPrefix(line, "dealloc ") &&
					(strings.Contains(line, "src/tactic/aig/aig_tactic.cpp:39") || strings.Contains(line, "src/tactic/aig/aig.cpp:1688")) {
				records = append(records, line)
			}
		}
		if len(records) != int(prefix) {
			t.Fatalf("prefix %d emitted %d source records: %q", prefix, len(records), records)
		}
		for i, source := range []string{"src/tactic/aig/aig_tactic.cpp:34 aig_manager", "src/tactic/aig/aig.cpp:1684 imp"}[:prefix] {
			if !strings.HasPrefix(records[i], "alloc ") || !strings.Contains(records[i], source) {
				t.Fatalf("prefix %d record %d: %q", prefix, i, records[i])
			}
		}
	}
	aigMust(t, m.Close())
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
