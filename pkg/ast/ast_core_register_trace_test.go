//go:build g3trace

package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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

func TestASTDeletionSourceTraceRecords(t *testing.T) {
	s, numerals, budget := astTestOwners(t)
	var buf bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&buf)
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	disabled, e := m.MkVar(11, sort)
	astTestMust(t, e)
	ref, e := m.NewRef(disabled)
	astTestMust(t, e)
	astTestMust(t, m.DelRef(&ref))
	if buf.Len() != 0 {
		t.Fatalf("disabled deletion tags emitted %q", buf.String())
	}
	variable, e := m.MkVar(12, sort)
	astTestMust(t, e)
	quantifier, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{sort}, Names: []usymbol.Symbol{{}}, Body: variable})
	astTestMust(t, e)
	ref, e = m.NewRef(quantifier)
	astTestMust(t, e)
	qid, e := m.GetASTID(quantifier)
	astTestMust(t, e)
	vid, e := m.GetASTID(variable)
	astTestMust(t, e)
	qp := fmt.Sprintf("%p", m.nodeAt(quantifier.slot))
	vp := fmt.Sprintf("%p", m.nodeAt(variable.slot))
	tags.Enable("ast")
	tags.Enable("del_quantifier")
	tags.Enable("mk_var_bug")
	astTestMust(t, m.DelRef(&ref))
	qi := strconv.FormatUint(uint64(qid), 10)
	vi := strconv.FormatUint(uint64(vid), 10)
	record := func(tag string, line int, body string) string {
		return "-------- [" + tag + "] delete_node src/ast/ast.cpp:" + strconv.Itoa(line) + " ---------\n" +
			body + "\n------------------------------------------------\n"
	}
	want := record("ast", 1790, "Deleting object "+qi+" "+qp) +
		record("del_quantifier", 1791, "deleting quantifier "+qi+" "+qp) +
		record("mk_var_bug", 1792, "del_ast: "+qi) +
		record("ast", 1790, "Deleting object "+vi+" "+vp) +
		record("mk_var_bug", 1792, "del_ast: "+vi)
	if got := buf.String(); got != want {
		t.Fatalf("deletion records:\n got %q\nwant %q", got, want)
	}
}

func TestASTDeleteNodeBugTraceBeforeWorklist(t *testing.T) {
	s, numerals, budget := astTestOwners(t)
	var buf bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&buf)
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	variable, e := m.MkVar(7, sort)
	astTestMust(t, e)
	ref, e := m.NewRef(variable)
	astTestMust(t, e)
	tags.Enable("delete_node_bug")
	tags.Enable("mk_var_bug")
	tags.Enable("ast_delete_node")
	id, e := m.GetASTID(variable)
	astTestMust(t, e)
	astTestMust(t, m.DelRef(&ref))
	want := "-------- [delete_node_bug] delete_node src/ast/ast.cpp:1782 ---------\n" +
		"(:var 7 Bool)\n\n" +
		"------------------------------------------------\n" +
		"-------- [mk_var_bug] delete_node src/ast/ast.cpp:1792 ---------\n" +
		"del_ast: " + strconv.FormatUint(uint64(id), 10) + "\n" +
		"------------------------------------------------\n" +
		"-------- [ast_delete_node] delete_node src/ast/ast.cpp:1793 ---------\n" +
		"(:var 7)\n" +
		"------------------------------------------------\n"
	if got := buf.String(); got != want {
		t.Fatalf("delete_node_bug record:\n got %q\nwant %q", got, want)
	}
}
