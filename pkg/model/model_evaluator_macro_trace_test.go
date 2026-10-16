package model_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestModelEvaluatorGetMacroSourceTraceBranches(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("model_evaluator")
	astTestMust(t, ev.BindTrace(tags))
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	decl := modelTestFunctionDecl(t, m, "macro-trace-existing")
	fi := modelTestFunction(t, m, 1)
	astTestMust(t, fi.SetElse(yes))
	astTestMust(t, md.Core().RegisterFunction(decl, fi))
	app, e := m.MkApp(decl, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, app)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(app, out))
	firstRecords := output.String()
	entry := func(name string) string {
		return "-------- [model_evaluator] operator() src/model/model_evaluator.cpp:566 ---------\n" +
			"(" + name + " false)\n" +
			"------------------------------------------------\n"
	}
	missing := modelTestFunctionDecl(t, m, "macro-trace-missing")
	app, e = m.MkApp(missing, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, app)
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(app, out))
	secondRecords := firstRecords
	if tags.ShouldTrace("model_evaluator") {
		secondRecords += entry("macro-trace-missing")
	}
	if output.String() != secondRecords {
		t.Fatal("absent interpretation traced with completion disabled", output.String())
	}
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.SetModelCompletion(true))
	astTestMust(t, ev.Evaluate(app, out))
	if !tags.ShouldTrace("model_evaluator") {
		if output.Len() != 0 {
			t.Fatal("erased trace", output.String())
		}
		return
	}
	want := entry("macro-trace-existing") +
		"-------- [model_evaluator] get_macro src/model/model_evaluator.cpp:228 ---------\n" +
		"get_macro for macro-trace-existing (model completion: 0)\n" +
		"------------------------------------------------\n" +
		entry("macro-trace-missing") + entry("macro-trace-missing") +
		"-------- [model_evaluator] get_macro src/model/model_evaluator.cpp:247 ---------\n" +
		"get_macro for macro-trace-missing (model completion: 1)\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatal("source get_macro records", strings.ReplaceAll(got, "\n", "\\n"), strings.ReplaceAll(want, "\n", "\\n"))
	}
}
