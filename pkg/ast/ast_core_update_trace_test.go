package ast

import (
	"bytes"
	"strconv"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTQuantifierWeightTraceBeforeReplacement(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	body, err := m.MkVar(0, sort)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{sort}, Names: []usymbol.Symbol{{}}, Body: body, Weight: 7})
	astTestMust(t, err)
	astTestPin(t, m, q)
	id, err := m.GetASTID(q)
	astTestMust(t, err)
	tags.Enable("update_quantifier_weight")
	tags.Enable("ast")
	if got, err := m.UpdateQuantifierWeight(q, 7); err != nil || got != q || output.Len() != 0 {
		t.Fatal("unchanged weight emitted or allocated", got, err, output.String())
	}
	updated, err := m.UpdateQuantifierWeight(q, -3)
	astTestMust(t, err)
	astTestPin(t, m, updated)
	updatedID, err := m.GetASTID(updated)
	astTestMust(t, err)
	weight, err := m.GetQuantifierWeight(updated)
	astTestMust(t, err)
	if weight != -3 {
		t.Fatal("replacement weight", weight)
	}
	if tags.ShouldTrace("update_quantifier_weight") {
		record := "-------- [update_quantifier_weight] update_quantifier_weight src/ast/ast.cpp:2454 ---------\n" +
			"#" + strconv.FormatUint(uint64(id), 10) + " 7 -> -3\n" +
			"------------------------------------------------\n"
		registered := "-------- [ast] register_node_core src/ast/ast.cpp:1696 ---------\n" +
			"Object " + strconv.FormatUint(uint64(updatedID), 10) + " was created.\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != record+registered {
			t.Fatalf("source record before replacement registration:\n got %q\nwant %q", got, record+registered)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased TRACE emitted", output.String())
	}
}
