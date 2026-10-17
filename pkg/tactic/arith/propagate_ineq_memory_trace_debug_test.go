//go:build g3debug && g3trace

package arith

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
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
	empty, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	borrowed, err := tactic.Apply(empty)
	equationMust(t, err)
	if borrowed != empty {
		t.Fatal("source no-bound branch must return the borrowed input")
	}
	equationMust(t, empty.Close())
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	xName, err := sy.Intern("memory-wrapper-x")
	equationMust(t, err)
	x, err := m.MkConst(xName, realSort)
	equationMust(t, err)
	n, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&n, "2"))
	two, err := m.ArithmeticNumeral(&n, false)
	equationMust(t, err)
	equationMust(t, q.Del(&n))
	bound, err := m.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, two})
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	equationMust(t, input.Assert(bound))
	result, err := tactic.Apply(input)
	equationMust(t, err)
	if result == input {
		t.Fatal("recognized bound must transfer the source result goal")
	}
	equationMust(t, result.Close())
	equationMust(t, input.Close())
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
	if strings.Count(got, " src/tactic/arith/propagate_ineqs_tactic.cpp:483 goal ") != 2 || strings.Count(got, " src/tactic/goal.h:85\n") != 2 {
		t.Fatal("no-bound and transferred goal allocation/deallocation records", got)
	}
	goalSize := strconv.FormatUint(uint64(unsafe.Sizeof(goaltactic.Goal{})), 10)
	pattern := regexp.MustCompile(`alloc (0x[0-9a-f]+) src/tactic/arith/propagate_ineqs_tactic\.cpp:483 goal ` + goalSize + `\n`)
	goals := pattern.FindAllStringSubmatch(got, -1)
	if len(goals) != 2 || goals[0][1] == fmt.Sprintf("%p", empty) || goals[0][1] == goals[1][1] || goals[1][1] != fmt.Sprintf("%p", result) {
		t.Fatal("source goal identity for borrowed and transferred paths", goals, got)
	}
	for _, match := range goals {
		alloc := match[0]
		dealloc := "dealloc " + match[1] + " src/tactic/goal.h:85\n"
		if strings.Count(got, dealloc) != 1 || strings.Index(got, alloc) >= strings.Index(got, dealloc) {
			t.Fatal("goal source lifetime", alloc, dealloc, got)
		}
	}
}
