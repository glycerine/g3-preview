package pattern

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternCollectGroundChildAndInfoMemoryLifetime(t *testing.T) {
	m := patternManager(t)
	boolSort, err := m.MkBoolSort()
	patternMust(t, err)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		patternMust(t, m.IncRef(h))
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
		return h
	}
	outerVar, err := m.MkVar(1, boolSort)
	patternMust(t, err)
	pin(outerVar)
	constantDecl, err := m.MkFuncDecl(symbol.Numerical(970), nil, boolSort, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	pin(constantDecl)
	constant, err := m.MkApp(constantDecl, nil)
	patternMust(t, err)
	pin(constant)
	pairDecl, err := m.MkFuncDecl(symbol.Numerical(971), []ast.ASTHandle{boolSort, boolSort}, boolSort, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	pin(pairDecl)
	body, err := m.MkApp(pairDecl, []ast.ASTHandle{outerVar, constant})
	patternMust(t, err)
	pin(body)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []symbol.Symbol{symbol.Numerical(972)}, Body: body})
	patternMust(t, err)
	pin(quantifier)
	c, err := newPatternCandidates(m, 1, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	c.traceTags = tags
	patternMust(t, c.collect(quantifier))
	if len(c.candidates) != 1 || len(c.collectRefs) != 0 {
		t.Fatal("ground child must survive collection and temporary info must be released", c.candidates, len(c.collectRefs))
	}
	mapped, err := m.GetArg(c.candidates[0], 0)
	patternMust(t, err)
	index, err := m.GetVarIndex(mapped)
	patternMust(t, err)
	second, err := m.GetArg(c.candidates[0], 1)
	patternMust(t, err)
	if index != 0 || second != constant {
		t.Fatal("source nested binder remapping or ground child", index, second)
	}
	if diagnostic.DebugBuild && tags.ShouldTrace("memory") {
		got := output.String()
		allocs := regexp.MustCompile(`alloc (0x[0-9a-f]+) src/ast/pattern/pattern_inference.cpp:(188|205|238) info ([0-9]+)\n`).FindAllStringSubmatch(got, -1)
		deallocs := regexp.MustCompile(`dealloc (0x[0-9a-f]+) src/util/util.h:171\n`).FindAllStringSubmatch(got, -1)
		if len(allocs) != 3 || len(deallocs) != 3 || allocs[0][2] != "188" || allocs[1][2] != "205" || allocs[2][2] != "238" {
			t.Fatalf("source collected-info allocation sites: %q", got)
		}
		for i := range allocs {
			if allocs[i][1] != deallocs[i][1] || allocs[i][3] == "0" {
				t.Fatalf("source collected-info release order/extent: %q", got)
			}
		}
		if strings.Index(got, "\ndealloc ") < strings.LastIndex(got, "\nalloc ") ||
			strings.Count(got, "[memory] allocate src/util/memory_manager.cpp:209") != 3 ||
			strings.Count(got, "[memory] deallocate src/util/memory_manager.cpp:204") != 3 {
			t.Fatalf("source collected-info memory records: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased collected-info memory trace emitted %q", output.String())
	}
}
