package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: bool_rewriter.cpp:592-599. The three unequal values are traced
// before false is made and published by the live equality reducer.
func TestBoolRewriterTryITEValueFalseSourceTrace(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	bindBVTraceGlobals(t, m)
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	values := [3]ast.ASTHandle{}
	for i := range values {
		values[i], err = m.BitVectorNumeralUnsigned(uint32(i), 8)
		astTestMust(t, err)
		astTestPin(t, m, values[i])
	}
	condition := goalTestAtom(t, m, "ite-value-condition")
	ite, err := m.MkIte(condition, values[0], values[1])
	astTestMust(t, err)
	astTestPin(t, m, ite)
	out := astTestPin(t, m, values[2])
	tags.Enable("try_ite_value")
	rw := boolRewriter{manager: m}
	status, err := rw.mkEqCore(ite, values[2], out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	falsity, err := m.MkFalse()
	astTestMust(t, err)
	if status != brDone || result != falsity {
		t.Fatal("source unequal ITE values", status, result, falsity)
	}
	if !tags.ShouldTrace("try_ite_value") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	wantValue := "-------- [try_ite_value] try_ite_value src/ast/rewriter/bool_rewriter.cpp:594 ---------\n" +
		"#x00 #x01 #x02\n" + boolTraceASTIdentity(values[0]) + " " + boolTraceASTIdentity(values[1]) + " " + boolTraceASTIdentity(values[2]) + "\n" +
		"------------------------------------------------\n"
	want := wantValue + "-------- [try_ite_value] mk_eq_core src/ast/rewriter/bool_rewriter.cpp:665 ---------\n" +
		"(if ite-value-condition bv[0:8] bv[1:8])\nbv[2:8]\n--->\nfalse\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source value trace got %q, want %q", got, want)
	}
	output.Reset()
	astTestMust(t, m.SetRef(out, values[2]))
	status, err = rw.mkEqCore(values[2], ite, out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	if status != brDone || result != falsity {
		t.Fatal("source reversed unequal ITE values", status, result, falsity)
	}
	want = wantValue + "-------- [try_ite_value] mk_eq_core src/ast/rewriter/bool_rewriter.cpp:670 ---------\n" +
		"bv[2:8]\n(if ite-value-condition bv[0:8] bv[1:8])\n--->\nfalse\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("reversed source value trace got %q, want %q", got, want)
	}
}
