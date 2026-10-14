//go:build g3trace

package ast

import (
	"bytes"
	"strconv"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTRegistrationSourceTraceRecords(t *testing.T) {
	s, q, budget := astTestOwners(t)
	var buf bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&buf)
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: budget, TraceTags: tags})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	if buf.Len() != 0 {
		t.Fatalf("disabled registration tags emitted %q", buf.String())
	}
	tags.Enable("ast")
	tags.Enable("mk_var_bug")
	variable, e := m.MkVar(37, sort)
	astTestMust(t, e)
	id, e := m.GetASTID(variable)
	astTestMust(t, e)
	n := strconv.FormatUint(uint64(id), 10)
	want := "-------- [ast] register_node_core src/ast/ast.cpp:1696 ---------\n" +
		"Object " + n + " was created.\n" +
		"------------------------------------------------\n" +
		"-------- [mk_var_bug] register_node_core src/ast/ast.cpp:1697 ---------\n" +
		"mk_ast: " + n + "\n" +
		"------------------------------------------------\n"
	if got := buf.String(); got != want {
		t.Fatalf("registration records:\n got %q\nwant %q", got, want)
	}
	again, e := m.MkVar(37, sort)
	astTestMust(t, e)
	if again != variable || buf.String() != want {
		t.Fatalf("duplicate registration changed node or emitted another record: %q", buf.String())
	}
}
