package pattern

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

// Source: src/ast/pattern/pattern_inference.cpp:123-138. Both occurrences of
// the shared child are queued before either is cached, then the parent is
// visited again after its children.
func TestPatternCollectSharedChildSourceStackTrace(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	patternMust(t, m.IncRef(x))
	t.Cleanup(func() { patternMust(t, m.DecRef(x)) })
	d, err := m.MkFuncDecl(symbol.Numerical(960), []ast.ASTHandle{b, b}, b, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	patternMust(t, m.IncRef(d))
	t.Cleanup(func() { patternMust(t, m.DecRef(d)) })
	app, err := m.MkApp(d, []ast.ASTHandle{x, x})
	patternMust(t, err)
	patternMust(t, m.IncRef(app))
	t.Cleanup(func() { patternMust(t, m.DecRef(app)) })
	c, err := newPatternCandidates(m, 1, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("collect")
	c.traceTags = tags
	patternMust(t, c.collect(app))
	if len(c.candidates) != 1 || c.candidates[0] != app {
		t.Fatal("shared child candidate", c.candidates)
	}
	if tags.ShouldTrace("collect") {
		appID, err := m.GetASTID(app)
		patternMust(t, err)
		xID, err := m.GetASTID(x)
		patternMust(t, err)
		record := func(id uint32, kind ast.ASTKind) string {
			return fmt.Sprintf("-------- [collect] operator() src/ast/pattern/pattern_inference.cpp:131 ---------\nprocessing: %d 0 kind: %d\n------------------------------------------------\n", id, kind)
		}
		want := record(appID, ast.ASTApp) + record(xID, ast.ASTVar) + record(xID, ast.ASTVar) + record(appID, ast.ASTApp)
		if output.String() != want {
			t.Fatalf("source stack trace: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased collect trace wrote output", output.String())
	}
	patternMust(t, c.resetCandidates())
	output.Reset()
	tags.Disable("collect")
	patternMust(t, c.collect(app))
	if output.Len() != 0 || len(c.candidates) != 1 {
		t.Fatal("disabled collect trace changed candidate or emitted output", c.candidates, output.String())
	}
}

func TestPatternCollectInfoSourceSMT2Trace(t *testing.T) {
	m, sy, q := patternManagerServices(t)
	b := m.BudgetField
	ctx, err := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: b})
	patternMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(sy, uparams.GlobalParamsCatalogConfig{
		Budget:       b,
		Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}},
		Text:         arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	})
	patternMust(t, err)
	g, err := uparams.NewGlobalParamsService(sy, ctx, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog})
	patternMust(t, err)
	patternMust(t, g.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(g)
	patternMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		patternMust(t, e)
		patternMust(t, g.Close())
		patternMust(t, catalog.Close())
		patternMust(t, ctx.Close())
	})
	boolSort, err := m.MkBoolSort()
	patternMust(t, err)
	x, err := m.MkVar(0, boolSort)
	patternMust(t, err)
	patternMust(t, m.IncRef(x))
	t.Cleanup(func() { patternMust(t, m.DecRef(x)) })
	c, err := newPatternCandidates(m, 1, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("collect")
	tags.Enable("collect_info")
	c.traceTags = tags
	patternMust(t, c.collect(x))
	if tags.ShouldTrace("collect_info") {
		id, err := m.GetASTID(x)
		patternMust(t, err)
		want := fmt.Sprintf("-------- [collect] operator() src/ast/pattern/pattern_inference.cpp:131 ---------\nprocessing: %d 0 kind: 1\n------------------------------------------------\n", id) +
			"-------- [collect_info] operator() src/ast/pattern/pattern_inference.cpp:132 ---------\n(:var 0)\n------------------------------------------------\n"
		if output.String() != want {
			t.Fatalf("source SMT2 body: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased collect_info trace wrote output", output.String())
	}
	output.Reset()
	tags.Disable("collect")
	tags.Disable("collect_info")
	patternMust(t, c.collect(x))
	if output.Len() != 0 {
		t.Fatal("disabled collect_info trace wrote output", output.String())
	}
	fName, err := sy.Intern("collect-f")
	patternMust(t, err)
	xName, err := sy.Intern("collect-x")
	patternMust(t, err)
	decl, err := m.MkFuncDecl(fName, []ast.ASTHandle{boolSort}, boolSort, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	patternMust(t, m.IncRef(decl))
	t.Cleanup(func() { patternMust(t, m.DecRef(decl)) })
	app, err := m.MkApp(decl, []ast.ASTHandle{x})
	patternMust(t, err)
	patternMust(t, m.IncRef(app))
	t.Cleanup(func() { patternMust(t, m.DecRef(app)) })
	forall, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []symbol.Symbol{xName}, Body: app})
	patternMust(t, err)
	patternMust(t, m.IncRef(forall))
	t.Cleanup(func() { patternMust(t, m.DecRef(forall)) })
	tags.Enable("collect_info")
	infoRecord := func(body string) string {
		return "-------- [collect_info] operator() src/ast/pattern/pattern_inference.cpp:132 ---------\n" + body + "\n------------------------------------------------\n"
	}
	tags.Enable("pattern_inference")
	patternMust(t, c.collect(app))
	if tags.ShouldTrace("collect_info") {
		appBody := "(collect-f (:var 0))"
		want := infoRecord(appBody) + infoRecord("(:var 0)") + infoRecord(appBody)
		want += "-------- [pattern_inference] save_candidate src/ast/pattern/pattern_inference.cpp:251 ---------\npotential candidate: \n" + appBody + "\n------------------------------------------------\n"
		if output.String() != want {
			t.Fatalf("application stack display: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased application display wrote output", output.String())
	}
	patternMust(t, c.resetCandidates())
	output.Reset()
	tags.Disable("pattern_inference")
	patternMust(t, c.collect(forall))
	if tags.ShouldTrace("collect_info") {
		appBody := "(collect-f (:var 0))"
		quantifierBody := "(forall ((collect-x Bool)) (collect-f collect-x))"
		want := infoRecord(quantifierBody) + infoRecord(appBody) + infoRecord("(:var 0)") + infoRecord(appBody) + infoRecord(quantifierBody)
		if output.String() != want {
			t.Fatalf("quantifier stack display: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased quantifier display wrote output", output.String())
	}
	tags.Disable("collect_info")
	innerName, err := sy.Intern("collect-inner")
	patternMust(t, err)
	outerName, err := sy.Intern("collect-outer")
	patternMust(t, err)
	makeUnary := func(name symbol.Symbol, child ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		d, err := m.MkFuncDecl(name, []ast.ASTHandle{boolSort}, boolSort, ast.ASTFuncDeclInfo{})
		patternMust(t, err)
		patternMust(t, m.IncRef(d))
		t.Cleanup(func() { patternMust(t, m.DecRef(d)) })
		a, err := m.MkApp(d, []ast.ASTHandle{child})
		patternMust(t, err)
		patternMust(t, m.IncRef(a))
		t.Cleanup(func() { patternMust(t, m.DecRef(a)) })
		return a
	}
	inner := makeUnary(innerName, x)
	outer := makeUnary(outerName, inner)
	patternMust(t, c.collect(outer))
	output.Reset()
	tags.Enable("pattern_inference")
	filtered, err := c.filterBigger([]ast.ASTHandle{outer, inner})
	patternMust(t, err)
	if len(filtered) != 1 || filtered[0] != inner {
		t.Fatal("source bigger-pattern filter", filtered)
	}
	if tags.ShouldTrace("pattern_inference") {
		want := "-------- [pattern_inference] operator() src/ast/pattern/pattern_inference.cpp:354 ---------\n" +
			"(collect-outer (collect-inner (:var 0)))\nis bigger than\n(collect-inner (:var 0))\n" +
			"------------------------------------------------\n"
		if output.String() != want {
			t.Fatalf("bigger-pattern trace: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased bigger-pattern trace wrote output", output.String())
	}
	output.Reset()
	tags.Disable("pattern_inference")
	filtered, err = c.filterBigger([]ast.ASTHandle{outer, inner})
	patternMust(t, err)
	if len(filtered) != 1 || filtered[0] != inner || output.Len() != 0 {
		t.Fatal("disabled bigger-pattern trace changed filtering or emitted output", filtered, output.String())
	}
	patternMust(t, c.resetCandidates())
	patternMust(t, c.addCandidate(app, []uint32{0}, 2))
	patternMust(t, c.registerPreferred(decl))
	output.Reset()
	tags.Enable("pattern_inference")
	preferred, found, err := c.selectPreferred([]ast.ASTHandle{app})
	patternMust(t, err)
	if !found || len(preferred) != 1 {
		t.Fatal("source preferred candidate", found, len(preferred))
	}
	if tags.ShouldTrace("pattern_inference") {
		want := "-------- [pattern_inference] has_preferred_patterns src/ast/pattern/pattern_inference.cpp:510 ---------\n" +
			"found preferred pattern:\n(collect-f (:var 0))\n------------------------------------------------\n"
		if output.String() != want {
			t.Fatalf("preferred-pattern trace: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased preferred-pattern trace wrote output", output.String())
	}
	patternMust(t, m.DelRef(&preferred[0]))
	output.Reset()
	tags.Disable("pattern_inference")
	preferred, found, err = c.selectPreferred([]ast.ASTHandle{app})
	patternMust(t, err)
	if !found || len(preferred) != 1 || output.Len() != 0 {
		t.Fatal("disabled preferred trace changed selection or emitted output", found, len(preferred), output.String())
	}
	patternMust(t, m.DelRef(&preferred[0]))
}
