package pattern

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
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
