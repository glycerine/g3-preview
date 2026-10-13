package pattern

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceLoopFilterConcreteCandidateOwnership(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		patternMust(t, m.IncRef(h))
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
		return h
	}
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	pin(x)
	y, err := m.MkVar(1, b)
	patternMust(t, err)
	pin(y)
	declare := func(name uint32) ast.ASTHandle {
		t.Helper()
		h, e := m.MkFuncDecl(symbol.Numerical(name), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		return pin(h)
	}
	app := func(decl, arg ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(decl, []ast.ASTHandle{arg})
		patternMust(t, e)
		return pin(h)
	}
	f, g := declare(711), declare(712)
	general := app(f, x)
	specific := app(f, app(g, x))
	excluded := app(f, y)
	otherVars := app(g, y)
	base, err := m.GetRefCount(general)
	patternMust(t, err)
	filter, err := newPatternCandidates(m, 2, []ast.ASTHandle{excluded}, true)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, filter.close()) })
	patternMust(t, filter.addCandidate(general, []uint32{0}, 1))
	patternMust(t, filter.addCandidate(general, []uint32{0}, 99)) // Duplicate keeps first metadata and one pin.
	patternMust(t, filter.addCandidate(specific, []uint32{0}, 2))
	patternMust(t, filter.addCandidate(excluded, []uint32{1}, 1))
	patternMust(t, filter.addCandidate(otherVars, []uint32{1}, 1))
	if len(filter.candidates) != 3 || filter.info[general].size != 1 {
		t.Fatal("candidate insertion order, exclusion or duplicate metadata", filter.candidates, filter.info)
	}
	if refs, e := m.GetRefCount(general); e != nil || refs != base+1 {
		t.Fatal("candidate vector must own one AST reference", refs, base, e)
	}
	got, err := filter.filterLooping()
	patternMust(t, err)
	if len(got) != 2 || got[0] != specific || got[1] != otherVars {
		t.Fatal("strict same-free-variable filtering and source order", got)
	}
	if _, found := filter.info[general]; found || len(filter.candidates) != 3 {
		t.Fatal("source erases metadata but retains vector ownership", filter.info, filter.candidates)
	}
	patternMust(t, filter.close())
	if refs, e := m.GetRefCount(general); e != nil || refs != base {
		t.Fatal("candidate vector close must release its reference", refs, base, e)
	}

	pass, err := newPatternCandidates(m, 2, nil, false)
	patternMust(t, err)
	patternMust(t, pass.addCandidate(general, []uint32{0}, 1))
	patternMust(t, pass.addCandidate(specific, []uint32{0}, 2))
	got, err = pass.filterLooping()
	patternMust(t, err)
	if len(got) != 2 || got[0] != general || got[1] != specific {
		t.Fatal("disabled loop filtering must preserve candidate order", got)
	}
	patternMust(t, pass.close())
}
