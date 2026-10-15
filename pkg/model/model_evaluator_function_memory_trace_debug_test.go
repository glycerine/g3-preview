//go:build g3debug && g3trace

package model_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestModelEvaluatorCompletedFunctionMemoryWrapperReleaseSites(t *testing.T) {
	m, md, original, _, _, config := evaluatorFixture(t)
	astTestMust(t, original.Close())
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	config.Trace = tags
	ev, err := model.NewModelEvaluator(md.Core(), config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, ev.Close()) })
	astTestMust(t, ev.SetModelCompletion(true))
	yes, err := m.MkTrue()
	astTestMust(t, err)
	astTestPin(t, m, yes)
	out := astTestPin(t, m, ast.ASTHandle{})
	var declarations [3]ast.ASTHandle
	var interpretations [3]*model.FunctionInterpretation
	for i, name := range [...]string{"trace-replace", "trace-unregister", "trace-close"} {
		d := modelTestFunctionDecl(t, m, name)
		declarations[i] = d
		app, e := m.MkApp(d, []ast.ASTHandle{yes})
		astTestMust(t, e)
		astTestPin(t, m, app)
		astTestMust(t, ev.Evaluate(app, out))
		fi, e := md.Core().FunctionInterpretation(d)
		astTestMust(t, e)
		if fi == nil {
			t.Fatal("completion did not publish a function", name)
		}
		interpretations[i] = fi
	}
	replacement, err := model.NewFunctionInterpretation(m, 1)
	astTestMust(t, err)
	astTestMust(t, replacement.SetElse(yes))
	astTestMust(t, md.Core().RegisterFunction(declarations[0], replacement))
	astTestMust(t, md.Core().UnregisterDeclaration(declarations[1]))
	astTestMust(t, ev.Close())
	astTestMust(t, md.Close())
	size := strconv.FormatUint(uint64(reflect.TypeOf(model.FunctionInterpretation{}).Size()), 10)
	got := output.String()
	for i, fi := range interpretations {
		id := fmt.Sprintf("%p", fi)
		alloc := "\nalloc " + id + " src/model/model_evaluator.cpp:250 func_interp " + size + "\n"
		line := [...]int{81, 105, 29}[i]
		dealloc := "\ndealloc " + id + " src/model/model_core.cpp:" + strconv.Itoa(line) + "\n"
		if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 {
			t.Fatalf("completed function memory records for %d: alloc=%q dealloc=%q output=%q", i, alloc, dealloc, got)
		}
	}
}

func TestModelCopiedFunctionMemoryWrapperOwnerTransfer(t *testing.T) {
	m, md, original, _, _, config := evaluatorFixture(t)
	astTestMust(t, original.Close())
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	config.Trace = tags
	ev, err := model.NewModelEvaluator(md.Core(), config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, ev.Close()) })
	astTestMust(t, ev.SetModelCompletion(true))
	yes, err := m.MkTrue()
	astTestMust(t, err)
	astTestPin(t, m, yes)
	d := modelTestFunctionDecl(t, m, "trace-copied-function")
	app, err := m.MkApp(d, []ast.ASTHandle{yes})
	astTestMust(t, err)
	astTestPin(t, m, app)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(app, out))
	source, err := md.Core().FunctionInterpretation(d)
	astTestMust(t, err)
	if source == nil {
		t.Fatal("completion did not publish source function")
	}
	copied, err := model.NewModelCore(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, copied.Close()) })
	astTestMust(t, copied.CopyFunctionInterpretation(d, source))
	owned, err := copied.FunctionInterpretation(d)
	astTestMust(t, err)
	if owned == nil || owned == source {
		t.Fatal("model copy did not transfer an independent graph")
	}
	astTestMust(t, copied.Close())
	id := fmt.Sprintf("%p", owned)
	size := strconv.FormatUint(uint64(reflect.TypeOf(model.FunctionInterpretation{}).Size()), 10)
	got := output.String()
	alloc := "\nalloc " + id + " src/model/func_interp.cpp:91 func_interp " + size + "\n"
	dealloc := "\ndealloc " + id + " src/model/model_core.cpp:29\n"
	if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 {
		t.Fatalf("copied function memory records: alloc=%q dealloc=%q output=%q", alloc, dealloc, got)
	}
}
