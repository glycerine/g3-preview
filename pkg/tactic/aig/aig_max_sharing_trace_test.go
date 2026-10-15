package aig

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGMaxSharingProcessingAndNullTraceSourceFallthrough(t *testing.T) {
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
	closeParams := aigTestBindSMT2Params(t, symbols, numbers, budget)
	tags.Enable("max_sharing")
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	g, err := NewManager(m)
	aigMust(t, err)
	var vars [3]*Ref
	for i, name := range []string{"a", "b", "c"} {
		h, e := m.MkFreshConst(name, sort)
		aigMust(t, e)
		vars[i], e = g.MakeVariable(h)
		aigMust(t, e)
	}
	ab, err := g.InternAnd(vars[0], vars[1])
	aigMust(t, err)
	abNode := ab.lit.node
	abc, err := g.InternAnd(ab, vars[2])
	aigMust(t, err)
	aigMust(t, ab.Close())
	nestedIndex := 0
	if abc.lit.node.children[1].node == abNode {
		nestedIndex = 1
	}
	rootName := "@" + strconv.FormatUint(uint64(abc.lit.node.id-firstNodeID), 10)
	nestedName := "@" + strconv.FormatUint(uint64(abNode.id-firstNodeID), 10)
	check := func(wantNull string, result *Ref) {
		t.Helper()
		var bodies []string
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.HasPrefix(line, "processing ") || strings.HasPrefix(line, "r.is_null(): ") {
				bodies = append(bodies, line)
			}
		}
		if !tags.ShouldTrace("max_sharing") {
			if len(bodies) != 0 || output.Len() != 0 {
				t.Fatalf("erased trace emitted %q", output.String())
			}
			return
		}
		resultRecord := "-------- [max_sharing] operator() src/tactic/aig/aig.cpp:1295 ---------\n" +
			"result:\n" + aigRefText(result.lit) + "\n"
		if !strings.Contains(output.String(), resultRecord) {
			t.Fatalf("source result display missing: %q", output.String())
		}
		if wantNull == "0" {
			improved := "-------- [max_sharing] improve_sharing_left src/tactic/aig/aig.cpp:1144 ---------\n" +
				"improved:\n" + rootName + "\n"
			if !strings.Contains(output.String(), improved) ||
				!strings.Contains(output.String(), "---->\n"+aigRefText(result.lit)+"\n") {
				t.Fatalf("source improved display missing: %q", output.String())
			}
		}
		want := []string{
			"processing " + rootName + " idx: 0",
			"processing " + nestedName + " idx: 0",
			"processing " + rootName + " idx: " + strconv.Itoa(nestedIndex+1),
			"r.is_null(): " + wantNull,
		}
		if len(bodies) != len(want) {
			t.Fatalf("source records %q, want %q", bodies, want)
		}
		for i := range want {
			if bodies[i] != want[i] {
				t.Fatalf("source record %d = %q, want %q", i, bodies[i], want[i])
			}
		}
	}
	unchanged, err := g.MaxSharing(abc)
	aigMust(t, err)
	check("1", unchanged)
	aigMust(t, unchanged.Close())
	output.Reset()
	bc, err := g.InternAnd(vars[1], vars[2])
	aigMust(t, err)
	shared, err := g.MaxSharing(abc)
	aigMust(t, err)
	check("0", shared)
	aigMust(t, shared.Close())
	aigMust(t, bc.Close())
	aigMust(t, abc.Close())
	for _, r := range vars {
		aigMust(t, r.Close())
	}
	aigMust(t, g.Close())
	aigMust(t, m.Close())
	closeParams()
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}

func TestAIGMaxSharingTryingTraceAllFourReassociationBranches(t *testing.T) {
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
	closeParams := aigTestBindSMT2Params(t, symbols, numbers, budget)
	tags.Enable("max_sharing")
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	g, err := NewManager(m)
	aigMust(t, err)
	var vars [4]*Ref
	for i, name := range []string{"a", "b", "c", "d"} {
		h, e := m.MkFreshConst(name, sort)
		aigMust(t, e)
		vars[i], e = g.MakeVariable(h)
		aigMust(t, e)
	}
	ab, err := g.InternAnd(vars[0], vars[1])
	aigMust(t, err)
	cd, err := g.InternAnd(vars[2], vars[3])
	aigMust(t, err)
	root, err := g.InternAnd(ab, cd)
	aigMust(t, err)
	if root.lit.node.children[0].node != ab.lit.node || root.lit.node.children[1].node != cd.lit.node {
		t.Fatal("source canonical order did not put earlier nested node on left")
	}
	left, right := ab.lit.node, cd.lit.node
	aigMust(t, ab.Close())
	aigMust(t, cd.Close())
	result, err := g.MaxSharing(root)
	aigMust(t, err)
	if tags.ShouldTrace("max_sharing") {
		name := func(l Literal) string {
			prefix, id := "#", l.node.id
			if l.node.and {
				prefix, id = "@", id-firstNodeID
			}
			if l.inverted {
				prefix = "-" + prefix
			}
			return prefix + strconv.FormatUint(uint64(id), 10)
		}
		a, b, c := name(left.children[0]), name(left.children[1]), name(root.lit.node.children[1])
		d, e, f := name(root.lit.node.children[0]), name(right.children[0]), name(right.children[1])
		want := []struct {
			line         int
			action, body string
		}{
			{1131, "improve_sharing_left", "trying (and " + a + " (and " + b + " " + c + "))"},
			{1149, "improve_sharing_left", "trying (and " + a + " (and " + c + " " + b + "))"},
			{1176, "improve_sharing_right", "trying (and (and " + d + " " + e + ") " + f + ")"},
			{1195, "improve_sharing_right", "trying (and (and " + d + " " + f + ") " + e + ")"},
		}
		var headers, bodies []string
		var previous string
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.HasPrefix(line, "trying ") {
				headers = append(headers, previous)
				bodies = append(bodies, line)
			}
			previous = line
		}
		if len(bodies) != len(want) {
			t.Fatalf("trying records %q, want four", bodies)
		}
		for i, row := range want {
			header := "-------- [max_sharing] " + row.action + " src/tactic/aig/aig.cpp:" + strconv.Itoa(row.line) + " ---------"
			if headers[i] != header || bodies[i] != row.body {
				t.Fatalf("trying %d: header=%q body=%q; want %q %q", i, headers[i], bodies[i], header, row.body)
			}
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace emitted %q", output.String())
	}
	aigMust(t, result.Close())
	aigMust(t, root.Close())
	for _, r := range vars {
		aigMust(t, r.Close())
	}
	aigMust(t, g.Close())
	aigMust(t, m.Close())
	closeParams()
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
