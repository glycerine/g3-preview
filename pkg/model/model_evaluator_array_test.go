package model_test

// Source: src/model/model_evaluator.cpp:66-93,146-203,254-276,292-302.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/model/model_evaluator.cpp:169-202. A successful array select
// rewrite reaches the shared conditional result record.
func TestModelEvaluatorArraySelectSourceStatusTrace(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	boolSort := mustBoolSort(t, m)
	bp, e := m.NewParameterAST(boolSort)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&bp)) }()
	arraySort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	astTestMust(t, e)
	ap, e := m.NewParameterAST(arraySort)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&ap)) }()
	no, e := m.MkFalse()
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{ap}, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, constant)
	term, e := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{constant, yes})
	astTestMust(t, e)
	astTestPin(t, m, term)
	out := astTestPin(t, m, ast.ASTHandle{})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	astTestMust(t, ev.BindTrace(tags))
	tags.Enable("model_evaluator")
	astTestMust(t, ev.Evaluate(term, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != no {
		t.Fatal("array select rewrite", got)
	}
	if !tags.ShouldTrace("model_evaluator") {
		if output.Len() != 0 {
			t.Fatal("erased trace", output.String())
		}
		return
	}
	want := "-------- [model_evaluator] reduce_app src/model/model_evaluator.cpp:200 ---------\n" +
		"false\n" +
		"------------------------------------------------\n"
	if got := output.String(); !strings.HasSuffix(got, want) {
		t.Fatalf("array select status trace: got %q, want suffix %q", got, want)
	}
}

func TestModelEvaluatorOriginalArrayStoreSelectDispatch(t *testing.T) {
	m, md, ev, c, p, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	bp, e := m.NewParameterAST(bs)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&bp)) }()
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	astTestMust(t, e)
	ap, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&ap)) }()
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{ap}, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, constant)
	mk := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, op, nil, args)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	stored := mk(ast.ASTArrayOpStore, constant, yes, yes)
	x := goalTestAtom(t, m, "evaluated-array-index")
	d, e := m.GetDecl(x)
	astTestMust(t, e)
	out := astTestPin(t, m, ast.ASTHandle{})
	check := func(term, want ast.ASTHandle) {
		t.Helper()
		astTestMust(t, ev.Evaluate(term, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != want {
			t.Fatal("array evaluation", got, want)
		}
	}
	check(mk(ast.ASTArrayOpSelect, stored, yes), yes)
	check(mk(ast.ASTArrayOpSelect, stored, no), no)
	// The full traversal also reduces the generated Boolean ITE to x.
	check(mk(ast.ASTArrayOpSelect, stored, x), x)
	astTestMust(t, md.Core().RegisterConstant(d, no))
	astTestMust(t, ev.Reset(p))
	check(mk(ast.ASTArrayOpSelect, stored, x), no)
	// Local array settings are read by evaluator construction/cleanup, not by
	// its own updt_params/reset, which only updates model_evaluator parameters.
	key := "sort_store"
	astTestMust(t, c.SetBoolName(p, &key, true))
	astTestMust(t, ev.Reset(p))
	reverseBefore := mk(ast.ASTArrayOpStore, mk(ast.ASTArrayOpStore, constant, yes, yes), no, yes)
	check(reverseBefore, reverseBefore)
	astTestMust(t, ev.Cleanup(p))
	reverse := mk(ast.ASTArrayOpStore, mk(ast.ASTArrayOpStore, constant, yes, yes), no, yes)
	sorted := mk(ast.ASTArrayOpStore, mk(ast.ASTArrayOpStore, constant, no, yes), yes, yes)
	check(reverse, sorted)
	z := goalTestAtom(t, m, "array-ite-index")
	branch, e := m.MkIte(z, stored, constant)
	astTestMust(t, e)
	astTestPin(t, m, branch)
	check(mk(ast.ASTArrayOpSelect, branch, yes), z)
}

func TestModelEvaluatorArrayIdentityAndDisabledEquality(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	bp, e := m.NewParameterAST(bs)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&bp)) }()
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	astTestMust(t, e)
	mk := func(name string) ast.ASTHandle {
		d, e := m.MkFuncDecl(astTestSymbol(t, m, name), nil, sort, ast.ASTFuncDeclInfo{})
		astTestMust(t, e)
		h, e := m.MkApp(d, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	a, b := mk("array-eq-a"), mk("array-eq-b")
	equal, e := m.MkEq(a, a)
	astTestMust(t, e)
	astTestPin(t, m, equal)
	unknown, e := m.MkEq(a, b)
	astTestMust(t, e)
	astTestPin(t, m, unknown)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	out := astTestPin(t, m, yes)
	astTestMust(t, ev.Evaluate(equal, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != yes {
		t.Fatal("array identity")
	}
	astTestMust(t, ev.SetExpandArrayEqualities(false))
	astTestMust(t, ev.Evaluate(unknown, out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != unknown {
		t.Fatal("disabled array equality")
	}
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, m.SetRef(out, yes))
	// Extraction now follows the source refusal/fallback path for bare arrays.
	astTestMust(t, ev.Evaluate(unknown, out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != unknown {
		t.Fatal("unextractable equality changed output")
	}

}

func TestModelEvaluatorArrayPartialTheoryGraphCompletion(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	bp, e := m.NewParameterAST(bs)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&bp)) }()
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "partial-array-base"), nil, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	base, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, base)
	x := goalTestAtom(t, m, "partial-array-index")
	selected, e := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{base, x})
	astTestMust(t, e)
	astTestPin(t, m, selected)
	selectDecl, e := m.GetDecl(selected)
	astTestMust(t, e)
	fi, e := model.NewFunctionInterpretation(m, 2)
	astTestMust(t, e)
	astTestMust(t, md.Core().RegisterFunction(selectDecl, fi))
	// Partial theory graphs acquire an else value even with model completion off.
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	astTestMust(t, ev.BindTrace(tags))
	tags.Enable("model_evaluator")
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(selected, out))
	no, e := m.MkFalse()
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	fallback, e := fi.Else()
	astTestMust(t, e)
	if got != no || fallback != no {
		t.Fatal("source theory graph completion", got, fallback)
	}
	if !tags.ShouldTrace("model_evaluator") {
		if output.Len() != 0 {
			t.Fatal("erased partial-theory trace", output.String())
		}
		return
	}
	want := "-------- [model_evaluator] operator() src/model/model_evaluator.cpp:566 ---------\n" +
		"(select partial-array-base partial-array-index)\n" +
		"------------------------------------------------\n" +
		"-------- [model_evaluator] reduce_app src/model/model_evaluator.cpp:200 ---------\n" +
		"false\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("partial-theory source trace: got %q, want %q", got, want)
	}
}

func TestModelEvaluatorOriginalFunctionBackedArraySelection(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "model-array-function"), []ast.ASTHandle{bs, bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	fi, e := model.NewFunctionInterpretation(m, 2)
	astTestMust(t, e)
	astTestMust(t, fi.SetElse(yes))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes, no}, no))
	astTestMust(t, md.Core().RegisterFunction(f, fi))
	array, e := m.MkAsArray(f)
	astTestMust(t, e)
	astTestPin(t, m, array)
	out := astTestPin(t, m, ast.ASTHandle{})
	for _, indices := range [][]ast.ASTHandle{{yes, no}, {no, yes}} {
		selected, e := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{array, indices[0], indices[1]})
		astTestMust(t, e)
		astTestPin(t, m, selected)
		astTestMust(t, ev.Evaluate(selected, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := yes
		if indices[0] == yes {
			want = no
		}
		if got != want {
			t.Fatal("function-backed array interpretation", got, want)
		}
	}
}
