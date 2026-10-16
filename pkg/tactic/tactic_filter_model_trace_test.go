package tactic

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/tactic/filter_model_converter.cpp:26,51,59-65. Both snapshots
// use actual model contents, and the pre-filter record includes display().
func TestFilterModelSourceBeforeAfterTrace(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, md, out, config := extensionFixture(t, tags)
	dA := modelTestDecl(t, m, "filter-trace-a")
	dB := modelTestDecl(t, m, "filter-trace-b")
	yes, err := m.MkTrue()
	satOK(t, err)
	no, err := m.MkFalse()
	satOK(t, err)
	satOK(t, md.Core().RegisterConstant(dA, yes))
	satOK(t, md.Core().RegisterConstant(dB, no))
	bs, err := m.MkBoolSort()
	satOK(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "filter-trace-f"), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
	satOK(t, err)
	fi, err := model.NewFunctionInterpretation(m, 1)
	satOK(t, err)
	satOK(t, fi.InsertEntry([]ast.ASTHandle{yes}, no))
	satOK(t, fi.SetElse(yes))
	satOK(t, md.Core().RegisterFunction(f, fi))
	c, err := NewFilterModel(m)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	satOK(t, c.Insert(dB))
	satOK(t, c.Insert(dB))
	satOK(t, c.BindTrace(tags, config.Globals))
	tags.Enable("filter_mc")
	satOK(t, c.ApplyModel(out, 0))
	next, err := out.Get()
	satOK(t, err)
	value, err := next.ConstantInterpretation(dA)
	satOK(t, err)
	if value != yes {
		t.Fatal("filter trace path changed retained model")
	}
	if copied, err := next.FunctionInterpretation(f); err != nil || copied == nil || copied == fi {
		t.Fatal("filter trace path did not copy the retained function", err)
	}
	if !tags.ShouldTrace("filter_mc") {
		if output.Len() != 0 {
			t.Fatal("erased filter trace", output.String())
		}
		return
	}
	want := "-------- [filter_mc] operator() src/tactic/filter_model_converter.cpp:26 ---------\n" +
		"before filter_model_converter\n" +
		"filter-trace-a -> true\nfilter-trace-b -> false\n" +
		"filter-trace-f -> {\n  true -> false\n  else -> true\n}\n" +
		"(filter-model-converter filter-trace-b filter-trace-b)\n" +
		"------------------------------------------------\n" +
		"-------- [filter_mc] operator() src/tactic/filter_model_converter.cpp:51 ---------\n" +
		"after filter_model_converter\n" +
		"filter-trace-a -> true\n" +
		"filter-trace-f -> {\n  true -> false\n  else -> true\n}\n" +
		"------------------------------------------------\n"
	if output.String() != want {
		t.Fatalf("source filter trace: got %q, want %q", output.String(), want)
	}
}
