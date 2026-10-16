package model_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/model/model_evaluator.cpp:435-497. Extraction emits entry,
// successful else, absent else, non-ground entry and non-ground else records
// from the real function-backed array path.
func TestModelEvaluatorArrayExtractionSourceTrace(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	_, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	bs := mustBoolSort(t, m)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "array-trace-f"), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, f)
	fi, err := model.NewFunctionInterpretation(m, 1)
	astTestMust(t, err)
	astTestMust(t, fi.SetElse(no))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, yes))
	astTestMust(t, md.Core().RegisterFunction(f, fi))
	array, err := m.MkAsArray(f)
	astTestMust(t, err)
	astTestPin(t, m, array)
	out := astTestPin(t, m, ast.ASTHandle{})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	astTestMust(t, ev.BindTrace(tags))
	astTestMust(t, ev.Evaluate(array, out))
	if output.Len() != 0 {
		t.Fatal("disabled array trace", output.String())
	}
	tags.Enable("model_evaluator")
	if !tags.ShouldTrace("model_evaluator") {
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(array, out))
		if output.Len() != 0 {
			t.Fatal("erased array trace", output.String())
		}
		return
	}
	check := func(line int, body string) {
		t.Helper()
		header := "[model_evaluator] extract_array_func_interp src/model/model_evaluator.cpp:"
		if !strings.Contains(output.String(), header+strconv.Itoa(line)+" ---------\n"+body) {
			t.Fatalf("missing source array trace line %d, body %q: %q", line, body, output.String())
		}
	}
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(array, out))
	check(435, "(_ as-array array-trace-f)\n")
	check(497, "else case: false\n")
	output.Reset()
	astTestMust(t, fi.SetElse(ast.ASTHandle{}))
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(array, out))
	check(476, "no else case (_ as-array array-trace-f)\n")
	output.Reset()
	v, err := m.MkVar(0, bs)
	astTestMust(t, err)
	astTestPin(t, m, v)
	astTestMust(t, fi.SetElse(v))
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(array, out))
	check(482, "non-ground else case (_ as-array array-trace-f)\n")
	output.Reset()
	arraySort, err := m.GetSort(array)
	astTestMust(t, err)
	plainDecl, err := m.MkFuncDecl(astTestSymbol(t, m, "untranslated-array"), nil, arraySort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, plainDecl)
	plain, err := m.MkApp(plainDecl, nil)
	astTestMust(t, err)
	astTestPin(t, m, plain)
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(plain, out))
	check(451, "no translation: untranslated-array\n")
	output.Reset()
	astTestMust(t, fi.SetElse(no))
	err = fi.InsertEntry([]ast.ASTHandle{no}, v)
	if ast.ASTSourceDebug {
		if err != ast.ErrASTDomain || output.Len() != 0 {
			t.Fatalf("source debug rejects non-ground function entry before extraction: %v %q", err, output.String())
		}
	} else {
		astTestMust(t, err)
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(array, out))
		check(468, "could not extract array interpretation: (_ as-array array-trace-f)\n(:var 0)\n")
	}
}
