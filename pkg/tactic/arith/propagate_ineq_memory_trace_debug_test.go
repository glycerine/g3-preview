//go:build g3debug && g3trace

package arith

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPropagateIneqImplementationMemoryWrapperSourceLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	equationMust(t, err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	equationMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b, TraceTags: tags})
	equationMust(t, err)
	t.Cleanup(func() {
		equationMust(t, m.Close())
		equationMust(t, q.Close())
		equationMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("AST owner retained bytes", b.Used())
		}
	})
	numbers, _ := equationManager(t)
	context, value := boundParams(t)
	tactic, err := NewPropagateIneqTactic(m, numbers, context, value)
	equationMust(t, err)
	first := tactic.current.traceID
	equationMust(t, tactic.Cleanup())
	second := tactic.current.traceID
	if first == 0 || second == 0 || first == second {
		t.Fatal("replacement must have a distinct implementation identity", first, second)
	}
	equationMust(t, tactic.Close())
	size := strconv.FormatUint(uint64(unsafe.Sizeof(inequalityImplementation{})), 10)
	id := func(n uint64) string { return "0x" + strconv.FormatUint(n, 16) }
	want := []string{
		"alloc " + id(first) + " src/tactic/arith/propagate_ineqs_tactic.cpp:517 imp " + size + "\n",
		"alloc " + id(second) + " src/tactic/arith/propagate_ineqs_tactic.cpp:546 imp " + size + "\n",
		"dealloc " + id(first) + " src/tactic/arith/propagate_ineqs_tactic.cpp:548\n",
		"dealloc " + id(second) + " src/tactic/arith/propagate_ineqs_tactic.cpp:521\n",
	}
	got := output.String()
	last := -1
	for _, line := range want {
		at := strings.Index(got, line)
		if at <= last || strings.Count(got, line) != 1 {
			t.Fatal("source allocation/replacement/destruction order", line, got)
		}
		last = at
	}
}
