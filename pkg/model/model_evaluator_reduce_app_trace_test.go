package model_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type refusingModelEvaluatorTraceWriter struct{ calls int }

func (w *refusingModelEvaluatorTraceWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, io.ErrClosedPipe
}

func TestModelEvaluatorTraceStreamRefusalKeepsEvaluation(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	d := modelTestFunctionDecl(t, m, "refused-trace-f")
	fi := modelTestFunction(t, m, 1)
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, no))
	astTestMust(t, md.Core().RegisterFunction(d, fi))
	app, e := m.MkApp(d, []ast.ASTHandle{yes})
	astTestMust(t, e)
	astTestPin(t, m, app)
	out := astTestPin(t, m, ast.ASTHandle{})
	w := &refusingModelEvaluatorTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("model_evaluator")
	astTestMust(t, ev.BindTrace(tags))
	astTestMust(t, ev.Evaluate(app, out))
	result, e := m.RefGet(out)
	astTestMust(t, e)
	if result != no {
		t.Fatal("failed trace stream changed direct-entry evaluation", result)
	}
	if tags.ShouldTrace("model_evaluator") && w.calls != 1 {
		t.Fatal("source trace stream did not retain bad state", w.calls)
	}
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(yes, out))
	result, e = m.RefGet(out)
	astTestMust(t, e)
	if result != yes {
		t.Fatal("failed trace stream changed subsequent evaluation", result)
	}
	if tags.ShouldTrace("model_evaluator") && w.calls != 1 {
		t.Fatal("sticky trace stream retried after refusal", w.calls)
	}
}

// Source: src/model/model_evaluator.cpp:187-191,566. The entry lookup must
// record the evaluated argument and result after the outer operator entry.
func TestModelEvaluatorDirectEntrySourceTrace(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	d := modelTestFunctionDecl(t, m, "entry-trace-f")
	fi := modelTestFunction(t, m, 1)
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, no))
	astTestMust(t, md.Core().RegisterFunction(d, fi))
	app, e := m.MkApp(d, []ast.ASTHandle{yes})
	astTestMust(t, e)
	astTestPin(t, m, app)
	out := astTestPin(t, m, ast.ASTHandle{})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	astTestMust(t, ev.BindTrace(tags))
	astTestMust(t, ev.Evaluate(app, out))
	if output.Len() != 0 {
		t.Fatal("disabled evaluator trace", output.String())
	}
	tags.Enable("model_evaluator")
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(app, out))
	result, e := m.RefGet(out)
	astTestMust(t, e)
	if result != no {
		t.Fatal("direct function entry", result)
	}
	if !tags.ShouldTrace("model_evaluator") {
		if output.Len() != 0 {
			t.Fatal("erased evaluator trace", output.String())
		}
		return
	}
	want := "-------- [model_evaluator] operator() src/model/model_evaluator.cpp:566 ---------\n" +
		"(entry-trace-f true)\n" +
		"------------------------------------------------\n" +
		"-------- [model_evaluator] reduce_app src/model/model_evaluator.cpp:187 ---------\n" +
		"reduce_app entry-trace-f\ntrue\n---->\nfalse\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source evaluator direct-entry trace: got %q, want %q", got, want)
	}
}
