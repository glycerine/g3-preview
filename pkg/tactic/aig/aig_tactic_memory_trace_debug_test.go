//go:build g3debug && g3trace

package aig

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGTacticMemoryWrapperSourceAllocationAndRelease(t *testing.T) {
	b := arena.NewBudget(1 << 30)
	s, e := symbol.NewManager(symbol.Config{Budget: b})
	aigMust(t, e)
	q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	aigMust(t, e)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	aigMust(t, e)
	tags.Enable("memory")
	sort, e := m.MkBoolSort()
	aigMust(t, e)
	p, e := m.MkFreshConst("p", sort)
	aigMust(t, e)
	qVar, e := m.MkFreshConst("q", sort)
	aigMust(t, e)
	or, e := m.MkOr(p, qVar)
	aigMust(t, e)
	ref, e := m.NewRef(or)
	aigMust(t, e)
	goal, e := tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	aigMust(t, goal.Assert(or))
	out, e := NewPass().Apply(goal)
	aigMust(t, e)
	aigMust(t, out.DecRef())
	goal, e = tactic.NewGoal(m, false, false, false)
	aigMust(t, e)
	aigMust(t, goal.Assert(or))
	m.Limit.Cancel()
	if out, e := NewPass().Apply(goal); out != nil || e == nil {
		t.Fatalf("canceled AIG pass: %v %v", out, e)
	}
	m.Limit.ResetCancel()
	aigMust(t, goal.Close())
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	var records []string
	for _, line := range lines {
		if strings.Contains(line, "src/tactic/aig/aig_tactic.cpp:") ||
			strings.Contains(line, "src/tactic/aig/aig.cpp:1684") ||
			strings.Contains(line, "src/tactic/aig/aig.cpp:1688") {
			records = append(records, line)
		}
	}
	if len(records) != 8 {
		t.Fatalf("tactic source memory records: %q", records)
	}
	wrapperSize := strconv.FormatUint(uint64(reflect.TypeOf(Manager{}).Size()), 10)
	impSize := strconv.FormatUint(uint64(reflect.TypeOf(managerImp{}).Size()), 10)
	for i := 0; i < 8; i += 4 {
		wrapper, imp := strings.Fields(records[i]), strings.Fields(records[i+1])
		if len(wrapper) != 5 || wrapper[0] != "alloc" || wrapper[2] != "src/tactic/aig/aig_tactic.cpp:34" || wrapper[3] != "aig_manager" || wrapper[4] != wrapperSize {
			t.Fatalf("wrapper allocation record: %q", records[i])
		}
		if len(imp) != 5 || imp[0] != "alloc" || imp[2] != "src/tactic/aig/aig.cpp:1684" || imp[3] != "imp" || imp[4] != impSize || imp[1] == wrapper[1] {
			t.Fatalf("imp allocation record: %q", records[i+1])
		}
		if want := "dealloc " + imp[1] + " src/tactic/aig/aig.cpp:1688"; records[i+2] != want {
			t.Fatalf("imp release %q, want %q", records[i+2], want)
		}
		if want := "dealloc " + wrapper[1] + " src/tactic/aig/aig_tactic.cpp:39"; records[i+3] != want {
			t.Fatalf("wrapper release %q, want %q", records[i+3], want)
		}
	}
	aigMust(t, m.DelRef(&ref))
	aigMust(t, m.Close())
	aigMust(t, q.Close())
	aigMust(t, s.Close())
	if b.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", b.Used())
	}
}
